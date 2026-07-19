// Package logging configures the process-wide structured logger (log/slog):
// human-readable text in development, JSON in production. It also provides the
// HTTP request-logging middleware that replaces chi's default printf logger,
// tagging every line with the chi request ID.
package logging

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// Setup installs the default slog logger for the whole process and returns it.
func Setup(production bool) *slog.Logger {
	var handler slog.Handler
	if production {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// RequestLogger logs one structured line per request: method, path, status,
// bytes, duration, remote addr, and the chi request ID (requires
// chi middleware.RequestID earlier in the chain).
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("http_request",
				"request_id", chimw.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", float64(time.Since(start).Microseconds())/1000.0,
				"remote", r.RemoteAddr,
			)
		})
	}
}
