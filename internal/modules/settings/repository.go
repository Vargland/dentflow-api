package settings

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
)

// ErrNotFound is returned when settings have not been created yet.
var ErrNotFound = errors.New("settings not found")

// Repository handles database operations for settings.
type Repository struct {
	q *db.Queries
}

// NewRepository creates a new settings Repository.
func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

// GetSettings returns the settings for a doctor, or ErrNotFound if none exist.
func (r *Repository) GetSettings(ctx context.Context, doctorID string) (db.UserSettings, error) {
	s, err := r.q.GetUserSettings(ctx, doctorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.UserSettings{}, ErrNotFound
	}
	return s, err
}

// UpsertSettings creates or updates the settings for a doctor.
func (r *Repository) UpsertSettings(ctx context.Context, doctorID string, req UpdateSettingsRequest) (db.UserSettings, error) {
	lang := req.EmailLanguage
	if lang != "es" && lang != "en" {
		lang = "es"
	}

	return r.q.UpsertUserSettings(ctx, db.UpsertUserSettingsParams{
		DoctorID:      doctorID,
		Timezone:      req.Timezone,
		DoctorName:    req.DoctorName,
		ClinicAddress: req.ClinicAddress,
		ClinicPhone:   req.ClinicPhone,
		EmailLanguage: lang,
	})
}

// GetGoogleToken returns the stored Google token, or ErrNotFound.
func (r *Repository) GetGoogleToken(ctx context.Context, doctorID string) (db.GoogleToken, error) {
	t, err := r.q.GetGoogleToken(ctx, doctorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GoogleToken{}, ErrNotFound
	}
	return t, err
}

// UpsertGoogleToken stores or refreshes a doctor's Google OAuth token.
func (r *Repository) UpsertGoogleToken(ctx context.Context, p db.UpsertGoogleTokenParams) (db.GoogleToken, error) {
	return r.q.UpsertGoogleToken(ctx, p)
}

// DeleteGoogleToken removes a doctor's Google Calendar connection.
func (r *Repository) DeleteGoogleToken(ctx context.Context, doctorID string) error {
	return r.q.DeleteGoogleToken(ctx, doctorID)
}
