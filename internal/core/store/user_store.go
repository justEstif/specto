package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// PgUserStore implements UserStore using sqlc-generated queries.
type PgUserStore struct {
	q Querier
}

// NewUserStore creates a new UserStore backed by PostgreSQL.
func NewUserStore(q Querier) *PgUserStore {
	return &PgUserStore{q: q}
}

var _ core.UserStore = (*PgUserStore)(nil)

func (s *PgUserStore) GetByID(ctx context.Context, id uuid.UUID) (*core.UserInfo, error) {
	row, err := s.q.GetUserByID(ctx, uuidToPgx(id))
	if err != nil {
		return nil, fmt.Errorf("getting user by ID: %w", err)
	}

	info := userFromDB(row)
	return &info, nil
}

func (s *PgUserStore) GetByProfileSlug(ctx context.Context, slug string) (*core.UserInfo, error) {
	row, err := s.q.GetUserByProfileSlug(ctx, pgtype.Text{String: slug, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("getting user by profile slug: %w", err)
	}

	info := userFromDB(row)
	return &info, nil
}

func (s *PgUserStore) GetByEmail(ctx context.Context, email string) (*core.UserInfo, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}

	info := userFromDB(row)
	return &info, nil
}

func (s *PgUserStore) GetByAuth(ctx context.Context, provider, subject string) (*core.UserInfo, error) {
	row, err := s.q.GetUserByAuth(ctx, database.GetUserByAuthParams{
		AuthProvider: provider,
		AuthSubject:  subject,
	})
	if err != nil {
		return nil, fmt.Errorf("getting user by auth: %w", err)
	}

	info := userFromDB(row)
	return &info, nil
}

func (s *PgUserStore) Create(ctx context.Context, email, displayName string, avatarURL *string, provider, subject string) (*core.UserInfo, error) {
	row, err := s.q.CreateUser(ctx, database.CreateUserParams{
		Email:        email,
		DisplayName:  displayName,
		AvatarUrl:    textPtr(avatarURL),
		AuthProvider: provider,
		AuthSubject:  subject,
	})
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	info := userFromDB(row)
	return &info, nil
}

func (s *PgUserStore) CreateWithPassword(ctx context.Context, email, displayName, passwordHash string) (*core.UserInfo, error) {
	row, err := s.q.CreateUserWithPassword(ctx, database.CreateUserWithPasswordParams{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("creating user with password: %w", err)
	}

	info := userFromDB(row)
	return &info, nil
}

func (s *PgUserStore) UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, avatarURL, profileSlug *string) (*core.UserInfo, error) {
	row, err := s.q.UpdateUserProfile(ctx, database.UpdateUserProfileParams{
		ID:          uuidToPgx(id),
		DisplayName: displayName,
		AvatarUrl:   textPtr(avatarURL),
		ProfileSlug: textPtr(profileSlug),
	})
	if err != nil {
		return nil, fmt.Errorf("updating user profile: %w", err)
	}

	info := userFromDB(row)
	return &info, nil
}

func (s *PgUserStore) MarkOnboarded(ctx context.Context, id uuid.UUID) error {
	return s.q.MarkUserOnboarded(ctx, uuidToPgx(id))
}
