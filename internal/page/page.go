package page

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"seshhub/internal/article"
)

var reserved = map[string]bool{
	"team": true, "skaters": true, "news": true, "videos": true, "auth": true,
	"access": true, "admin": true, "dashboard": true, "static": true, "healthz": true, "spot": true, "episodes": true,
}

type Page struct {
	ID, Slug, Title, ContentRaw, ContentHTML, CSS string
	Published                                     bool
}

func Reserved(slug string) bool {
	slug = strings.Trim(slug, "/")
	if slug == "about/privacy" || slug == "about/tos" {
		return true
	}
	first, _, _ := strings.Cut(slug, "/")
	return reserved[first]
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func List(db *sql.DB) ([]Page, error) {
	rows, err := db.Query(`SELECT id, slug, title, content_raw, content_html, IFNULL(custom_css,''), is_published FROM pages ORDER BY slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Page
	for rows.Next() {
		var p Page
		var pub int
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.ContentRaw, &p.ContentHTML, &p.CSS, &pub); err != nil {
			return nil, err
		}
		p.Published = pub != 0
		out = append(out, p)
	}
	return out, rows.Err()
}

func Get(db *sql.DB, by, val string) (Page, error) {
	col := "id"
	if by == "slug" {
		col = "slug"
	}
	var p Page
	var pub int
	err := db.QueryRow(`SELECT id, slug, title, content_raw, content_html, IFNULL(custom_css,''), is_published FROM pages WHERE `+col+` = ?`, val).
		Scan(&p.ID, &p.Slug, &p.Title, &p.ContentRaw, &p.ContentHTML, &p.CSS, &pub)
	p.Published = pub != 0
	return p, err
}

func Save(db *sql.DB, p Page) (Page, error) {
	p.Title = strings.TrimSpace(p.Title)
	if p.Title == "" || strings.TrimSpace(p.ContentRaw) == "" {
		return p, fmt.Errorf("title and body required")
	}
	base := slugify(p.Slug)
	if base == "" {
		base = slugify(p.Title)
	}
	if base == "" {
		return p, fmt.Errorf("slug required")
	}
	if Reserved(base) {
		return p, fmt.Errorf("slug reserved")
	}
	slug, err := unique(db, base, p.ID)
	if err != nil {
		return p, err
	}
	p.Slug = slug
	p.ContentHTML = article.Render(p.ContentRaw)
	pub := 0
	if p.Published {
		pub = 1
	}
	if p.ID == "" {
		p.ID = newID()
		_, err = db.Exec(`INSERT INTO pages (id, slug, title, content_raw, content_html, custom_css, is_published) VALUES (?,?,?,?,?,?,?)`,
			p.ID, p.Slug, p.Title, p.ContentRaw, p.ContentHTML, nullEmpty(p.CSS), pub)
	} else {
		_, err = db.Exec(`UPDATE pages SET slug=?, title=?, content_raw=?, content_html=?, custom_css=?, is_published=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			p.Slug, p.Title, p.ContentRaw, p.ContentHTML, nullEmpty(p.CSS), pub, p.ID)
	}
	return p, err
}

func Delete(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM pages WHERE id = ?`, id)
	return err
}

func unique(db *sql.DB, base, exceptID string) (string, error) {
	slug := base
	for n := 2; n < 100; n++ {
		var existing string
		err := db.QueryRow(`SELECT id FROM pages WHERE slug = ?`, slug).Scan(&existing)
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

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var parts []string
	for _, seg := range strings.Split(s, "/") {
		p := slugSeg(seg)
		if p == "" {
			continue
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, "/")
}

func slugSeg(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func nullEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
