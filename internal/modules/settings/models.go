package settings

// SettingsResponse is returned by GET /api/v1/settings.
type SettingsResponse struct {
	Timezone          string `json:"timezone"`
	CalendarConnected bool   `json:"calendarConnected"`
	CalendarEmail     string `json:"calendarEmail,omitempty"`
}

// UpdateSettingsRequest is the body for PUT /api/v1/settings.
type UpdateSettingsRequest struct {
	Timezone string `json:"timezone"`
}
