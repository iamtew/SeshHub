-- Forum mention pings. Off until an admin turns it on.
ALTER TABLE bot_settings ADD COLUMN mentions INTEGER NOT NULL DEFAULT 0;
