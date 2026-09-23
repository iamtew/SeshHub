package forum

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"seshhub/internal/skater"
)

const PhotoMax = 3
const photoEdge = 1600

type FileIn struct {
	R    io.Reader
	Name string
}

func PhotoURL(id, ext string) string {
	return "/media/forum/" + id + "." + ext
}

func PhotoPath(id, ext string) string {
	return filepath.Join("data", "forum", id+"."+ext)
}

var attachExt = map[string]string{
	"jpg":  "image/jpeg",
	"gif":  "image/gif",
	"webp": "image/webp",
	"mp3":  "audio/mpeg",
	"wav":  "audio/wav",
	"ogg":  "audio/ogg",
	"ogv":  "video/ogg",
	"flac": "audio/flac",
	"m4a":  "audio/mp4",
	"mp4":  "video/mp4",
	"webm": "video/webm",
	"pdf":  "application/pdf",
}

func MimeFor(file string) string {
	i := strings.LastIndexByte(file, '.')
	if i < 0 {
		return ""
	}
	return attachExt[file[i+1:]]
}

func PhotoFile(file string) (string, bool) {
	i := strings.LastIndexByte(file, '.')
	if i <= 0 {
		return "", false
	}
	id, ext := file[:i], file[i+1:]
	if _, ok := attachExt[ext]; !ok {
		return "", false
	}
	if !skater.ValidPhotoID(id) {
		return "", false
	}
	return PhotoPath(id, ext), true
}

func RemovePhoto(file string) {
	path, ok := PhotoFile(file)
	if !ok {
		return
	}
	_ = os.Remove(path)
}

func RemovePhotos(files []string) {
	for _, f := range files {
		RemovePhoto(f)
	}
}

func AttachName(raw, id, ext string) string {
	fallback := id + "." + ext
	base := path.Base(strings.ReplaceAll(raw, `\`, "/"))
	base = strings.Trim(base, " .")
	base = strings.ReplaceAll(base, "\x00", "")
	if i := strings.LastIndexByte(base, '.'); i > 0 {
		base = base[:i]
	}
	if base == "" || base == "." || base == ".." {
		return fallback
	}
	if len(base) > 180 {
		base = base[:180]
	}
	return base + "." + ext
}

func PhotoDownloadName(db *sql.DB, file string) string {
	i := strings.LastIndexByte(file, '.')
	if i <= 0 {
		return file
	}
	id, ext := file[:i], file[i+1:]
	var name string
	_ = db.QueryRow(`SELECT name FROM forum_post_photos WHERE id=?`, id).Scan(&name)
	if name == "" {
		return id + "." + ext
	}
	return name
}

func (p Photo) IsImage() bool { return p.Kind == "image" }
func (p Photo) IsAudio() bool { return p.Kind == "audio" }
func (p Photo) IsVideo() bool { return p.Kind == "video" }
func (p Photo) IsPDF() bool   { return p.Kind == "pdf" }

func Sniff(raw []byte) (kind, ext, mime string, err error) {
	if len(raw) < 12 {
		return "", "", "", fmt.Errorf("unsupported file")
	}
	switch {
	case raw[0] == 0xff && raw[1] == 0xd8 && raw[2] == 0xff:
		return "image", "jpg", "image/jpeg", nil
	case bytes.HasPrefix(raw, []byte{0x89, 'P', 'N', 'G'}):
		return "image", "png", "image/png", nil
	case bytes.HasPrefix(raw, []byte("GIF87a")) || bytes.HasPrefix(raw, []byte("GIF89a")):
		return "image", "gif", "image/gif", nil
	case bytes.HasPrefix(raw, []byte("RIFF")) && bytes.Equal(raw[8:12], []byte("WEBP")):
		return "image", "webp", "image/webp", nil
	case bytes.HasPrefix(raw, []byte("RIFF")) && bytes.Equal(raw[8:12], []byte("WAVE")):
		return "audio", "wav", "audio/wav", nil
	case bytes.HasPrefix(raw, []byte("%PDF")):
		return "pdf", "pdf", "application/pdf", nil
	case bytes.HasPrefix(raw, []byte("OggS")):
		if bytes.Contains(raw[:min(len(raw), 4096)], []byte("theora")) {
			return "video", "ogv", "video/ogg", nil
		}
		return "audio", "ogg", "audio/ogg", nil
	case bytes.HasPrefix(raw, []byte("fLaC")):
		return "audio", "flac", "audio/flac", nil
	case bytes.HasPrefix(raw, []byte("ID3")) || (raw[0] == 0xff && raw[1]&0xe0 == 0xe0):
		return "audio", "mp3", "audio/mpeg", nil
	case bytes.Equal(raw[4:8], []byte("ftyp")):
		return sniffISO(raw)
	case raw[0] == 0x1a && raw[1] == 0x45 && raw[2] == 0xdf && raw[3] == 0xa3:
		return "video", "webm", "video/webm", nil
	default:
		return "", "", "", fmt.Errorf("unsupported file")
	}
}

func sniffISO(raw []byte) (kind, ext, mime string, err error) {
	n := int(raw[0])<<24 | int(raw[1])<<16 | int(raw[2])<<8 | int(raw[3])
	if n < 16 || n > len(raw) {
		n = min(len(raw), 256)
	}
	if bytes.Contains(raw[8:n], []byte("M4A ")) || bytes.Contains(raw[8:n], []byte("M4B ")) {
		return "audio", "m4a", "audio/mp4", nil
	}
	return sniffFtyp(raw[8:12])
}

func sniffFtyp(brand []byte) (kind, ext, mime string, err error) {
	switch string(brand) {
	case "M4A ", "M4B ":
		return "audio", "m4a", "audio/mp4", nil
	case "isom", "iso2", "mp41", "mp42", "avc1", "dash", "MSNV", "M4V ":
		return "video", "mp4", "video/mp4", nil
	default:
		return "", "", "", fmt.Errorf("unsupported file")
	}
}

func attachPhotos(db *sql.DB, posts []Post) error {
	if len(posts) == 0 {
		return nil
	}
	ids := make([]string, len(posts))
	idx := make(map[string]int, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
		idx[p.ID] = i
	}
	photos, err := photosByPosts(db, ids)
	if err != nil {
		return err
	}
	for id, list := range photos {
		i := idx[id]
		posts[i].Photos = list
	}
	return nil
}

func photosByPosts(db *sql.DB, postIDs []string) (map[string][]Photo, error) {
	out := map[string][]Photo{}
	if len(postIDs) == 0 {
		return out, nil
	}
	args := make([]any, len(postIDs))
	ph := make([]string, len(postIDs))
	for i, id := range postIDs {
		args[i] = id
		ph[i] = "?"
	}
	rows, err := db.Query(`SELECT id, post_id, kind, ext, mime, name FROM forum_post_photos WHERE post_id IN (`+strings.Join(ph, ",")+`) ORDER BY pos, id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, postID, kind, ext, mime, name string
		if err := rows.Scan(&id, &postID, &kind, &ext, &mime, &name); err != nil {
			return nil, err
		}
		out[postID] = append(out[postID], Photo{ID: id, URL: PhotoURL(id, ext), Kind: kind, Ext: ext, MIME: mime, Name: name})
	}
	return out, rows.Err()
}

func addPhotos(tx *sql.Tx, postID string, start int, files []FileIn) error {
	for i, f := range files {
		raw, err := io.ReadAll(io.LimitReader(f.R, int64(skater.PhotoMax)+1))
		if err != nil {
			return err
		}
		if len(raw) == 0 {
			return fmt.Errorf("empty")
		}
		if len(raw) > skater.PhotoMax {
			return fmt.Errorf("too large")
		}
		kind, ext, mime, err := Sniff(raw)
		if err != nil {
			return err
		}
		id := newID()
		if kind == "image" && (ext == "jpg" || ext == "png") {
			ext, mime = "jpg", "image/jpeg"
			if err := skater.SaveFittedJPEG(PhotoPath(id, ext), bytes.NewReader(raw), skater.PhotoMax, photoEdge); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(PhotoPath(id, ext)), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(PhotoPath(id, ext), raw, 0o644); err != nil {
				return err
			}
		}
		name := AttachName(f.Name, id, ext)
		if _, err := tx.Exec(`INSERT INTO forum_post_photos (id, post_id, pos, kind, ext, mime, name) VALUES (?,?,?,?,?,?,?)`, id, postID, start+i, kind, ext, mime, name); err != nil {
			RemovePhoto(id + "." + ext)
			return err
		}
	}
	return nil
}

func dropPhotos(tx *sql.Tx, postID string, ids []string) ([]string, error) {
	var gone []string
	for _, id := range ids {
		if !skater.ValidPhotoID(id) {
			continue
		}
		var ext string
		if err := tx.QueryRow(`SELECT ext FROM forum_post_photos WHERE id=? AND post_id=?`, id, postID).Scan(&ext); err != nil {
			continue
		}
		res, err := tx.Exec(`DELETE FROM forum_post_photos WHERE id=? AND post_id=?`, id, postID)
		if err != nil {
			return nil, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			gone = append(gone, id+"."+ext)
		}
	}
	return gone, nil
}

func photoIDsIn(tx *sql.Tx, q string, arg any) ([]string, error) {
	rows, err := tx.Query(q, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
