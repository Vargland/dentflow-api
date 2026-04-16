package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
	oauth2v2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
	"github.com/psi-germanr/dentflow-api/internal/gcal"
	"github.com/psi-germanr/dentflow-api/internal/shared"
)

// Handler handles HTTP requests for settings and Google Calendar OAuth.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new settings Handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterProtectedRoutes mounts the auth-protected settings routes on r.
// Call this inside the /api/v1 JWT-protected subrouter.
func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/settings", h.GetSettings)
	r.Put("/settings", h.UpdateSettings)
	r.Delete("/settings/calendar", h.DisconnectCalendar)
}

// GetSettings handles GET /api/v1/settings.
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())

	s, err := h.repo.GetSettings(r.Context(), doctorID)
	if errors.Is(err, ErrNotFound) {
		shared.JSON(w, http.StatusOK, SettingsResponse{
			Timezone:          "America/Argentina/Buenos_Aires",
			EmailLanguage:     "es",
			CalendarConnected: false,
		})
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	resp := SettingsResponse{
		Timezone:          s.Timezone,
		DoctorName:        s.DoctorName,
		ClinicAddress:     s.ClinicAddress,
		ClinicPhone:       s.ClinicPhone,
		EmailLanguage:     s.EmailLanguage,
		CalendarConnected: false,
	}

	tok, err := h.repo.GetGoogleToken(r.Context(), doctorID)
	if err == nil {
		resp.CalendarConnected = true
		resp.CalendarEmail = tok.CalendarID
	}

	shared.JSON(w, http.StatusOK, resp)
}

// UpdateSettings handles PUT /api/v1/settings.
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	if req.Timezone == "" {
		shared.ValidationError(w, "timezone is required")
		return
	}

	s, err := h.repo.UpsertSettings(r.Context(), doctorID, req)
	if err != nil {
		shared.InternalError(w)
		return
	}

	resp := SettingsResponse{
		Timezone:          s.Timezone,
		DoctorName:        s.DoctorName,
		ClinicAddress:     s.ClinicAddress,
		ClinicPhone:       s.ClinicPhone,
		EmailLanguage:     s.EmailLanguage,
		CalendarConnected: false,
	}

	tok, err := h.repo.GetGoogleToken(r.Context(), doctorID)
	if err == nil {
		resp.CalendarConnected = true
		resp.CalendarEmail = tok.CalendarID
	}

	shared.JSON(w, http.StatusOK, resp)
}

// StartCalendarOAuth handles GET /auth/google/calendar.
// doctor_id query param must be provided (set by the frontend from the session).
func (h *Handler) StartCalendarOAuth(w http.ResponseWriter, r *http.Request) {
	doctorID := r.URL.Query().Get("doctor_id")
	if doctorID == "" {
		shared.BadRequest(w, "doctor_id query param required")
		return
	}

	url := gcal.AuthURL(doctorID)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// CalendarOAuthCallback handles GET /auth/google/calendar/callback.
func (h *Handler) CalendarOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state") // doctorID encoded in state

	if code == "" || state == "" {
		shared.BadRequest(w, "missing code or state")
		return
	}

	doctorID := state

	tok, err := gcal.TokenFromCode(r.Context(), code)
	if err != nil {
		log.Printf("CalendarOAuthCallback: token exchange error: %v", err)
		shared.ErrorResponse(w, http.StatusBadGateway, "OAUTH_FAILED", "failed to exchange code")
		return
	}

	calEmail, err := fetchGoogleEmail(r.Context(), tok.AccessToken)
	if err != nil {
		log.Printf("CalendarOAuthCallback: fetchGoogleEmail error: %v", err)
		calEmail = "primary"
	}

	// Preserve existing refresh token if Google didn't return a new one
	refreshToken := tok.RefreshToken
	if refreshToken == "" {
		existing, rerr := h.repo.GetGoogleToken(r.Context(), doctorID)
		if rerr == nil {
			refreshToken = existing.RefreshToken
		}
	}

	_, err = h.repo.UpsertGoogleToken(r.Context(), db.UpsertGoogleTokenParams{
		DoctorID:     doctorID,
		AccessToken:  tok.AccessToken,
		RefreshToken: refreshToken,
		Expiry:       tok.Expiry,
		CalendarID:   calEmail,
	})
	if err != nil {
		log.Printf("CalendarOAuthCallback: UpsertGoogleToken error: %v", err)
		shared.InternalError(w)
		return
	}

	frontendURL := envOrDefault("FRONTEND_URL", "http://localhost:3000")
	http.Redirect(w, r, fmt.Sprintf("%s/settings?calendar=connected", frontendURL), http.StatusTemporaryRedirect)
}

// DisconnectCalendar handles DELETE /api/v1/settings/calendar.
func (h *Handler) DisconnectCalendar(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())

	if err := h.repo.DeleteGoogleToken(r.Context(), doctorID); err != nil {
		shared.InternalError(w)
		return
	}

	shared.NoContent(w)
}

// fetchGoogleEmail retrieves the Google account email using the access token.
func fetchGoogleEmail(ctx context.Context, accessToken string) (string, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})

	svc, err := oauth2v2.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return "", err
	}

	info, err := svc.Userinfo.Get().Do()
	if err != nil {
		return "", err
	}

	return info.Email, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
