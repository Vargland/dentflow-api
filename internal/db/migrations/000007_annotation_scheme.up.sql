ALTER TABLE user_settings
    ADD COLUMN IF NOT EXISTS annotation_scheme TEXT NOT NULL DEFAULT 'international';
