package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/justestif/specto/internal/core"
)

func testOAuthConfig() *core.OAuthConfig {
	return &core.OAuthConfig{
		ProviderName: "TestProvider",
		AuthURL:      "https://provider.example.com/authorize",
		TokenURL:     "https://provider.example.com/token",
		Scopes:       []string{"read", "write"},
	}
}

func testOAuthService(clients map[string]OAuthClientCredentials) *OAuthService {
	return NewOAuthService("http://localhost:3000", clients, nil)
}

func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}
	if state1 == "" {
		t.Fatal("GenerateState() returned empty string")
	}

	// Each call should produce a different state (randomness).
	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}
	if state1 == state2 {
		t.Error("GenerateState() returned the same value twice")
	}

	// State should be long enough (32 bytes base64-encoded = 44 chars).
	if len(state1) < 40 {
		t.Errorf("state too short: %d chars", len(state1))
	}
}

func TestBuildAuthURL(t *testing.T) {
	svc := testOAuthService(map[string]OAuthClientCredentials{
		"spotify": {ClientID: "test-client-id", ClientSecret: "test-secret"},
	})
	cfg := testOAuthConfig()

	authURL, err := svc.BuildAuthURL("spotify", cfg, "test-state-123")
	if err != nil {
		t.Fatalf("BuildAuthURL() error: %v", err)
	}

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("returned URL is not valid: %v", err)
	}

	// Verify the base URL is preserved.
	if u.Scheme != "https" || u.Host != "provider.example.com" || u.Path != "/authorize" {
		t.Errorf("unexpected base URL: %s://%s%s", u.Scheme, u.Host, u.Path)
	}

	q := u.Query()

	tests := []struct {
		param    string
		expected string
	}{
		{"client_id", "test-client-id"},
		{"redirect_uri", "http://localhost:3000/api/v1/plugins/spotify/callback"},
		{"response_type", "code"},
		{"scope", "read write"},
		{"state", "test-state-123"},
	}

	for _, tt := range tests {
		got := q.Get(tt.param)
		if got != tt.expected {
			t.Errorf("param %q = %q, want %q", tt.param, got, tt.expected)
		}
	}
}

func TestBuildAuthURLMissingClient(t *testing.T) {
	svc := testOAuthService(map[string]OAuthClientCredentials{})
	cfg := testOAuthConfig()

	_, err := svc.BuildAuthURL("spotify", cfg, "state")
	if err != ErrMissingOAuthClient {
		t.Errorf("expected ErrMissingOAuthClient, got %v", err)
	}
}

func TestExchangeCode(t *testing.T) {
	// Mock OAuth token endpoint.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}

		// Verify all expected form fields.
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q, want authorization_code", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "test-auth-code" {
			t.Errorf("code = %q, want test-auth-code", r.FormValue("code"))
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("client_id = %q, want test-client-id", r.FormValue("client_id"))
		}
		if r.FormValue("client_secret") != "test-secret" {
			t.Errorf("client_secret = %q, want test-secret", r.FormValue("client_secret"))
		}
		if !strings.HasSuffix(r.FormValue("redirect_uri"), "/api/v1/plugins/spotify/callback") {
			t.Errorf("redirect_uri = %q, want suffix /api/v1/plugins/spotify/callback", r.FormValue("redirect_uri"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"expires_in":    3600,
			"token_type":    "Bearer",
		})
	}))
	defer server.Close()

	svc := NewOAuthService("http://localhost:3000", map[string]OAuthClientCredentials{
		"spotify": {ClientID: "test-client-id", ClientSecret: "test-secret"},
	}, server.Client())

	cfg := &core.OAuthConfig{
		ProviderName: "Spotify",
		AuthURL:      "https://accounts.spotify.com/authorize",
		TokenURL:     server.URL + "/token",
		Scopes:       []string{"user-read-recently-played"},
	}

	resp, err := svc.ExchangeCode("spotify", cfg, "test-auth-code")
	if err != nil {
		t.Fatalf("ExchangeCode() error: %v", err)
	}

	if resp.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q, want new-access-token", resp.AccessToken)
	}
	if resp.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %q, want new-refresh-token", resp.RefreshToken)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", resp.ExpiresIn)
	}
}

func TestExchangeCodeError(t *testing.T) {
	// Mock server returns an OAuth error response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "Authorization code expired",
		})
	}))
	defer server.Close()

	svc := NewOAuthService("http://localhost:3000", map[string]OAuthClientCredentials{
		"spotify": {ClientID: "id", ClientSecret: "secret"},
	}, server.Client())

	cfg := &core.OAuthConfig{
		TokenURL: server.URL + "/token",
	}

	_, err := svc.ExchangeCode("spotify", cfg, "bad-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Errorf("error should mention status 400: %v", err)
	}
}

func TestExchangeCodeMissingClient(t *testing.T) {
	svc := testOAuthService(map[string]OAuthClientCredentials{})
	cfg := testOAuthConfig()

	_, err := svc.ExchangeCode("spotify", cfg, "code")
	if err != ErrMissingOAuthClient {
		t.Errorf("expected ErrMissingOAuthClient, got %v", err)
	}
}

func TestExchangeCodeEmptyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	svc := NewOAuthService("http://localhost:3000", map[string]OAuthClientCredentials{
		"spotify": {ClientID: "id", ClientSecret: "secret"},
	}, server.Client())

	cfg := &core.OAuthConfig{TokenURL: server.URL + "/token"}

	_, err := svc.ExchangeCode("spotify", cfg, "code")
	if err == nil {
		t.Fatal("expected error for empty access_token")
	}
	if !strings.Contains(err.Error(), "empty access_token") {
		t.Errorf("error should mention empty access_token: %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}

		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", r.FormValue("grant_type"))
		}
		if r.FormValue("refresh_token") != "old-refresh-token" {
			t.Errorf("refresh_token = %q, want old-refresh-token", r.FormValue("refresh_token"))
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("client_id = %q, want test-client-id", r.FormValue("client_id"))
		}
		if r.FormValue("client_secret") != "test-secret" {
			t.Errorf("client_secret = %q, want test-secret", r.FormValue("client_secret"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "refreshed-access-token",
			"refresh_token": "rotated-refresh-token",
			"expires_in":    7200,
		})
	}))
	defer server.Close()

	svc := NewOAuthService("http://localhost:3000", map[string]OAuthClientCredentials{
		"spotify": {ClientID: "test-client-id", ClientSecret: "test-secret"},
	}, server.Client())

	cfg := &core.OAuthConfig{TokenURL: server.URL + "/token"}

	resp, err := svc.RefreshToken("spotify", cfg, "old-refresh-token")
	if err != nil {
		t.Fatalf("RefreshToken() error: %v", err)
	}

	if resp.AccessToken != "refreshed-access-token" {
		t.Errorf("AccessToken = %q, want refreshed-access-token", resp.AccessToken)
	}
	if resp.RefreshToken != "rotated-refresh-token" {
		t.Errorf("RefreshToken = %q, want rotated-refresh-token", resp.RefreshToken)
	}
	if resp.ExpiresIn != 7200 {
		t.Errorf("ExpiresIn = %d, want 7200", resp.ExpiresIn)
	}
}

func TestRefreshTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "Refresh token has been revoked",
		})
	}))
	defer server.Close()

	svc := NewOAuthService("http://localhost:3000", map[string]OAuthClientCredentials{
		"spotify": {ClientID: "id", ClientSecret: "secret"},
	}, server.Client())

	cfg := &core.OAuthConfig{TokenURL: server.URL + "/token"}

	_, err := svc.RefreshToken("spotify", cfg, "revoked-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Errorf("error should mention status 401: %v", err)
	}
}

func TestRefreshTokenMissingClient(t *testing.T) {
	svc := testOAuthService(map[string]OAuthClientCredentials{})
	cfg := testOAuthConfig()

	_, err := svc.RefreshToken("spotify", cfg, "token")
	if err != ErrMissingOAuthClient {
		t.Errorf("expected ErrMissingOAuthClient, got %v", err)
	}
}

func TestRefreshTokenNoRefreshTokenInResponse(t *testing.T) {
	// Some providers (like Spotify) may not return a new refresh_token.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access-token",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	svc := NewOAuthService("http://localhost:3000", map[string]OAuthClientCredentials{
		"spotify": {ClientID: "id", ClientSecret: "secret"},
	}, server.Client())

	cfg := &core.OAuthConfig{TokenURL: server.URL + "/token"}

	resp, err := svc.RefreshToken("spotify", cfg, "old-token")
	if err != nil {
		t.Fatalf("RefreshToken() error: %v", err)
	}
	if resp.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q, want new-access-token", resp.AccessToken)
	}
	if resp.RefreshToken != "" {
		t.Errorf("RefreshToken should be empty when not returned, got %q", resp.RefreshToken)
	}
}

func TestRedirectURI(t *testing.T) {
	svc := testOAuthService(nil)
	uri := svc.RedirectURI("spotify")
	expected := "http://localhost:3000/api/v1/plugins/spotify/callback"
	if uri != expected {
		t.Errorf("RedirectURI() = %q, want %q", uri, expected)
	}
}

func TestTokenResponseOAuthError(t *testing.T) {
	// Provider returns 200 but with an error field in the JSON body.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		})
	}))
	defer server.Close()

	svc := NewOAuthService("http://localhost:3000", map[string]OAuthClientCredentials{
		"test": {ClientID: "id", ClientSecret: "secret"},
	}, server.Client())

	cfg := &core.OAuthConfig{TokenURL: server.URL + "/token"}

	_, err := svc.ExchangeCode("test", cfg, "code")
	if err == nil {
		t.Fatal("expected error for OAuth error in response body")
	}
	if !strings.Contains(err.Error(), "invalid_client") {
		t.Errorf("error should contain OAuth error code: %v", err)
	}
}
