ALTER TABLE forum_post_photos ADD COLUMN kind TEXT NOT NULL DEFAULT 'image';
ALTER TABLE forum_post_photos ADD COLUMN ext TEXT NOT NULL DEFAULT 'jpg';
ALTER TABLE forum_post_photos ADD COLUMN mime TEXT NOT NULL DEFAULT 'image/jpeg';
