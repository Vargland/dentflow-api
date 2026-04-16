package appointments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
	"github.com/psi-germanr/dentflow-api/internal/gcal"
	"github.com/psi-germanr/dentflow-api/internal/shared"
)

// Handler handles HTTP requests for the appointments module.
type Handler struct {
	repo      *Repository
	settingsQ *db.Queries
}

// NewHandler creates a new appointments Handler.
func NewHandler(repo *Repository, settingsQ *db.Queries) *Handler {
	return &Handler{repo: repo, settingsQ: settingsQ}
}

// RegisterRoutes attaches appointment routes to the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

// List handles GET /api/v1/appointments?start=...&end=...
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		shared.BadRequest(w, "start and end query params are required")
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		shared.BadRequest(w, "invalid start date format (RFC3339 required)")
		return
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		shared.BadRequest(w, "invalid end date format (RFC3339 required)")
		return
	}

	items, err := h.repo.List(r.Context(), doctorID, start, end)
	if err != nil {
		shared.InternalError(w)
		return
	}

	if items == nil {
		items = []AppointmentResponse{}
	}

	shared.JSON(w, http.StatusOK, items)
}

// Get handles GET /api/v1/appointments/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	appt, err := h.repo.Get(r.Context(), id, doctorID)
	if errors.Is(err, ErrNotFound) {
		shared.NotFound(w, "appointment")
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusOK, appt)
}

// Create handles POST /api/v1/appointments
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())

	var req CreateAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	if err := validateCreate(req); err != nil {
		shared.ValidationError(w, err.Error())
		return
	}

	appt, err := h.repo.Create(r.Context(), doctorID, req)
	if errors.Is(err, ErrOverlap) {
		shared.ErrorResponse(w, http.StatusConflict, "OVERLAP", "time slot overlaps an existing appointment")
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	// Best-effort Google Calendar sync (non-blocking)
	go h.syncCreate(context.Background(), doctorID, appt)

	shared.JSON(w, http.StatusCreated, appt)
}

// Update handles PUT /api/v1/appointments/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req UpdateAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	if err := validateUpdate(req); err != nil {
		shared.ValidationError(w, err.Error())
		return
	}

	// Fetch existing row before update to keep the google_event_id
	existing, _ := h.repo.GetRaw(r.Context(), id, doctorID)

	appt, err := h.repo.Update(r.Context(), id, doctorID, req)
	if errors.Is(err, ErrNotFound) {
		shared.NotFound(w, "appointment")
		return
	}
	if errors.Is(err, ErrOverlap) {
		shared.ErrorResponse(w, http.StatusConflict, "OVERLAP", "time slot overlaps an existing appointment")
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	go h.syncUpdate(context.Background(), doctorID, appt, existing)

	shared.JSON(w, http.StatusOK, appt)
}

// Delete handles DELETE /api/v1/appointments/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	existing, _ := h.repo.GetRaw(r.Context(), id, doctorID)

	if err := h.repo.Delete(r.Context(), id, doctorID); err != nil {
		shared.InternalError(w)
		return
	}

	go h.syncDelete(context.Background(), doctorID, existing)

	shared.NoContent(w)
}

// ---------- Google Calendar sync helpers ----------

func (h *Handler) syncCreate(ctx context.Context, doctorID string, appt AppointmentResponse) {
	tok, err := h.settingsQ.GetGoogleToken(ctx, doctorID)
	if err != nil {
		return // Calendar not connected
	}

	timezone := h.getTimezone(ctx, doctorID)

	svc, newTok, err := gcal.NewService(ctx, tok)
	if err != nil {
		return
	}

	if newTok.AccessToken != tok.AccessToken {
		h.persistRefreshedToken(ctx, doctorID, tok, newTok.AccessToken, newTok.Expiry)
	}

	var notes string
	if appt.Notes != nil {
		notes = *appt.Notes
	}

	eventID, err := gcal.CreateEvent(svc, tok.CalendarID, appt.Title, notes, timezone, appt.StartTime, appt.EndTime)
	if err != nil {
		return
	}

	_ = h.repo.UpdateGoogleEventID(ctx, appt.ID, doctorID, &eventID)
}

func (h *Handler) syncUpdate(ctx context.Context, doctorID string, appt AppointmentResponse, existing db.AppointmentRow) {
	tok, err := h.settingsQ.GetGoogleToken(ctx, doctorID)
	if err != nil {
		return
	}

	if existing.GoogleEventID == nil {
		return // Was never synced to GCal
	}

	timezone := h.getTimezone(ctx, doctorID)

	svc, newTok, err := gcal.NewService(ctx, tok)
	if err != nil {
		return
	}

	if newTok.AccessToken != tok.AccessToken {
		h.persistRefreshedToken(ctx, doctorID, tok, newTok.AccessToken, newTok.Expiry)
	}

	var notes string
	if appt.Notes != nil {
		notes = *appt.Notes
	}

	_ = gcal.UpdateEvent(svc, tok.CalendarID, *existing.GoogleEventID, appt.Title, notes, timezone, appt.StartTime, appt.EndTime)
}

func (h *Handler) syncDelete(ctx context.Context, doctorID string, existing db.AppointmentRow) {
	tok, err := h.settingsQ.GetGoogleToken(ctx, doctorID)
	if err != nil {
		return
	}

	if existing.GoogleEventID == nil {
		return
	}

	svc, newTok, err := gcal.NewService(ctx, tok)
	if err != nil {
		return
	}

	if newTok.AccessToken != tok.AccessToken {
		h.persistRefreshedToken(ctx, doctorID, tok, newTok.AccessToken, newTok.Expiry)
	}

	_ = gcal.DeleteEvent(svc, tok.CalendarID, *existing.GoogleEventID)
}

func (h *Handler) getTimezone(ctx context.Context, doctorID string) string {
	s, err := h.settingsQ.GetUserSettings(ctx, doctorID)
	if err != nil {
		return "America/Argentina/Buenos_Aires"
	}
	return s.Timezone
}

func (h *Handler) persistRefreshedToken(ctx context.Context, doctorID string, tok db.GoogleToken, newAccessToken string, newExpiry time.Time) {
	_, _ = h.settingsQ.UpsertGoogleToken(ctx, db.UpsertGoogleTokenParams{
		DoctorID:     doctorID,
		AccessToken:  newAccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       newExpiry,
		CalendarID:   tok.CalendarID,
	})
}

// ---------- Validation ----------

func validateCreate(req CreateAppointmentRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Title, validation.Required, validation.Length(1, 200)),
		validation.Field(&req.StartTime, validation.Required),
		validation.Field(&req.EndTime, validation.Required),
		validation.Field(&req.DurationMinutes, validation.Min(1)),
	)
}

func validateUpdate(req UpdateAppointmentRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Title, validation.Required, validation.Length(1, 200)),
		validation.Field(&req.StartTime, validation.Required),
		validation.Field(&req.EndTime, validation.Required),
		validation.Field(&req.DurationMinutes, validation.Min(1)),
	)
}
