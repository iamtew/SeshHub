CREATE TABLE IF NOT EXISTS forum_sections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS forum_threads (
    id           TEXT PRIMARY KEY,
    section_id   TEXT NOT NULL REFERENCES forum_sections(id),
    user_id      TEXT NOT NULL REFERENCES users(id),
    title        TEXT NOT NULL,
    slug         TEXT NOT NULL,
    is_locked    INTEGER NOT NULL DEFAULT 0,
    is_sticky    INTEGER NOT NULL DEFAULT 0,
    last_post_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(section_id, slug)
);

CREATE TABLE IF NOT EXISTS forum_posts (
    id            TEXT PRIMARY KEY,
    thread_id     TEXT NOT NULL REFERENCES forum_threads(id),
    user_id       TEXT NOT NULL REFERENCES users(id),
    body_raw      TEXT NOT NULL,
    body_html     TEXT NOT NULL,
    is_first_post INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS forum_thread_reads (
    user_id      TEXT NOT NULL REFERENCES users(id),
    thread_id    TEXT NOT NULL REFERENCES forum_threads(id),
    last_read_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, thread_id)
);

CREATE INDEX IF NOT EXISTS idx_forum_threads_section_last
    ON forum_threads(section_id, last_post_at DESC);
CREATE INDEX IF NOT EXISTS idx_forum_posts_thread_created
    ON forum_posts(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_forum_thread_reads_user
    ON forum_thread_reads(user_id);

INSERT OR IGNORE INTO forum_sections (id, name, slug, description, sort_order) VALUES
    ('forum-general', 'General', 'general', '', 0),
    ('forum-skateboarding', 'Skateboarding', 'skateboarding', '', 1),
    ('forum-off-topic', 'Off topic', 'off-topic', '', 2);
