package auth

import (
	"encoding/gob"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

func init() {
	gob.Register([16]byte{})
}

const (
	SessionName        = "specto_session"
	sessionUserID      = "user_id"
	sessionOAuthState  = "oauth_state"
	sessionOAuthPlugin = "oauth_plugin"
)

// SessionManager wraps a session store, eliminating the package-level global.
type SessionManager struct {
	store sessions.Store
}

// NewSessionManager creates a SessionManager with a cookie-based store.
// In production, consider a server-side store (PostgreSQL, Redis).
func NewSessionManager(secret []byte) *SessionManager {
	store := sessions.NewCookieStore(secret)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set true in production (HTTPS)
	}
	return &SessionManager{store: store}
}

// SetUserSession stores the user ID in the session.
func (sm *SessionManager) SetUserSession(w http.ResponseWriter, r *http.Request, userID uuid.UUID) error {
	session, err := sm.store.Get(r, SessionName)
	if err != nil {
		return err
	}
	session.Values[sessionUserID] = [16]byte(userID)
	return session.Save(r, w)
}

// GetUserIDFromSession retrieves the user ID from the session.
func (sm *SessionManager) GetUserIDFromSession(r *http.Request) (uuid.UUID, error) {
	session, err := sm.store.Get(r, SessionName)
	if err != nil {
		return uuid.Nil, err
	}
	bytes, ok := session.Values[sessionUserID].([16]byte)
	if !ok {
		return uuid.Nil, http.ErrNoCookie
	}
	return uuid.UUID(bytes), nil
}

// ClearSession removes the user session.
func (sm *SessionManager) ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := sm.store.Get(r, SessionName)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// SetOAuthState stores the OAuth state token and associated plugin name
// in the session. Used for CSRF protection during the OAuth flow.
func (sm *SessionManager) SetOAuthState(w http.ResponseWriter, r *http.Request, state, pluginName string) error {
	session, err := sm.store.Get(r, SessionName)
	if err != nil {
		return err
	}
	session.Values[sessionOAuthState] = state
	session.Values[sessionOAuthPlugin] = pluginName
	return session.Save(r, w)
}

// GetOAuthState retrieves and clears the OAuth state and plugin name
// from the session. Returns empty strings if not found.
func (sm *SessionManager) GetOAuthState(w http.ResponseWriter, r *http.Request) (state, pluginName string, err error) {
	session, err := sm.store.Get(r, SessionName)
	if err != nil {
		return "", "", err
	}

	s, _ := session.Values[sessionOAuthState].(string)
	p, _ := session.Values[sessionOAuthPlugin].(string)

	// Clear the OAuth state after reading (one-time use)
	delete(session.Values, sessionOAuthState)
	delete(session.Values, sessionOAuthPlugin)
	if err := session.Save(r, w); err != nil {
		return "", "", err
	}

	return s, p, nil
}
