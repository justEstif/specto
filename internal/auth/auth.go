package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/database"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
)

type contextKey string

const userContextKey contextKey = "user"

// UserFromContext retrieves the authenticated user from the request context.
func UserFromContext(ctx context.Context) (*database.User, bool) {
	u, ok := ctx.Value(userContextKey).(*database.User)
	return u, ok
}

// ContextWithUser stores the user in the context.
func ContextWithUser(ctx context.Context, user *database.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// Register creates a new user with email/password.
func Register(ctx context.Context, q *database.Queries, email, displayName, password string) (*database.User, error) {
	// Check if email is already taken
	_, err := q.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user, err := q.CreateUserWithPassword(ctx, database.CreateUserWithPasswordParams{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: pgtype.Text{String: string(hash), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return &user, nil
}

// Login authenticates a user by email/password.
func Login(ctx context.Context, q *database.Queries, email, password string) (*database.User, error) {
	user, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.PasswordHash.Valid {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

// GetUserByID looks up a user by their ID (for session restoration).
func GetUserByID(ctx context.Context, q *database.Queries, id pgtype.UUID) (*database.User, error) {
	user, err := q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
