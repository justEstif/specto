package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/database"
)

func TestUserStore_GetByID(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC()

	mock := &mockQuerier{
		getUserByIDFn: func(_ context.Context, uid pgtype.UUID) (database.User, error) {
			if pgxToUUID(uid) != id {
				t.Errorf("ID: want %v, got %v", id, pgxToUUID(uid))
			}
			return database.User{
				ID:           uuidToPgx(id),
				Email:        "test@example.com",
				DisplayName:  "Test User",
				AuthProvider: "email",
				AuthSubject:  "test@example.com",
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewUserStore(mock)
	info, err := store.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != id {
		t.Errorf("ID: want %v, got %v", id, info.ID)
	}
	if info.Email != "test@example.com" {
		t.Errorf("Email: want 'test@example.com', got %q", info.Email)
	}
}

func TestUserStore_GetByEmail(t *testing.T) {
	now := time.Now().UTC()

	mock := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, email string) (database.User, error) {
			if email != "user@test.com" {
				t.Errorf("Email: want 'user@test.com', got %q", email)
			}
			return database.User{
				ID:           uuidToPgx(uuid.New()),
				Email:        email,
				DisplayName:  "User",
				AuthProvider: "email",
				AuthSubject:  email,
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewUserStore(mock)
	info, err := store.GetByEmail(context.Background(), "user@test.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Email != "user@test.com" {
		t.Errorf("Email: want 'user@test.com', got %q", info.Email)
	}
}

func TestUserStore_GetByAuth(t *testing.T) {
	now := time.Now().UTC()

	mock := &mockQuerier{
		getUserByAuthFn: func(_ context.Context, arg database.GetUserByAuthParams) (database.User, error) {
			if arg.AuthProvider != "github" {
				t.Errorf("Provider: want 'github', got %q", arg.AuthProvider)
			}
			if arg.AuthSubject != "12345" {
				t.Errorf("Subject: want '12345', got %q", arg.AuthSubject)
			}
			return database.User{
				ID:           uuidToPgx(uuid.New()),
				Email:        "github@test.com",
				DisplayName:  "GitHub User",
				AuthProvider: "github",
				AuthSubject:  "12345",
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewUserStore(mock)
	info, err := store.GetByAuth(context.Background(), "github", "12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.AuthProvider != "github" {
		t.Errorf("AuthProvider: want 'github', got %q", info.AuthProvider)
	}
}

func TestUserStore_Create(t *testing.T) {
	now := time.Now().UTC()
	avatarURL := "https://example.com/avatar.png"

	mock := &mockQuerier{
		createUserFn: func(_ context.Context, arg database.CreateUserParams) (database.User, error) {
			if arg.Email != "new@example.com" {
				t.Errorf("Email: want 'new@example.com', got %q", arg.Email)
			}
			if arg.DisplayName != "New User" {
				t.Errorf("DisplayName: want 'New User', got %q", arg.DisplayName)
			}
			if !arg.AvatarUrl.Valid || arg.AvatarUrl.String != avatarURL {
				t.Errorf("AvatarUrl: want %q, got %+v", avatarURL, arg.AvatarUrl)
			}
			if arg.AuthProvider != "google" {
				t.Errorf("AuthProvider: want 'google', got %q", arg.AuthProvider)
			}
			return database.User{
				ID:           uuidToPgx(uuid.New()),
				Email:        arg.Email,
				DisplayName:  arg.DisplayName,
				AvatarUrl:    arg.AvatarUrl,
				AuthProvider: arg.AuthProvider,
				AuthSubject:  arg.AuthSubject,
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewUserStore(mock)
	info, err := store.Create(context.Background(), "new@example.com", "New User", &avatarURL, "google", "goog-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Email != "new@example.com" {
		t.Errorf("Email: want 'new@example.com', got %q", info.Email)
	}
	if info.AvatarURL == nil || *info.AvatarURL != avatarURL {
		t.Errorf("AvatarURL: want %q, got %v", avatarURL, info.AvatarURL)
	}
}

func TestUserStore_CreateWithPassword(t *testing.T) {
	now := time.Now().UTC()

	mock := &mockQuerier{
		createUserWithPasswordFn: func(_ context.Context, arg database.CreateUserWithPasswordParams) (database.User, error) {
			if arg.Email != "pwd@example.com" {
				t.Errorf("Email: want 'pwd@example.com', got %q", arg.Email)
			}
			if !arg.PasswordHash.Valid || arg.PasswordHash.String != "$2a$10$hash" {
				t.Errorf("PasswordHash: want '$2a$10$hash', got %+v", arg.PasswordHash)
			}
			return database.User{
				ID:           uuidToPgx(uuid.New()),
				Email:        arg.Email,
				DisplayName:  arg.DisplayName,
				AuthProvider: "email",
				AuthSubject:  arg.Email,
				PasswordHash: arg.PasswordHash,
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewUserStore(mock)
	info, err := store.CreateWithPassword(context.Background(), "pwd@example.com", "Password User", "$2a$10$hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PasswordHash == nil || *info.PasswordHash != "$2a$10$hash" {
		t.Errorf("PasswordHash: want hash, got %v", info.PasswordHash)
	}
}

func TestUserStore_UpdateProfile(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC()
	slug := "new-slug"

	mock := &mockQuerier{
		updateUserProfileFn: func(_ context.Context, arg database.UpdateUserProfileParams) (database.User, error) {
			if pgxToUUID(arg.ID) != id {
				t.Errorf("ID: want %v, got %v", id, pgxToUUID(arg.ID))
			}
			if arg.DisplayName != "Updated Name" {
				t.Errorf("DisplayName: want 'Updated Name', got %q", arg.DisplayName)
			}
			if arg.AvatarUrl.Valid {
				t.Error("AvatarUrl: expected invalid (nil)")
			}
			if !arg.ProfileSlug.Valid || arg.ProfileSlug.String != slug {
				t.Errorf("ProfileSlug: want %q, got %+v", slug, arg.ProfileSlug)
			}
			return database.User{
				ID:           uuidToPgx(id),
				Email:        "test@example.com",
				DisplayName:  arg.DisplayName,
				ProfileSlug:  arg.ProfileSlug,
				AuthProvider: "email",
				AuthSubject:  "test@example.com",
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewUserStore(mock)
	info, err := store.UpdateProfile(context.Background(), id, "Updated Name", nil, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.DisplayName != "Updated Name" {
		t.Errorf("DisplayName: want 'Updated Name', got %q", info.DisplayName)
	}
	if info.ProfileSlug == nil || *info.ProfileSlug != slug {
		t.Errorf("ProfileSlug: want %q, got %v", slug, info.ProfileSlug)
	}
}

func TestUserStore_Create_NilAvatar(t *testing.T) {
	now := time.Now().UTC()

	mock := &mockQuerier{
		createUserFn: func(_ context.Context, arg database.CreateUserParams) (database.User, error) {
			if arg.AvatarUrl.Valid {
				t.Error("AvatarUrl: expected invalid for nil")
			}
			return database.User{
				ID:           uuidToPgx(uuid.New()),
				Email:        arg.Email,
				DisplayName:  arg.DisplayName,
				AuthProvider: arg.AuthProvider,
				AuthSubject:  arg.AuthSubject,
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewUserStore(mock)
	info, err := store.Create(context.Background(), "test@example.com", "Test", nil, "email", "test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.AvatarURL != nil {
		t.Errorf("AvatarURL: want nil, got %v", info.AvatarURL)
	}
}
