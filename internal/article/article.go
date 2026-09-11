package article

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"seshhub/internal/skater"
)

type Article struct {
	ID          string
	Slug        string
	Title       string
	Excerpt     string
	ContentRaw  string
	ContentHTML string
	ImageURL    string
	AuthorID    string
	AuthorName  string
	Status      string
	Tags        string
	PublishedAt string
}

func CanEdit(role, userID string, a Article) bool {
	if role == "admin" {
		return true
	}
	return role == "skater" && userID == a.AuthorID && a.Status == "draft"
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func uniqueSlug(db *sql.DB, base, exceptID string) (string, error) {
	slug := base
	for n := 2; n < 100; n++ {
		var existing string
		err := db.QueryRow(`SELECT id FROM articles WHERE slug = ?`, slug).Scan(&existing)
		if err == sql.ErrNoRows || (err == nil && existing == exceptID) {
			return slug, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
		slug = fmt.Sprintf("%s-%d", base, n)
	}
	return "", fmt.Errorf("slug taken")
}

const cols = `a.id, a.slug, a.title, IFNULL(a.excerpt,''), a.content_raw, a.content_html, IFNULL(a.featured_image_url,''), a.author_id, u.display_name, a.status, IFNULL(a.tags,''), IFNULL(a.published_at,'')`

func scan(s func(dest ...any) error) (Article, error) {
	var a Article
	err := s(&a.ID, &a.Slug, &a.Title, &a.Excerpt, &a.ContentRaw, &a.ContentHTML, &a.ImageURL, &a.AuthorID, &a.AuthorName, &a.Status, &a.Tags, &a.PublishedAt)
	return a, err
}

func ListPublished(db *sql.DB) ([]Article, error) {
	return query(db, `SELECT `+cols+` FROM articles a JOIN users u ON u.id = a.author_id WHERE a.status = 'published' ORDER BY a.published_at DESC`)
}

func ListAll(db *sql.DB) ([]Article, error) {
	return query(db, `SELECT `+cols+` FROM articles a JOIN users u ON u.id = a.author_id ORDER BY a.updated_at DESC`)
}

func ListByAuthor(db *sql.DB, authorID string) ([]Article, error) {
	return query(db, `SELECT `+cols+` FROM articles a JOIN users u ON u.id = a.author_id WHERE a.author_id = ? ORDER BY a.updated_at DESC`, authorID)
}

func query(db *sql.DB, q string, args ...any) ([]Article, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Article
	for rows.Next() {
		a, err := scan(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func Get(db *sql.DB, by, val string) (Article, error) {
	col := "a.id"
	if by == "slug" {
		col = "a.slug"
	}
	row := db.QueryRow(`SELECT `+cols+` FROM articles a JOIN users u ON u.id = a.author_id WHERE `+col+` = ?`, val)
	return scan(row.Scan)
}

func Save(db *sql.DB, a Article, asAdmin bool) (Article, error) {
	a.Title = strings.TrimSpace(a.Title)
	if a.Title == "" || strings.TrimSpace(a.ContentRaw) == "" {
		return a, fmt.Errorf("title and body required")
	}
	if !asAdmin {
		a.Status = "draft"
	}
	switch a.Status {
	case "draft", "published", "archived":
	default:
		a.Status = "draft"
	}
	base := a.Slug
	if base == "" {
		base = skater.Slugify(a.Title)
	} else {
		base = skater.Slugify(base)
	}
	slug, err := uniqueSlug(db, base, a.ID)
	if err != nil {
		return a, err
	}
	a.Slug = slug
	a.ContentHTML = Render(a.ContentRaw)
	a.Excerpt = Excerpt(a.ContentRaw, a.Excerpt)
	var published any
	if a.Status == "published" {
		if a.PublishedAt == "" {
			a.PublishedAt = time.Now().UTC().Format("2006-01-02 15:04:05")
		}
		published = a.PublishedAt
	}
	if a.ID == "" {
		a.ID = newID()
		_, err = db.Exec(`INSERT INTO articles (id, slug, title, excerpt, content_raw, content_html, featured_image_url, author_id, status, tags, published_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			a.ID, a.Slug, a.Title, nullEmpty(a.Excerpt), a.ContentRaw, a.ContentHTML, nullEmpty(a.ImageURL), a.AuthorID, a.Status, nullEmpty(a.Tags), published)
	} else {
		_, err = db.Exec(`UPDATE articles SET slug=?, title=?, excerpt=?, content_raw=?, content_html=?, featured_image_url=?, status=?, tags=?, published_at=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			a.Slug, a.Title, nullEmpty(a.Excerpt), a.ContentRaw, a.ContentHTML, nullEmpty(a.ImageURL), a.Status, nullEmpty(a.Tags), published, a.ID)
	}
	return a, err
}

func Delete(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM articles WHERE id = ?`, id)
	return err
}

func nullEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
