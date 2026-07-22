package api

import (
	"net/http"
	"time"

	"github.com/architsharma/keyway/internal/version"
	"github.com/architsharma/keyway/pkg/apitypes"
)

// handleHealth is fully implemented; the rest return 501 until their milestone
// wires the underlying logic (see PROGRESS.md).
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, apitypes.HealthResponse{
		Status:  "ok",
		Version: version.Version,
		Time:    time.Now().UTC(),
	})
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M4", "POST /v1/snapshots")
}
func (s *Server) handleLatestSnapshot(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M4", "GET /v1/snapshots/latest")
}
func (s *Server) handleGetSnapshot(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M4", "GET /v1/snapshots/{id}")
}
func (s *Server) handleListConsumers(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M4", "GET /v1/consumers")
}
func (s *Server) handleGetConsumer(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M4", "GET /v1/consumers/{stable_id}")
}
func (s *Server) handleRunProbes(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M3", "POST /v1/probes/run")
}
func (s *Server) handleGetProbeRun(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M3", "GET /v1/probes/runs/{run_id}")
}
func (s *Server) handleListChanges(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M4", "GET /v1/changes")
}
func (s *Server) handleBlastRadius(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M7", "POST /v1/blast-radius")
}
func (s *Server) handleCanaryAnnounce(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M6", "POST /v1/canary/announce")
}
func (s *Server) handleCanaryStatus(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M6", "GET /v1/canary/status")
}
func (s *Server) handleCanaryPromote(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M6", "POST /v1/canary/promote")
}
func (s *Server) handleCoverage(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w, "M9", "GET /v1/coverage")
}

func notImplemented(w http.ResponseWriter, milestone, what string) {
	writeError(w, http.StatusNotImplemented, "not_implemented",
		what+" is not implemented yet ("+milestone+" — see PROGRESS.md)")
}
