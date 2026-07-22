// Package api serves the Keyway HTTP API (PRD §12) and, in production builds,
// the embedded web dashboard.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/architsharma/keyway/internal/store"
	"github.com/architsharma/keyway/pkg/apitypes"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config configures the API server.
type Config struct {
	Addr  string
	Token string // bearer token required by all /v1 endpoints
}

// Server holds server dependencies.
type Server struct {
	cfg   Config
	store store.Store
}

// NewServer constructs an API server.
func NewServer(cfg Config, st store.Store) *Server {
	return &Server{cfg: cfg, store: st}
}

// Routes builds the HTTP handler.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Health is unauthenticated for probes/liveness.
	r.Get("/v1/health", s.handleHealth)

	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		r.Post("/v1/snapshots", s.handleCreateSnapshot)
		r.Get("/v1/snapshots/latest", s.handleLatestSnapshot)
		r.Get("/v1/snapshots/{id}", s.handleGetSnapshot)

		r.Get("/v1/consumers", s.handleListConsumers)
		r.Get("/v1/consumers/{stable_id}", s.handleGetConsumer)

		r.Post("/v1/probes/run", s.handleRunProbes)
		r.Get("/v1/probes/runs/{run_id}", s.handleGetProbeRun)

		r.Get("/v1/changes", s.handleListChanges)

		r.Post("/v1/blast-radius", s.handleBlastRadius)

		r.Post("/v1/canary/announce", s.handleCanaryAnnounce)
		r.Get("/v1/canary/status", s.handleCanaryStatus)
		r.Post("/v1/canary/promote", s.handleCanaryPromote)

		r.Get("/v1/coverage", s.handleCoverage)
	})

	// TODO(M9): serve the embedded web dashboard (web/dist) for non-/v1 paths.
	return r
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	return srv.ListenAndServe()
}

// authMiddleware enforces the bearer token on protected routes.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Token == "" {
			writeError(w, http.StatusInternalServerError, "server_misconfigured", "no API token configured")
			return
		}
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" || token != s.cfg.Token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "valid bearer token required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- helpers ----------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apitypes.ErrorResponse{Error: msg, Code: code})
}
