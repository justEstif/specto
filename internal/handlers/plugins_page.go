package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/csrf"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// PluginsPage renders GET /plugins — the plugin management page.
func (h *Handler) PluginsPage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	names := h.App.Registry.List()

	// Load per-user plugin states.
	states, err := h.App.PluginStates.ListStates(r.Context(), user.ID)
	if err != nil {
		// Non-fatal — show plugins without state data.
		states = nil
	}
	stateMap := make(map[string]*core.PluginStateInfo, len(states))
	for i := range states {
		stateMap[states[i].Plugin] = &states[i]
	}

	var connected, available []components.PluginViewData
	for _, name := range names {
		p := h.App.Registry.Get(name)
		if p == nil {
			continue
		}
		view := buildPluginView(p, stateMap[name])
		if view.Connected {
			connected = append(connected, view)
		}
		// Always show in available list (connected plugins can re-upload or reconnect)
		available = append(available, view)
	}

	data := components.PluginsPageData{
		User:      user,
		Connected: connected,
		Available: available,
		CSRFToken: csrf.Token(r),
	}
	components.PluginsPage(data).Render(r.Context(), w)
}

// buildPluginView creates display data for a single plugin.
func buildPluginView(p core.SourcePlugin, state *core.PluginStateInfo) components.PluginViewData {
	view := components.PluginViewData{
		Name:        p.Name(),
		DisplayName: pluginDisplayName(p),
		AuthType:    p.AuthType(),
		Status:      "disconnected",
	}

	if state != nil {
		view.Status = state.Status
		view.Connected = state.Status == "connected"
		if state.LastSyncedAt != nil {
			view.LastSynced = formatRelativeTime(*state.LastSyncedAt)
		}
		if state.ErrorMessage != nil && *state.ErrorMessage != "" {
			view.ErrorMsg = *state.ErrorMessage
		}
	}

	return view
}

// formatRelativeTime formats a time.Time as a relative string.
func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}
