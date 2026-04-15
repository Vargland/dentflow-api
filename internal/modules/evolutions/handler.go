package evolutions

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/psi-germanr/dentflow-api/internal/shared"
)

// Handler handles HTTP requests for the evolutions module.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new evolutions Handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes attaches evolution routes under /patients/{patientId}/evolutions.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{eid}", h.Update)
	r.Delete("/{eid}", h.Delete)
}

// List handles GET /patients/:id/evolutions
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	patientID := chi.URLParam(r, "id")

	evolutions, err := h.repo.List(r.Context(), patientID, doctorID)
	if err != nil {
		shared.InternalError(w)
		return
	}
	if evolutions == nil {
		evolutions = []EvolutionResponse{}
	}
	shared.JSON(w, http.StatusOK, evolutions)
}

// Create handles POST /patients/:id/evolutions
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	patientID := chi.URLParam(r, "id")

	var req CreateEvolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	if err := validateCreate(req); err != nil {
		shared.ValidationError(w, err.Error())
		return
	}

	ev, err := h.repo.Create(r.Context(), patientID, doctorID, req)
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusCreated, ev)
}

// Update handles PUT /patients/:id/evolutions/:eid
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	eid := chi.URLParam(r, "eid")

	var req UpdateEvolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	ev, err := h.repo.Update(r.Context(), eid, doctorID, req)
	if errors.Is(err, ErrNotFound) {
		shared.NotFound(w, "evolution")
		return
	}
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusOK, ev)
}

// Delete handles DELETE /patients/:id/evolutions/:eid
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	doctorID := shared.DoctorIDFromContext(r.Context())
	eid := chi.URLParam(r, "eid")

	if err := h.repo.Delete(r.Context(), eid, doctorID); err != nil {
		shared.InternalError(w)
		return
	}
	shared.NoContent(w)
}

// validateCreate validates required fields on CreateEvolutionRequest.
func validateCreate(req CreateEvolutionRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Descripcion, validation.Required, validation.Length(1, 2000)),
	)
}
