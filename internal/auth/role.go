package auth

const (
	RoleAdmin   = "admin"
	RoleSkater  = "skater"
	RoleMember  = "member"
	RolePending = "pending"
)

func DiscordRole(discordID string, inGuild bool, guildRoleIDs, superAdmins []string, adminRoleID, skaterRoleID string) string {
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
	if skaterRoleID != "" && contains(guildRoleIDs, skaterRoleID) {
		return RoleSkater
	}
	return RoleMember
}

// KeepRole stops a later Discord login from wiping an admin-approved member (or higher) back to pending.
func KeepRole(existing, computed string) string {
	if computed != RolePending {
		return computed
	}
	switch existing {
	case RoleAdmin, RoleSkater, RoleMember:
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
	case RoleMember:
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
