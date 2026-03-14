package handlers

import (
	"fmt"
	"net/http"

	"github.com/justestif/specto/internal/auth"
)

// --- Google OAuth App Login ---

// GoogleLogin handles GET /auth/google/login
// Initiates the Google OAuth flow for app login.
func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	h.oauthLogin(w, r, "google")
}

// GoogleCallback handles GET /auth/google/callback
// Processes the Google OAuth callback after user authorization.
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	tokenResp, err := h.oauthCallback(w, r, "google")
	if err != nil {
		http.Redirect(w, r, "/login?error="+err.Error(), http.StatusFound)
		return
	}

	// Fetch user info from Google
	userInfo, err := auth.FetchGoogleUserInfo(tokenResp.AccessToken)
	if err != nil {
		http.Redirect(w, r, "/login?error=failed+to+fetch+user+info", http.StatusFound)
		return
	}

	h.completeOAuthLogin(w, r, "google", userInfo)
}

// --- GitHub OAuth App Login ---

// GithubLogin handles GET /auth/github/login
// Initiates the GitHub OAuth flow for app login.
func (h *Handler) GithubLogin(w http.ResponseWriter, r *http.Request) {
	h.oauthLogin(w, r, "github")
}

// GithubCallback handles GET /auth/github/callback
// Processes the GitHub OAuth callback after user authorization.
func (h *Handler) GithubCallback(w http.ResponseWriter, r *http.Request) {
	tokenResp, err := h.oauthCallback(w, r, "github")
	if err != nil {
		http.Redirect(w, r, "/login?error="+err.Error(), http.StatusFound)
		return
	}

	// Fetch user info from GitHub
	userInfo, err := auth.FetchGithubUserInfo(tokenResp.AccessToken)
	if err != nil {
		http.Redirect(w, r, "/login?error=failed+to+fetch+user+info", http.StatusFound)
		return
	}

	h.completeOAuthLogin(w, r, "github", userInfo)
}

// --- Shared helpers ---

// oauthLogin initiates an OAuth login flow for the given provider.
func (h *Handler) oauthLogin(w http.ResponseWriter, r *http.Request, provider string) {
	cfg := h.App.OAuth.AppAuthConfig(provider)
	if cfg == nil {
		http.Redirect(w, r, "/login?error=provider+not+configured", http.StatusFound)
		return
	}

	state, err := auth.GenerateState()
	if err != nil {
		http.Redirect(w, r, "/login?error=internal+error", http.StatusFound)
		return
	}

	if err := h.App.Auth.Sessions.SetOAuthState(w, r, state, "app:"+provider); err != nil {
		http.Redirect(w, r, "/login?error=internal+error", http.StatusFound)
		return
	}

	redirectURL, err := h.App.OAuth.BuildAppAuthURL(provider, cfg, state)
	if err != nil {
		http.Redirect(w, r, "/login?error=internal+error", http.StatusFound)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// oauthCallback validates the OAuth callback and exchanges the code for tokens.
func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request, provider string) (*auth.TokenResponse, error) {
	// Check for provider-side errors
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		return nil, fmt.Errorf("oauth+denied")
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		return nil, fmt.Errorf("missing+parameters")
	}

	// Validate state
	expectedState, expectedPlugin, err := h.App.Auth.Sessions.GetOAuthState(w, r)
	if err != nil || expectedState == "" || state != expectedState {
		return nil, fmt.Errorf("invalid+state")
	}
	if expectedPlugin != "app:"+provider {
		return nil, fmt.Errorf("provider+mismatch")
	}

	cfg := h.App.OAuth.AppAuthConfig(provider)
	if cfg == nil {
		return nil, fmt.Errorf("provider+not+configured")
	}

	tokenResp, err := h.App.OAuth.ExchangeAppAuthCode(provider, cfg, code)
	if err != nil {
		return nil, fmt.Errorf("token+exchange+failed")
	}

	return tokenResp, nil
}

// completeOAuthLogin upserts the user and creates a session.
func (h *Handler) completeOAuthLogin(w http.ResponseWriter, r *http.Request, provider string, info *auth.ProviderUserInfo) {
	ctx := r.Context()

	// Try to find existing user by auth provider + subject
	user, err := h.App.Users.GetByAuth(ctx, provider, info.Subject)
	if err != nil {
		// User doesn't exist — create
		user, err = h.App.Users.Create(ctx, info.Email, info.DisplayName, info.AvatarURL, provider, info.Subject)
		if err != nil {
			http.Redirect(w, r, "/login?error=account+creation+failed", http.StatusFound)
			return
		}
	}

	// Create session
	if err := h.App.Auth.Sessions.SetUserSession(w, r, user.ID); err != nil {
		http.Redirect(w, r, "/login?error=session+creation+failed", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
