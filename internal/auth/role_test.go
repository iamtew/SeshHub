package auth

import "testing"

func TestDiscordRole(t *testing.T) {
	supers := []string{"super1"}
	roles := []string{"adminR", "other"}
	cases := []struct {
		id, want     string
		inGuild      bool
		guildRoles   []string
		admin, skate string
	}{
		{id: "super1", inGuild: false, want: RoleAdmin},
		{id: "u", inGuild: false, want: RolePending},
		{id: "u", inGuild: true, guildRoles: roles, admin: "adminR", skate: "skateR", want: RoleAdmin},
		{id: "u", inGuild: true, guildRoles: []string{"skateR"}, admin: "adminR", skate: "skateR", want: RoleSkater},
		{id: "u", inGuild: true, guildRoles: []string{"x"}, admin: "adminR", skate: "skateR", want: RoleMember},
	}
	for _, c := range cases {
		got := DiscordRole(c.id, c.inGuild, c.guildRoles, supers, c.admin, c.skate)
		if got != c.want {
			t.Fatalf("id=%s inGuild=%v roles=%v: got %s want %s", c.id, c.inGuild, c.guildRoles, got, c.want)
		}
	}
}

func TestKeepRole(t *testing.T) {
	if KeepRole(RoleMember, RolePending) != RoleMember {
		t.Fatal("approved member must survive pending discord re-login")
	}
	if KeepRole(RolePending, RoleMember) != RoleMember {
		t.Fatal("guild member must still promote")
	}
	if KeepRole(RolePending, RolePending) != RolePending {
		t.Fatal("pending stays pending")
	}
}
