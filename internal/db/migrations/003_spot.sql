-- Special page: Spot (the public homepage)
CREATE TABLE IF NOT EXISTS spot (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    show_episode INTEGER NOT NULL DEFAULT 1,
    content_raw TEXT NOT NULL DEFAULT '',
    content_html TEXT NOT NULL DEFAULT '',
    spot_image TEXT NOT NULL DEFAULT '',
    sub_images TEXT NOT NULL DEFAULT '[]'
);
INSERT OR IGNORE INTO spot (id) VALUES (1);
