-- Discord bot announcements. One row. Toggles default off; empty channel posts nothing.
CREATE TABLE IF NOT EXISTS bot_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    channel_id TEXT NOT NULL DEFAULT '',
    news INTEGER NOT NULL DEFAULT 0,
    forum INTEGER NOT NULL DEFAULT 0,
    access INTEGER NOT NULL DEFAULT 0
);
INSERT OR IGNORE INTO bot_settings (id) VALUES (1);
