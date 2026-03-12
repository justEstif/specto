package handlers

import (
	"net/http"

	"github.com/justestif/specto/components"
)

// Home renders the landing page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	components.Home().Render(r.Context(), w)
}
