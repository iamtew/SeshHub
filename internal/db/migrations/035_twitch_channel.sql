-- Twitch going-live posts to this channel, not the general announce channel.
ALTER TABLE bot_settings ADD COLUMN twitch_channel_id TEXT NOT NULL DEFAULT '';
