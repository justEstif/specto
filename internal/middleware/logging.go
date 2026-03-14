package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/justestif/specto/internal/logger"
)

// responseRecorder wraps http.ResponseWriter to capture status code and bytes written.
type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// WideEventLogger returns middleware that emits a single wide event per
// request at completion. It replaces chi's middleware.Logger with a
// structured, context-rich alternative.
//
// Handlers can add business context to the wide event via:
//
//	we := logger.FromContext(r.Context())
//	we.Set("user_id", userID)
func WideEventLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ctx, we := logger.NewWideEvent(r.Context())

			// Pre-populate request fields.
			we.Set("method", r.Method)
			we.Set("path", r.URL.Path)
			if r.URL.RawQuery != "" {
				we.Set("query", r.URL.RawQuery)
			}
			we.Set("remote_addr", r.RemoteAddr)

			// Use chi's request ID if present.
			if reqID := chimw.GetReqID(ctx); reqID != "" {
				we.Set("request_id", reqID)
			}

			rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

			defer func() {
				we.Set("status", rec.status)
				we.Set("bytes", rec.bytes)
				we.Set("duration_ms", time.Since(start).Milliseconds())

				level := slog.LevelInfo
				if rec.status >= 500 {
					level = slog.LevelError
				}

				log.Log(ctx, level, "http_request", we.SlogAttrs()...)
			}()

			next.ServeHTTP(rec, r.WithContext(ctx))
		})
	}
}
