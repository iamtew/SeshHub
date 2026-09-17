package skater

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func TestFrameCSS(t *testing.T) {
	if got := string(FrameCSS(0, 0, 0, 0, "off", "", 0, 0)); got != "border-radius:0% 0% 0% 0%" {
		t.Fatalf("square %q", got)
	}
	if got := string(FrameCSS(50, 12, 0, 99, "default", "", 0, 0)); got != "border-radius:50% 12% 0% 50%;--avw:3px;--avblur:0px" {
		t.Fatalf("frame %q", got)
	}
	if got := string(FrameCSS(50, 50, 50, 50, "custom", "#FF00AA", 8, 4)); got != "border-radius:50% 50% 50% 50%;--avw:8px;--avblur:4px;--avb:#ff00aa" {
		t.Fatalf("custom %q", got)
	}
	if FrameClass("pulse") != "avb-pulse" || FrameClass("chroma") != "avb-chroma" || FrameClass("nope") != "" {
		t.Fatal("class")
	}
	st, col, width := NormalizeBorder("nope", "#gggggg", 1)
	if st != "off" || col != "" || width != 0 {
		t.Fatalf("junk %s %s %d", st, col, width)
	}
	st, col, width = NormalizeBorder("", "", 1)
	if st != "default" || width != 1 {
		t.Fatalf("legacy %s %d", st, width)
	}
	st, col, width = NormalizeBorder("custom", "nope", 0)
	if st != "custom" || col != "#e5f20d" || width != 3 {
		t.Fatalf("hex %s %s %d", st, col, width)
	}
}

func TestClampRadius(t *testing.T) {
	if ClampRadius(-1) != 0 || ClampRadius(0) != 0 || ClampRadius(50) != 50 || ClampRadius(99) != 50 {
		t.Fatal("clamp")
	}
	if ClampWidth(0) != 3 || ClampWidth(1) != 1 || ClampWidth(12) != 12 || ClampWidth(99) != 12 {
		t.Fatal("width")
	}
	if ClampBlur(-1) != 0 || ClampBlur(0) != 0 || ClampBlur(16) != 16 || ClampBlur(99) != 16 {
		t.Fatal("blur")
	}
}

func TestSavePhotoSquareJPEG(t *testing.T) {
	t.Chdir(t.TempDir())

	src := image.NewRGBA(image.Rect(0, 0, 80, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			src.Set(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	id := "0123456789abcdef0123456789abcdef"
	url, err := SavePhoto(id, bytes.NewReader(buf.Bytes()))
	if err != nil || url != PhotoURL(id) {
		t.Fatalf("save %q %v", url, err)
	}
	f, err := os.Open(PhotoPath(id))
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(f)
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != b.Dy() || b.Dx() != 40 {
		t.Fatalf("size %dx%d", b.Dx(), b.Dy())
	}
	if _, err := SavePhoto(id, bytes.NewReader([]byte("nope"))); err == nil {
		t.Fatal("garbage")
	}
	if _, err := SavePhoto(id, bytes.NewReader(bytes.Repeat([]byte("x"), PhotoMax+1))); err == nil {
		t.Fatal("oversize")
	}
	RemovePhoto(id)
	if _, err := os.Stat(PhotoPath(id)); !os.IsNotExist(err) {
		t.Fatal("left file")
	}
}

func TestPhotoFile(t *testing.T) {
	if _, ok := PhotoFile("../etc/passwd.jpg"); ok {
		t.Fatal("dotdot")
	}
	if _, ok := PhotoFile("not-hex.jpg"); ok {
		t.Fatal("hex")
	}
	p, ok := PhotoFile("0123456789abcdef0123456789abcdef.jpg")
	if !ok || p != PhotoPath("0123456789abcdef0123456789abcdef") {
		t.Fatalf("got %q %v", p, ok)
	}
}
