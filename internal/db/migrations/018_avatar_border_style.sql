ALTER TABLE skater_profiles ADD COLUMN avatar_border_style TEXT NOT NULL DEFAULT '';
ALTER TABLE skater_profiles ADD COLUMN avatar_border_color TEXT NOT NULL DEFAULT '';
UPDATE skater_profiles SET avatar_border_style='default' WHERE avatar_border=1 AND avatar_border_style='';
