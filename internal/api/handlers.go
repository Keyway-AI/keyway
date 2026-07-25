package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nometria/keyway/internal/app"
	"github.com/nometria/keyway/internal/blastradius"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/version"
	"github.com/nometria/keyway/pkg/apitypes"
)

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, apitypes.HealthResponse{
		Status: "ok", Version: version.Version, Time: time.Now().UTC(),
	})
}

// POST /v1/snapshots — discover, build, and store a contract version.
func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	res, err := s.deps.Snapshot(r.Context(), "manual")
	if err != nil {
		writeError(w, http.StatusBadGateway, "snapshot_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apitypes.SnapshotResponse{
		VersionID: res.Version.ID, Hash: res.Version.Hash, IsBaseline: res.IsBaseline,
	})
}

func (s *Server) handleLatestSnapshot(w http.ResponseWriter, r *http.Request) {
	v, err := s.deps.Store.LatestVersion(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "no snapshot yet")
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, err := s.deps.Store.GetContractVersion(r.Context(), id)
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "version not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleListConsumers(w http.ResponseWriter, r *http.Request) {
	v, err := s.deps.Store.LatestVersion(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, apitypes.ConsumerList{})
		return
	}
	issuerFilter := r.URL.Query().Get("issuer_id")
	probeableFilter := r.URL.Query().Get("probeable")
	var out []model.Consumer
	for _, c := range v.Consumers {
		if issuerFilter != "" && !contains(c.Expects.Issuers, issuerFilter) {
			continue
		}
		if probeableFilter == "true" && !c.Probeable {
			continue
		}
		if probeableFilter == "false" && c.Probeable {
			continue
		}
		out = append(out, c)
	}
	writeJSON(w, http.StatusOK, apitypes.ConsumerList{Consumers: out, Total: len(out)})
}

// handleConsumerProbes returns a consumer's recent probe results (newest first),
// powering the consumer detail drawer's probe-history panel.
func (s *Server) handleConsumerProbes(w http.ResponseWriter, r *http.Request) {
	stableID := chi.URLParam(r, "stable_id")
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	results, err := s.deps.Store.ProbeHistory(r.Context(), stableID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "could not load probe history")
		return
	}
	if results == nil {
		results = []model.ProbeResult{}
	}
	writeJSON(w, http.StatusOK, apitypes.ConsumerProbesResponse{ConsumerID: stableID, Results: results})
}

func (s *Server) handleGetConsumer(w http.ResponseWriter, r *http.Request) {
	stableID := chi.URLParam(r, "stable_id")
	v, err := s.deps.Store.LatestVersion(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "no snapshot yet")
		return
	}
	for _, c := range v.Consumers {
		if c.StableID == stableID {
			writeJSON(w, http.StatusOK, c)
			return
		}
	}
	writeError(w, http.StatusNotFound, "not_found", "consumer not found")
}

// POST /v1/probes/run — run probes against the latest consumers.
func (s *Server) handleRunProbes(w http.ResponseWriter, r *http.Request) {
	var req apitypes.RunProbesRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	results, err := s.deps.ProbeRun(r.Context(), req.ConsumerIDs)
	switch {
	case errors.Is(err, app.ErrNoSnapshot):
		writeError(w, http.StatusBadRequest, "no_snapshot", "take a snapshot first")
		return
	case errors.Is(err, app.ErrNoIssuer):
		writeError(w, http.StatusPreconditionFailed, "no_issuer", "no issuer configured to mint tokens")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "probe_failed", err.Error())
		return
	}
	runID := uuid.NewString()
	s.runs.put(runID, results)
	writeJSON(w, http.StatusOK, apitypes.RunProbesResponse{RunID: runID, Results: results})
}

func (s *Server) handleGetProbeRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "run_id")
	results, ok := s.runs.get(runID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "run not found (results are also queryable per consumer)")
		return
	}
	writeJSON(w, http.StatusOK, apitypes.RunProbesResponse{RunID: runID, Results: results})
}

func (s *Server) handleListChanges(w http.ResponseWriter, r *http.Request) {
	since := parseSince(r.URL.Query().Get("since"))
	events, err := s.deps.Store.ListChangeEvents(r.Context(), since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	classF := r.URL.Query().Get("class")
	sevF := r.URL.Query().Get("severity")
	var out []model.ChangeEvent
	for _, e := range events {
		if classF != "" && string(e.Class) != classF {
			continue
		}
		if sevF != "" && string(e.Severity) != sevF {
			continue
		}
		out = append(out, e)
	}
	writeJSON(w, http.StatusOK, apitypes.ChangeList{Changes: out, Total: len(out)})
}

func (s *Server) handleBlastRadius(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req apitypes.BlastRadiusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	proposal := blastradius.ChangeProposal{
		Kind: req.Proposal.Kind, IssuerID: req.Proposal.IssuerID, KID: req.Proposal.KID,
		ClaimName: req.Proposal.ClaimName, NewIssuerURL: req.Proposal.NewIssuerURL, Algorithm: req.Proposal.Algorithm,
	}
	res, err := s.deps.BlastRadius(ctx, proposal)
	switch {
	case errors.Is(err, app.ErrNoSnapshot):
		writeError(w, http.StatusBadRequest, "no_snapshot", "take a snapshot first")
		return
	case err != nil:
		writeError(w, http.StatusBadRequest, "bad_proposal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleCanaryAnnounce(w http.ResponseWriter, r *http.Request) {
	var req apitypes.AnnounceKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	iss, ok := s.deps.Issuers.Get(req.IssuerID)
	if !ok {
		writeError(w, http.StatusNotFound, "no_issuer", "issuer not found: "+req.IssuerID)
		return
	}
	alg := req.Alg
	if alg == "" {
		alg = "RS256"
	}
	key, err := iss.AnnounceKey(r.Context(), alg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "announce_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (s *Server) handleCanaryStatus(w http.ResponseWriter, r *http.Request) {
	iss, ok := s.deps.Issuers.Get(r.URL.Query().Get("issuer_id"))
	if !ok {
		writeError(w, http.StatusNotFound, "no_issuer", "issuer not found")
		return
	}
	desc, err := iss.Describe(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "describe_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer": desc.Name, "keys": desc.Keys, "announced_kid": iss.KeySet().AnnouncedKID(),
	})
}

func (s *Server) handleCanaryPromote(w http.ResponseWriter, r *http.Request) {
	var req apitypes.PromoteKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	iss, ok := s.deps.Issuers.Get(req.IssuerID)
	if !ok {
		writeError(w, http.StatusNotFound, "no_issuer", "issuer not found")
		return
	}
	if err := iss.PromoteKey(r.Context(), req.KID); err != nil {
		writeError(w, http.StatusInternalServerError, "promote_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"promoted": req.KID})
}

func (s *Server) handleCoverage(w http.ResponseWriter, r *http.Request) {
	v, err := s.deps.Store.LatestVersion(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, apitypes.CoverageResponse{})
		return
	}
	cov := apitypes.CoverageResponse{Total: len(v.Consumers)}
	for _, c := range v.Consumers {
		switch {
		case !c.Probeable:
			cov.Unresolved++
		case c.Confidence["overall"] < 0.6:
			cov.LowConfidence++
		default:
			cov.Resolved++
		}
	}
	writeJSON(w, http.StatusOK, cov)
}

// --- helpers ----------------------------------------------------------------

func parseSince(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d)
	}
	// Support a "7d" shorthand that time.ParseDuration does not.
	if n := len(s); n > 1 && s[n-1] == 'd' {
		if hours, err := time.ParseDuration(s[:n-1] + "h"); err == nil {
			return time.Now().Add(-hours * 24)
		}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
