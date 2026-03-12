package core

import (
	"fmt"
	"time"
)

// PluginError is the normalized error type all plugins must use.
// Core uses the Code to decide how to handle failures (retry, disconnect, etc.).
type PluginError struct {
	Code    ErrorCode     // normalized error category
	Message string        // human-readable description
	Retry   bool          // should core retry this operation?
	After   time.Duration // retry delay hint (for rate limits)
	Raw     error         // original platform-specific error (for logging)
}

// Error implements the error interface.
func (e *PluginError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying platform error for errors.Is/As chains.
func (e *PluginError) Unwrap() error {
	return e.Raw
}

// ErrorCode categorizes plugin errors so core can handle them uniformly.
type ErrorCode string

const (
	// ErrAuthExpired means the token needs refresh or re-authorization.
	// Core will attempt a token refresh via OAuth; if that fails, it marks
	// the plugin as disconnected and notifies the user.
	ErrAuthExpired ErrorCode = "auth_expired"

	// ErrRateLimit means the platform's API rate limit was hit.
	// Core backs off and retries after the After duration.
	ErrRateLimit ErrorCode = "rate_limit"

	// ErrPartialSync means some items were fetched before a failure.
	// Core stores the returned items, saves the cursor, and logs the error.
	ErrPartialSync ErrorCode = "partial_sync"

	// ErrUpstream means the platform API returned an error (500, timeout, etc.).
	// Core retries with exponential backoff up to a maximum.
	ErrUpstream ErrorCode = "upstream"

	// ErrInvalidData means the platform returned an unexpected response format.
	// Core logs the error with the raw response and does not retry.
	ErrInvalidData ErrorCode = "invalid_data"

	// ErrPermissionDenied means the OAuth scopes are insufficient.
	// Core notifies the user to re-authorize with the correct scopes.
	ErrPermissionDenied ErrorCode = "permission_denied"

	// ErrFileParseError means the uploaded import file is malformed.
	// Core returns the error to the user with position details.
	ErrFileParseError ErrorCode = "file_parse_error"
)

// ValidErrorCodes contains all recognized error codes.
var ValidErrorCodes = map[ErrorCode]bool{
	ErrAuthExpired:      true,
	ErrRateLimit:        true,
	ErrPartialSync:      true,
	ErrUpstream:         true,
	ErrInvalidData:      true,
	ErrPermissionDenied: true,
	ErrFileParseError:   true,
}

// Valid returns true if the error code is one of the recognized values.
func (ec ErrorCode) Valid() bool {
	return ValidErrorCodes[ec]
}
