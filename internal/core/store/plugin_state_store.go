package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// PgPluginStateStore implements PluginStateStore using sqlc-generated queries.
type PgPluginStateStore struct {
	q      Querier
	encKey string // hex-encoded AES-256 key for credential encryption
}

// NewPluginStateStore creates a new PluginStateStore backed by PostgreSQL.
// The encKey is a 64-character hex string (32 bytes) used for AES-256-GCM
// encryption of plugin credentials.
func NewPluginStateStore(q Querier, encKey string) *PgPluginStateStore {
	return &PgPluginStateStore{q: q, encKey: encKey}
}

var _ core.PluginStateStore = (*PgPluginStateStore)(nil)

func (s *PgPluginStateStore) GetState(ctx context.Context, userID uuid.UUID, plugin string) (*core.PluginStateInfo, error) {
	row, err := s.q.GetPluginState(ctx, database.GetPluginStateParams{
		UserID: uuidToPgx(userID),
		Plugin: plugin,
	})
	if err != nil {
		return nil, fmt.Errorf("getting plugin state: %w", err)
	}

	info := pluginStateFromDB(row)
	return &info, nil
}

func (s *PgPluginStateStore) UpsertState(ctx context.Context, userID uuid.UUID, plugin, status string, enabled bool) (*core.PluginStateInfo, error) {
	row, err := s.q.UpsertPluginState(ctx, database.UpsertPluginStateParams{
		UserID:  uuidToPgx(userID),
		Plugin:  plugin,
		Status:  status,
		Enabled: enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("upserting plugin state: %w", err)
	}

	info := pluginStateFromDB(row)
	return &info, nil
}

func (s *PgPluginStateStore) UpdateStatus(ctx context.Context, userID uuid.UUID, plugin, status string, errMsg *string) (*core.PluginStateInfo, error) {
	row, err := s.q.UpdatePluginStateStatus(ctx, database.UpdatePluginStateStatusParams{
		UserID:       uuidToPgx(userID),
		Plugin:       plugin,
		Status:       status,
		ErrorMessage: textPtr(errMsg),
	})
	if err != nil {
		return nil, fmt.Errorf("updating plugin status: %w", err)
	}

	info := pluginStateFromDB(row)
	return &info, nil
}

func (s *PgPluginStateStore) UpdateSynced(ctx context.Context, userID uuid.UUID, plugin string, cursor *string) (*core.PluginStateInfo, error) {
	row, err := s.q.UpdatePluginStateSynced(ctx, database.UpdatePluginStateSyncedParams{
		UserID: uuidToPgx(userID),
		Plugin: plugin,
		Cursor: textPtr(cursor),
	})
	if err != nil {
		return nil, fmt.Errorf("updating plugin synced: %w", err)
	}

	info := pluginStateFromDB(row)
	return &info, nil
}

func (s *PgPluginStateStore) ListStates(ctx context.Context, userID uuid.UUID) ([]core.PluginStateInfo, error) {
	rows, err := s.q.ListPluginStates(ctx, uuidToPgx(userID))
	if err != nil {
		return nil, fmt.Errorf("listing plugin states: %w", err)
	}

	states := make([]core.PluginStateInfo, len(rows))
	for i, row := range rows {
		states[i] = pluginStateFromDB(row)
	}
	return states, nil
}

// credentialPayload is the JSON structure encrypted and stored in the database.
type credentialPayload struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	APIKey       string `json:"api_key,omitempty"`
}

func (s *PgPluginStateStore) GetCredentials(ctx context.Context, userID uuid.UUID, plugin string) (*core.Credentials, error) {
	row, err := s.q.GetPluginCredentials(ctx, database.GetPluginCredentialsParams{
		UserID: uuidToPgx(userID),
		Plugin: plugin,
	})
	if err != nil {
		return nil, fmt.Errorf("getting plugin credentials: %w", err)
	}

	plaintext, err := Decrypt(row.EncryptedData, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypting credentials: %w", err)
	}

	var payload credentialPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling credentials: %w", err)
	}

	return &core.Credentials{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		APIKey:       payload.APIKey,
	}, nil
}

func (s *PgPluginStateStore) UpsertCredentials(ctx context.Context, userID uuid.UUID, plugin string, authType core.AuthType, creds core.Credentials, expiresAt *time.Time) error {
	payload := credentialPayload{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
		APIKey:       creds.APIKey,
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	encrypted, err := Encrypt(plaintext, s.encKey)
	if err != nil {
		return fmt.Errorf("encrypting credentials: %w", err)
	}

	_, err = s.q.UpsertPluginCredentials(ctx, database.UpsertPluginCredentialsParams{
		UserID:        uuidToPgx(userID),
		Plugin:        plugin,
		AuthType:      authType.String(),
		EncryptedData: encrypted,
		ExpiresAt:     timestamptzPtr(expiresAt),
	})
	if err != nil {
		return fmt.Errorf("upserting plugin credentials: %w", err)
	}

	return nil
}

func (s *PgPluginStateStore) DeleteCredentials(ctx context.Context, userID uuid.UUID, plugin string) error {
	err := s.q.DeletePluginCredentials(ctx, database.DeletePluginCredentialsParams{
		UserID: uuidToPgx(userID),
		Plugin: plugin,
	})
	if err != nil {
		return fmt.Errorf("deleting plugin credentials: %w", err)
	}
	return nil
}
