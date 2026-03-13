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
	SessionName   = "specto_session"
	sessionUserID = "user_id"
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
