ALTER TABLE forum_posts ADD COLUMN parent_id TEXT REFERENCES forum_posts(id);

CREATE TABLE IF NOT EXISTS forum_post_photos (
    id      TEXT PRIMARY KEY,
    post_id TEXT NOT NULL REFERENCES forum_posts(id),
    pos     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS forum_mentions (
    post_id    TEXT NOT NULL REFERENCES forum_posts(id),
    user_id    TEXT NOT NULL REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (post_id, user_id)
);

CREATE TABLE IF NOT EXISTS forum_mention_reads (
    user_id      TEXT PRIMARY KEY REFERENCES users(id),
    last_read_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_forum_post_photos_post ON forum_post_photos(post_id, pos);
CREATE INDEX IF NOT EXISTS idx_forum_mentions_user ON forum_mentions(user_id, created_at);
