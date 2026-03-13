package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/justestif/specto/internal/core"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
)

type contextKey string

const userContextKey contextKey = "user"

// UserFromContext retrieves the authenticated user from the request context.
func UserFromContext(ctx context.Context) (*core.UserInfo, bool) {
	u, ok := ctx.Value(userContextKey).(*core.UserInfo)
	return u, ok
}

// ContextWithUser stores the user in the context.
func ContextWithUser(ctx context.Context, user *core.UserInfo) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// Service handles authentication operations using core store interfaces.
type Service struct {
	Users    core.UserStore
	Sessions *SessionManager
}

// NewService creates an auth Service.
func NewService(users core.UserStore, sessions *SessionManager) *Service {
	return &Service{Users: users, Sessions: sessions}
}

// Register creates a new user with email/password.
func (s *Service) Register(ctx context.Context, email, displayName, password string) (*core.UserInfo, error) {
	// Check if email is already taken
	_, err := s.Users.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user, err := s.Users.CreateWithPassword(ctx, email, displayName, string(hash))
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return user, nil
}

// Login authenticates a user by email/password.
func (s *Service) Login(ctx context.Context, email, password string) (*core.UserInfo, error) {
	user, err := s.Users.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.PasswordHash == nil || *user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

// GetUserByID looks up a user by their ID (for session restoration).
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*core.UserInfo, error) {
	return s.Users.GetByID(ctx, id)
}
