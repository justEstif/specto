package auth

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/gorilla/sessions"
)

const (
	SessionName   = "specto_session"
	sessionUserID = "user_id"
)

// SessionStore is the application's session store.
var SessionStore sessions.Store

// InitSessions creates a cookie-based session store.
// In production, use a server-side store (PostgreSQL, Redis).
func InitSessions(secret []byte) {
	store := sessions.NewCookieStore(secret)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set true in production (HTTPS)
	}
	SessionStore = store
}

// SetUserSession stores the user ID in the session.
func SetUserSession(w http.ResponseWriter, r *http.Request, userID pgtype.UUID) error {
	session, err := SessionStore.Get(r, SessionName)
	if err != nil {
		return err
	}
	// Store as [16]byte
	session.Values[sessionUserID] = userID.Bytes
	return session.Save(r, w)
}

// GetUserIDFromSession retrieves the user ID from the session.
func GetUserIDFromSession(r *http.Request) (pgtype.UUID, error) {
	session, err := SessionStore.Get(r, SessionName)
	if err != nil {
		return pgtype.UUID{}, err
	}
	bytes, ok := session.Values[sessionUserID].([16]byte)
	if !ok {
		return pgtype.UUID{}, http.ErrNoCookie
	}
	return pgtype.UUID{Bytes: bytes, Valid: true}, nil
}

// ClearSession removes the user session.
func ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := SessionStore.Get(r, SessionName)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}
