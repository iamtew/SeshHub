package skater

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"seshhub/internal/db"
)

func tinyPNG() []byte {
	src := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			src.Set(x, y, color.RGBA{B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, src)
	return buf.Bytes()
}

func TestGalleryCapAndDelete(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if _, err := sqldb.Exec(`INSERT INTO skater_profiles (id, slug, skater_name) VALUES ('p-gal', 'gal', 'Gal')`); err != nil {
		t.Fatal(err)
	}
	raw := tinyPNG()
	var last Photo
	for i := 0; i < GalleryMax; i++ {
		last, err = AddGallery(sqldb, "p-gal", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	if _, err := AddGallery(sqldb, "p-gal", bytes.NewReader(raw)); err == nil {
		t.Fatal("11th")
	}
	list, err := ListByProfile(sqldb, "p-gal")
	if err != nil || len(list) != GalleryMax {
		t.Fatalf("list %d %v", len(list), err)
	}
	if _, err := os.Stat(GalleryPath(last.ID)); err != nil {
		t.Fatal(err)
	}
	if _, ok := GalleryFile("../x.jpg"); ok {
		t.Fatal("dotdot")
	}
	if err := Delete(sqldb, "p-gal"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(GalleryPath(last.ID)); !os.IsNotExist(err) {
		t.Fatal("left file")
	}
	var n int
	_ = sqldb.QueryRow(`SELECT COUNT(*) FROM gallery_photos WHERE profile_id='p-gal'`).Scan(&n)
	if n != 0 {
		t.Fatal("rows")
	}
}

func TestDisplaceOldest(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if _, err := sqldb.Exec(`INSERT INTO skater_profiles (id, slug, skater_name) VALUES ('p-disp', 'disp', 'Disp')`); err != nil {
		t.Fatal(err)
	}
	raw := tinyPNG()
	first, err := AddGallery(sqldb, "p-disp", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < GalleryMax; i++ {
		if _, err := AddGallery(sqldb, "p-disp", bytes.NewReader(raw)); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	extra, err := AddGalleryDisplace(sqldb, "p-disp", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	list, err := ListByProfile(sqldb, "p-disp")
	if err != nil || len(list) != GalleryMax {
		t.Fatalf("list %d %v", len(list), err)
	}
	if _, err := os.Stat(GalleryPath(first.ID)); !os.IsNotExist(err) {
		t.Fatal("oldest file")
	}
	if _, err := os.Stat(GalleryPath(extra.ID)); err != nil {
		t.Fatal(err)
	}
	for _, p := range list {
		if p.ID == first.ID {
			t.Fatal("oldest row")
		}
	}
}

func TestReorderGallery(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if _, err := sqldb.Exec(`INSERT INTO skater_profiles (id, slug, skater_name) VALUES ('p-ord', 'ord', 'Ord')`); err != nil {
		t.Fatal(err)
	}
	raw := tinyPNG()
	a, err := AddGallery(sqldb, "p-ord", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	b, err := AddGallery(sqldb, "p-ord", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	list, err := ListByProfile(sqldb, "p-ord")
	if err != nil || len(list) != 2 || list[0].ID != b.ID {
		t.Fatalf("newest first %+v %v", list, err)
	}
	if err := ReorderGallery(sqldb, "p-ord", []string{a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	list, err = ListByProfile(sqldb, "p-ord")
	if err != nil || len(list) != 2 || list[0].ID != a.ID || list[1].ID != b.ID {
		t.Fatalf("swapped %+v %v", list, err)
	}
	if err := ReorderGallery(sqldb, "p-ord", []string{a.ID}); err == nil {
		t.Fatal("partial")
	}
}
