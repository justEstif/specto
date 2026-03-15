package core_test

import (
	"context"
	"testing"

	"github.com/justestif/specto/internal/core"
)

// --- test helpers ---

// fakePlugin is a configurable SourcePlugin for registry tests.
type fakePlugin struct {
	name       string
	authType   core.AuthType
	authConfig *core.OAuthConfig
}

func (f *fakePlugin) Name() string                  { return f.name }
func (f *fakePlugin) AuthType() core.AuthType       { return f.authType }
func (f *fakePlugin) AuthConfig() *core.OAuthConfig { return f.authConfig }
func (f *fakePlugin) Sync(_ context.Context, _ core.Credentials, _ string) core.SyncResult {
	return core.SyncResult{}
}
func (f *fakePlugin) Enrich(_ context.Context, _ core.Credentials, items []core.MediaItem) ([]core.MediaItem, error) {
	return items, nil
}

func validOAuthConfig() *core.OAuthConfig {
	return &core.OAuthConfig{
		ProviderName: "TestProvider",
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		Scopes:       []string{"read"},
	}
}

// --- NewPluginRegistry ---

func TestNewPluginRegistry(t *testing.T) {
	r := core.NewPluginRegistry()
	if r == nil {
		t.Fatal("NewPluginRegistry returned nil")
	}
	if names := r.List(); len(names) != 0 {
		t.Errorf("new registry should be empty, got %v", names)
	}
}

// --- Register ---

func TestRegisterSuccess(t *testing.T) {
	r := core.NewPluginRegistry()
	p := &fakePlugin{name: "spotify", authType: core.AuthOAuth, authConfig: validOAuthConfig()}

	if err := r.Register(p); err != nil {
		t.Fatalf("Register() returned unexpected error: %v", err)
	}

	got := r.Get("spotify")
	if got == nil {
		t.Fatal("Get() returned nil after successful Register")
	}
	if got.Name() != "spotify" {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), "spotify")
	}
}

func TestRegisterDuplicateName(t *testing.T) {
	r := core.NewPluginRegistry()
	p1 := &fakePlugin{name: "netflix", authType: core.AuthFileImport}
	p2 := &fakePlugin{name: "netflix", authType: core.AuthFileImport}

	if err := r.Register(p1); err != nil {
		t.Fatalf("first Register() failed: %v", err)
	}
	if err := r.Register(p2); err == nil {
		t.Fatal("second Register() with same name should fail")
	}
}

func TestRegisterEmptyName(t *testing.T) {
	r := core.NewPluginRegistry()
	p := &fakePlugin{name: "", authType: core.AuthNone}

	if err := r.Register(p); err == nil {
		t.Fatal("Register() with empty name should fail")
	}
}

func TestRegisterMultiplePlugins(t *testing.T) {
	r := core.NewPluginRegistry()

	plugins := []*fakePlugin{
		{name: "spotify", authType: core.AuthOAuth, authConfig: validOAuthConfig()},
		{name: "netflix", authType: core.AuthFileImport},
		{name: "youtube", authType: core.AuthOAuth, authConfig: validOAuthConfig()},
	}

	for _, p := range plugins {
		if err := r.Register(p); err != nil {
			t.Fatalf("Register(%q) failed: %v", p.name, err)
		}
	}

	names := r.List()
	if len(names) != 3 {
		t.Fatalf("List() len = %d, want 3", len(names))
	}
}

// --- OAuth validation ---

func TestRegisterOAuthNilConfig(t *testing.T) {
	r := core.NewPluginRegistry()
	p := &fakePlugin{name: "bad-oauth", authType: core.AuthOAuth, authConfig: nil}

	if err := r.Register(p); err == nil {
		t.Fatal("Register() should fail for OAuth plugin with nil AuthConfig")
	}
}

func TestRegisterOAuthEmptyAuthURL(t *testing.T) {
	r := core.NewPluginRegistry()
	cfg := validOAuthConfig()
	cfg.AuthURL = ""
	p := &fakePlugin{name: "bad-oauth", authType: core.AuthOAuth, authConfig: cfg}

	if err := r.Register(p); err == nil {
		t.Fatal("Register() should fail for OAuth plugin with empty AuthURL")
	}
}

func TestRegisterOAuthEmptyTokenURL(t *testing.T) {
	r := core.NewPluginRegistry()
	cfg := validOAuthConfig()
	cfg.TokenURL = ""
	p := &fakePlugin{name: "bad-oauth", authType: core.AuthOAuth, authConfig: cfg}

	if err := r.Register(p); err == nil {
		t.Fatal("Register() should fail for OAuth plugin with empty TokenURL")
	}
}

func TestRegisterOAuthEmptyScopes(t *testing.T) {
	r := core.NewPluginRegistry()
	cfg := validOAuthConfig()
	cfg.Scopes = nil
	p := &fakePlugin{name: "no-scopes-oauth", authType: core.AuthOAuth, authConfig: cfg}

	if err := r.Register(p); err != nil {
		t.Fatalf("Register() should succeed for OAuth plugin with nil Scopes: %v", err)
	}
}

func TestRegisterNonOAuthIgnoresConfig(t *testing.T) {
	r := core.NewPluginRegistry()
	// A file-import plugin with no OAuth config should be fine.
	p := &fakePlugin{name: "netflix", authType: core.AuthFileImport, authConfig: nil}

	if err := r.Register(p); err != nil {
		t.Fatalf("Register() should succeed for non-OAuth plugin: %v", err)
	}
}

// --- Get ---

func TestGetNotFound(t *testing.T) {
	r := core.NewPluginRegistry()
	if got := r.Get("nonexistent"); got != nil {
		t.Errorf("Get() for missing plugin should return nil, got %v", got)
	}
}

// --- List ---

func TestListSorted(t *testing.T) {
	r := core.NewPluginRegistry()
	// Register in non-alphabetical order.
	for _, name := range []string{"youtube", "apple-music", "netflix", "spotify"} {
		p := &fakePlugin{name: name, authType: core.AuthNone}
		if err := r.Register(p); err != nil {
			t.Fatalf("Register(%q) failed: %v", name, err)
		}
	}

	names := r.List()
	want := []string{"apple-music", "netflix", "spotify", "youtube"}
	if len(names) != len(want) {
		t.Fatalf("List() len = %d, want %d", len(names), len(want))
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("List()[%d] = %q, want %q", i, name, want[i])
		}
	}
}
