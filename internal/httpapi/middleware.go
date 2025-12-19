package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type ctxKey int

const requestIDKey ctxKey = iota

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if id == "" {
				id = newRequestID()
			}
			w.Header().Set("X-Request-Id", id)
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: 200}

			next.ServeHTTP(rec, r)

			fields := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
			}
			if rid, ok := GetRequestID(r.Context()); ok {
				fields = append(fields, "request_id", rid)
			}
			logger.Info("http request", fields...)
		})
	}
}

func Recoverer(logger *slog.Logger, isProd bool) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if isProd {
						logger.Error("panic", "panic", rec)
					} else {
						logger.Error("panic", "panic", rec, "stack", string(debug.Stack()))
					}
					WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

func newRequestID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return time.Now().UTC().Format("20060102T150405.000000000Z07:00")
	}
	return hex.EncodeToString(b[:])
}
