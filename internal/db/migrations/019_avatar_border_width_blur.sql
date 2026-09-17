ALTER TABLE skater_profiles ADD COLUMN avatar_border_blur INTEGER NOT NULL DEFAULT 0;
UPDATE skater_profiles SET avatar_border=3 WHERE avatar_border=1;
