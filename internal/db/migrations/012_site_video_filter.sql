CREATE TABLE IF NOT EXISTS site_video_filter (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    rules TEXT
);
INSERT OR IGNORE INTO site_video_filter (id) VALUES (1);
