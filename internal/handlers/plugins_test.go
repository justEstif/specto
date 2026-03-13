package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/justestif/specto/internal/app"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
	"github.com/justestif/specto/internal/handlers"
)

// --- test helpers ---

func newTestHandler(registry *core.PluginRegistry, pluginStates core.PluginStateStore, syncLogs core.SyncLogStore) *handlers.Handler {
	application := &app.App{
		Registry:     registry,
		PluginStates: pluginStates,
		SyncLogs:     syncLogs,
	}
	return handlers.New(application)
}

func newTestHandlerWithOAuth(registry *core.PluginRegistry, pluginStates core.PluginStateStore, oauthSvc *auth.OAuthService) *handlers.Handler {
	sessions := auth.NewSessionManager([]byte("test-session-secret-that-is-32b!"))
	authSvc := auth.NewService(nil, sessions)
	application := &app.App{
		Auth:         authSvc,
		OAuth:        oauthSvc,
		Registry:     registry,
		PluginStates: pluginStates,
	}
	return handlers.New(application)
}

func authenticatedRequest(method, path string, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	user := &core.UserInfo{
		ID:          userID,
		Email:       "test@example.com",
		DisplayName: "Test User",
	}
	ctx := auth.ContextWithUser(req.Context(), user)
	return req.WithContext(ctx)
}

func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	return result
}

// --- mock plugin ---

type mockPlugin struct {
	name       string
	authType   core.AuthType
	authConfig *core.OAuthConfig
}

func (m *mockPlugin) Name() string                  { return m.name }
func (m *mockPlugin) AuthType() core.AuthType       { return m.authType }
func (m *mockPlugin) AuthConfig() *core.OAuthConfig { return m.authConfig }
func (m *mockPlugin) Sync(_ context.Context, _ core.Credentials, _ string) core.SyncResult {
	return core.SyncResult{}
}
func (m *mockPlugin) Enrich(_ context.Context, _ core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	return items, nil
}

var _ core.SourcePlugin = (*mockPlugin)(nil)

// --- mock plugin state store ---

type mockPluginStateStore struct {
	listStatesFn     func(ctx context.Context, userID uuid.UUID) ([]core.PluginStateInfo, error)
	getStateFn       func(ctx context.Context, userID uuid.UUID, plugin string) (*core.PluginStateInfo, error)
	upsertStateFn    func(ctx context.Context, userID uuid.UUID, plugin, status string, enabled bool) (*core.PluginStateInfo, error)
	updateStatusFn   func(ctx context.Context, userID uuid.UUID, plugin, status string, errMsg *string) (*core.PluginStateInfo, error)
	updateSyncedFn   func(ctx context.Context, userID uuid.UUID, plugin string, cursor *string) (*core.PluginStateInfo, error)
	getCredentialsFn func(ctx context.Context, userID uuid.UUID, plugin string) (*core.Credentials, error)
	upsertCredsFn    func(ctx context.Context, userID uuid.UUID, plugin string, authType core.AuthType, creds core.Credentials, expiresAt *time.Time) error
	deleteCredsFn    func(ctx context.Context, userID uuid.UUID, plugin string) error
}

func (m *mockPluginStateStore) ListStates(ctx context.Context, userID uuid.UUID) ([]core.PluginStateInfo, error) {
	if m.listStatesFn != nil {
		return m.listStatesFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockPluginStateStore) GetState(ctx context.Context, userID uuid.UUID, plugin string) (*core.PluginStateInfo, error) {
	if m.getStateFn != nil {
		return m.getStateFn(ctx, userID, plugin)
	}
	return nil, nil
}
func (m *mockPluginStateStore) UpsertState(ctx context.Context, userID uuid.UUID, plugin, status string, enabled bool) (*core.PluginStateInfo, error) {
	if m.upsertStateFn != nil {
		return m.upsertStateFn(ctx, userID, plugin, status, enabled)
	}
	return &core.PluginStateInfo{}, nil
}
func (m *mockPluginStateStore) UpdateStatus(ctx context.Context, userID uuid.UUID, plugin, status string, errMsg *string) (*core.PluginStateInfo, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, userID, plugin, status, errMsg)
	}
	return &core.PluginStateInfo{}, nil
}
func (m *mockPluginStateStore) UpdateSynced(ctx context.Context, userID uuid.UUID, plugin string, cursor *string) (*core.PluginStateInfo, error) {
	if m.updateSyncedFn != nil {
		return m.updateSyncedFn(ctx, userID, plugin, cursor)
	}
	return &core.PluginStateInfo{}, nil
}
func (m *mockPluginStateStore) GetCredentials(ctx context.Context, userID uuid.UUID, plugin string) (*core.Credentials, error) {
	if m.getCredentialsFn != nil {
		return m.getCredentialsFn(ctx, userID, plugin)
	}
	return &core.Credentials{}, nil
}
func (m *mockPluginStateStore) UpsertCredentials(ctx context.Context, userID uuid.UUID, plugin string, authType core.AuthType, creds core.Credentials, expiresAt *time.Time) error {
	if m.upsertCredsFn != nil {
		return m.upsertCredsFn(ctx, userID, plugin, authType, creds, expiresAt)
	}
	return nil
}
func (m *mockPluginStateStore) DeleteCredentials(ctx context.Context, userID uuid.UUID, plugin string) error {
	if m.deleteCredsFn != nil {
		return m.deleteCredsFn(ctx, userID, plugin)
	}
	return nil
}

// --- mock sync log store ---

type mockSyncLogStore struct {
	beginFn    func(ctx context.Context, userID uuid.UUID, plugin string) (uuid.UUID, error)
	completeFn func(ctx context.Context, logID uuid.UUID, result core.SyncLogResult) error
	failFn     func(ctx context.Context, logID uuid.UUID, result core.SyncLogResult) error
	listFn     func(ctx context.Context, userID uuid.UUID, plugin string, limit int32) ([]core.SyncLogEntry, error)
}

func (m *mockSyncLogStore) Begin(ctx context.Context, userID uuid.UUID, plugin string) (uuid.UUID, error) {
	if m.beginFn != nil {
		return m.beginFn(ctx, userID, plugin)
	}
	return uuid.New(), nil
}
func (m *mockSyncLogStore) Complete(ctx context.Context, logID uuid.UUID, result core.SyncLogResult) error {
	if m.completeFn != nil {
		return m.completeFn(ctx, logID, result)
	}
	return nil
}
func (m *mockSyncLogStore) Fail(ctx context.Context, logID uuid.UUID, result core.SyncLogResult) error {
	if m.failFn != nil {
		return m.failFn(ctx, logID, result)
	}
	return nil
}
func (m *mockSyncLogStore) List(ctx context.Context, userID uuid.UUID, plugin string, limit int32) ([]core.SyncLogEntry, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, plugin, limit)
	}
	return nil, nil
}

// --- tests ---

func TestListPlugins(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "https://accounts.spotify.com/api/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})
	registry.Register(&mockPlugin{name: "netflix", authType: core.AuthFileImport})

	userID := uuid.New()
	lastSynced := time.Now().Add(-1 * time.Hour)

	states := &mockPluginStateStore{
		listStatesFn: func(_ context.Context, _ uuid.UUID) ([]core.PluginStateInfo, error) {
			return []core.PluginStateInfo{
				{Plugin: "spotify", Status: "connected", Enabled: true, LastSyncedAt: &lastSynced},
			}, nil
		},
	}

	h := newTestHandler(registry, states, nil)
	req := authenticatedRequest("GET", "/api/v1/plugins", userID)
	w := httptest.NewRecorder()

	h.ListPlugins(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(data))
	}

	// Plugins should be sorted alphabetically
	first := data[0].(map[string]any)
	if first["name"] != "netflix" {
		t.Errorf("expected first plugin to be netflix, got %s", first["name"])
	}

	second := data[1].(map[string]any)
	if second["name"] != "spotify" {
		t.Errorf("expected second plugin to be spotify, got %s", second["name"])
	}
	if second["connected"] != true {
		t.Error("expected spotify to be connected")
	}
	if second["display_name"] != "Spotify" {
		t.Errorf("expected display_name Spotify, got %s", second["display_name"])
	}
}

func TestListPluginsUnauthenticated(t *testing.T) {
	h := newTestHandler(core.NewPluginRegistry(), &mockPluginStateStore{}, nil)
	req := httptest.NewRequest("GET", "/api/v1/plugins", nil)
	w := httptest.NewRecorder()

	h.ListPlugins(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetPlugin(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "https://accounts.spotify.com/api/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	userID := uuid.New()
	h := newTestHandler(registry, &mockPluginStateStore{}, nil)
	req := authenticatedRequest("GET", "/api/v1/plugins/spotify", userID)
	req = withChiParam(req, "plugin", "spotify")
	w := httptest.NewRecorder()

	h.GetPlugin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].(map[string]any)
	if data["name"] != "spotify" {
		t.Errorf("expected plugin name spotify, got %s", data["name"])
	}
	caps := data["capabilities"].(map[string]any)
	if caps["can_connect"] != true {
		t.Error("expected can_connect to be true for OAuth plugin")
	}
	if caps["can_import"] != false {
		t.Error("expected can_import to be false for OAuth plugin")
	}
}

func TestGetPluginNotFound(t *testing.T) {
	h := newTestHandler(core.NewPluginRegistry(), &mockPluginStateStore{}, nil)
	req := authenticatedRequest("GET", "/api/v1/plugins/nonexistent", uuid.New())
	req = withChiParam(req, "plugin", "nonexistent")
	w := httptest.NewRecorder()

	h.GetPlugin(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestConnectPluginOAuth(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "https://accounts.spotify.com/api/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	oauthSvc := auth.NewOAuthService("http://localhost:3000", map[string]auth.OAuthClientCredentials{
		"spotify": {ClientID: "test-client-id", ClientSecret: "test-secret"},
	}, nil)

	h := newTestHandlerWithOAuth(registry, &mockPluginStateStore{}, oauthSvc)
	req := authenticatedRequest("POST", "/api/v1/plugins/spotify/connect", uuid.New())
	req = withChiParam(req, "plugin", "spotify")
	w := httptest.NewRecorder()

	h.ConnectPlugin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].(map[string]any)
	redirectURL := data["redirect_url"].(string)

	// The redirect URL should be the full OAuth URL with query params.
	u, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("redirect_url is not a valid URL: %v", err)
	}

	if u.Host != "accounts.spotify.com" {
		t.Errorf("unexpected host: %s", u.Host)
	}

	q := u.Query()
	if q.Get("client_id") != "test-client-id" {
		t.Errorf("client_id = %q, want test-client-id", q.Get("client_id"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q, want code", q.Get("response_type"))
	}
	if q.Get("scope") != "user-read-recently-played" {
		t.Errorf("scope = %q, want user-read-recently-played", q.Get("scope"))
	}
	if q.Get("state") == "" {
		t.Error("state parameter is missing")
	}
	if q.Get("redirect_uri") != "http://localhost:3000/api/v1/plugins/spotify/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
}

func TestConnectPluginNonOAuth(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{name: "netflix", authType: core.AuthFileImport})

	h := newTestHandler(registry, &mockPluginStateStore{}, nil)
	req := authenticatedRequest("POST", "/api/v1/plugins/netflix/connect", uuid.New())
	req = withChiParam(req, "plugin", "netflix")
	w := httptest.NewRecorder()

	h.ConnectPlugin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDisconnectPlugin(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "https://accounts.spotify.com/api/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	deleteCalled := false
	updateCalled := false
	states := &mockPluginStateStore{
		deleteCredsFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			deleteCalled = true
			return nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ string, status string, _ *string) (*core.PluginStateInfo, error) {
			updateCalled = true
			if status != "disconnected" {
				t.Errorf("expected status disconnected, got %s", status)
			}
			return &core.PluginStateInfo{Status: "disconnected"}, nil
		},
	}

	h := newTestHandler(registry, states, nil)
	req := authenticatedRequest("DELETE", "/api/v1/plugins/spotify/disconnect", uuid.New())
	req = withChiParam(req, "plugin", "spotify")
	w := httptest.NewRecorder()

	h.DisconnectPlugin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !deleteCalled {
		t.Error("expected credentials to be deleted")
	}
	if !updateCalled {
		t.Error("expected status to be updated")
	}

	resp := parseResponse(t, w)
	data := resp["data"].(map[string]any)
	if data["status"] != "disconnected" {
		t.Errorf("expected status disconnected, got %s", data["status"])
	}
}

func TestSyncHistory(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{name: "netflix", authType: core.AuthFileImport})

	now := time.Now()
	completedAt := now.Add(10 * time.Second)
	logs := &mockSyncLogStore{
		listFn: func(_ context.Context, _ uuid.UUID, _ string, _ int32) ([]core.SyncLogEntry, error) {
			return []core.SyncLogEntry{
				{
					ID:           uuid.New(),
					Plugin:       "netflix",
					StartedAt:    now,
					CompletedAt:  &completedAt,
					Status:       "completed",
					ItemsAdded:   42,
					ItemsSkipped: 5,
				},
			}, nil
		},
	}

	h := newTestHandler(registry, &mockPluginStateStore{}, logs)
	req := authenticatedRequest("GET", "/api/v1/plugins/netflix/sync-history", uuid.New())
	req = withChiParam(req, "plugin", "netflix")
	w := httptest.NewRecorder()

	h.SyncHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(data))
	}

	entry := data[0].(map[string]any)
	if entry["status"] != "completed" {
		t.Errorf("expected status completed, got %s", entry["status"])
	}
	if int(entry["items_added"].(float64)) != 42 {
		t.Errorf("expected items_added 42, got %v", entry["items_added"])
	}
}

func TestSyncHistoryPluginNotFound(t *testing.T) {
	h := newTestHandler(core.NewPluginRegistry(), &mockPluginStateStore{}, &mockSyncLogStore{})
	req := authenticatedRequest("GET", "/api/v1/plugins/nonexistent/sync-history", uuid.New())
	req = withChiParam(req, "plugin", "nonexistent")
	w := httptest.NewRecorder()

	h.SyncHistory(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- OAuth callback tests ---

// setupOAuthCallbackTest creates a handler with a mock token server and
// returns the handler, a function to make an authenticated callback request
// with the state already stored in the session, and a cleanup function.
func setupOAuthCallbackTest(t *testing.T, tokenHandler http.HandlerFunc) (h *handlers.Handler, makeReq func(code, state string) (*http.Request, *httptest.ResponseRecorder), cleanup func()) {
	t.Helper()

	tokenServer := httptest.NewServer(tokenHandler)

	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     tokenServer.URL + "/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	oauthSvc := auth.NewOAuthService("http://localhost:3000", map[string]auth.OAuthClientCredentials{
		"spotify": {ClientID: "test-client-id", ClientSecret: "test-secret"},
	}, tokenServer.Client())

	states := &mockPluginStateStore{}
	h = newTestHandlerWithOAuth(registry, states, oauthSvc)

	makeReq = func(code, state string) (*http.Request, *httptest.ResponseRecorder) {
		callbackURL := fmt.Sprintf("/api/v1/plugins/spotify/callback?code=%s&state=%s",
			url.QueryEscape(code), url.QueryEscape(state))
		req := authenticatedRequest("GET", callbackURL, uuid.New())
		req = withChiParam(req, "plugin", "spotify")
		return req, httptest.NewRecorder()
	}

	return h, makeReq, func() { tokenServer.Close() }
}

func TestOAuthCallback(t *testing.T) {
	tokenHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"expires_in":    3600,
		})
	})

	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "", // will be set below
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	tokenServer := httptest.NewServer(tokenHandler)
	defer tokenServer.Close()

	// Reconfigure with the correct TokenURL.
	registry = core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     tokenServer.URL + "/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	oauthSvc := auth.NewOAuthService("http://localhost:3000", map[string]auth.OAuthClientCredentials{
		"spotify": {ClientID: "test-client-id", ClientSecret: "test-secret"},
	}, tokenServer.Client())

	var upsertCredsCalled bool
	var upsertStateCalled bool
	states := &mockPluginStateStore{
		upsertCredsFn: func(_ context.Context, _ uuid.UUID, plugin string, authType core.AuthType, creds core.Credentials, expiresAt *time.Time) error {
			upsertCredsCalled = true
			if plugin != "spotify" {
				t.Errorf("upsert plugin = %q, want spotify", plugin)
			}
			if authType != core.AuthOAuth {
				t.Errorf("authType = %v, want AuthOAuth", authType)
			}
			if creds.AccessToken != "new-access-token" {
				t.Errorf("AccessToken = %q, want new-access-token", creds.AccessToken)
			}
			if creds.RefreshToken != "new-refresh-token" {
				t.Errorf("RefreshToken = %q, want new-refresh-token", creds.RefreshToken)
			}
			if expiresAt == nil {
				t.Error("expiresAt should not be nil when expires_in is set")
			}
			return nil
		},
		upsertStateFn: func(_ context.Context, _ uuid.UUID, plugin, status string, enabled bool) (*core.PluginStateInfo, error) {
			upsertStateCalled = true
			if status != "connected" {
				t.Errorf("status = %q, want connected", status)
			}
			if !enabled {
				t.Error("expected enabled = true")
			}
			return &core.PluginStateInfo{Status: "connected"}, nil
		},
	}

	h := newTestHandlerWithOAuth(registry, states, oauthSvc)

	// Step 1: Call ConnectPlugin to set the state in the session.
	connectReq := authenticatedRequest("POST", "/api/v1/plugins/spotify/connect", uuid.New())
	connectReq = withChiParam(connectReq, "plugin", "spotify")
	connectW := httptest.NewRecorder()
	h.ConnectPlugin(connectW, connectReq)

	if connectW.Code != http.StatusOK {
		t.Fatalf("ConnectPlugin: expected 200, got %d: %s", connectW.Code, connectW.Body.String())
	}

	// Extract the state from the redirect URL.
	connectResp := parseResponse(t, connectW)
	connectData := connectResp["data"].(map[string]any)
	redirectURL := connectData["redirect_url"].(string)
	u, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect_url: %v", err)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("state parameter is empty")
	}

	// Step 2: Simulate the OAuth callback with the correct state.
	// Transfer session cookies from connect response to callback request.
	callbackURL := fmt.Sprintf("/api/v1/plugins/spotify/callback?code=test-auth-code&state=%s", url.QueryEscape(state))
	callbackReq := authenticatedRequest("GET", callbackURL, uuid.New())
	callbackReq = withChiParam(callbackReq, "plugin", "spotify")

	// Copy cookies from connect response to callback request.
	for _, cookie := range connectW.Result().Cookies() {
		callbackReq.AddCookie(cookie)
	}

	callbackW := httptest.NewRecorder()
	h.OAuthCallback(callbackW, callbackReq)

	if callbackW.Code != http.StatusSeeOther {
		t.Fatalf("OAuthCallback: expected 303, got %d: %s", callbackW.Code, callbackW.Body.String())
	}

	if loc := callbackW.Header().Get("Location"); loc != "/plugins" {
		t.Errorf("redirect location = %q, want /plugins", loc)
	}
	if !upsertCredsCalled {
		t.Error("expected UpsertCredentials to be called")
	}
	if !upsertStateCalled {
		t.Error("expected UpsertState to be called")
	}
}

func TestOAuthCallbackInvalidState(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "https://accounts.spotify.com/api/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	oauthSvc := auth.NewOAuthService("http://localhost:3000", map[string]auth.OAuthClientCredentials{
		"spotify": {ClientID: "id", ClientSecret: "secret"},
	}, nil)

	h := newTestHandlerWithOAuth(registry, &mockPluginStateStore{}, oauthSvc)

	// First set a state via ConnectPlugin.
	connectReq := authenticatedRequest("POST", "/api/v1/plugins/spotify/connect", uuid.New())
	connectReq = withChiParam(connectReq, "plugin", "spotify")
	connectW := httptest.NewRecorder()
	h.ConnectPlugin(connectW, connectReq)

	// Use a WRONG state in the callback.
	callbackReq := authenticatedRequest("GET", "/api/v1/plugins/spotify/callback?code=test-code&state=wrong-state", uuid.New())
	callbackReq = withChiParam(callbackReq, "plugin", "spotify")
	for _, cookie := range connectW.Result().Cookies() {
		callbackReq.AddCookie(cookie)
	}

	callbackW := httptest.NewRecorder()
	h.OAuthCallback(callbackW, callbackReq)

	if callbackW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", callbackW.Code, callbackW.Body.String())
	}

	resp := parseResponse(t, callbackW)
	errData := resp["error"].(map[string]any)
	if errData["code"] != "validation_error" {
		t.Errorf("error code = %q, want validation_error", errData["code"])
	}
}

func TestOAuthCallbackMissingCode(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "https://accounts.spotify.com/api/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	oauthSvc := auth.NewOAuthService("http://localhost:3000", map[string]auth.OAuthClientCredentials{
		"spotify": {ClientID: "id", ClientSecret: "secret"},
	}, nil)

	h := newTestHandlerWithOAuth(registry, &mockPluginStateStore{}, oauthSvc)

	// Missing code param.
	req := authenticatedRequest("GET", "/api/v1/plugins/spotify/callback?state=some-state", uuid.New())
	req = withChiParam(req, "plugin", "spotify")
	w := httptest.NewRecorder()
	h.OAuthCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOAuthCallbackProviderError(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "https://accounts.spotify.com/api/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	oauthSvc := auth.NewOAuthService("http://localhost:3000", map[string]auth.OAuthClientCredentials{
		"spotify": {ClientID: "id", ClientSecret: "secret"},
	}, nil)

	h := newTestHandlerWithOAuth(registry, &mockPluginStateStore{}, oauthSvc)

	// Provider sent an error (user denied consent).
	req := authenticatedRequest("GET", "/api/v1/plugins/spotify/callback?error=access_denied&error_description=User+denied+access", uuid.New())
	req = withChiParam(req, "plugin", "spotify")
	w := httptest.NewRecorder()
	h.OAuthCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	errData := resp["error"].(map[string]any)
	if errData["code"] != "oauth_error" {
		t.Errorf("error code = %q, want oauth_error", errData["code"])
	}
}

func TestOAuthCallbackPluginNotFound(t *testing.T) {
	oauthSvc := auth.NewOAuthService("http://localhost:3000", nil, nil)
	h := newTestHandlerWithOAuth(core.NewPluginRegistry(), &mockPluginStateStore{}, oauthSvc)

	req := authenticatedRequest("GET", "/api/v1/plugins/nonexistent/callback?code=x&state=y", uuid.New())
	req = withChiParam(req, "plugin", "nonexistent")
	w := httptest.NewRecorder()

	h.OAuthCallback(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestOAuthCallbackUnauthenticated(t *testing.T) {
	oauthSvc := auth.NewOAuthService("http://localhost:3000", nil, nil)
	h := newTestHandlerWithOAuth(core.NewPluginRegistry(), &mockPluginStateStore{}, oauthSvc)

	req := httptest.NewRequest("GET", "/api/v1/plugins/spotify/callback?code=x&state=y", nil)
	req = withChiParam(req, "plugin", "spotify")
	w := httptest.NewRecorder()

	h.OAuthCallback(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestConnectPluginOAuthUnauthenticated(t *testing.T) {
	registry := core.NewPluginRegistry()
	registry.Register(&mockPlugin{
		name:     "spotify",
		authType: core.AuthOAuth,
		authConfig: &core.OAuthConfig{
			ProviderName: "Spotify",
			AuthURL:      "https://accounts.spotify.com/authorize",
			TokenURL:     "https://accounts.spotify.com/api/token",
			Scopes:       []string{"user-read-recently-played"},
		},
	})

	oauthSvc := auth.NewOAuthService("http://localhost:3000", map[string]auth.OAuthClientCredentials{
		"spotify": {ClientID: "id", ClientSecret: "secret"},
	}, nil)

	h := newTestHandlerWithOAuth(registry, &mockPluginStateStore{}, oauthSvc)
	req := httptest.NewRequest("POST", "/api/v1/plugins/spotify/connect", nil)
	req = withChiParam(req, "plugin", "spotify")
	w := httptest.NewRecorder()

	h.ConnectPlugin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
