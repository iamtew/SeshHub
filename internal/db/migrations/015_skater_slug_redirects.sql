CREATE TABLE IF NOT EXISTS skater_slug_redirects (
    slug TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES skater_profiles(id) ON DELETE CASCADE
);
