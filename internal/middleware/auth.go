// Package middleware provides HTTP middleware for the DentFlow API.
package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/psi-germanr/dentflow-api/internal/shared"
)

// Auth returns a middleware that validates the Auth.js JWT and injects
// the doctor ID (JWT sub claim) into the request context.
//
// The token must be present as: Authorization: Bearer <token>
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				shared.Unauthorized(w)
				return
			}

			raw := strings.TrimPrefix(header, "Bearer ")

			token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				shared.Unauthorized(w)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				shared.Unauthorized(w)
				return
			}

			sub, ok := claims["sub"].(string)
			if !ok || sub == "" {
				shared.Unauthorized(w)
				return
			}

			next.ServeHTTP(w, shared.WithDoctorID(r, sub))
		})
	}
}
