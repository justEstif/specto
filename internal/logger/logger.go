// Package logger provides a centralized, structured logger for the
// application. It configures a single slog.Logger with JSON output and
// environment context, and exposes a wide-event builder for HTTP
// request logging.
package logger

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Config holds logger configuration set once at startup.
type Config struct {
	// Level sets the minimum log level. Defaults to slog.LevelInfo.
	Level slog.Level
	// ServiceName identifies this service in log events.
	ServiceName string
	// Version is the application version string (e.g. "1.0.0").
	Version string
}

// New creates a JSON slog.Logger with environment context baked in.
// It also sets slog.SetDefault so any code using slog.Default() gets
// the same configured logger.
func New(cfg Config) *slog.Logger {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "specto"
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.Level,
	})

	logger := slog.New(handler).With(
		"service", cfg.ServiceName,
		"version", cfg.Version,
		"commit", commitHash(),
	)

	slog.SetDefault(logger)
	return logger
}

var (
	cachedCommit string
	commitOnce   sync.Once
)

// commitHash returns the short git commit hash, cached after first call.
func commitHash() string {
	commitOnce.Do(func() {
		out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
		if err != nil {
			cachedCommit = "unknown"
			return
		}
		cachedCommit = strings.TrimSpace(string(out))
	})
	return cachedCommit
}

// --- Wide Event ---

// wideEventKey is the context key for the wide event map.
type contextKey string

const wideEventCtxKey contextKey = "wide_event"

// WideEvent is a map of fields accumulated over a request's lifetime
// and emitted as a single structured log event at request completion.
type WideEvent map[string]any

// NewWideEvent creates a new wide event and stores it in the context.
func NewWideEvent(ctx context.Context) (context.Context, WideEvent) {
	we := make(WideEvent, 16)
	return context.WithValue(ctx, wideEventCtxKey, we), we
}

// FromContext retrieves the wide event from the context. Returns nil
// if no wide event exists (e.g. outside the logging middleware).
func FromContext(ctx context.Context) WideEvent {
	we, _ := ctx.Value(wideEventCtxKey).(WideEvent)
	return we
}

// Set adds a key-value pair to the wide event. Safe to call when we is nil.
func (we WideEvent) Set(key string, value any) {
	if we == nil {
		return
	}
	we[key] = value
}

// SlogAttrs converts the wide event to slog key-value pairs for emission.
func (we WideEvent) SlogAttrs() []any {
	attrs := make([]any, 0, len(we)*2)
	for k, v := range we {
		attrs = append(attrs, k, v)
	}
	return attrs
}
