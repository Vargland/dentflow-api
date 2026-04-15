package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/golang-jwt/jwt/v5"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/psi-germanr/dentflow-api/internal/shared"
)

// Handler holds dependencies for the auth HTTP handlers.
type Handler struct {
	repo *Repository
}

// NewHandler constructs a Handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes mounts auth endpoints on r (no JWT required).
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
}

// Register handles POST /auth/register.
// Creates a new user, hashes the password, returns a signed JWT.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if err := validateRegister(req); err != nil {
		shared.ValidationError(w, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		shared.InternalError(w)
		return
	}

	user, err := h.repo.Create(r.Context(), req.Email, req.Name, string(hash))
	if errors.Is(err, ErrEmailTaken) {
		shared.ErrorResponse(w, http.StatusConflict, "EMAIL_TAKEN", "email already registered")
		return
	}

	if err != nil {
		shared.InternalError(w)
		return
	}

	token, err := mintToken(user.ID, user.Email, user.Name)
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusCreated, TokenResponse{
		Token: token,
		Name:  user.Name,
		Email: user.Email,
	})
}

// Login handles POST /auth/login.
// Validates credentials and returns a signed JWT.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, "invalid JSON body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.Password == "" {
		shared.ErrorResponse(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}

	user, err := h.repo.GetByEmail(r.Context(), req.Email)
	if errors.Is(err, ErrNotFound) {
		shared.ErrorResponse(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}

	if err != nil {
		shared.InternalError(w)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		shared.ErrorResponse(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}

	token, err := mintToken(user.ID, user.Email, user.Name)
	if err != nil {
		shared.InternalError(w)
		return
	}

	shared.JSON(w, http.StatusOK, TokenResponse{
		Token: token,
		Name:  user.Name,
		Email: user.Email,
	})
}

// mintToken creates a signed HS256 JWT compatible with the Auth.js frontend.
func mintToken(sub, email, name string) (string, error) {
	secret := os.Getenv("AUTH_SECRET")

	claims := jwt.MapClaims{
		"sub":   sub,
		"email": email,
		"name":  name,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// validateRegister checks that all required fields are present and valid.
func validateRegister(req RegisterRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Email, validation.Required, is.Email),
		validation.Field(&req.Name, validation.Required, validation.Length(2, 100)),
		validation.Field(&req.Password, validation.Required, validation.Length(8, 128)),
	)
}
