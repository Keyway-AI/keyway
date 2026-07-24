// Package api serves the Keyway HTTP API (PRD §12) and the embedded web
// dashboard.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nometria/keyway/internal/contract"
	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/issuerregistry"
	"github.com/nometria/keyway/internal/libdefaults"
	"github.com/nometria/keyway/internal/probe"
	"github.com/nometria/keyway/internal/store"
	"github.com/nometria/keyway/pkg/apitypes"
)

// Config configures the API server.
type Config struct {
	Addr  string
	Token string // bearer token required by all /v1 endpoints
}

// Deps are the runtime dependencies the handlers operate on.
type Deps struct {
	Store       store.Store
	Issuers     *issuerregistry.Registry
	Libs        *libdefaults.DB
	Discoverers []discovery.Discoverer
	Scope       discovery.Scope
	Probe       probe.EngineConfig
	// Attributor, when set, binds each change event to its cause (commit/deploy/
	// IdP admin action) as snapshots are taken. Optional.
	Attributor contract.Attributor
}

// Server holds server dependencies.
type Server struct {
	cfg  Config
	deps Deps
	idem *idemStore
	runs *runIndex
}

// NewServer constructs an API server.
func NewServer(cfg Config, deps Deps) *Server {
	return &Server{cfg: cfg, deps: deps, idem: newIdemStore(24 * time.Hour), runs: newRunIndex(256)}
}

// Routes builds the HTTP handler.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/v1/health", s.handleHealth)

	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(maxBytes(1 << 20)) // cap request bodies at 1 MiB before they are read
		r.Use(s.idempotency)     // replays POSTs that repeat an Idempotency-Key

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

	// Everything else serves the embedded web dashboard.
	r.NotFound(spaHandler().ServeHTTP)
	return r
}

// ListenAndServe starts the HTTP server and shuts down on ctx cancellation.
func (s *Server) ListenAndServe(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second, // > the 60s request timeout, so slow probes can still respond
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Token == "" {
			writeError(w, http.StatusInternalServerError, "server_misconfigured", "no API token configured")
			return
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		// Constant-time comparison to avoid leaking the token via timing.
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Token)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "valid bearer token required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// maxBytes caps the size of a request body a handler can read, so a client
// cannot exhaust memory by streaming an unbounded payload. It sets a hard limit
// via http.MaxBytesReader; reads past the limit fail and the handler surfaces a
// 413 through its normal decode-error path.
func maxBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apitypes.ErrorResponse{Error: msg, Code: code})
}
