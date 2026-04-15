//go:build ignore

// gen_token generates a test JWT signed with the same AUTH_SECRET as Auth.js.
// Usage: go run tools/gen_token.go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	secret := os.Getenv("AUTH_SECRET")
	if secret == "" {
		fmt.Fprintln(os.Stderr, "AUTH_SECRET not set")
		os.Exit(1)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "test-doctor-id-001",
		"email": "psi.germanr@gmail.com",
		"name":  "Dr. Test",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(signed)
}
