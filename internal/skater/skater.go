package skater

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

type Profile struct {
	ID              string
	UserID          string
	Slug            string
	SkaterName      string
	RealName        string
	Bio             string
	Stance          string
	Status          string
	AvatarURL       string
	BannerURL       string
	Location        string
	Sponsors        string
	SocialLinks     string
	SignatureTricks string
	FeaturedVideoID string
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
	return role == "skater" || role == "admin"
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
			IFNULL(p.stance,'regular'), IFNULL(p.status,'active'), IFNULL(NULLIF(p.avatar_url,''), IFNULL(u.avatar_url,'')), IFNULL(p.banner_url,''),
			IFNULL(p.location,''), IFNULL(p.sponsors,''), IFNULL(p.social_links,''), IFNULL(p.signature_tricks,''),
			IFNULL(p.featured_video_id,'')
		FROM skater_profiles p LEFT JOIN users u ON u.id = p.user_id`

func List(db *sql.DB) ([]Profile, error) {
	return list(db, false)
}

func ListTeam(db *sql.DB) ([]Profile, error) {
	return list(db, true)
}

func list(db *sql.DB, linkedOnly bool) ([]Profile, error) {
	q := profileSelect + ` ORDER BY COALESCE(NULLIF(p.real_name,''), p.skater_name)`
	if linkedOnly {
		q = profileSelect + ` WHERE p.user_id IS NOT NULL AND p.user_id != '' ORDER BY COALESCE(NULLIF(p.real_name,''), p.skater_name)`
	}
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.UserID, &p.Slug, &p.SkaterName, &p.RealName, &p.Bio, &p.Stance, &p.Status,
			&p.AvatarURL, &p.BannerURL, &p.Location, &p.Sponsors, &p.SocialLinks, &p.SignatureTricks, &p.FeaturedVideoID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func Get(db *sql.DB, by, val string) (Profile, error) {
	col := "p.id"
	switch by {
	case "slug":
		col = "p.slug"
	case "user_id":
		col = "p.user_id"
	}
	var p Profile
	err := db.QueryRow(profileSelect+` WHERE `+col+` = ?`, val).
		Scan(&p.ID, &p.UserID, &p.Slug, &p.SkaterName, &p.RealName, &p.Bio, &p.Stance, &p.Status,
			&p.AvatarURL, &p.BannerURL, &p.Location, &p.Sponsors, &p.SocialLinks, &p.SignatureTricks, &p.FeaturedVideoID)
	return p, err
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
	uid := any(nil)
	if p.UserID != "" {
		uid = p.UserID
	}
	if p.ID == "" {
		p.ID = newID()
		_, err = db.Exec(`INSERT INTO skater_profiles (id, user_id, slug, skater_name, real_name, bio, stance, status, avatar_url, banner_url, location, sponsors, social_links, signature_tricks, featured_video_id)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			p.ID, uid, p.Slug, p.SkaterName, nullEmpty(p.RealName), nullEmpty(p.Bio), p.Stance, p.Status,
			nullEmpty(p.AvatarURL), nullEmpty(p.BannerURL), nullEmpty(p.Location), nullEmpty(p.Sponsors),
			nullEmpty(p.SocialLinks), nullEmpty(p.SignatureTricks), nullEmpty(p.FeaturedVideoID))
	} else {
		_, err = db.Exec(`UPDATE skater_profiles SET user_id=?, slug=?, skater_name=?, real_name=?, bio=?, stance=?, status=?, avatar_url=?, banner_url=?, location=?, sponsors=?, social_links=?, signature_tricks=?, featured_video_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			uid, p.Slug, p.SkaterName, nullEmpty(p.RealName), nullEmpty(p.Bio), p.Stance, p.Status,
			nullEmpty(p.AvatarURL), nullEmpty(p.BannerURL), nullEmpty(p.Location), nullEmpty(p.Sponsors),
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
