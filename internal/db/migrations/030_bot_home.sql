-- Home channel: where the bot listens. Empty = it does not listen.
ALTER TABLE bot_settings ADD COLUMN home_channel_id TEXT NOT NULL DEFAULT '';
