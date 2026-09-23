package forum

import (
	"database/sql"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"sort"
	"strings"

	"seshhub/internal/article"
	"seshhub/internal/auth"
)

var mentionAt = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])@([A-Za-z0-9_]{2,32})\b`)

type UserHit struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type MentionItem struct {
	PostID      string `json:"post_id"`
	SectionSlug string `json:"section_slug"`
	ThreadSlug  string `json:"thread_slug"`
	ThreadTitle string `json:"thread_title"`
	AuthorName  string `json:"author_name"`
	Excerpt     string `json:"excerpt"`
	CreatedAt   string `json:"created_at"`
	Index       int    `json:"-"`
}

func (m MentionItem) When() template.HTML { return LocalTime(m.CreatedAt) }

func extractHandles(raw string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range mentionAt.FindAllStringSubmatch(raw, -1) {
		h := strings.ToLower(m[1])
		if seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, m[1])
	}
	return out
}

type mentionUser struct {
	ID, Username, DisplayName, Role, Slug string
}

func mentionSelect() string {
	return `SELECT u.id, u.username, u.display_name, u.role, IFNULL(sp.slug,'')
		FROM users u LEFT JOIN skater_profiles sp ON sp.user_id = u.id`
}

func scanMention(row *sql.Row) (mentionUser, error) {
	var u mentionUser
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Slug)
	return u, err
}

func findMentionUser(db *sql.DB, handle string) (mentionUser, error) {
	u, err := scanMention(db.QueryRow(mentionSelect()+`
		WHERE u.id != ? AND lower(u.username) = lower(?)`, auth.TombstoneID, handle))
	if err == nil {
		return u, nil
	}
	if err != sql.ErrNoRows {
		return mentionUser{}, err
	}
	rows, err := db.Query(mentionSelect()+`
		WHERE u.id != ? AND lower(u.display_name) = lower(?)`, auth.TombstoneID, handle)
	if err != nil {
		return mentionUser{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return mentionUser{}, fmt.Errorf("unknown @%s", handle)
	}
	if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Slug); err != nil {
		return mentionUser{}, err
	}
	if rows.Next() {
		return mentionUser{}, fmt.Errorf("ambiguous @%s", handle)
	}
	return u, rows.Err()
}

func cookBody(db *sql.DB, authorID, body string) ([]string, string, error) {
	handles := extractHandles(body)
	users := make([]mentionUser, 0, len(handles))
	var mentionIDs []string
	for _, h := range handles {
		u, err := findMentionUser(db, h)
		if err != nil {
			return nil, "", err
		}
		users = append(users, u)
		if u.ID != authorID {
			mentionIDs = append(mentionIDs, u.ID)
		}
	}
	return mentionIDs, linkMentions(article.Render(body), users), nil
}

func mentionNeedles(u mentionUser) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range []string{u.Username, u.DisplayName} {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
		low := strings.ToLower(n)
		if low != n && !seen[low] {
			seen[low] = true
			out = append(out, low)
		}
	}
	sort.Slice(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

func linkMentions(htmlBody string, users []mentionUser) string {
	sort.Slice(users, func(i, j int) bool {
		return len(users[i].DisplayName)+len(users[i].Username) > len(users[j].DisplayName)+len(users[j].Username)
	})
	for _, u := range users {
		name := u.DisplayName
		if name == "" {
			name = u.Username
		}
		label := "@" + html.EscapeString(name)
		var repl string
		if href := authorURL(u.Role, u.Slug); href != "" {
			repl = `<a class="forum-mention" href="` + href + `">` + label + `</a>`
		} else {
			repl = `<span class="forum-mention">` + label + `</span>`
		}
		for _, n := range mentionNeedles(u) {
			htmlBody = strings.ReplaceAll(htmlBody, "@"+n, repl)
		}
	}
	return htmlBody
}

// MentionUserIDs are the hub accounts @mentioned on a post.
func MentionUserIDs(db *sql.DB, postID string) ([]string, error) {
	rows, err := db.Query(`SELECT user_id FROM forum_mentions WHERE post_id=?`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// MentionDiscords are Discord user ids for mentions on a post. skip are hub user ids already pinged. Accounts without Discord are left out.
func MentionDiscords(db *sql.DB, postID string, skip []string) ([]string, error) {
	skipSet := map[string]bool{}
	for _, id := range skip {
		skipSet[id] = true
	}
	rows, err := db.Query(`
		SELECT m.user_id, IFNULL(u.discord_id,'')
		FROM forum_mentions m JOIN users u ON u.id = m.user_id
		WHERE m.post_id=?`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var uid, discord string
		if err := rows.Scan(&uid, &discord); err != nil {
			return nil, err
		}
		if skipSet[uid] || !snowflake(discord) {
			continue
		}
		out = append(out, discord)
	}
	return out, rows.Err()
}

func snowflake(s string) bool {
	if len(s) < 5 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func FirstPostID(db *sql.DB, threadID string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM forum_posts WHERE thread_id=? AND is_first_post=1`, threadID).Scan(&id)
	return id, err
}

func LatestPostID(db *sql.DB, threadID, userID string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM forum_posts WHERE thread_id=? AND user_id=? ORDER BY rowid DESC LIMIT 1`, threadID, userID).Scan(&id)
	return id, err
}

func saveMentions(tx *sql.Tx, postID string, userIDs []string) error {
	if _, err := tx.Exec(`DELETE FROM forum_mentions WHERE post_id=?`, postID); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, id := range userIDs {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		if _, err := tx.Exec(`INSERT INTO forum_mentions (post_id, user_id) VALUES (?,?)`, postID, id); err != nil {
			return err
		}
	}
	return nil
}

func likeHas(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

func SearchUsers(db *sql.DB, q string, limit int) ([]UserHit, error) {
	q = strings.TrimSpace(q)
	if limit < 1 || limit > 20 {
		limit = 8
	}
	like := likeHas(q)
	rows, err := db.Query(`
		SELECT username, display_name FROM users
		WHERE id != ? AND (username LIKE ? ESCAPE '\' OR display_name LIKE ? ESCAPE '\')
		ORDER BY display_name, username LIMIT ?`, auth.TombstoneID, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserHit
	for rows.Next() {
		var h UserHit
		if err := rows.Scan(&h.Username, &h.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func UnreadMentions(db *sql.DB, userID string) (int, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM forum_mentions m
		JOIN forum_posts p ON p.id = m.post_id
		JOIN forum_threads t ON t.id = p.thread_id
		JOIN forum_sections s ON s.id = t.section_id AND s.is_active = 1
		LEFT JOIN forum_mention_reads r ON r.user_id = m.user_id
		WHERE m.user_id = ? AND (r.last_read_at IS NULL OR m.created_at > r.last_read_at)`, userID).Scan(&n)
	return n, err
}

func MarkMentionsRead(db *sql.DB, userID string) error {
	_, err := db.Exec(`
		INSERT INTO forum_mention_reads (user_id, last_read_at)
		VALUES (?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET last_read_at = CURRENT_TIMESTAMP`, userID)
	return err
}

func ListMentions(db *sql.DB, userID string) ([]MentionItem, error) {
	rows, err := db.Query(`
		SELECT p.id, s.slug, t.slug, t.title, u.display_name, p.body_raw, p.created_at,
			(SELECT COUNT(*) FROM forum_posts x WHERE x.thread_id = p.thread_id AND (x.created_at < p.created_at OR (x.created_at = p.created_at AND x.id <= p.id)))
		FROM forum_mentions m
		JOIN forum_posts p ON p.id = m.post_id
		JOIN forum_threads t ON t.id = p.thread_id
		JOIN forum_sections s ON s.id = t.section_id AND s.is_active = 1
		JOIN users u ON u.id = p.user_id
		WHERE m.user_id = ?
		ORDER BY m.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MentionItem
	for rows.Next() {
		var it MentionItem
		var raw string
		if err := rows.Scan(&it.PostID, &it.SectionSlug, &it.ThreadSlug, &it.ThreadTitle, &it.AuthorName, &raw, &it.CreatedAt, &it.Index); err != nil {
			return nil, err
		}
		it.Excerpt = article.Excerpt(raw, "")
		out = append(out, it)
	}
	return out, rows.Err()
}

func ExportMentions(db *sql.DB, userID string) ([]MentionItem, error) {
	return ListMentions(db, userID)
}

func MentionHref(it MentionItem) string {
	return PostURL(it.SectionSlug, it.ThreadSlug, it.PostID)
}

func PostURL(sectionSlug, threadSlug, postID string) string {
	return "/forum/" + sectionSlug + "/" + threadSlug + "?post=" + postID
}
