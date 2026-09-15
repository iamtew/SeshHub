-- Seed About / Privacy / ToS CMS pages. Skip if the slug already exists.
INSERT INTO pages (id, slug, title, content_raw, content_html, is_published)
SELECT 'page-about', 'about', 'About', '# About', '<h1>About</h1>', 1
WHERE NOT EXISTS (SELECT 1 FROM pages WHERE slug = 'about');

INSERT INTO pages (id, slug, title, content_raw, content_html, is_published)
SELECT 'page-about-privacy', 'about/privacy', 'Privacy Policy', '# Privacy Policy', '<h1>Privacy Policy</h1>', 1
WHERE NOT EXISTS (SELECT 1 FROM pages WHERE slug = 'about/privacy');

INSERT INTO pages (id, slug, title, content_raw, content_html, is_published)
SELECT 'page-about-tos', 'about/tos', 'Terms of Service', '# Terms of Service', '<h1>Terms of Service</h1>', 1
WHERE NOT EXISTS (SELECT 1 FROM pages WHERE slug = 'about/tos');
