package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/google/uuid"
	"github.com/gorilla/csrf"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// ListPlugins handles GET /api/v1/plugins
// Returns all registered plugins with per-user connection state.
func (h *Handler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	names := h.App.Registry.List()

	// Load per-user plugin states
	states, err := h.App.PluginStates.ListStates(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load plugin states")
		return
	}
	stateMap := make(map[string]*core.PluginStateInfo, len(states))
	for i := range states {
		stateMap[states[i].Plugin] = &states[i]
	}

	plugins := make([]map[string]any, 0, len(names))
	for _, name := range names {
		p := h.App.Registry.Get(name)
		if p == nil {
			continue
		}
		entry := pluginListEntry(p, stateMap[name])
		plugins = append(plugins, entry)
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": plugins})
}

// GetPlugin handles GET /api/v1/plugins/{plugin}
// Returns a single plugin's state and capabilities.
func (h *Handler) GetPlugin(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	pluginName := chi.URLParam(r, "plugin")
	p := h.App.Registry.Get(pluginName)
	if p == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Plugin %q not found", pluginName))
		return
	}

	state, _ := h.App.PluginStates.GetState(r.Context(), user.ID, pluginName)

	entry := pluginDetailEntry(p, state)
	writeJSON(w, http.StatusOK, map[string]any{"data": entry})
}

// ConnectPlugin handles POST /api/v1/plugins/{plugin}/connect
// Starts an OAuth connection flow for OAuth plugins.
func (h *Handler) ConnectPlugin(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	pluginName := chi.URLParam(r, "plugin")
	p := h.App.Registry.Get(pluginName)
	if p == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Plugin %q not found", pluginName))
		return
	}

	if p.AuthType() != core.AuthOAuth {
		writeError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("Plugin %q does not use OAuth", pluginName))
		return
	}

	cfg := p.AuthConfig()
	if cfg == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Plugin OAuth config is missing")
		return
	}

	// Generate a cryptographically random state for CSRF protection.
	state, err := auth.GenerateState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate OAuth state")
		return
	}

	// Store the state in the user's session so the callback can validate it.
	if err := h.App.Auth.Sessions.SetOAuthState(w, r, state, pluginName); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to save OAuth state")
		return
	}

	// Build the full OAuth authorization URL with all query parameters.
	redirectURL, err := h.App.OAuth.BuildAuthURL(pluginName, cfg, state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Failed to build OAuth URL: %s", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{
			"redirect_url": redirectURL,
		},
	})
}

// OAuthCallback handles GET /api/v1/plugins/{plugin}/callback
// Processes the OAuth provider's redirect after user authorization.
func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	pluginName := chi.URLParam(r, "plugin")
	p := h.App.Registry.Get(pluginName)
	if p == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Plugin %q not found", pluginName))
		return
	}

	if p.AuthType() != core.AuthOAuth {
		writeError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("Plugin %q does not use OAuth", pluginName))
		return
	}

	cfg := p.AuthConfig()
	if cfg == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Plugin OAuth config is missing")
		return
	}

	// Check for provider-side errors (e.g., user denied consent).
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		errDesc := r.URL.Query().Get("error_description")
		if errDesc == "" {
			errDesc = errCode
		}
		writeError(w, http.StatusBadRequest, "oauth_error", fmt.Sprintf("OAuth provider error: %s", errDesc))
		return
	}

	// Extract the authorization code and state from the query string.
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Missing code or state parameter")
		return
	}

	// Validate the state parameter against the session-stored value.
	expectedState, expectedPlugin, err := h.App.Auth.Sessions.GetOAuthState(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to read OAuth state from session")
		return
	}
	if expectedState == "" || state != expectedState {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid OAuth state parameter")
		return
	}
	if pluginName != expectedPlugin {
		writeError(w, http.StatusBadRequest, "validation_error", "OAuth callback plugin mismatch")
		return
	}

	// Exchange the authorization code for tokens.
	tokenResp, err := h.App.OAuth.ExchangeCode(pluginName, cfg, code)
	if err != nil {
		writeError(w, http.StatusBadGateway, "oauth_error", fmt.Sprintf("Token exchange failed: %s", err.Error()))
		return
	}

	// Compute expiry time from expires_in.
	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	// Store the OAuth credentials.
	creds := core.Credentials{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
	}
	if err := h.App.PluginStates.UpsertCredentials(r.Context(), user.ID, pluginName, core.AuthOAuth, creds, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to store credentials")
		return
	}

	// Mark plugin as connected.
	if _, err := h.App.PluginStates.UpsertState(r.Context(), user.ID, pluginName, "connected", true); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update plugin state")
		return
	}

	// Redirect back to the plugins page after successful OAuth connection.
	http.Redirect(w, r, "/plugins", http.StatusSeeOther)
}

// ImportPlugin handles POST /api/v1/plugins/{plugin}/import
// Uploads a file for file-import plugins.
func (h *Handler) ImportPlugin(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	pluginName := chi.URLParam(r, "plugin")
	p := h.App.Registry.Get(pluginName)
	if p == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Plugin %q not found", pluginName))
		return
	}

	if p.AuthType() != core.AuthFileImport {
		writeError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("Plugin %q does not support file import", pluginName))
		return
	}

	// Parse multipart form (max 32 MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid multipart form data")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "File field is required")
		return
	}
	defer file.Close()

	// Ensure plugin state exists as connected
	_, err = h.App.PluginStates.UpsertState(r.Context(), user.ID, pluginName, "connected", true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update plugin state")
		return
	}

	// Run sync with the file reader directly — io.Reader can't be
	// serialized to the credential store, so we pass it explicitly.
	summary, err := h.App.Syncer.SyncPluginWithFile(r.Context(), user.ID, pluginName, file)
	if err != nil {
		var rateLimitErr *core.RateLimitError
		if errors.As(err, &rateLimitErr) {
			writeError(w, http.StatusTooManyRequests, "rate_limit", rateLimitErr.Reason)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Import failed: %s", err.Error()))
		return
	}

	if h.renderPluginCard(w, r, user.ID, pluginName) {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"plugin":        pluginName,
			"status":        "connected",
			"imported":      true,
			"items_added":   summary.ItemsAdded,
			"items_skipped": summary.ItemsSkipped,
		},
	})
}

// DisconnectPlugin handles DELETE /api/v1/plugins/{plugin}/disconnect
// Disconnects a plugin and deletes stored credentials.
func (h *Handler) DisconnectPlugin(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	pluginName := chi.URLParam(r, "plugin")
	p := h.App.Registry.Get(pluginName)
	if p == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Plugin %q not found", pluginName))
		return
	}

	// Delete credentials
	if err := h.App.PluginStates.DeleteCredentials(r.Context(), user.ID, pluginName); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete credentials")
		return
	}

	// Update state to disconnected
	_, err := h.App.PluginStates.UpdateStatus(r.Context(), user.ID, pluginName, "disconnected", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update plugin state")
		return
	}

	if h.renderPluginCard(w, r, user.ID, pluginName) {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"plugin": pluginName,
			"status": "disconnected",
		},
	})
}

// SyncPlugin handles POST /api/v1/plugins/{plugin}/sync
// Triggers a sync for a connected plugin.
func (h *Handler) SyncPlugin(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	pluginName := chi.URLParam(r, "plugin")
	p := h.App.Registry.Get(pluginName)
	if p == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Plugin %q not found", pluginName))
		return
	}

	summary, err := h.App.Syncer.SyncPlugin(r.Context(), user.ID, pluginName)
	if err != nil {
		var rateLimitErr *core.RateLimitError
		if errors.As(err, &rateLimitErr) {
			retryAfter := int(rateLimitErr.RetryAfter.Seconds())
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error": map[string]any{
					"code":    "rate_limit",
					"message": rateLimitErr.Reason,
					"details": map[string]int{
						"retry_after_seconds": retryAfter,
					},
				},
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Sync failed: %s", err.Error()))
		return
	}

	// Determine status from summary
	status := "success"
	if summary.Error != nil {
		status = "partial"
	}

	// Get updated state for last_synced_at
	state, _ := h.App.PluginStates.GetState(r.Context(), user.ID, pluginName)

	if h.renderPluginCard(w, r, user.ID, pluginName) {
		return
	}

	resp := map[string]any{
		"plugin":        pluginName,
		"status":        status,
		"items_added":   summary.ItemsAdded,
		"items_skipped": summary.ItemsSkipped,
		"items_updated": summary.ItemsUpdated,
	}
	if state != nil && state.LastSyncedAt != nil {
		resp["last_synced_at"] = state.LastSyncedAt
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": resp})
}

// SyncHistory handles GET /api/v1/plugins/{plugin}/sync-history
// Returns recent sync runs for this plugin.
func (h *Handler) SyncHistory(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	pluginName := chi.URLParam(r, "plugin")
	p := h.App.Registry.Get(pluginName)
	if p == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Plugin %q not found", pluginName))
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := int32(20)
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = int32(v)
		}
	}

	logs, err := h.App.SyncLogs.List(r.Context(), user.ID, pluginName, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to load sync history")
		return
	}

	entries := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		entry := map[string]any{
			"started_at":    l.StartedAt,
			"status":        l.Status,
			"items_added":   l.ItemsAdded,
			"items_skipped": l.ItemsSkipped,
			"items_updated": l.ItemsUpdated,
			"error_code":    l.ErrorCode,
			"error_message": l.ErrorMessage,
		}
		if l.CompletedAt != nil {
			entry["completed_at"] = l.CompletedAt
		}
		entries = append(entries, entry)
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": entries})
}

// --- helpers ---

// pluginListEntry builds a plugin list response entry.
func pluginListEntry(p core.SourcePlugin, state *core.PluginStateInfo) map[string]any {
	entry := map[string]any{
		"name":         p.Name(),
		"auth_type":    p.AuthType().String(),
		"status":       "disconnected",
		"enabled":      true,
		"connected":    false,
		"display_name": pluginDisplayName(p),
	}

	if state != nil {
		entry["status"] = state.Status
		entry["enabled"] = state.Enabled
		entry["connected"] = state.Status == "connected"
		if state.LastSyncedAt != nil {
			entry["last_synced_at"] = state.LastSyncedAt
		}
		entry["error_message"] = state.ErrorMessage
	}

	return entry
}

// pluginDetailEntry builds a plugin detail response entry with capabilities.
func pluginDetailEntry(p core.SourcePlugin, state *core.PluginStateInfo) map[string]any {
	entry := pluginListEntry(p, state)

	authType := p.AuthType()
	entry["capabilities"] = map[string]bool{
		"can_connect":               authType == core.AuthOAuth,
		"can_disconnect":            authType == core.AuthOAuth || authType == core.AuthAPIKey,
		"can_import":                authType == core.AuthFileImport,
		"can_sync":                  authType == core.AuthOAuth || authType == core.AuthAPIKey,
		"supports_incremental_sync": authType == core.AuthOAuth || authType == core.AuthAPIKey,
	}

	return entry
}

// pluginDisplayName returns a display name from the plugin.
// If the plugin has an OAuthConfig with ProviderName, use that;
// otherwise capitalize the plugin name.
func pluginDisplayName(p core.SourcePlugin) string {
	if cfg := p.AuthConfig(); cfg != nil && cfg.ProviderName != "" {
		return cfg.ProviderName
	}
	name := p.Name()
	if len(name) == 0 {
		return name
	}
	// Capitalize first letter
	return strings.ToUpper(name[:1]) + name[1:]
}

// renderPluginCard re-renders a single PluginCard for HTMX responses.
// Returns true if it handled the response (HTMX request), false if the
// caller should fall back to JSON.
func (h *Handler) renderPluginCard(w http.ResponseWriter, r *http.Request, userID uuid.UUID, pluginName string) bool {
	if r.Header.Get("HX-Request") != "true" {
		return false
	}

	p := h.App.Registry.Get(pluginName)
	if p == nil {
		return false
	}

	state, _ := h.App.PluginStates.GetState(r.Context(), userID, pluginName)
	var stateInfo *core.PluginStateInfo
	if state != nil {
		stateInfo = state
	}
	view := buildPluginView(p, stateInfo)
	csrfToken := csrf.Token(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	components.PluginCard(view, csrfToken).Render(r.Context(), w)
	return true
}
