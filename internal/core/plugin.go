// Package core defines the domain types and interfaces for the specto platform.
// It contains no implementation details — only the contracts that plugins,
// the store layer, sync orchestration, and the API layer depend on.
package core

import (
	"context"
	"io"
)

// SourcePlugin is the interface every media source must implement.
type SourcePlugin interface {
	// Name returns the unique identifier for this plugin (e.g., "spotify", "youtube").
	Name() string

	// AuthType returns how this plugin authenticates with its platform.
	AuthType() AuthType

	// AuthConfig returns OAuth configuration if AuthType is AuthOAuth.
	// Returns nil for non-OAuth plugins.
	AuthConfig() *OAuthConfig

	// Sync fetches media items from the platform.
	// Core provides credentials and the cursor from the last successful sync.
	// An empty cursor means first sync (full fetch).
	Sync(ctx context.Context, creds Credentials, cursor string) SyncResult

	// Enrich adds platform-specific tags and metadata using external APIs
	// (e.g., Spotify plugin calls Last.fm, Netflix plugin calls TMDB).
	// Core runs universal LLM enrichment separately after this.
	// Optional — return items unchanged if no platform-specific enrichment is needed.
	Enrich(ctx context.Context, creds Credentials, items []MediaItem) ([]MediaItem, error)
}

// AuthType describes how a plugin authenticates with its platform.
type AuthType int

const (
	AuthOAuth      AuthType = iota // Platform OAuth flow (Spotify, YouTube)
	AuthFileImport                 // User uploads an export file (Netflix CSV, TikTok JSON)
	AuthAPIKey                     // User provides an API key
	AuthNone                       // No auth needed
)

// String returns the human-readable name of the auth type.
func (a AuthType) String() string {
	switch a {
	case AuthOAuth:
		return "oauth"
	case AuthFileImport:
		return "file_import"
	case AuthAPIKey:
		return "api_key"
	case AuthNone:
		return "none"
	default:
		return "unknown"
	}
}

// OAuthConfig defines the OAuth parameters for a plugin.
// Core uses this to initiate the OAuth flow and handle callbacks.
type OAuthConfig struct {
	ProviderName string   // Display name (e.g., "Spotify")
	AuthURL      string   // Authorization endpoint
	TokenURL     string   // Token exchange endpoint
	Scopes       []string // Required OAuth scopes
}

// Credentials are passed to plugins by core. The plugin never stores these.
type Credentials struct {
	// OAuth plugins
	AccessToken  string
	RefreshToken string

	// File import plugins
	File io.Reader

	// API key plugins
	APIKey string
}
