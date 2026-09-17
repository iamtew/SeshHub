package skater

import (
	"database/sql"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const GalleryMax = 10
const galleryEdge = 1600

type Photo struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name,omitempty"`
	Href string `json:"href,omitempty"`
}

func GalleryURL(id string) string {
	return "/media/gallery/" + id + ".jpg"
}

func GalleryPath(id string) string {
	return filepath.Join("data", "gallery", id+".jpg")
}

func GalleryFile(file string) (string, bool) {
	if !strings.HasSuffix(file, ".jpg") {
		return "", false
	}
	id := strings.TrimSuffix(file, ".jpg")
	if !ValidPhotoID(id) {
		return "", false
	}
	return GalleryPath(id), true
}

func RemoveGallery(id string) {
	if !ValidPhotoID(id) {
		return
	}
	_ = os.Remove(GalleryPath(id))
}

func GalleryIDs(db *sql.DB, profileID string) []string {
	rows, err := db.Query(`SELECT id FROM gallery_photos WHERE profile_id=?`, profileID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ids
		}
		ids = append(ids, id)
	}
	return ids
}

func RemoveGalleries(ids []string) {
	for _, id := range ids {
		RemoveGallery(id)
	}
}

func saveGalleryFile(id string, r io.Reader) error {
	if !ValidPhotoID(id) {
		return fmt.Errorf("bad id")
	}
	raw, err := io.ReadAll(io.LimitReader(r, PhotoMax+1))
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return fmt.Errorf("empty")
	}
	if len(raw) > PhotoMax {
		return fmt.Errorf("too large")
	}
	img, err := decodePhoto(raw)
	if err != nil {
		return err
	}
	return writeJPEG(GalleryPath(id), fitJPEG(img, galleryEdge))
}

func AddGallery(db *sql.DB, profileID string, r io.Reader) (Photo, error) {
	id := newID()
	if err := saveGalleryFile(id, r); err != nil {
		return Photo{}, err
	}
	res, err := db.Exec(`INSERT INTO gallery_photos (id, profile_id, pos) SELECT ?, ?, IFNULL((SELECT MIN(pos) FROM gallery_photos WHERE profile_id=?), 0) - 1 WHERE (SELECT COUNT(*) FROM gallery_photos WHERE profile_id=?) < ?`,
		id, profileID, profileID, profileID, GalleryMax)
	if err != nil {
		RemoveGallery(id)
		return Photo{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		RemoveGallery(id)
		return Photo{}, fmt.Errorf("gallery full")
	}
	return Photo{ID: id, URL: GalleryURL(id)}, nil
}

func DeleteGallery(db *sql.DB, profileID, id string) error {
	res, err := db.Exec(`DELETE FROM gallery_photos WHERE id=? AND profile_id=?`, id, profileID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	RemoveGallery(id)
	return nil
}

func ReorderGallery(db *sql.DB, profileID string, ids []string) error {
	if len(ids) == 0 || len(ids) > GalleryMax {
		return fmt.Errorf("bad order")
	}
	have, err := ListByProfile(db, profileID)
	if err != nil {
		return err
	}
	if len(have) != len(ids) {
		return fmt.Errorf("bad order")
	}
	set := make(map[string]bool, len(have))
	for _, p := range have {
		set[p.ID] = true
	}
	for _, id := range ids {
		if !ValidPhotoID(id) || !set[id] {
			return fmt.Errorf("bad order")
		}
		delete(set, id)
	}
	if len(set) != 0 {
		return fmt.Errorf("bad order")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.Exec(`UPDATE gallery_photos SET pos=? WHERE id=? AND profile_id=?`, i, id, profileID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func ListByProfile(db *sql.DB, profileID string) ([]Photo, error) {
	rows, err := db.Query(`SELECT id FROM gallery_photos WHERE profile_id=? ORDER BY pos ASC, created_at DESC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Photo
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, Photo{ID: id, URL: GalleryURL(id)})
	}
	return out, rows.Err()
}

func ListPublicGallery(db *sql.DB) ([]Photo, error) {
	rows, err := db.Query(`
		SELECT g.id, p.slug, p.skater_name, IFNULL(p.real_name,''), IFNULL(u.role,'')
		FROM gallery_photos g
		JOIN skater_profiles p ON p.id = g.profile_id
		LEFT JOIN users u ON u.id = p.user_id
		WHERE p.user_id IS NOT NULL AND p.user_id != ''
		ORDER BY g.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Photo
	for rows.Next() {
		var id, slug, skaterName, realName, role string
		if err := rows.Scan(&id, &slug, &skaterName, &realName, &role); err != nil {
			return nil, err
		}
		name := strings.TrimSpace(realName)
		if name == "" {
			name = skaterName
		}
		out = append(out, Photo{
			ID: id, URL: GalleryURL(id), Name: name,
			Href: RosterPath(role) + "/" + slug,
		})
	}
	return out, rows.Err()
}

func fitJPEG(src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 || (w <= maxEdge && h <= maxEdge) {
		return src
	}
	nw, nh := w, h
	if w >= h {
		nw = maxEdge
		nh = h * maxEdge / w
	} else {
		nh = maxEdge
		nw = w * maxEdge / h
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := b.Min.Y + y*h/nh
		for x := 0; x < nw; x++ {
			dst.Set(x, y, src.At(b.Min.X+x*w/nw, sy))
		}
	}
	return dst
}
