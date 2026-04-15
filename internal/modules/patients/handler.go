package patients

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/psi-germanr/dentflow-api/internal/shared"
)

// Handler handles HTTP requests for the patients module.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new patients Handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes attaches patient routes to the given chi router.
// All routes require the auth middleware to be applied at the parent router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/odontogram", h.GetOdontogram)
	r.Put("/{id}/odontogram", h.SaveOdontogram)
}

// List handles GET /patients
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())

	var query *string
	if q := r.URL.Query().Get("q"); q != "" {
		query = &q
	}

	patients, err := h.repo.List(r.Context(), doctorID, query)
	if err != nil {
		shared.InternalError(w)
		return
	}

	// Return empty array instead of null
	if patients == nil {
		patients = []PatientListItem{}
	}

	shared.JSON(w, http.StatusOK, patients)
}

// Get handles GET /patients/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	patient, err := h.repo.Get(r.Context(), id, doctorID)
	if errors.Is(err, ErrNotFound) {
		shared.NotFound(w, "patient")
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusOK, patient)
}

// Create handles POST /patients
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())

	var req CreatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	if err := validateCreate(req); err != nil {
		shared.ValidationError(w, err.Error())
		return
	}

	patient, err := h.repo.Create(r.Context(), doctorID, req)
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusCreated, patient)
}

// Update handles PUT /patients/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req UpdatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	patient, err := h.repo.Update(r.Context(), id, doctorID, req)
	if errors.Is(err, ErrNotFound) {
		shared.NotFound(w, "patient")
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusOK, patient)
}

// Delete handles DELETE /patients/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(r.Context(), id, doctorID); err != nil {
		shared.InternalError(w)
		return
	}

	shared.NoContent(w)
}

// GetOdontogram handles GET /patients/:id/odontogram
func (h *Handler) GetOdontogram(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	resp, err := h.repo.GetOdontogram(r.Context(), id, doctorID)
	if errors.Is(err, ErrNotFound) {
		shared.NotFound(w, "patient")
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusOK, resp)
}

// SaveOdontogram handles PUT /patients/:id/odontogram
func (h *Handler) SaveOdontogram(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req SaveOdontogramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	if len(req.Data) == 0 {
		shared.ValidationError(w, "data is required")
		return
	}

	resp, err := h.repo.SaveOdontogram(r.Context(), id, doctorID, req.Data)
	if errors.Is(err, ErrNotFound) {
		shared.NotFound(w, "patient")
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusOK, resp)
}

// validateCreate validates the required fields on a CreatePatientRequest.
func validateCreate(req CreatePatientRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Nombre,   validation.Required, validation.Length(1, 100)),
		validation.Field(&req.Apellido, validation.Required, validation.Length(1, 100)),
		validation.Field(&req.Email,    is.Email),
		validation.Field(&req.Dni,      validation.Length(0, 20)),
	)
}
