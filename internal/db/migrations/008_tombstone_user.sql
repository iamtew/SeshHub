INSERT INTO users (id, username, display_name, role)
SELECT 'deleted-user', 'deleted', 'Former member', 'member'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE id = 'deleted-user');
