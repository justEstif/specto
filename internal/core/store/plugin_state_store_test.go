package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/database"
)

// testEncKey generates a random 32-byte key as hex for tests.
func testEncKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generating test key: %v", err)
	}
	return hex.EncodeToString(key)
}

func TestPluginStateStore_GetState(t *testing.T) {
	userID := uuid.New()
	stateID := uuid.New()
	now := time.Now().UTC()

	mock := &mockQuerier{
		getPluginStateFn: func(_ context.Context, arg database.GetPluginStateParams) (database.PluginState, error) {
			if arg.Plugin != "spotify" {
				t.Errorf("Plugin: want 'spotify', got %q", arg.Plugin)
			}
			return database.PluginState{
				ID:        uuidToPgx(stateID),
				UserID:    uuidToPgx(userID),
				Plugin:    "spotify",
				Status:    "connected",
				Enabled:   true,
				Cursor:    pgtype.Text{String: "cursor-abc", Valid: true},
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewPluginStateStore(mock, testEncKey(t))
	info, err := store.GetState(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != stateID {
		t.Errorf("ID: want %v, got %v", stateID, info.ID)
	}
	if info.Status != "connected" {
		t.Errorf("Status: want 'connected', got %q", info.Status)
	}
	if info.Cursor == nil || *info.Cursor != "cursor-abc" {
		t.Errorf("Cursor: want 'cursor-abc', got %v", info.Cursor)
	}
}

func TestPluginStateStore_UpsertState(t *testing.T) {
	userID := uuid.New()

	mock := &mockQuerier{
		upsertPluginStateFn: func(_ context.Context, arg database.UpsertPluginStateParams) (database.PluginState, error) {
			if arg.Plugin != "youtube" {
				t.Errorf("Plugin: want 'youtube', got %q", arg.Plugin)
			}
			if arg.Status != "connecting" {
				t.Errorf("Status: want 'connecting', got %q", arg.Status)
			}
			if !arg.Enabled {
				t.Error("Enabled: want true")
			}
			now := time.Now().UTC()
			return database.PluginState{
				ID:        uuidToPgx(uuid.New()),
				UserID:    uuidToPgx(userID),
				Plugin:    arg.Plugin,
				Status:    arg.Status,
				Enabled:   arg.Enabled,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewPluginStateStore(mock, testEncKey(t))
	info, err := store.UpsertState(context.Background(), userID, "youtube", "connecting", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Plugin != "youtube" {
		t.Errorf("Plugin: want 'youtube', got %q", info.Plugin)
	}
}

func TestPluginStateStore_UpdateStatus(t *testing.T) {
	userID := uuid.New()
	errMsg := "token expired"

	mock := &mockQuerier{
		updatePluginStateStatusFn: func(_ context.Context, arg database.UpdatePluginStateStatusParams) (database.PluginState, error) {
			if arg.Status != "error" {
				t.Errorf("Status: want 'error', got %q", arg.Status)
			}
			if !arg.ErrorMessage.Valid || arg.ErrorMessage.String != "token expired" {
				t.Errorf("ErrorMessage: want 'token expired', got %+v", arg.ErrorMessage)
			}
			now := time.Now().UTC()
			return database.PluginState{
				ID:           uuidToPgx(uuid.New()),
				UserID:       uuidToPgx(userID),
				Plugin:       "spotify",
				Status:       arg.Status,
				ErrorMessage: arg.ErrorMessage,
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewPluginStateStore(mock, testEncKey(t))
	info, err := store.UpdateStatus(context.Background(), userID, "spotify", "error", &errMsg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ErrorMessage == nil || *info.ErrorMessage != "token expired" {
		t.Errorf("ErrorMessage: want 'token expired', got %v", info.ErrorMessage)
	}
}

func TestPluginStateStore_UpdateSynced(t *testing.T) {
	userID := uuid.New()
	cursor := "new-cursor-xyz"

	mock := &mockQuerier{
		updatePluginStateSyncedFn: func(_ context.Context, arg database.UpdatePluginStateSyncedParams) (database.PluginState, error) {
			if !arg.Cursor.Valid || arg.Cursor.String != cursor {
				t.Errorf("Cursor: want %q, got %+v", cursor, arg.Cursor)
			}
			now := time.Now().UTC()
			return database.PluginState{
				ID:           uuidToPgx(uuid.New()),
				UserID:       uuidToPgx(userID),
				Plugin:       "spotify",
				Status:       "connected",
				Cursor:       arg.Cursor,
				LastSyncedAt: pgtype.Timestamptz{Time: now, Valid: true},
				CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	store := NewPluginStateStore(mock, testEncKey(t))
	info, err := store.UpdateSynced(context.Background(), userID, "spotify", &cursor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Cursor == nil || *info.Cursor != cursor {
		t.Errorf("Cursor: want %q, got %v", cursor, info.Cursor)
	}
}

func TestPluginStateStore_ListStates(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	mock := &mockQuerier{
		listPluginStatesFn: func(_ context.Context, uid pgtype.UUID) ([]database.PluginState, error) {
			return []database.PluginState{
				{
					ID: uuidToPgx(uuid.New()), UserID: uuidToPgx(userID),
					Plugin: "spotify", Status: "connected", Enabled: true,
					CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				},
				{
					ID: uuidToPgx(uuid.New()), UserID: uuidToPgx(userID),
					Plugin: "youtube", Status: "disconnected", Enabled: false,
					CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				},
			}, nil
		},
	}

	store := NewPluginStateStore(mock, testEncKey(t))
	states, err := store.ListStates(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	if states[0].Plugin != "spotify" {
		t.Errorf("first state plugin: want 'spotify', got %q", states[0].Plugin)
	}
}

func TestPluginStateStore_CredentialRoundTrip(t *testing.T) {
	userID := uuid.New()
	encKey := testEncKey(t)

	var storedEncrypted []byte

	mock := &mockQuerier{
		upsertPluginCredentialsFn: func(_ context.Context, arg database.UpsertPluginCredentialsParams) (database.PluginCredential, error) {
			if arg.AuthType != "oauth" {
				t.Errorf("AuthType: want 'oauth', got %q", arg.AuthType)
			}
			if arg.EncryptedData == nil {
				t.Fatal("EncryptedData: expected non-nil")
			}
			storedEncrypted = arg.EncryptedData
			return database.PluginCredential{}, nil
		},
		getPluginCredentialsFn: func(_ context.Context, arg database.GetPluginCredentialsParams) (database.PluginCredential, error) {
			return database.PluginCredential{
				ID:            uuidToPgx(uuid.New()),
				UserID:        uuidToPgx(userID),
				Plugin:        "spotify",
				AuthType:      "oauth",
				EncryptedData: storedEncrypted,
			}, nil
		},
	}

	store := NewPluginStateStore(mock, encKey)

	// Store credentials
	creds := core.Credentials{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
	}
	expires := time.Now().UTC().Add(time.Hour)
	err := store.UpsertCredentials(context.Background(), userID, "spotify", core.AuthOAuth, creds, &expires)
	if err != nil {
		t.Fatalf("UpsertCredentials: %v", err)
	}

	// Verify the stored data is actually encrypted (not plaintext JSON)
	var payload credentialPayload
	if json.Unmarshal(storedEncrypted, &payload) == nil && payload.AccessToken == "access-token-123" {
		t.Fatal("stored data appears to be unencrypted plaintext")
	}

	// Retrieve and verify credentials
	got, err := store.GetCredentials(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if got.AccessToken != "access-token-123" {
		t.Errorf("AccessToken: want 'access-token-123', got %q", got.AccessToken)
	}
	if got.RefreshToken != "refresh-token-456" {
		t.Errorf("RefreshToken: want 'refresh-token-456', got %q", got.RefreshToken)
	}
}

func TestPluginStateStore_CredentialAPIKey(t *testing.T) {
	encKey := testEncKey(t)
	var storedEncrypted []byte

	mock := &mockQuerier{
		upsertPluginCredentialsFn: func(_ context.Context, arg database.UpsertPluginCredentialsParams) (database.PluginCredential, error) {
			if arg.AuthType != "api_key" {
				t.Errorf("AuthType: want 'api_key', got %q", arg.AuthType)
			}
			storedEncrypted = arg.EncryptedData
			return database.PluginCredential{}, nil
		},
		getPluginCredentialsFn: func(_ context.Context, _ database.GetPluginCredentialsParams) (database.PluginCredential, error) {
			return database.PluginCredential{
				EncryptedData: storedEncrypted,
			}, nil
		},
	}

	store := NewPluginStateStore(mock, encKey)

	creds := core.Credentials{APIKey: "my-secret-key"}
	err := store.UpsertCredentials(context.Background(), uuid.New(), "custom-api", core.AuthAPIKey, creds, nil)
	if err != nil {
		t.Fatalf("UpsertCredentials: %v", err)
	}

	got, err := store.GetCredentials(context.Background(), uuid.New(), "custom-api")
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if got.APIKey != "my-secret-key" {
		t.Errorf("APIKey: want 'my-secret-key', got %q", got.APIKey)
	}
}

func TestPluginStateStore_DeleteCredentials(t *testing.T) {
	userID := uuid.New()
	deleted := false

	mock := &mockQuerier{
		deletePluginCredentialsFn: func(_ context.Context, arg database.DeletePluginCredentialsParams) error {
			if arg.Plugin != "spotify" {
				t.Errorf("Plugin: want 'spotify', got %q", arg.Plugin)
			}
			deleted = true
			return nil
		},
	}

	store := NewPluginStateStore(mock, testEncKey(t))
	err := store.DeleteCredentials(context.Background(), userID, "spotify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("DeletePluginCredentials was not called")
	}
}
