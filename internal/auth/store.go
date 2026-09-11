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

type User struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	Role        string
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
	err := db.QueryRow(`SELECT id, username, display_name, IFNULL(avatar_url,''), role FROM users WHERE discord_id = ?`, discordID).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role)
	if err == sql.ErrNoRows {
		u = User{ID: newID(), Username: username, DisplayName: display, AvatarURL: avatar, Role: role}
		_, err = db.Exec(`INSERT INTO users (id, username, display_name, avatar_url, role, discord_id, discord_username) VALUES (?,?,?,?,?,?,?)`,
			u.ID, u.Username, u.DisplayName, nullIfEmpty(avatar), role, discordID, username)
		return u, err
	}
	if err != nil {
		return u, err
	}
	u.Username, u.DisplayName, u.AvatarURL, u.Role = username, display, avatar, role
	_, err = db.Exec(`UPDATE users SET username=?, display_name=?, avatar_url=?, role=?, discord_username=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		username, display, nullIfEmpty(avatar), role, username, u.ID)
	return u, err
}

func UpsertYouTube(db *sql.DB, channelID, title, avatar string) (User, error) {
	var u User
	err := db.QueryRow(`SELECT id, username, display_name, IFNULL(avatar_url,''), role FROM users WHERE youtube_channel_id = ?`, channelID).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role)
	if err == sql.ErrNoRows {
		u = User{ID: newID(), Username: title, DisplayName: title, AvatarURL: avatar, Role: RolePending}
		_, err = db.Exec(`INSERT INTO users (id, username, display_name, avatar_url, role, youtube_channel_id, youtube_channel_title) VALUES (?,?,?,?,?,?,?)`,
			u.ID, title, title, nullIfEmpty(avatar), RolePending, channelID, title)
		return u, err
	}
	if err != nil {
		return u, err
	}
	u.Username, u.DisplayName, u.AvatarURL = title, title, avatar
	_, err = db.Exec(`UPDATE users SET username=?, display_name=?, avatar_url=?, youtube_channel_title=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		title, title, nullIfEmpty(avatar), title, u.ID)
	return u, err
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
		SELECT u.id, u.username, u.display_name, IFNULL(u.avatar_url,''), u.role
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > datetime('now')`, hashToken(token)).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role)
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
