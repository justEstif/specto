package core_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

// --- AuthType ---

func TestAuthTypeString(t *testing.T) {
	tests := []struct {
		at   core.AuthType
		want string
	}{
		{core.AuthOAuth, "oauth"},
		{core.AuthFileImport, "file_import"},
		{core.AuthAPIKey, "api_key"},
		{core.AuthNone, "none"},
		{core.AuthType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.at.String(); got != tt.want {
			t.Errorf("AuthType(%d).String() = %q, want %q", tt.at, got, tt.want)
		}
	}
}

// --- MediaType ---

func TestMediaTypeValid(t *testing.T) {
	valid := []core.MediaType{core.MediaMusic, core.MediaVideo, core.MediaArticle, core.MediaPodcast}
	for _, mt := range valid {
		if !mt.Valid() {
			t.Errorf("MediaType(%q).Valid() = false, want true", mt)
		}
	}

	invalid := core.MediaType("game")
	if invalid.Valid() {
		t.Errorf("MediaType(%q).Valid() = true, want false", invalid)
	}
}

// --- ErrorCode ---

func TestErrorCodeValid(t *testing.T) {
	codes := []core.ErrorCode{
		core.ErrAuthExpired,
		core.ErrRateLimit,
		core.ErrPartialSync,
		core.ErrUpstream,
		core.ErrInvalidData,
		core.ErrPermissionDenied,
		core.ErrFileParseError,
	}
	for _, c := range codes {
		if !c.Valid() {
			t.Errorf("ErrorCode(%q).Valid() = false, want true", c)
		}
	}

	invalid := core.ErrorCode("bogus")
	if invalid.Valid() {
		t.Errorf("ErrorCode(%q).Valid() = true, want false", invalid)
	}
}

// --- PluginError ---

func TestPluginErrorMessage(t *testing.T) {
	pe := &core.PluginError{
		Code:    core.ErrAuthExpired,
		Message: "token expired",
	}
	want := "[auth_expired] token expired"
	if got := pe.Error(); got != want {
		t.Errorf("PluginError.Error() = %q, want %q", got, want)
	}
}

func TestPluginErrorUnwrap(t *testing.T) {
	underlying := errors.New("connection reset")
	pe := &core.PluginError{
		Code: core.ErrUpstream,
		Raw:  underlying,
	}
	if !errors.Is(pe, underlying) {
		t.Error("PluginError.Unwrap() should make underlying error reachable via errors.Is")
	}
}

func TestPluginErrorUnwrapNil(t *testing.T) {
	pe := &core.PluginError{Code: core.ErrRateLimit}
	if pe.Unwrap() != nil {
		t.Error("PluginError.Unwrap() should return nil when Raw is nil")
	}
}

// --- MediaItem ---

func TestMediaItemDefaults(t *testing.T) {
	item := core.MediaItem{
		Platform:   "spotify",
		Type:       core.MediaMusic,
		Title:      "Test Track",
		ExternalID: "spotify:track:123",
	}

	if item.Duration != nil {
		t.Error("Duration should be nil by default")
	}
	if item.TimeSpent != nil {
		t.Error("TimeSpent should be nil by default")
	}
	if item.Tags != nil {
		t.Error("Tags should be nil by default")
	}
	if item.RawMetadata != nil {
		t.Error("RawMetadata should be nil by default")
	}
}

func TestMediaItemWithOptionalFields(t *testing.T) {
	dur := 3 * time.Minute
	spent := 2 * time.Minute
	item := core.MediaItem{
		Platform:   "youtube",
		Type:       core.MediaVideo,
		Title:      "Test Video",
		Creator:    "Test Channel",
		ConsumedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		Duration:   &dur,
		TimeSpent:  &spent,
		Tags:       []string{"comedy", "tech"},
		URL:        "https://youtube.com/watch?v=abc",
		ExternalID: "abc",
		RawMetadata: map[string]any{
			"views": 1000,
		},
	}

	if *item.Duration != 3*time.Minute {
		t.Errorf("Duration = %v, want 3m", *item.Duration)
	}
	if *item.TimeSpent != 2*time.Minute {
		t.Errorf("TimeSpent = %v, want 2m", *item.TimeSpent)
	}
	if len(item.Tags) != 2 {
		t.Errorf("Tags len = %d, want 2", len(item.Tags))
	}
	if item.RawMetadata["views"] != 1000 {
		t.Error("RawMetadata should preserve platform-specific fields")
	}
}

// --- SyncResult ---

func TestSyncResultSuccess(t *testing.T) {
	result := core.SyncResult{
		Items:      []core.MediaItem{{Title: "item1"}, {Title: "item2"}},
		NextCursor: "cursor-abc",
		HasMore:    false,
		Err:        nil,
	}

	if result.Err != nil {
		t.Error("Successful SyncResult should have nil Err")
	}
	if len(result.Items) != 2 {
		t.Errorf("Items len = %d, want 2", len(result.Items))
	}
	if result.NextCursor != "cursor-abc" {
		t.Errorf("NextCursor = %q, want %q", result.NextCursor, "cursor-abc")
	}
}

func TestSyncResultPartialFailure(t *testing.T) {
	result := core.SyncResult{
		Items:      []core.MediaItem{{Title: "partial"}},
		NextCursor: "resume-here",
		HasMore:    true,
		Err: &core.PluginError{
			Code:    core.ErrPartialSync,
			Message: "upstream error after page 3",
			Retry:   true,
		},
	}

	if result.Err == nil {
		t.Fatal("Partial SyncResult should have non-nil Err")
	}
	if result.Err.Code != core.ErrPartialSync {
		t.Errorf("Err.Code = %q, want %q", result.Err.Code, core.ErrPartialSync)
	}
	if !result.HasMore {
		t.Error("HasMore should be true for partial sync")
	}
	if len(result.Items) != 1 {
		t.Error("Partial sync should still carry collected items")
	}
}

// --- SourcePlugin (verify interface is implementable) ---

// stubPlugin is a minimal SourcePlugin used to verify the interface compiles.
type stubPlugin struct{}

func (s *stubPlugin) Name() string                  { return "stub" }
func (s *stubPlugin) AuthType() core.AuthType       { return core.AuthNone }
func (s *stubPlugin) AuthConfig() *core.OAuthConfig { return nil }
func (s *stubPlugin) Sync(_ context.Context, _ core.Credentials, _ string) core.SyncResult {
	return core.SyncResult{}
}
func (s *stubPlugin) Enrich(_ context.Context, _ core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	return items, nil
}

// Compile-time check that stubPlugin implements SourcePlugin.
var _ core.SourcePlugin = (*stubPlugin)(nil)

func TestStubPluginInterface(t *testing.T) {
	var p core.SourcePlugin = &stubPlugin{}
	if p.Name() != "stub" {
		t.Errorf("Name() = %q, want %q", p.Name(), "stub")
	}
	if p.AuthType() != core.AuthNone {
		t.Errorf("AuthType() = %v, want AuthNone", p.AuthType())
	}
	if p.AuthConfig() != nil {
		t.Error("AuthConfig() should be nil for AuthNone plugins")
	}

	result := p.Sync(context.Background(), core.Credentials{}, "")
	if result.Err != nil {
		t.Error("Stub Sync should succeed")
	}

	items, err := p.Enrich(context.Background(), core.Credentials{}, nil)
	if err != nil {
		t.Errorf("Stub Enrich returned error: %v", err)
	}
	if items != nil {
		t.Error("Stub Enrich should pass through nil items")
	}
}

// --- Credentials ---

func TestCredentialsOAuth(t *testing.T) {
	creds := core.Credentials{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
	}
	if creds.AccessToken != "access-123" {
		t.Error("AccessToken not set correctly")
	}
	if creds.RefreshToken != "refresh-456" {
		t.Error("RefreshToken not set correctly")
	}
}

func TestCredentialsFileImport(t *testing.T) {
	creds := core.Credentials{
		File: strings.NewReader("csv,data\n1,2"),
	}
	if creds.File == nil {
		t.Error("File should be non-nil")
	}
}

func TestCredentialsAPIKey(t *testing.T) {
	creds := core.Credentials{
		APIKey: "key-789",
	}
	if creds.APIKey != "key-789" {
		t.Error("APIKey not set correctly")
	}
}

// --- OAuthConfig ---

func TestOAuthConfig(t *testing.T) {
	cfg := core.OAuthConfig{
		ProviderName: "Spotify",
		AuthURL:      "https://accounts.spotify.com/authorize",
		TokenURL:     "https://accounts.spotify.com/api/token",
		Scopes:       []string{"user-read-recently-played", "user-library-read"},
	}
	if cfg.ProviderName != "Spotify" {
		t.Errorf("ProviderName = %q, want %q", cfg.ProviderName, "Spotify")
	}
	if len(cfg.Scopes) != 2 {
		t.Errorf("Scopes len = %d, want 2", len(cfg.Scopes))
	}
}
