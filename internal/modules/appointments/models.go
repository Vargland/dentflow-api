package appointments

import "time"

// AppointmentResponse is the API representation of an appointment.
type AppointmentResponse struct {
	ID              string    `json:"id"`
	DoctorID        string    `json:"doctor_id"`
	PatientID       *string   `json:"patient_id"`
	PatientName     *string   `json:"patient_name,omitempty"`
	GoogleEventID   *string   `json:"google_event_id,omitempty"`
	Title           string    `json:"title"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	DurationMinutes int       `json:"duration_minutes"`
	Status          string    `json:"status"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateAppointmentRequest is the body for POST /api/v1/appointments.
type CreateAppointmentRequest struct {
	PatientID       *string `json:"patient_id"`
	Title           string  `json:"title"`
	StartTime       string  `json:"start_time"` // RFC3339 UTC
	EndTime         string  `json:"end_time"`   // RFC3339 UTC
	DurationMinutes int     `json:"duration_minutes"`
	Status          string  `json:"status"`
	Notes           *string `json:"notes"`
	// AllowOverlap skips the overlap check — used for emergency over-bookings.
	AllowOverlap bool `json:"allow_overlap"`
}

// UpdateAppointmentRequest is the body for PUT /api/v1/appointments/:id.
type UpdateAppointmentRequest struct {
	PatientID       *string `json:"patient_id"`
	Title           string  `json:"title"`
	StartTime       string  `json:"start_time"` // RFC3339 UTC
	EndTime         string  `json:"end_time"`   // RFC3339 UTC
	DurationMinutes int     `json:"duration_minutes"`
	Status          string  `json:"status"`
	Notes           *string `json:"notes"`
	// AllowOverlap skips the overlap check — used for emergency over-bookings.
	AllowOverlap bool `json:"allow_overlap"`
}
