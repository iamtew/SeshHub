package skater

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const PhotoMax = 15 << 20
const photoEdge = 1024

func PhotoURL(id string) string {
	return "/media/avatars/" + id + ".jpg"
}

func Bust(url, path string) string {
	if url == "" || strings.Contains(url, "?") {
		return url
	}
	st, err := os.Stat(path)
	if err != nil {
		return url
	}
	return url + "?t=" + strconv.FormatInt(st.ModTime().Unix(), 10)
}

func PhotoPath(id string) string {
	return filepath.Join("data", "avatars", id+".jpg")
}

func ValidPhotoID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' {
			continue
		}
		return false
	}
	return true
}

func PhotoFile(file string) (string, bool) {
	if !strings.HasSuffix(file, ".jpg") {
		return "", false
	}
	id := strings.TrimSuffix(file, ".jpg")
	if !ValidPhotoID(id) {
		return "", false
	}
	return PhotoPath(id), true
}

func RemovePhoto(id string) {
	if !ValidPhotoID(id) {
		return
	}
	_ = os.Remove(PhotoPath(id))
}

func SavePhoto(id string, r io.Reader) (string, error) {
	if !ValidPhotoID(id) {
		return "", fmt.Errorf("bad id")
	}
	raw, err := io.ReadAll(io.LimitReader(r, PhotoMax+1))
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("empty")
	}
	if len(raw) > PhotoMax {
		return "", fmt.Errorf("too large")
	}
	img, err := decodePhoto(raw)
	if err != nil {
		return "", err
	}
	if err := writeJPEG(PhotoPath(id), squareJPEG(img)); err != nil {
		return "", err
	}
	return PhotoURL(id), nil
}

func writeJPEG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 88})
}

func decodePhoto(raw []byte) (image.Image, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("not an image")
	}
	if raw[0] != 0xff && !bytes.HasPrefix(raw, []byte{0x89, 'P', 'N', 'G'}) {
		return nil, fmt.Errorf("not jpeg or png")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("not an image")
	}
	return img, nil
}

func squareJPEG(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	side := w
	if h < w {
		side = h
	}
	if side < 1 {
		side = 1
	}
	x0 := b.Min.X + (w-side)/2
	y0 := b.Min.Y + (h-side)/2
	cropped := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(cropped, cropped.Bounds(), src, image.Pt(x0, y0), draw.Src)
	if side <= photoEdge {
		return cropped
	}
	dst := image.NewRGBA(image.Rect(0, 0, photoEdge, photoEdge))
	for y := 0; y < photoEdge; y++ {
		sy := y * side / photoEdge
		for x := 0; x < photoEdge; x++ {
			dst.Set(x, y, cropped.At(x*side/photoEdge, sy))
		}
	}
	return dst
}

func ClampRadius(n int) int {
	if n < 0 {
		return 0
	}
	if n > 50 {
		return 50
	}
	return n
}

func ClampWidth(n int) int {
	if n <= 0 {
		return 3
	}
	if n > 12 {
		return 12
	}
	return n
}

func ClampBlur(n int) int {
	if n < 0 {
		return 0
	}
	if n > 16 {
		return 16
	}
	return n
}
