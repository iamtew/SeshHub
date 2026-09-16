UPDATE users SET role = 'friend', updated_at = CURRENT_TIMESTAMP WHERE role = 'member' AND id != 'deleted-user';

-- ponytail: slug = user id (unique); they can rename on /dashboard/profile
INSERT INTO skater_profiles (id, user_id, slug, skater_name, stance, status)
SELECT lower(hex(randomblob(16))), u.id, u.id,
	CASE WHEN trim(u.display_name) != '' THEN u.display_name ELSE u.username END,
	'regular', 'active'
FROM users u
WHERE u.role = 'friend'
	AND NOT EXISTS (SELECT 1 FROM skater_profiles p WHERE p.user_id = u.id)
	AND NOT EXISTS (SELECT 1 FROM skater_profiles p WHERE p.slug = u.id);
