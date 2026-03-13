package core

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// PluginRegistry holds all registered source plugins.
// Plugins are registered once at startup; after that the registry is read-only.
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]SourcePlugin
}

// NewPluginRegistry creates an empty plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]SourcePlugin),
	}
}

// Register adds a plugin to the registry.
// It returns an error if:
//   - a plugin with the same name is already registered
//   - an OAuth plugin provides a nil or invalid OAuthConfig
func (r *PluginRegistry) Register(p SourcePlugin) error {
	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin name must not be empty")
	}

	if err := validateOAuth(p); err != nil {
		return fmt.Errorf("plugin %q: %w", name, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q is already registered", name)
	}

	r.plugins[name] = p
	return nil
}

// Get returns the plugin with the given name, or nil if not found.
func (r *PluginRegistry) Get(name string) SourcePlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.plugins[name]
}

// List returns the names of all registered plugins, sorted alphabetically.
func (r *PluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Platforms returns the unique platform names derived from registered plugins.
// Plugin names like "spotify-api" are normalized to "spotify" by stripping
// any "-api" suffix, so the result contains only base platform names.
func (r *PluginRegistry) Platforms() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]struct{})
	for name := range r.plugins {
		platform := strings.TrimSuffix(name, "-api")
		seen[platform] = struct{}{}
	}

	platforms := make([]string, 0, len(seen))
	for p := range seen {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)
	return platforms
}

// validateOAuth checks that OAuth plugins provide a valid OAuthConfig.
func validateOAuth(p SourcePlugin) error {
	if p.AuthType() != AuthOAuth {
		return nil
	}

	cfg := p.AuthConfig()
	if cfg == nil {
		return fmt.Errorf("OAuth plugin must provide a non-nil AuthConfig")
	}
	if cfg.AuthURL == "" {
		return fmt.Errorf("OAuthConfig.AuthURL must not be empty")
	}
	if cfg.TokenURL == "" {
		return fmt.Errorf("OAuthConfig.TokenURL must not be empty")
	}
	if len(cfg.Scopes) == 0 {
		return fmt.Errorf("OAuthConfig.Scopes must not be empty")
	}
	return nil
}
