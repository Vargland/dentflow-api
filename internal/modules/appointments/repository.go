package appointments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
)

// ErrNotFound is returned when an appointment does not exist or doesn't belong to the doctor.
var ErrNotFound = errors.New("appointment not found")

// ErrOverlap is returned when the requested time slot conflicts with an existing appointment.
var ErrOverlap = errors.New("time slot overlaps an existing appointment")

// Repository handles database operations for appointments.
type Repository struct {
	q *db.Queries
}

// NewRepository creates a new appointments Repository.
func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

// List returns appointments for a doctor within the given UTC time range.
func (r *Repository) List(ctx context.Context, doctorID string, start, end time.Time) ([]AppointmentResponse, error) {
	rows, err := r.q.ListAppointments(ctx, doctorID, start, end)
	if err != nil {
		return nil, err
	}

	results := make([]AppointmentResponse, 0, len(rows))
	for _, row := range rows {
		results = append(results, toResponse(row))
	}
	return results, nil
}

// Get returns a single appointment by ID scoped to the doctor.
func (r *Repository) Get(ctx context.Context, id, doctorID string) (AppointmentResponse, error) {
	row, err := r.q.GetAppointment(ctx, id, doctorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AppointmentResponse{}, ErrNotFound
	}
	if err != nil {
		return AppointmentResponse{}, err
	}
	return toResponse(row), nil
}

// Create inserts a new appointment after checking for overlaps.
// excludeID is empty for new appointments.
func (r *Repository) Create(ctx context.Context, doctorID string, req CreateAppointmentRequest) (AppointmentResponse, error) {
	start, end, err := parseTimes(req.StartTime, req.EndTime)
	if err != nil {
		return AppointmentResponse{}, fmt.Errorf("invalid time: %w", err)
	}

	n, err := r.q.CountOverlapping(ctx, doctorID, "", start, end)
	if err != nil {
		return AppointmentResponse{}, err
	}
	if n > 0 {
		return AppointmentResponse{}, ErrOverlap
	}

	status := req.Status
	if status == "" {
		status = "scheduled"
	}

	a, err := r.q.CreateAppointment(ctx, db.CreateAppointmentParams{
		DoctorID:        doctorID,
		PatientID:       req.PatientID,
		GoogleEventID:   nil,
		Title:           req.Title,
		StartTime:       start,
		EndTime:         end,
		DurationMinutes: req.DurationMinutes,
		Status:          status,
		Notes:           req.Notes,
	})
	if err != nil {
		return AppointmentResponse{}, err
	}

	return toResponseFromAppointment(a, nil, nil), nil
}

// Update modifies an existing appointment.
func (r *Repository) Update(ctx context.Context, id, doctorID string, req UpdateAppointmentRequest) (AppointmentResponse, error) {
	start, end, err := parseTimes(req.StartTime, req.EndTime)
	if err != nil {
		return AppointmentResponse{}, fmt.Errorf("invalid time: %w", err)
	}

	n, err := r.q.CountOverlapping(ctx, doctorID, id, start, end)
	if err != nil {
		return AppointmentResponse{}, err
	}
	if n > 0 {
		return AppointmentResponse{}, ErrOverlap
	}

	// Preserve existing google_event_id
	existing, err := r.q.GetAppointment(ctx, id, doctorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AppointmentResponse{}, ErrNotFound
	}
	if err != nil {
		return AppointmentResponse{}, err
	}

	status := req.Status
	if status == "" {
		status = "scheduled"
	}

	a, err := r.q.UpdateAppointment(ctx, db.UpdateAppointmentParams{
		ID:              id,
		DoctorID:        doctorID,
		Title:           req.Title,
		PatientID:       req.PatientID,
		GoogleEventID:   existing.GoogleEventID,
		StartTime:       start,
		EndTime:         end,
		DurationMinutes: req.DurationMinutes,
		Status:          status,
		Notes:           req.Notes,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return AppointmentResponse{}, ErrNotFound
	}
	if err != nil {
		return AppointmentResponse{}, err
	}

	return toResponseFromAppointment(a, existing.PatientNombre, existing.PatientApellido), nil
}

// UpdateGoogleEventID sets the google_event_id on an appointment.
func (r *Repository) UpdateGoogleEventID(ctx context.Context, id, doctorID string, eventID *string) error {
	existing, err := r.q.GetAppointment(ctx, id, doctorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	_, err = r.q.UpdateAppointment(ctx, db.UpdateAppointmentParams{
		ID:              id,
		DoctorID:        doctorID,
		Title:           existing.Title,
		PatientID:       existing.PatientID,
		GoogleEventID:   eventID,
		StartTime:       existing.StartTime,
		EndTime:         existing.EndTime,
		DurationMinutes: existing.DurationMinutes,
		Status:          existing.Status,
		Notes:           existing.Notes,
	})
	return err
}

// Delete removes an appointment.
func (r *Repository) Delete(ctx context.Context, id, doctorID string) error {
	return r.q.DeleteAppointment(ctx, id, doctorID)
}

// GetRaw returns the raw DB row for an appointment (used by the handler for GCal sync).
func (r *Repository) GetRaw(ctx context.Context, id, doctorID string) (db.AppointmentRow, error) {
	row, err := r.q.GetAppointment(ctx, id, doctorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.AppointmentRow{}, ErrNotFound
	}
	return row, err
}

// parseTimes parses RFC3339 start and end strings.
func parseTimes(startStr, endStr string) (time.Time, time.Time, error) {
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, end, nil
}

// toResponse converts an AppointmentRow to an AppointmentResponse.
func toResponse(row db.AppointmentRow) AppointmentResponse {
	return toResponseFromAppointment(row.Appointment, row.PatientNombre, row.PatientApellido)
}

func toResponseFromAppointment(a db.Appointment, nombre, apellido *string) AppointmentResponse {
	resp := AppointmentResponse{
		ID:              a.ID,
		DoctorID:        a.DoctorID,
		PatientID:       a.PatientID,
		GoogleEventID:   a.GoogleEventID,
		Title:           a.Title,
		StartTime:       a.StartTime,
		EndTime:         a.EndTime,
		DurationMinutes: a.DurationMinutes,
		Status:          a.Status,
		Notes:           a.Notes,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}

	if nombre != nil || apellido != nil {
		parts := []string{}
		if nombre != nil {
			parts = append(parts, *nombre)
		}
		if apellido != nil {
			parts = append(parts, *apellido)
		}
		name := strings.Join(parts, " ")
		resp.PatientName = &name
	}

	return resp
}
