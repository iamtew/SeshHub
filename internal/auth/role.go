package auth

const (
	RoleAdmin   = "admin"
	RoleSkater  = "skater"
	RoleFriend  = "friend"
	RoleMember  = "member"
	RolePending = "pending"
)

func DiscordRole(discordID string, inGuild bool, guildRoleIDs, superAdmins []string, adminRoleID, skaterRoleID, friendsRoleID string) string {
	for _, id := range superAdmins {
		if id == discordID {
			return RoleAdmin
		}
	}
	if !inGuild {
		return RolePending
	}
	if adminRoleID != "" && contains(guildRoleIDs, adminRoleID) {
		return RoleAdmin
	}
	if HasGuildRole(guildRoleIDs, skaterRoleID) {
		return RoleSkater
	}
	if HasGuildRole(guildRoleIDs, friendsRoleID) {
		return RoleFriend
	}
	return RolePending
}

func HasGuildRole(guildRoleIDs []string, roleID string) bool {
	return roleID != "" && contains(guildRoleIDs, roleID)
}

// KeepRole stops a later Discord login from wiping an approved friend/member (or higher) back to pending.
func KeepRole(existing, computed string) string {
	if computed != RolePending {
		return computed
	}
	switch existing {
	case RoleAdmin, RoleSkater, RoleFriend, RoleMember:
		return existing
	}
	return RolePending
}

func roleRank(r string) int {
	switch r {
	case RoleAdmin:
		return 3
	case RoleSkater:
		return 2
	case RoleFriend, RoleMember:
		return 1
	default:
		return 0
	}
}

func HigherRole(a, b string) string {
	if roleRank(a) >= roleRank(b) {
		return a
	}
	return b
}

func contains(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func HasPublicRoster(role string) bool {
	return role == RoleSkater || role == RoleAdmin || role == RoleFriend
}
