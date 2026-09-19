package auth

import (
	"database/sql"
	"testing"

	"seshhub/internal/db"
)

func TestAccessApproveGrantsFriend(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	userID, adminID := newID(), newID()
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role) VALUES (?,?,?,?), (?,?,?,?)`,
		userID, "u", "User", RolePending, adminID, "a", "Admin", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if err := RequestAccess(sqldb, userID); err != nil {
		t.Fatal(err)
	}
	list, err := PendingAccess(sqldb)
	if err != nil || len(list) != 1 {
		t.Fatalf("pending: %v %#v", err, list)
	}
	if err := DecideAccess(sqldb, list[0].ID, adminID, "approved"); err != nil {
		t.Fatal(err)
	}
	var role string
	if err := sqldb.QueryRow(`SELECT role FROM users WHERE id = ?`, userID).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if role != RoleFriend {
		t.Fatalf("role %s", role)
	}
}

func TestAccessRejectYouTubeNukes(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	adminID := newID()
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role) VALUES (?,?,?,?)`, adminID, "a", "Admin", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	u, err := UpsertYouTube(sqldb, "ch-deny", "Chan", "", "refresh")
	if err != nil {
		t.Fatal(err)
	}
	if err := RequestAccess(sqldb, u.ID); err != nil {
		t.Fatal(err)
	}
	list, err := PendingAccess(sqldb)
	if err != nil || len(list) != 1 {
		t.Fatalf("pending: %v %#v", err, list)
	}
	if err := DecideAccess(sqldb, list[0].ID, adminID, "rejected"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetUser(sqldb, u.ID); err != sql.ErrNoRows {
		t.Fatalf("user leftover %v", err)
	}
	denied, err := ConsumeDenial(sqldb, "ch-deny")
	if err != nil || !denied {
		t.Fatalf("denial %v %v", denied, err)
	}
	denied, err = ConsumeDenial(sqldb, "ch-deny")
	if err != nil || denied {
		t.Fatalf("oneshot leftover %v %v", denied, err)
	}
}

func TestAccessRejectDiscordKeepsUser(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	adminID := newID()
	u, err := UpsertDiscord(sqldb, "d-pend", "pal", "Pal", "", RolePending, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role) VALUES (?,?,?,?)`, adminID, "a", "Admin", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if err := RequestAccess(sqldb, u.ID); err != nil {
		t.Fatal(err)
	}
	list, err := PendingAccess(sqldb)
	if err != nil || len(list) != 1 {
		t.Fatalf("pending: %v %#v", err, list)
	}
	if err := DecideAccess(sqldb, list[0].ID, adminID, "rejected"); err != nil {
		t.Fatal(err)
	}
	got, err := GetUser(sqldb, u.ID)
	if err != nil || got.Role != RolePending {
		t.Fatalf("kept %+v %v", got, err)
	}
	st, err := AccessStatus(sqldb, u.ID)
	if err != nil || st != "rejected" {
		t.Fatalf("status %q %v", st, err)
	}
}
