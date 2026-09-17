INSERT INTO pages (id, slug, title, content_raw, content_html, is_published)
SELECT 'page-team', 'team', 'FS Team', '# FS Team', '<h1>FS Team</h1>', 1
WHERE NOT EXISTS (SELECT 1 FROM pages WHERE id = 'page-team' OR slug = 'team');
