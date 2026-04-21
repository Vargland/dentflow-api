package appointments

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/jackc/pgx/v5"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
	"github.com/psi-germanr/dentflow-api/internal/email"
	"github.com/psi-germanr/dentflow-api/internal/gcal"
	"github.com/psi-germanr/dentflow-api/internal/gmail"
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
	r.Post("/{id}/send-invite", h.SendInvite)
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

// SendInvite handles POST /api/v1/appointments/:id/send-invite
// Sends (or resends) the email invitation to the linked patient via the doctor's Gmail account.
func (h *Handler) SendInvite(w http.ResponseWriter, r *http.Request) {
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

	if appt.PatientID == nil {
		shared.ErrorResponse(w, http.StatusUnprocessableEntity, "NO_PATIENT", "appointment has no linked patient")
		return
	}

	tok, err := h.settingsQ.GetGoogleToken(r.Context(), doctorID)
	if err != nil {
		shared.ErrorResponse(w, http.StatusUnprocessableEntity, "GMAIL_NOT_CONNECTED", "Google account not connected — connect it in Settings")
		return
	}

	patient, err := h.settingsQ.GetPatient(r.Context(), *appt.PatientID, doctorID)
	if err != nil {
		shared.NotFound(w, "patient")
		return
	}

	if patient.Email == nil || *patient.Email == "" {
		shared.ErrorResponse(w, http.StatusUnprocessableEntity, "NO_EMAIL", "patient has no email address")
		return
	}

	settings, err := h.settingsQ.GetUserSettings(r.Context(), doctorID)
	if err != nil {
		shared.InternalError(w)
		return
	}

	timezone := settings.Timezone
	if timezone == "" {
		timezone = "America/Argentina/Buenos_Aires"
	}

	lang := settings.EmailLanguage
	if lang == "" {
		lang = "es"
	}

	loc, locErr := time.LoadLocation(timezone)
	if locErr != nil {
		loc = time.UTC
	}

	localStart := appt.StartTime.In(loc)
	localEnd := appt.EndTime.In(loc)

	patientName := patient.Nombre
	if patient.Apellido != "" {
		patientName = patient.Nombre + " " + patient.Apellido
	}

	params := email.InviteParams{
		PatientName:   patientName,
		PatientEmail:  *patient.Email,
		DoctorName:    settings.DoctorName,
		ClinicAddress: settings.ClinicAddress,
		ClinicPhone:   settings.ClinicPhone,
		Title:         appt.Title,
		StartTime:     localStart,
		EndTime:       localEnd,
		StartUTC:      appt.StartTime,
		EndUTC:        appt.EndTime,
		Duration:      appt.DurationMinutes,
		Language:      lang,
	}

	refreshed, err := email.SendInvite(r.Context(), tok, params)
	if err != nil {
		if errors.Is(err, gmail.ErrInsufficientScope) {
			shared.ErrorResponse(w, http.StatusForbidden, "GMAIL_SCOPE_MISSING", "Google account requires reconnection to grant email permissions")
			return
		}
		log.Printf("SendInvite handler: Gmail error: %v", err)
		shared.ErrorResponse(w, http.StatusBadGateway, "EMAIL_FAILED", err.Error())
		return
	}

	if refreshed.Changed {
		h.persistRefreshedToken(r.Context(), doctorID, tok, refreshed.AccessToken, refreshed.Expiry)
	}

	shared.JSON(w, http.StatusOK, map[string]string{"sent_to": *patient.Email})
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

	// Best-effort patient email invite (non-blocking)
	go h.sendInvite(context.Background(), doctorID, appt)

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
		log.Printf("syncCreate: no google token for doctor %s: %v", doctorID, err)
		return // Calendar not connected
	}
	log.Printf("syncCreate: found token for doctor %s, calendar %s", doctorID, tok.CalendarID)

	timezone := h.getTimezone(ctx, doctorID)

	svc, newTok, err := gcal.NewService(ctx, tok)
	if err != nil {
		log.Printf("syncCreate: NewService error: %v", err)
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
		log.Printf("syncCreate: CreateEvent error: %v", err)
		return
	}

	log.Printf("syncCreate: created event %s", eventID)
	_ = h.repo.UpdateGoogleEventID(ctx, appt.ID, doctorID, &eventID)
}

func (h *Handler) sendInvite(ctx context.Context, doctorID string, appt AppointmentResponse) {
	if appt.PatientID == nil {
		return
	}

	tok, err := h.settingsQ.GetGoogleToken(ctx, doctorID)
	if err != nil {
		log.Printf("sendInvite: no google token for doctor %s, skipping email", doctorID)
		return
	}

	patient, err := h.settingsQ.GetPatient(ctx, *appt.PatientID, doctorID)
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		log.Printf("sendInvite: could not fetch patient %s: %v", *appt.PatientID, err)
		return
	}

	if patient.Email == nil || *patient.Email == "" {
		log.Printf("sendInvite: patient %s has no email, skipping", *appt.PatientID)
		return
	}

	settings, err := h.settingsQ.GetUserSettings(ctx, doctorID)
	if err != nil {
		log.Printf("sendInvite: could not fetch settings for doctor %s: %v", doctorID, err)
		return
	}

	timezone := settings.Timezone
	if timezone == "" {
		timezone = "America/Argentina/Buenos_Aires"
	}

	lang := settings.EmailLanguage
	if lang == "" {
		lang = "es"
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	localStart := appt.StartTime.In(loc)
	localEnd := appt.EndTime.In(loc)

	patientName := patient.Nombre
	if patient.Apellido != "" {
		patientName = patient.Nombre + " " + patient.Apellido
	}

	params := email.InviteParams{
		PatientName:   patientName,
		PatientEmail:  *patient.Email,
		DoctorName:    settings.DoctorName,
		ClinicAddress: settings.ClinicAddress,
		ClinicPhone:   settings.ClinicPhone,
		Title:         appt.Title,
		StartTime:     localStart,
		EndTime:       localEnd,
		StartUTC:      appt.StartTime,
		EndUTC:        appt.EndTime,
		Duration:      appt.DurationMinutes,
		Language:      lang,
	}

	refreshed, err := email.SendInvite(ctx, tok, params)
	if err != nil {
		log.Printf("sendInvite: Gmail error for patient %s: %v", *appt.PatientID, err)
		return
	}

	if refreshed.Changed {
		h.persistRefreshedToken(ctx, doctorID, tok, refreshed.AccessToken, refreshed.Expiry)
	}

	log.Printf("sendInvite: invitation sent to %s for appointment %s", *patient.Email, appt.ID)
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
