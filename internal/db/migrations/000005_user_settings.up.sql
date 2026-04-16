CREATE TABLE user_settings (
    doctor_id  TEXT        NOT NULL PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    timezone   TEXT        NOT NULL DEFAULT 'America/Argentina/Buenos_Aires',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
