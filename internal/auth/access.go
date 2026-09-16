package auth

import (
	"database/sql"
	"fmt"
)

type AccessRequest struct {
	ID          string
	UserID      string
	Status      string
	DisplayName string
	Username    string
	CreatedAt   string
}

func RequestAccess(db *sql.DB, userID string) error {
	_, err := db.Exec(`
		INSERT INTO access_requests (id, user_id, status) VALUES (?, ?, 'pending')
		ON CONFLICT(user_id) DO UPDATE SET
			status = 'pending',
			reviewed_by = NULL,
			reviewed_at = NULL,
			created_at = CURRENT_TIMESTAMP
		WHERE access_requests.status = 'rejected'`, newID(), userID)
	return err
}

func AccessStatus(db *sql.DB, userID string) (string, error) {
	var status string
	err := db.QueryRow(`SELECT status FROM access_requests WHERE user_id = ?`, userID).Scan(&status)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return status, err
}

func PendingAccess(db *sql.DB) ([]AccessRequest, error) {
	rows, err := db.Query(`
		SELECT r.id, r.user_id, r.status, u.display_name, u.username, r.created_at
		FROM access_requests r JOIN users u ON u.id = r.user_id
		WHERE r.status = 'pending'
		ORDER BY r.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccessRequest
	for rows.Next() {
		var a AccessRequest
		if err := rows.Scan(&a.ID, &a.UserID, &a.Status, &a.DisplayName, &a.Username, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func DecideAccess(db *sql.DB, requestID, reviewerID, status string) error {
	if status != "approved" && status != "rejected" {
		return fmt.Errorf("bad status")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var userID string
	err = tx.QueryRow(`SELECT user_id FROM access_requests WHERE id = ? AND status = 'pending'`, requestID).Scan(&userID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE access_requests SET status = ?, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, reviewerID, requestID)
	if err != nil {
		return err
	}
	if status == "approved" {
		_, err = tx.Exec(`UPDATE users SET role = 'friend', updated_at = CURRENT_TIMESTAMP WHERE id = ? AND role = 'pending'`, userID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
