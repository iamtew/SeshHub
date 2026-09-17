INSERT INTO pages (id, slug, title, content_raw, content_html, is_published)
SELECT 'page-friends', 'friends', 'Friends', '# Friends', '<h1>Friends</h1>', 1
WHERE NOT EXISTS (SELECT 1 FROM pages WHERE id = 'page-friends' OR slug = 'friends');
