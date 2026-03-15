package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/csrf"

	"github.com/justestif/specto/components"
	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

// ConfirmEra handles POST /api/v1/eras/{id}/confirm — confirms a suggested era
// with its suggested title (or existing title).
func (h *Handler) ConfirmEra(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	eraID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid era ID")
		return
	}

	// Get the era to find its suggested title.
	era, err := h.App.Eras.Get(r.Context(), user.ID, eraID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Era not found")
		return
	}

	title := "Untitled era"
	if era.SuggestedTitle != nil {
		title = *era.SuggestedTitle
	}

	updated, err := h.App.Eras.UpdateTitle(r.Context(), eraID, title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update_failed", "Failed to confirm era")
		return
	}

	// Load tags for the re-rendered segment.
	tags, _ := h.App.Eras.GetTags(r.Context(), updated.ID)
	updated.Tags = tags

	renderEraSegment(w, r, *updated)
}

// UpdateEraTitle handles PUT /api/v1/eras/{id}/title — renames an era with a
// user-provided title (confirms it in the process).
func (h *Handler) UpdateEraTitle(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	eraID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid era ID")
		return
	}

	// Verify the era belongs to this user.
	if _, err := h.App.Eras.Get(r.Context(), user.ID, eraID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Era not found")
		return
	}

	title := r.FormValue("title")
	if title == "" {
		writeError(w, http.StatusBadRequest, "missing_title", "Title is required")
		return
	}
	if len(title) > 100 {
		title = title[:100]
	}

	updated, err := h.App.Eras.UpdateTitle(r.Context(), eraID, title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update_failed", "Failed to rename era")
		return
	}

	tags, _ := h.App.Eras.GetTags(r.Context(), updated.ID)
	updated.Tags = tags

	renderEraSegment(w, r, *updated)
}

// DismissEra handles DELETE /api/v1/eras/{id} — dismisses an era so it won't
// be suggested again.
func (h *Handler) DismissEra(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	eraID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid era ID")
		return
	}

	if err := h.App.Eras.Dismiss(r.Context(), user.ID, eraID); err != nil {
		writeError(w, http.StatusInternalServerError, "dismiss_failed", "Failed to dismiss era")
		return
	}

	// Return empty response — HTMX will remove the element via outerHTML swap.
	w.WriteHeader(http.StatusOK)
}

// renderEraSegment renders a single era segment partial for HTMX swap.
func renderEraSegment(w http.ResponseWriter, r *http.Request, era core.Era) {
	csrfToken := csrf.Token(r)
	// We render with index=0 and total=1 since these are standalone swaps.
	components.EraSegmentPartial(era, csrfToken).Render(r.Context(), w)
}
