package skater

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"strings"
	"unicode"
)

type Profile struct {
	ID                string
	UserID            string
	Slug              string
	SkaterName        string
	RealName          string
	Bio               string
	Stance            string
	Status            string
	AvatarURL         string
	PhotoURL          string
	AvatarR1          int
	AvatarR2          int
	AvatarR3          int
	AvatarR4          int
	AvatarBorder      int
	AvatarBorderBlur  int
	AvatarBorderStyle string
	AvatarBorderColor string
	BannerURL         string
	Location          string
	Sponsors          string
	SocialLinks       string
	SignatureTricks   string
	FeaturedVideoID   string
	Role              string
}

func FrameCSS(r1, r2, r3, r4 int, style, color string, width, blur int) template.CSS {
	r1, r2, r3, r4 = ClampRadius(r1), ClampRadius(r2), ClampRadius(r3), ClampRadius(r4)
	s := fmt.Sprintf("border-radius:%d%% %d%% %d%% %d%%", r1, r2, r3, r4)
	style, color, width = NormalizeBorder(style, color, width)
	if style == "off" {
		return template.CSS(s)
	}
	s += fmt.Sprintf(";--avw:%dpx;--avblur:%dpx", width, ClampBlur(blur))
	if style == "custom" {
		s += ";--avb:" + color
	}
	return template.CSS(s)
}

func FrameClass(style string) string {
	style, _, _ = NormalizeBorder(style, "", 0)
	if style == "off" {
		return ""
	}
	return "avb-" + style
}

func (p Profile) AvatarStyle() template.CSS {
	st, col, w := NormalizeBorder(p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorder)
	return FrameCSS(p.AvatarR1, p.AvatarR2, p.AvatarR3, p.AvatarR4, st, col, w, p.AvatarBorderBlur)
}

func (p Profile) AvatarClass() string {
	st, _, _ := NormalizeBorder(p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorder)
	return FrameClass(st)
}

var borderStyles = map[string]bool{
	"off": true, "default": true, "custom": true,
	"pulse": true, "strobe": true, "fire": true, "neon": true, "orbit": true, "chroma": true,
}

func ClampHex(s string) string {
	s = strings.TrimSpace(s)
	if len(s) != 7 || s[0] != '#' {
		return ""
	}
	for i := 1; i < 7; i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return ""
		}
	}
	return strings.ToLower(s)
}

func NormalizeBorder(style, color string, flag int) (string, string, int) {
	style = strings.ToLower(strings.TrimSpace(style))
	if style == "" {
		if flag != 0 {
			style = "default"
		} else {
			style = "off"
		}
	}
	if !borderStyles[style] {
		style = "off"
	}
	color = ClampHex(color)
	if style == "off" {
		return "off", "", 0
	}
	if style != "custom" {
		color = ""
	} else if color == "" {
		color = "#e5f20d"
	}
	return style, color, ClampWidth(flag)
}

func (p Profile) PublicName() string {
	if s := strings.TrimSpace(p.RealName); s != "" {
		return s
	}
	return p.SkaterName
}

func CanEdit(role, userID, profileUserID string) bool {
	if userID == "" || userID != profileUserID {
		return false
	}
	return role == "skater" || role == "admin" || role == "friend"
}

func RosterPath(role string) string {
	if role == "friend" {
		return "/friends"
	}
	return "/team"
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
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
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "skater"
	}
	return out
}

func Lines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
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
		err := db.QueryRow(`SELECT id FROM skater_profiles WHERE slug = ?`, slug).Scan(&existing)
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

const profileSelect = `
		SELECT p.id, IFNULL(p.user_id,''), p.slug, p.skater_name, IFNULL(p.real_name,''), IFNULL(p.bio,''),
			IFNULL(p.stance,'regular'), IFNULL(p.status,'active'), IFNULL(NULLIF(p.avatar_url,''), IFNULL(u.avatar_url,'')), IFNULL(p.avatar_url,''),
			IFNULL(p.avatar_r1,50), IFNULL(p.avatar_r2,50), IFNULL(p.avatar_r3,50), IFNULL(p.avatar_r4,50), IFNULL(p.avatar_border,0),
			IFNULL(p.avatar_border_style,''), IFNULL(p.avatar_border_color,''), IFNULL(p.avatar_border_blur,0),
			IFNULL(p.banner_url,''),
			IFNULL(p.location,''), IFNULL(p.sponsors,''), IFNULL(p.social_links,''), IFNULL(p.signature_tricks,''),
			IFNULL(p.featured_video_id,''), IFNULL(u.role,'')
		FROM skater_profiles p LEFT JOIN users u ON u.id = p.user_id`

func List(db *sql.DB) ([]Profile, error) {
	return list(db, "")
}

func ListTeam(db *sql.DB) ([]Profile, error) {
	return list(db, "team")
}

func ListFriends(db *sql.DB) ([]Profile, error) {
	return list(db, "friend")
}

func list(db *sql.DB, roster string) ([]Profile, error) {
	q := profileSelect + ` ORDER BY COALESCE(NULLIF(p.real_name,''), p.skater_name)`
	switch roster {
	case "team":
		q = profileSelect + ` WHERE u.role IN ('skater','admin') ORDER BY COALESCE(NULLIF(p.real_name,''), p.skater_name)`
	case "friend":
		q = profileSelect + ` WHERE u.role = 'friend' ORDER BY COALESCE(NULLIF(p.real_name,''), p.skater_name)`
	}
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanProfile(sc interface{ Scan(dest ...any) error }) (Profile, error) {
	var p Profile
	err := sc.Scan(&p.ID, &p.UserID, &p.Slug, &p.SkaterName, &p.RealName, &p.Bio, &p.Stance, &p.Status,
		&p.AvatarURL, &p.PhotoURL, &p.AvatarR1, &p.AvatarR2, &p.AvatarR3, &p.AvatarR4, &p.AvatarBorder,
		&p.AvatarBorderStyle, &p.AvatarBorderColor, &p.AvatarBorderBlur,
		&p.BannerURL, &p.Location, &p.Sponsors, &p.SocialLinks, &p.SignatureTricks, &p.FeaturedVideoID, &p.Role)
	if err != nil {
		return p, err
	}
	p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorder = NormalizeBorder(p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorder)
	p.AvatarBorderBlur = ClampBlur(p.AvatarBorderBlur)
	if p.AvatarBorderStyle == "off" {
		p.AvatarBorderBlur = 0
	}
	return p, nil
}

func Get(db *sql.DB, by, val string) (Profile, error) {
	col := "p.id"
	switch by {
	case "slug":
		col = "p.slug"
	case "user_id":
		col = "p.user_id"
	}
	return scanProfile(db.QueryRow(profileSelect+` WHERE `+col+` = ?`, val))
}

func EnsureForUser(db *sql.DB, userID, discordName string) (Profile, error) {
	discordName = strings.TrimSpace(discordName)
	if userID == "" || discordName == "" {
		return Profile{}, fmt.Errorf("user and name required")
	}
	p, err := Get(db, "user_id", userID)
	if err == sql.ErrNoRows {
		return Save(db, Profile{UserID: userID, SkaterName: discordName})
	}
	if err != nil {
		return p, err
	}
	if p.SkaterName == discordName {
		return p, nil
	}
	p.SkaterName = discordName
	return Save(db, p)
}

func Save(db *sql.DB, p Profile) (Profile, error) {
	p.SkaterName = strings.TrimSpace(p.SkaterName)
	if p.SkaterName == "" {
		return p, fmt.Errorf("name required")
	}
	base := p.Slug
	if base == "" {
		if strings.TrimSpace(p.RealName) != "" {
			base = Slugify(p.RealName)
		} else {
			base = Slugify(p.SkaterName)
		}
	} else {
		base = Slugify(base)
	}
	slug, err := uniqueSlug(db, base, p.ID)
	if err != nil {
		return p, err
	}
	oldSlug := ""
	if p.ID != "" {
		if prev, err := Get(db, "id", p.ID); err == nil {
			oldSlug = prev.Slug
		}
	}
	p.Slug = slug
	if p.Stance == "" {
		p.Stance = "regular"
	}
	if p.Status == "" {
		p.Status = "active"
	}
	if p.ID == "" && p.AvatarR1 == 0 && p.AvatarR2 == 0 && p.AvatarR3 == 0 && p.AvatarR4 == 0 {
		p.AvatarR1, p.AvatarR2, p.AvatarR3, p.AvatarR4 = 50, 50, 50, 50
	}
	p.AvatarR1, p.AvatarR2, p.AvatarR3, p.AvatarR4 = ClampRadius(p.AvatarR1), ClampRadius(p.AvatarR2), ClampRadius(p.AvatarR3), ClampRadius(p.AvatarR4)
	p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorder = NormalizeBorder(p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorder)
	p.AvatarBorderBlur = ClampBlur(p.AvatarBorderBlur)
	if p.AvatarBorderStyle == "off" {
		p.AvatarBorderBlur = 0
	}
	uid := any(nil)
	if p.UserID != "" {
		uid = p.UserID
	}
	if p.ID == "" {
		p.ID = newID()
		_, err = db.Exec(`INSERT INTO skater_profiles (id, user_id, slug, skater_name, real_name, bio, stance, status, avatar_url, avatar_r1, avatar_r2, avatar_r3, avatar_r4, avatar_border, avatar_border_style, avatar_border_color, avatar_border_blur, banner_url, location, sponsors, social_links, signature_tricks, featured_video_id)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			p.ID, uid, p.Slug, p.SkaterName, nullEmpty(p.RealName), nullEmpty(p.Bio), p.Stance, p.Status,
			nullEmpty(p.PhotoURL), p.AvatarR1, p.AvatarR2, p.AvatarR3, p.AvatarR4, p.AvatarBorder, p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorderBlur,
			nullEmpty(p.BannerURL), nullEmpty(p.Location), nullEmpty(p.Sponsors),
			nullEmpty(p.SocialLinks), nullEmpty(p.SignatureTricks), nullEmpty(p.FeaturedVideoID))
	} else {
		_, err = db.Exec(`UPDATE skater_profiles SET user_id=?, slug=?, skater_name=?, real_name=?, bio=?, stance=?, status=?, avatar_url=?, avatar_r1=?, avatar_r2=?, avatar_r3=?, avatar_r4=?, avatar_border=?, avatar_border_style=?, avatar_border_color=?, avatar_border_blur=?, banner_url=?, location=?, sponsors=?, social_links=?, signature_tricks=?, featured_video_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			uid, p.Slug, p.SkaterName, nullEmpty(p.RealName), nullEmpty(p.Bio), p.Stance, p.Status,
			nullEmpty(p.PhotoURL), p.AvatarR1, p.AvatarR2, p.AvatarR3, p.AvatarR4, p.AvatarBorder, p.AvatarBorderStyle, p.AvatarBorderColor, p.AvatarBorderBlur,
			nullEmpty(p.BannerURL), nullEmpty(p.Location), nullEmpty(p.Sponsors),
			nullEmpty(p.SocialLinks), nullEmpty(p.SignatureTricks), nullEmpty(p.FeaturedVideoID), p.ID)
	}
	if err != nil {
		return p, err
	}
	if err := rememberSlug(db, p.ID, oldSlug, p.Slug); err != nil {
		return p, err
	}
	return p, nil
}

func rememberSlug(db *sql.DB, profileID, oldSlug, newSlug string) error {
	if _, err := db.Exec(`DELETE FROM skater_slug_redirects WHERE slug = ?`, newSlug); err != nil {
		return err
	}
	if oldSlug == "" || oldSlug == newSlug {
		return nil
	}
	_, err := db.Exec(`INSERT INTO skater_slug_redirects (slug, profile_id) VALUES (?,?)
		ON CONFLICT(slug) DO UPDATE SET profile_id=excluded.profile_id`, oldSlug, profileID)
	return err
}

func CurrentSlug(db *sql.DB, old string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT profile_id FROM skater_slug_redirects WHERE slug = ?`, old).Scan(&id)
	if err != nil {
		return "", err
	}
	p, err := Get(db, "id", id)
	if err != nil {
		return "", err
	}
	if p.Slug == "" || p.Slug == old {
		return "", sql.ErrNoRows
	}
	return p.Slug, nil
}

func FormerSlugs(db *sql.DB, profileID string) ([]string, error) {
	rows, err := db.Query(`SELECT slug FROM skater_slug_redirects WHERE profile_id = ?`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func Delete(db *sql.DB, id string) error {
	RemoveGalleries(GalleryIDs(db, id))
	_, _ = db.Exec(`DELETE FROM gallery_photos WHERE profile_id = ?`, id)
	RemovePhoto(id)
	_, _ = db.Exec(`DELETE FROM skater_slug_redirects WHERE profile_id = ?`, id)
	_, err := db.Exec(`DELETE FROM skater_profiles WHERE id = ?`, id)
	return err
}

func nullEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
