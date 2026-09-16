package auth

import "testing"

func TestDiscordRole(t *testing.T) {
	supers := []string{"super1"}
	roles := []string{"adminR", "other"}
	cases := []struct {
		id, want              string
		inGuild               bool
		guildRoles            []string
		admin, skate, friends string
	}{
		{id: "super1", inGuild: false, want: RoleAdmin},
		{id: "u", inGuild: false, want: RolePending},
		{id: "u", inGuild: true, guildRoles: roles, admin: "adminR", skate: "skateR", friends: "friendR", want: RoleAdmin},
		{id: "u", inGuild: true, guildRoles: []string{"skateR"}, admin: "adminR", skate: "skateR", friends: "friendR", want: RoleSkater},
		{id: "u", inGuild: true, guildRoles: []string{"friendR", "skateR"}, admin: "adminR", skate: "skateR", friends: "friendR", want: RoleSkater},
		{id: "u", inGuild: true, guildRoles: []string{"friendR"}, admin: "adminR", skate: "skateR", friends: "friendR", want: RoleFriend},
		{id: "u", inGuild: true, guildRoles: []string{"x"}, admin: "adminR", skate: "skateR", friends: "friendR", want: RolePending},
	}
	for _, c := range cases {
		got := DiscordRole(c.id, c.inGuild, c.guildRoles, supers, c.admin, c.skate, c.friends)
		if got != c.want {
			t.Fatalf("id=%s inGuild=%v roles=%v: got %s want %s", c.id, c.inGuild, c.guildRoles, got, c.want)
		}
	}
}

func TestKeepRole(t *testing.T) {
	if KeepRole(RoleFriend, RolePending) != RoleFriend {
		t.Fatal("approved friend must survive pending discord re-login")
	}
	if KeepRole(RolePending, RoleFriend) != RoleFriend {
		t.Fatal("friends role must still promote")
	}
	if KeepRole(RoleFriend, RoleSkater) != RoleSkater {
		t.Fatal("skater role must promote friend")
	}
	if KeepRole(RolePending, RolePending) != RolePending {
		t.Fatal("pending stays pending")
	}
}
