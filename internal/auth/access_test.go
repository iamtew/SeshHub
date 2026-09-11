package auth

import (
	"testing"

	"seshhub/internal/db"
)

func TestAccessApproveGrantsMember(t *testing.T) {
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
	if role != RoleMember {
		t.Fatalf("role %s", role)
	}
}
