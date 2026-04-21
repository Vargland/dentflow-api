// Package gmail provides email sending via the Gmail API using the doctor's OAuth token.
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/psi-germanr/dentflow-api/internal/gcal"
	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
)

// RefreshedToken carries the possibly-updated token after a Gmail API call.
type RefreshedToken struct {
	AccessToken string
	Expiry      time.Time
	Changed     bool
}

// SendEmail sends an HTML email on behalf of the doctor using their Gmail account.
// It returns a RefreshedToken so callers can persist a new access token if it was refreshed.
func SendEmail(ctx context.Context, tok db.GoogleToken, to, subject, htmlBody string) (RefreshedToken, error) {
	cfg := gcal.OAuthConfig()

	oauthTok := &oauth2.Token{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
		TokenType:    "Bearer",
	}

	ts := cfg.TokenSource(ctx, oauthTok)

	newTok, err := ts.Token()
	if err != nil {
		return RefreshedToken{}, fmt.Errorf("gmail: token refresh: %w", err)
	}

	refreshed := RefreshedToken{
		AccessToken: newTok.AccessToken,
		Expiry:      newTok.Expiry,
		Changed:     newTok.AccessToken != tok.AccessToken,
	}

	svc, err := gmail.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return refreshed, fmt.Errorf("gmail: new service: %w", err)
	}

	raw := buildRaw(tok.CalendarID, to, subject, htmlBody)

	msg := &gmail.Message{Raw: raw}
	if _, err := svc.Users.Messages.Send("me", msg).Do(); err != nil {
		log.Printf("gmail.SendEmail: send error to=%q: %v", to, err)
		return refreshed, fmt.Errorf("gmail: send: %w", err)
	}

	log.Printf("gmail.SendEmail: sent to=%q subject=%q", to, subject)
	return refreshed, nil
}

// buildRaw constructs a base64url-encoded RFC 2822 message.
func buildRaw(from, to, subject, htmlBody string) string {
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, htmlBody,
	)
	return base64.URLEncoding.EncodeToString([]byte(msg))
}
