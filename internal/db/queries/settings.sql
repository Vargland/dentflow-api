-- name: GetUserSettings :one
SELECT * FROM user_settings WHERE doctor_id = $1 LIMIT 1;

-- name: UpsertUserSettings :one
INSERT INTO user_settings (doctor_id, timezone)
VALUES ($1, $2)
ON CONFLICT (doctor_id) DO UPDATE SET timezone = EXCLUDED.timezone
RETURNING *;

-- name: GetGoogleToken :one
SELECT * FROM google_tokens WHERE doctor_id = $1 LIMIT 1;

-- name: UpsertGoogleToken :one
INSERT INTO google_tokens (doctor_id, access_token, refresh_token, expiry, calendar_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (doctor_id) DO UPDATE
SET access_token  = EXCLUDED.access_token,
    refresh_token = EXCLUDED.refresh_token,
    expiry        = EXCLUDED.expiry,
    calendar_id   = EXCLUDED.calendar_id
RETURNING *;

-- name: DeleteGoogleToken :exec
DELETE FROM google_tokens WHERE doctor_id = $1;
