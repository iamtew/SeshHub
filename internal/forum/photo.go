package forum

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"

	"seshhub/internal/skater"
)

const PhotoMax = 3
const photoEdge = 1600

type FileIn struct {
	R io.Reader
}

func PhotoURL(id string) string {
	return "/media/forum/" + id + ".jpg"
}

func PhotoPath(id string) string {
	return filepath.Join("data", "forum", id+".jpg")
}

func PhotoFile(file string) (string, bool) {
	if !strings.HasSuffix(file, ".jpg") {
		return "", false
	}
	id := strings.TrimSuffix(file, ".jpg")
	if !skater.ValidPhotoID(id) {
		return "", false
	}
	return PhotoPath(id), true
}

func RemovePhoto(id string) {
	if !skater.ValidPhotoID(id) {
		return
	}
	_ = os.Remove(PhotoPath(id))
}

func RemovePhotos(ids []string) {
	for _, id := range ids {
		RemovePhoto(id)
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
	rows, err := db.Query(`SELECT id, post_id FROM forum_post_photos WHERE post_id IN (`+strings.Join(ph, ",")+`) ORDER BY pos, id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, postID string
		if err := rows.Scan(&id, &postID); err != nil {
			return nil, err
		}
		out[postID] = append(out[postID], Photo{ID: id, URL: PhotoURL(id)})
	}
	return out, rows.Err()
}

func photoURLsByPosts(db *sql.DB, postIDs []string) (map[string][]string, error) {
	photos, err := photosByPosts(db, postIDs)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for id, list := range photos {
		urls := make([]string, len(list))
		for i, p := range list {
			urls[i] = p.URL
		}
		out[id] = urls
	}
	return out, nil
}

func addPhotos(tx *sql.Tx, postID string, start int, files []FileIn) error {
	for i, f := range files {
		id := newID()
		if err := skater.SaveFittedJPEG(PhotoPath(id), f.R, skater.PhotoMax, photoEdge); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO forum_post_photos (id, post_id, pos) VALUES (?,?,?)`, id, postID, start+i); err != nil {
			RemovePhoto(id)
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
		res, err := tx.Exec(`DELETE FROM forum_post_photos WHERE id=? AND post_id=?`, id, postID)
		if err != nil {
			return nil, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			gone = append(gone, id)
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
