-- Twitch login on roster profiles; live title/started_at are the last Helix snapshot.
ALTER TABLE skater_profiles ADD COLUMN twitch_login TEXT NOT NULL DEFAULT '';
ALTER TABLE skater_profiles ADD COLUMN twitch_title TEXT NOT NULL DEFAULT '';
ALTER TABLE skater_profiles ADD COLUMN twitch_started_at DATETIME;
CREATE UNIQUE INDEX IF NOT EXISTS skater_twitch_login ON skater_profiles(twitch_login) WHERE twitch_login != '';

-- Discord going-live announce. Off until an admin turns it on. Empty message uses the default in code.
ALTER TABLE bot_settings ADD COLUMN twitch INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bot_settings ADD COLUMN twitch_msg TEXT NOT NULL DEFAULT '';
