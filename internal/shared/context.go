package shared

import (
	"context"
	"net/http"
)

type contextKey string

const doctorIDKey contextKey = "doctorID"

// WithDoctorID stores the doctor ID in the request context.
func WithDoctorID(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), doctorIDKey, id))
}

// DoctorIDFromContext retrieves the doctor ID from the request context.
// Returns an empty string if not set.
func DoctorIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(doctorIDKey).(string)
	return v
}
