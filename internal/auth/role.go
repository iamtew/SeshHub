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

func contains(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
