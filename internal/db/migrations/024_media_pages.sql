INSERT INTO pages (id, slug, title, content_raw, content_html, is_published)
SELECT 'page-photos', 'photos', 'Photos', '# Photos', '<h1>Photos</h1>', 1
WHERE NOT EXISTS (SELECT 1 FROM pages WHERE id = 'page-photos' OR slug = 'photos');

INSERT INTO pages (id, slug, title, content_raw, content_html, is_published)
SELECT 'page-videos', 'videos', 'Videos', '# Videos', '<h1>Videos</h1>', 1
WHERE NOT EXISTS (SELECT 1 FROM pages WHERE id = 'page-videos' OR slug = 'videos');
