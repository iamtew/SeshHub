package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

const CookieName = "seshhub_session"

var ErrTaken = fmt.Errorf("already linked to another account")
var ErrMerge = fmt.Errorf("cannot merge")
var ErrHasArticles = fmt.Errorf("user authored articles")
var ErrDeleteSelf = fmt.Errorf("cannot delete yourself")

type User struct {
	ID                  string
	Username            string
	DisplayName         string
	AvatarURL           string
	Role                string
	DiscordID           string
	YouTubeChannelID    string
	YouTubeChannelTitle string
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func UpsertDiscord(db *sql.DB, discordID, username, display, avatar, role string) (User, error) {
	var u User
	err := db.QueryRow(`SELECT id, username, display_name, IFNULL(avatar_url,''), role, IFNULL(discord_id,''), IFNULL(youtube_channel_id,''), IFNULL(youtube_channel_title,'') FROM users WHERE discord_id = ?`, discordID).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role, &u.DiscordID, &u.YouTubeChannelID, &u.YouTubeChannelTitle)
	if err == sql.ErrNoRows {
		u = User{ID: newID(), Username: username, DisplayName: display, AvatarURL: avatar, Role: role, DiscordID: discordID}
		_, err = db.Exec(`INSERT INTO users (id, username, display_name, avatar_url, role, discord_id, discord_username) VALUES (?,?,?,?,?,?,?)`,
			u.ID, u.Username, u.DisplayName, nullIfEmpty(avatar), role, discordID, username)
		return u, err
	}
	if err != nil {
		return u, err
	}
	role = KeepRole(u.Role, role)
	u.Username, u.DisplayName, u.AvatarURL, u.Role = username, display, avatar, role
	_, err = db.Exec(`UPDATE users SET username=?, display_name=?, avatar_url=?, role=?, discord_username=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		username, display, nullIfEmpty(avatar), role, username, u.ID)
	return u, err
}

func UpsertYouTube(db *sql.DB, channelID, title, avatar, refresh string) (User, error) {
	var u User
	err := db.QueryRow(`SELECT id, username, display_name, IFNULL(avatar_url,''), role, IFNULL(discord_id,''), IFNULL(youtube_channel_id,''), IFNULL(youtube_channel_title,'') FROM users WHERE youtube_channel_id = ?`, channelID).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role, &u.DiscordID, &u.YouTubeChannelID, &u.YouTubeChannelTitle)
	if err == sql.ErrNoRows {
		u = User{ID: newID(), Username: title, DisplayName: title, AvatarURL: avatar, Role: RolePending, YouTubeChannelID: channelID, YouTubeChannelTitle: title}
		_, err = db.Exec(`INSERT INTO users (id, username, display_name, avatar_url, role, youtube_channel_id, youtube_channel_title, youtube_refresh_token) VALUES (?,?,?,?,?,?,?,?)`,
			u.ID, title, title, nullIfEmpty(avatar), RolePending, channelID, title, nullIfEmpty(refresh))
		return u, err
	}
	if err != nil {
		return u, err
	}
	u.Username, u.DisplayName, u.AvatarURL, u.YouTubeChannelTitle = title, title, avatar, title
	_, err = db.Exec(`UPDATE users SET username=?, display_name=?, avatar_url=?, youtube_channel_title=?, youtube_refresh_token=COALESCE(NULLIF(?, ''), youtube_refresh_token), updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		title, title, nullIfEmpty(avatar), title, refresh, u.ID)
	return u, err
}

func idTaken(db *sql.DB, col, val, exceptID string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE `+col+` = ? AND id != ?`, val, exceptID).Scan(&n)
	return n > 0, err
}

func LinkDiscord(db *sql.DB, userID, discordID, username, display, avatar, role string) (User, error) {
	taken, err := idTaken(db, "discord_id", discordID, userID)
	if err != nil {
		return User{}, err
	}
	if taken {
		return User{}, ErrTaken
	}
	u, err := GetUser(db, userID)
	if err != nil {
		return u, err
	}
	role = KeepRole(u.Role, role)
	_, err = db.Exec(`UPDATE users SET discord_id=?, discord_username=?, username=?, display_name=?, avatar_url=?, role=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		discordID, username, username, display, nullIfEmpty(avatar), role, userID)
	if err != nil {
		return u, err
	}
	return GetUser(db, userID)
}

func LinkYouTube(db *sql.DB, userID, channelID, title, avatar, refresh string) (User, error) {
	taken, err := idTaken(db, "youtube_channel_id", channelID, userID)
	if err != nil {
		return User{}, err
	}
	if taken {
		return User{}, ErrTaken
	}
	_, err = db.Exec(`UPDATE users SET youtube_channel_id=?, youtube_channel_title=?, avatar_url=COALESCE(NULLIF(?, ''), avatar_url), youtube_refresh_token=COALESCE(NULLIF(?, ''), youtube_refresh_token), updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		channelID, title, avatar, refresh, userID)
	if err != nil {
		return User{}, err
	}
	return GetUser(db, userID)
}

func GetUser(db *sql.DB, id string) (User, error) {
	var u User
	err := db.QueryRow(`SELECT id, username, display_name, IFNULL(avatar_url,''), role, IFNULL(discord_id,''), IFNULL(youtube_channel_id,''), IFNULL(youtube_channel_title,'') FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role, &u.DiscordID, &u.YouTubeChannelID, &u.YouTubeChannelTitle)
	return u, err
}

func ListUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`SELECT id, username, display_name, IFNULL(avatar_url,''), role, IFNULL(discord_id,''), IFNULL(youtube_channel_id,''), IFNULL(youtube_channel_title,'') FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role, &u.DiscordID, &u.YouTubeChannelID, &u.YouTubeChannelTitle); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func UnlinkDiscord(db *sql.DB, userID string) error {
	_, err := db.Exec(`UPDATE users SET discord_id=NULL, discord_username=NULL, updated_at=CURRENT_TIMESTAMP WHERE id=?`, userID)
	return err
}

func UnlinkYouTube(db *sql.DB, userID string) error {
	_, err := db.Exec(`UPDATE users SET youtube_channel_id=NULL, youtube_channel_title=NULL, youtube_refresh_token=NULL, youtube_synced_at=NULL, updated_at=CURRENT_TIMESTAMP WHERE id=?`, userID)
	return err
}

func DeleteUser(db *sql.DB, id, actorID string) error {
	if id == actorID {
		return ErrDeleteSelf
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM articles WHERE author_id = ?`, id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrHasArticles
	}
	if _, err := db.Exec(`UPDATE access_requests SET reviewed_by=NULL WHERE reviewed_by=?`, id); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM access_requests WHERE user_id=?`, id); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM sessions WHERE user_id=?`, id); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

func MergeUsers(db *sql.DB, keepID, fromID string) error {
	if keepID == "" || fromID == "" || keepID == fromID {
		return ErrMerge
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	type row struct {
		role, discordID, discordUser, ytID, ytTitle, ytRefresh, ytSynced string
	}
	load := func(id string) (row, error) {
		var r row
		err := tx.QueryRow(`SELECT role, IFNULL(discord_id,''), IFNULL(discord_username,''), IFNULL(youtube_channel_id,''), IFNULL(youtube_channel_title,''), IFNULL(youtube_refresh_token,''), IFNULL(youtube_synced_at,'') FROM users WHERE id=?`, id).
			Scan(&r.role, &r.discordID, &r.discordUser, &r.ytID, &r.ytTitle, &r.ytRefresh, &r.ytSynced)
		return r, err
	}
	keep, err := load(keepID)
	if err != nil {
		return err
	}
	from, err := load(fromID)
	if err != nil {
		return err
	}
	if keep.discordID != "" && from.discordID != "" {
		return fmt.Errorf("%w: both have Discord", ErrMerge)
	}
	if keep.ytID != "" && from.ytID != "" {
		return fmt.Errorf("%w: both have YouTube", ErrMerge)
	}
	if keep.discordID == "" && from.discordID != "" {
		keep.discordID, keep.discordUser = from.discordID, from.discordUser
	}
	if keep.ytID == "" && from.ytID != "" {
		keep.ytID, keep.ytTitle, keep.ytRefresh, keep.ytSynced = from.ytID, from.ytTitle, from.ytRefresh, from.ytSynced
	}
	role := HigherRole(keep.role, from.role)
	_, err = tx.Exec(`UPDATE users SET discord_id=NULL, discord_username=NULL, youtube_channel_id=NULL, youtube_channel_title=NULL, youtube_refresh_token=NULL, youtube_synced_at=NULL WHERE id=?`, fromID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE users SET discord_id=?, discord_username=?, youtube_channel_id=?, youtube_channel_title=?, youtube_refresh_token=?, youtube_synced_at=?, role=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		nullIfEmpty(keep.discordID), nullIfEmpty(keep.discordUser), nullIfEmpty(keep.ytID), nullIfEmpty(keep.ytTitle), nullIfEmpty(keep.ytRefresh), nullIfEmpty(keep.ytSynced), role, keepID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE articles SET author_id=? WHERE author_id=?`, keepID, fromID); err != nil {
		return err
	}
	var keepProf, fromProf sql.NullString
	_ = tx.QueryRow(`SELECT id FROM skater_profiles WHERE user_id=?`, keepID).Scan(&keepProf)
	_ = tx.QueryRow(`SELECT id FROM skater_profiles WHERE user_id=?`, fromID).Scan(&fromProf)
	if fromProf.Valid {
		if keepProf.Valid {
			_, err = tx.Exec(`UPDATE skater_profiles SET user_id=NULL WHERE id=?`, fromProf.String)
		} else {
			_, err = tx.Exec(`UPDATE skater_profiles SET user_id=? WHERE id=?`, keepID, fromProf.String)
		}
		if err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`UPDATE access_requests SET reviewed_by=NULL WHERE reviewed_by=?`, fromID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM access_requests WHERE user_id=?`, fromID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM sessions WHERE user_id=?`, fromID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM users WHERE id=?`, fromID); err != nil {
		return err
	}
	return tx.Commit()
}

func CreateSession(db *sql.DB, userID, ip, ua string) (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw[:])
	exp := time.Now().UTC().Add(30 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO sessions (id, user_id, ip_address, user_agent, expires_at) VALUES (?,?,?,?,?)`,
		hashToken(token), userID, ip, ua, exp)
	return token, err
}

func UserByToken(db *sql.DB, token string) (User, error) {
	var u User
	err := db.QueryRow(`
		SELECT u.id, u.username, u.display_name, IFNULL(u.avatar_url,''), u.role,
			IFNULL(u.discord_id,''), IFNULL(u.youtube_channel_id,''), IFNULL(u.youtube_channel_title,'')
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > datetime('now')`, hashToken(token)).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role, &u.DiscordID, &u.YouTubeChannelID, &u.YouTubeChannelTitle)
	if err != nil {
		return u, err
	}
	return u, nil
}

func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, hashToken(token))
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
