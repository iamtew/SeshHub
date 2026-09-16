CREATE TABLE IF NOT EXISTS site_video_filter (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    rules TEXT
);
INSERT OR IGNORE INTO site_video_filter (id) VALUES (1);
UPDATE site_video_filter SET rules = (
    SELECT video_filter FROM users WHERE IFNULL(video_filter,'') != '' LIMIT 1
) WHERE id = 1 AND IFNULL(rules,'') = '';
UPDATE users SET video_filter = NULL;
