-- Users and authentication
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    display_name TEXT NOT NULL,
    avatar_url TEXT,
    role TEXT NOT NULL DEFAULT 'member',
    discord_id TEXT UNIQUE,
    discord_username TEXT,
    youtube_channel_id TEXT UNIQUE,
    youtube_channel_title TEXT,
    youtube_refresh_token TEXT,
    youtube_synced_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address TEXT,
    user_agent TEXT,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Skater profiles and roster
CREATE TABLE IF NOT EXISTS skater_profiles (
    id TEXT PRIMARY KEY,
    user_id TEXT UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    slug TEXT UNIQUE NOT NULL,
    skater_name TEXT NOT NULL,
    real_name TEXT,
    bio TEXT,
    stance TEXT DEFAULT 'regular',
    status TEXT DEFAULT 'active',
    avatar_url TEXT,
    banner_url TEXT,
    location TEXT,
    sponsors TEXT,
    social_links TEXT,
    signature_tricks TEXT,
    featured_video_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Articles and announcements
CREATE TABLE IF NOT EXISTS articles (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    excerpt TEXT,
    content_raw TEXT NOT NULL,
    content_html TEXT NOT NULL,
    featured_image_url TEXT,
    author_id TEXT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'draft',
    tags TEXT,
    published_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Static custom pages
CREATE TABLE IF NOT EXISTS pages (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    content_raw TEXT NOT NULL,
    content_html TEXT NOT NULL,
    custom_css TEXT,
    is_published BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- YouTube videos and sync logs
CREATE TABLE IF NOT EXISTS youtube_videos (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    published_at DATETIME NOT NULL,
    thumbnail_url TEXT NOT NULL,
    duration_seconds INTEGER DEFAULT 0,
    view_count INTEGER DEFAULT 0,
    like_count INTEGER DEFAULT 0,
    tags TEXT,
    category TEXT,
    is_featured BOOLEAN NOT NULL DEFAULT 0,
    is_hidden BOOLEAN NOT NULL DEFAULT 0,
    synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sync_logs (
    id TEXT PRIMARY KEY,
    service TEXT NOT NULL,
    status TEXT NOT NULL,
    videos_fetched INTEGER DEFAULT 0,
    videos_inserted INTEGER DEFAULT 0,
    videos_updated INTEGER DEFAULT 0,
    error_message TEXT,
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);
