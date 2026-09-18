package forum

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"strings"

	"seshhub/internal/article"
	"seshhub/internal/skater"
)

type Section struct {
	ID          string
	Name        string
	Slug        string
	Description string
	SortOrder   int
	Active      bool
	ThreadCount int
	LastPostAt  string
}

type Thread struct {
	ID         string
	SectionID  string
	UserID     string
	Title      string
	Slug       string
	Locked     bool
	Sticky     bool
	LastPostAt string
	CreatedAt  string
	AuthorName string
	AuthorURL  string
	ReplyCount int
}

type Post struct {
	ID                string
	ThreadID          string
	UserID            string
	BodyRaw           string
	BodyHTML          string
	First             bool
	CreatedAt         string
	UpdatedAt         string
	AuthorName        string
	AuthorURL         string
	AuthorAvatar      string
	AvatarR1          int
	AvatarR2          int
	AvatarR3          int
	AvatarR4          int
	AvatarBorder      int
	AvatarBorderBlur  int
	AvatarBorderStyle string
	AvatarBorderColor string
}

func (p Post) AvatarStyle() template.CSS {
	st, col, w := skater.NormalizeBorder(p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorder)
	return skater.FrameCSS(p.AvatarR1, p.AvatarR2, p.AvatarR3, p.AvatarR4, st, col, w, p.AvatarBorderBlur)
}

func (p Post) AvatarClass() string {
	st, _, _ := skater.NormalizeBorder(p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorder)
	return skater.FrameClass(st)
}

type OwnPost struct {
	ThreadTitle string `json:"thread_title"`
	BodyRaw     string `json:"body_raw"`
	CreatedAt   string `json:"created_at"`
	First       bool   `json:"is_first_post"`
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func authorURL(role, slug string) string {
	if slug == "" {
		return ""
	}
	switch role {
	case "skater", "admin", "friend":
		return skater.RosterPath(role) + "/" + slug
	}
	return ""
}

func UnreadCount(db *sql.DB, userID string) (int, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM forum_posts p
		JOIN forum_threads t ON t.id = p.thread_id
		JOIN forum_sections s ON s.id = t.section_id AND s.is_active = 1
		LEFT JOIN forum_thread_reads r ON r.thread_id = p.thread_id AND r.user_id = ?
		WHERE p.user_id != ? AND (r.last_read_at IS NULL OR p.created_at > r.last_read_at)`, userID, userID).Scan(&n)
	return n, err
}

func MarkRead(db *sql.DB, userID, threadID string) error {
	_, err := db.Exec(`
		INSERT INTO forum_thread_reads (user_id, thread_id, last_read_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, thread_id) DO UPDATE SET last_read_at = CURRENT_TIMESTAMP`, userID, threadID)
	return err
}

func ListSections(db *sql.DB, activeOnly bool) ([]Section, error) {
	q := `SELECT s.id, s.name, s.slug, s.description, s.sort_order, s.is_active,
		(SELECT COUNT(*) FROM forum_threads t WHERE t.section_id = s.id),
		IFNULL((SELECT MAX(t.last_post_at) FROM forum_threads t WHERE t.section_id = s.id),'')
		FROM forum_sections s`
	if activeOnly {
		q += ` WHERE s.is_active = 1`
	}
	q += ` ORDER BY s.sort_order, s.name`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Section
	for rows.Next() {
		var s Section
		var active int
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.Description, &s.SortOrder, &active, &s.ThreadCount, &s.LastPostAt); err != nil {
			return nil, err
		}
		s.Active = active != 0
		out = append(out, s)
	}
	return out, rows.Err()
}

func GetSection(db *sql.DB, slug string) (Section, error) {
	var s Section
	var active int
	err := db.QueryRow(`SELECT id, name, slug, description, sort_order, is_active FROM forum_sections WHERE slug = ?`, slug).
		Scan(&s.ID, &s.Name, &s.Slug, &s.Description, &s.SortOrder, &active)
	s.Active = active != 0
	return s, err
}

func GetSectionByID(db *sql.DB, id string) (Section, error) {
	var s Section
	var active int
	err := db.QueryRow(`SELECT id, name, slug, description, sort_order, is_active FROM forum_sections WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Slug, &s.Description, &s.SortOrder, &active)
	s.Active = active != 0
	return s, err
}

const threadCols = `t.id, t.section_id, t.user_id, t.title, t.slug, t.is_locked, t.is_sticky, t.last_post_at, t.created_at,
	u.display_name, u.role, IFNULL(sp.slug,''),
	(SELECT COUNT(*) FROM forum_posts p WHERE p.thread_id = t.id)`

func scanThread(s func(dest ...any) error) (Thread, error) {
	var t Thread
	var locked, sticky int
	var role, slug string
	err := s(&t.ID, &t.SectionID, &t.UserID, &t.Title, &t.Slug, &locked, &sticky, &t.LastPostAt, &t.CreatedAt, &t.AuthorName, &role, &slug, &t.ReplyCount)
	t.Locked = locked != 0
	t.Sticky = sticky != 0
	if t.ReplyCount > 0 {
		t.ReplyCount--
	}
	t.AuthorURL = authorURL(role, slug)
	return t, err
}

func ListThreads(db *sql.DB, sectionID string) ([]Thread, error) {
	rows, err := db.Query(`SELECT `+threadCols+`
		FROM forum_threads t
		JOIN users u ON u.id = t.user_id
		LEFT JOIN skater_profiles sp ON sp.user_id = t.user_id
		WHERE t.section_id = ?
		ORDER BY t.is_sticky DESC, t.last_post_at DESC`, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Thread
	for rows.Next() {
		t, err := scanThread(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func GetThread(db *sql.DB, sectionID, slug string) (Thread, error) {
	return scanThread(db.QueryRow(`SELECT `+threadCols+`
		FROM forum_threads t
		JOIN users u ON u.id = t.user_id
		LEFT JOIN skater_profiles sp ON sp.user_id = t.user_id
		WHERE t.section_id = ? AND t.slug = ?`, sectionID, slug).Scan)
}

func ListPosts(db *sql.DB, threadID string) ([]Post, error) {
	rows, err := db.Query(`
		SELECT p.id, p.thread_id, p.user_id, p.body_raw, p.body_html, p.is_first_post, p.created_at, p.updated_at,
			u.display_name, u.role, IFNULL(sp.slug,''), IFNULL(NULLIF(sp.avatar_url,''), IFNULL(u.avatar_url,'')),
			IFNULL(sp.avatar_r1,50), IFNULL(sp.avatar_r2,50), IFNULL(sp.avatar_r3,50), IFNULL(sp.avatar_r4,50),
			IFNULL(sp.avatar_border,0), IFNULL(sp.avatar_border_style,''), IFNULL(sp.avatar_border_color,''), IFNULL(sp.avatar_border_blur,0)
		FROM forum_posts p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN skater_profiles sp ON sp.user_id = p.user_id
		WHERE p.thread_id = ?
		ORDER BY p.created_at`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Post
	for rows.Next() {
		var p Post
		var first int
		var role, slug string
		if err := rows.Scan(&p.ID, &p.ThreadID, &p.UserID, &p.BodyRaw, &p.BodyHTML, &first, &p.CreatedAt, &p.UpdatedAt, &p.AuthorName, &role, &slug, &p.AuthorAvatar,
			&p.AvatarR1, &p.AvatarR2, &p.AvatarR3, &p.AvatarR4, &p.AvatarBorder, &p.AvatarBorderStyle, &p.AvatarBorderColor, &p.AvatarBorderBlur); err != nil {
			return nil, err
		}
		p.First = first != 0
		p.AuthorURL = authorURL(role, slug)
		out = append(out, p)
	}
	return out, rows.Err()
}

func GetPost(db *sql.DB, id string) (Post, error) {
	var p Post
	var first int
	err := db.QueryRow(`SELECT id, thread_id, user_id, body_raw, body_html, is_first_post, created_at, updated_at FROM forum_posts WHERE id = ?`, id).
		Scan(&p.ID, &p.ThreadID, &p.UserID, &p.BodyRaw, &p.BodyHTML, &first, &p.CreatedAt, &p.UpdatedAt)
	p.First = first != 0
	return p, err
}

func uniqueSlug(db *sql.DB, sectionID, title, exceptID string) (string, error) {
	base := skater.Slugify(title)
	if base == "" {
		base = "thread"
	}
	if base == "new" {
		base = "thread"
	}
	slug := base
	for n := 2; n < 100; n++ {
		var existing string
		err := db.QueryRow(`SELECT id FROM forum_threads WHERE section_id = ? AND slug = ?`, sectionID, slug).Scan(&existing)
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

func CreateThread(db *sql.DB, sectionID, userID, title, body string) (Thread, error) {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" || body == "" {
		return Thread{}, fmt.Errorf("title and body required")
	}
	slug, err := uniqueSlug(db, sectionID, title, "")
	if err != nil {
		return Thread{}, err
	}
	tid, pid := newID(), newID()
	html := article.Render(body)
	tx, err := db.Begin()
	if err != nil {
		return Thread{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO forum_threads (id, section_id, user_id, title, slug) VALUES (?,?,?,?,?)`, tid, sectionID, userID, title, slug); err != nil {
		return Thread{}, err
	}
	if _, err := tx.Exec(`INSERT INTO forum_posts (id, thread_id, user_id, body_raw, body_html, is_first_post) VALUES (?,?,?,?,?,1)`, pid, tid, userID, body, html); err != nil {
		return Thread{}, err
	}
	if err := tx.Commit(); err != nil {
		return Thread{}, err
	}
	return GetThread(db, sectionID, slug)
}

func Reply(db *sql.DB, threadID, userID, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("body required")
	}
	html := article.Render(body)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var locked int
	if err := tx.QueryRow(`SELECT is_locked FROM forum_threads WHERE id = ?`, threadID).Scan(&locked); err != nil {
		return err
	}
	if locked != 0 {
		return fmt.Errorf("thread locked")
	}
	if _, err := tx.Exec(`INSERT INTO forum_posts (id, thread_id, user_id, body_raw, body_html) VALUES (?,?,?,?,?)`, newID(), threadID, userID, body, html); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE forum_threads SET last_post_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, threadID); err != nil {
		return err
	}
	return tx.Commit()
}

func UpdateThreadTitle(db *sql.DB, t Thread, title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("title required")
	}
	slug, err := uniqueSlug(db, t.SectionID, title, t.ID)
	if err != nil {
		return "", err
	}
	_, err = db.Exec(`UPDATE forum_threads SET title=?, slug=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, title, slug, t.ID)
	return slug, err
}

func UpdatePost(db *sql.DB, id, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("body required")
	}
	_, err := db.Exec(`UPDATE forum_posts SET body_raw=?, body_html=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, body, article.Render(body), id)
	return err
}

func DeletePost(db *sql.DB, p Post) (deletedThread bool, err error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if p.First {
		if _, err := tx.Exec(`DELETE FROM forum_thread_reads WHERE thread_id=?`, p.ThreadID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(`DELETE FROM forum_posts WHERE thread_id=?`, p.ThreadID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(`DELETE FROM forum_threads WHERE id=?`, p.ThreadID); err != nil {
			return false, err
		}
		return true, tx.Commit()
	}
	if _, err := tx.Exec(`DELETE FROM forum_posts WHERE id=?`, p.ID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`UPDATE forum_threads SET last_post_at = IFNULL((SELECT MAX(created_at) FROM forum_posts WHERE thread_id=?), last_post_at), updated_at=CURRENT_TIMESTAMP WHERE id=?`, p.ThreadID, p.ThreadID); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

func (p Post) Edited() bool {
	return p.UpdatedAt != "" && p.CreatedAt != "" && p.UpdatedAt > p.CreatedAt
}

func CanEdit(role, userID string, ownerID string) bool {
	return role == "admin" || userID == ownerID
}

func uniqueSectionSlug(db *sql.DB, name, exceptID string) (string, error) {
	base := skater.Slugify(name)
	if base == "" {
		base = "section"
	}
	if base == "new" {
		base = "section"
	}
	slug := base
	for n := 2; n < 100; n++ {
		var existing string
		err := db.QueryRow(`SELECT id FROM forum_sections WHERE slug = ?`, slug).Scan(&existing)
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

func CreateSection(db *sql.DB, name, desc string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name required")
	}
	slug, err := uniqueSectionSlug(db, name, "")
	if err != nil {
		return err
	}
	var max int
	_ = db.QueryRow(`SELECT IFNULL(MAX(sort_order),-1) FROM forum_sections`).Scan(&max)
	_, err = db.Exec(`INSERT INTO forum_sections (id, name, slug, description, sort_order) VALUES (?,?,?,?,?)`, newID(), name, slug, strings.TrimSpace(desc), max+1)
	return err
}

func SaveSection(db *sql.DB, id, name, desc string, active bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name required")
	}
	slug, err := uniqueSectionSlug(db, name, id)
	if err != nil {
		return err
	}
	on := 0
	if active {
		on = 1
	}
	_, err = db.Exec(`UPDATE forum_sections SET name=?, slug=?, description=?, is_active=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, name, slug, strings.TrimSpace(desc), on, id)
	return err
}

func MoveSection(db *sql.DB, id string, dir int) error {
	var order int
	if err := db.QueryRow(`SELECT sort_order FROM forum_sections WHERE id=?`, id).Scan(&order); err != nil {
		return err
	}
	var oid string
	var oorder int
	q := `SELECT id, sort_order FROM forum_sections WHERE sort_order > ? ORDER BY sort_order LIMIT 1`
	if dir < 0 {
		q = `SELECT id, sort_order FROM forum_sections WHERE sort_order < ? ORDER BY sort_order DESC LIMIT 1`
	}
	err := db.QueryRow(q, order).Scan(&oid, &oorder)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE forum_sections SET sort_order=? WHERE id=?`, oorder, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE forum_sections SET sort_order=? WHERE id=?`, order, oid); err != nil {
		return err
	}
	return tx.Commit()
}

func Export(db *sql.DB, userID string) ([]OwnPost, error) {
	rows, err := db.Query(`
		SELECT t.title, p.body_raw, p.created_at, p.is_first_post
		FROM forum_posts p JOIN forum_threads t ON t.id = p.thread_id
		WHERE p.user_id = ? ORDER BY p.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OwnPost
	for rows.Next() {
		var p OwnPost
		var first int
		if err := rows.Scan(&p.ThreadTitle, &p.BodyRaw, &p.CreatedAt, &first); err != nil {
			return nil, err
		}
		p.First = first != 0
		out = append(out, p)
	}
	return out, rows.Err()
}
