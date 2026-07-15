package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"example.com/m/v2/internal/http/handlers"
	"example.com/m/v2/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	st := store.NewMemoryStore()
	h := handlers.NewHandler(st)

	mux := http.NewServeMux()
	mux.Handle("/api/v1/", h)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           withMiddleware(mux, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting api", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "err", err)
	}
}

func withMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return recoveryMiddleware(logger)(
		requestIDMiddleware(logger)(
			loggingMiddleware(logger)(
				timeoutMiddleware(5*time.Second, logger)(next),
			),
		),
	)
}

func requestIDMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = "dev"
			}
			w.Header().Set("X-Request-ID", requestID)
			ctx := r.Context()
			ctx = contextWithValue(ctx, "request_id", requestID)
			r = r.WithContext(ctx)
			logger.Debug("request started", "request_id", requestID, "method", r.Method, "path", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			logger.Info("request completed", "method", r.Method, "path", r.URL.Path, "status", recorder.Status(), "duration", time.Since(start))
		})
	}
}

func recoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "recover", recovered, "path", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "internal server error"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func timeoutMiddleware(timeout time.Duration, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			done := make(chan struct{})
			writer := &timeoutWriter{ResponseWriter: w}

			go func() {
				next.ServeHTTP(writer, r)
				close(done)
			}()

			select {
			case <-done:
				return
			case <-time.After(timeout):
				logger.Warn("request timed out", "path", r.URL.Path)
				writer.mu.Lock()
				writer.timedOut = true
				writer.mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "request timed out"})
			}
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

type timeoutWriter struct {
	http.ResponseWriter
	mu       sync.Mutex
	timedOut bool
}

func (w *timeoutWriter) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *timeoutWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	return w.ResponseWriter.Write(b)
}

func contextWithValue(ctx context.Context, key string, value any) context.Context {
	return context.WithValue(ctx, key, value)
}
