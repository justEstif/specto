package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/justestif/specto/internal/auth"
	"github.com/justestif/specto/internal/core"
)

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles POST /api/v1/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Email, display_name, and password are required")
		return
	}

	if len(req.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Password must be at least 8 characters")
		return
	}

	user, err := h.App.Auth.Register(r.Context(), req.Email, req.DisplayName, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "validation_error", "Email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create account")
		return
	}

	if err := h.App.Auth.Sessions.SetUserSession(w, r, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create session")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"data": userResponse(user),
	})
}

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Email and password are required")
		return
	}

	user, err := h.App.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Login failed")
		return
	}

	if err := h.App.Auth.Sessions.SetUserSession(w, r, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create session")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": userResponse(user),
	})
}

// Logout handles DELETE /api/v1/session
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.App.Auth.Sessions.ClearSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// Session handles GET /api/v1/session
func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{
			"user": userResponse(user),
		},
	})
}

func userResponse(u *core.UserInfo) map[string]any {
	resp := map[string]any{
		"id":           fmt.Sprintf("%x", u.ID),
		"email":        u.Email,
		"display_name": u.DisplayName,
	}
	if u.AvatarURL != nil {
		resp["avatar_url"] = *u.AvatarURL
	}
	if u.ProfileSlug != nil {
		resp["profile_slug"] = *u.ProfileSlug
	}
	return resp
}
