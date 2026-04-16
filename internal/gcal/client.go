// Package gcal provides a Google Calendar API client factory.
package gcal

import (
	"context"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
)

// OAuthConfig returns the OAuth2 config for Google Calendar.
func OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URI"),
		Scopes: []string{
			calendar.CalendarEventsScope,
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
}

// AuthURL returns the Google OAuth consent page URL.
func AuthURL(state string) string {
	return OAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// TokenFromCode exchanges an auth code for an OAuth2 token.
func TokenFromCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return OAuthConfig().Exchange(ctx, code)
}

// NewService creates an authenticated Google Calendar service from stored token.
func NewService(ctx context.Context, tok db.GoogleToken) (*calendar.Service, *oauth2.Token, error) {
	cfg := OAuthConfig()

	oauthTok := &oauth2.Token{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
		TokenType:    "Bearer",
	}

	ts := cfg.TokenSource(ctx, oauthTok)

	// Trigger a refresh if needed and get the (possibly new) token
	newTok, err := ts.Token()
	if err != nil {
		return nil, nil, err
	}

	svc, err := calendar.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, nil, err
	}

	return svc, newTok, nil
}

// CreateEvent creates a Google Calendar event and returns its ID.
func CreateEvent(svc *calendar.Service, calendarID, title, notes, timezone string, start, end time.Time) (string, error) {
	description := notes
	if description == "" {
		description = "Turno creado desde DentFlow"
	}

	event := &calendar.Event{
		Summary:     title,
		Description: description,
		Start:       &calendar.EventDateTime{DateTime: start.Format(time.RFC3339), TimeZone: timezone},
		End:         &calendar.EventDateTime{DateTime: end.Format(time.RFC3339), TimeZone: timezone},
		ColorId:     "1",
	}

	created, err := svc.Events.Insert(calendarID, event).Do()
	if err != nil {
		return "", err
	}

	return created.Id, nil
}

// UpdateEvent updates an existing Google Calendar event.
func UpdateEvent(svc *calendar.Service, calendarID, eventID, title, notes, timezone string, start, end time.Time) error {
	description := notes
	if description == "" {
		description = "Turno creado desde DentFlow"
	}

	event := &calendar.Event{
		Summary:     title,
		Description: description,
		Start:       &calendar.EventDateTime{DateTime: start.Format(time.RFC3339), TimeZone: timezone},
		End:         &calendar.EventDateTime{DateTime: end.Format(time.RFC3339), TimeZone: timezone},
	}

	_, err := svc.Events.Update(calendarID, eventID, event).Do()

	return err
}

// DeleteEvent removes a Google Calendar event.
func DeleteEvent(svc *calendar.Service, calendarID, eventID string) error {
	return svc.Events.Delete(calendarID, eventID).Do()
}
