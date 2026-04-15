package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
)

// ErrEmailTaken is returned when registering with an already-used email.
var ErrEmailTaken = errors.New("email already registered")

// ErrNotFound is returned when no user matches the given credentials.
var ErrNotFound = errors.New("user not found")

// Repository handles user persistence.
type Repository struct {
	q *db.Queries
}

// NewRepository constructs a Repository from the shared db pool.
func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

// Create inserts a new user. Returns ErrEmailTaken if the email is already in use.
func (r *Repository) Create(ctx context.Context, email, name, hashedPassword string) (db.User, error) {
	user, err := r.q.CreateUser(ctx, email, name, hashedPassword)
	if err != nil {
		var pgErr *pgconn.PgError
		// 23505 = unique_violation
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return db.User{}, ErrEmailTaken
		}

		return db.User{}, err
	}

	return user, nil
}

// GetByEmail fetches a user by email. Returns ErrNotFound if missing.
func (r *Repository) GetByEmail(ctx context.Context, email string) (db.User, error) {
	user, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, ErrNotFound
	}

	return user, err
}
