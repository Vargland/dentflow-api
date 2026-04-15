package shared

import "net/http"

// APIError represents a structured API error response.
type APIError struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// ErrorResponse writes a structured JSON error to the response writer.
func ErrorResponse(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, APIError{
		Error: errorBody{
			Code:    code,
			Message: message,
			Status:  status,
		},
	})
}

// NotFound writes a 404 JSON error.
func NotFound(w http.ResponseWriter, resource string) {
	ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", resource+" not found")
}

// Unauthorized writes a 401 JSON error.
func Unauthorized(w http.ResponseWriter) {
	ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

// Forbidden writes a 403 JSON error.
func Forbidden(w http.ResponseWriter) {
	ErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "access denied")
}

// BadRequest writes a 400 JSON error.
func BadRequest(w http.ResponseWriter, message string) {
	ErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// ValidationError writes a 422 JSON error.
func ValidationError(w http.ResponseWriter, message string) {
	ErrorResponse(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
}

// InternalError writes a 500 JSON error.
func InternalError(w http.ResponseWriter) {
	ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}
