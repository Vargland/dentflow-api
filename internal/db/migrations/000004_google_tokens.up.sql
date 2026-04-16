CREATE TABLE google_tokens (
    id            TEXT        NOT NULL DEFAULT gen_random_uuid()::text PRIMARY KEY,
    doctor_id     TEXT        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    access_token  TEXT        NOT NULL,
    refresh_token TEXT        NOT NULL,
    expiry        TIMESTAMPTZ NOT NULL,
    calendar_id   TEXT        NOT NULL DEFAULT 'primary',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER google_tokens_updated_at
    BEFORE UPDATE ON google_tokens
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
