package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nometria/keyway/internal/blastradius"
	"github.com/nometria/keyway/internal/contract"
	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/issuerregistry"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/probe"
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
	res, err := s.Snapshot(r.Context(), "manual")
	if err != nil {
		writeError(w, http.StatusBadGateway, "snapshot_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apitypes.SnapshotResponse{
		VersionID: res.Version.ID, Hash: res.Version.Hash, IsBaseline: res.IsBaseline,
	})
}

// Snapshot runs discovery, enriches with library defaults, builds a contract
// version, and stores it via the baseline flow. Shared by the HTTP handler and
// the scheduler.
func (s *Server) Snapshot(ctx context.Context, trigger string) (contract.SnapshotResult, error) {
	var consumers []model.Consumer
	if len(s.deps.Discoverers) > 0 {
		cs, err := discovery.Run(ctx, s.deps.Scope, s.deps.Discoverers...)
		if err != nil {
			return contract.SnapshotResult{}, err
		}
		consumers = cs
	}
	if s.deps.Libs != nil {
		for i := range consumers {
			s.deps.Libs.Enrich(&consumers[i])
		}
	}
	v := contract.Build(contract.BuildInput{Consumers: consumers, TriggerKind: trigger})
	return contract.Snapshot(ctx, s.deps.Store, v)
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
	ctx := r.Context()
	var req apitypes.RunProbesRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	v, err := s.deps.Store.LatestVersion(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "no_snapshot", "take a snapshot first")
		return
	}
	consumers := v.Consumers
	if len(req.ConsumerIDs) > 0 {
		consumers = filterByStableID(consumers, req.ConsumerIDs)
	}

	iss, ok := s.firstIssuer()
	if !ok {
		writeError(w, http.StatusPreconditionFailed, "no_issuer", "no issuer configured to mint tokens")
		return
	}
	issModel, err := iss.Describe(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "issuer_describe_failed", err.Error())
		return
	}
	eng := probe.NewEngine(s.deps.Probe)
	results, _, err := eng.Run(ctx, issModel, mintFuncOf(iss), consumers)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "probe_failed", err.Error())
		return
	}
	if err := s.deps.Store.SaveProbeResults(ctx, results); err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	runID := uuid.NewString()
	s.runs.put(runID, results)
	writeJSON(w, http.StatusOK, map[string]any{"run_id": runID, "results": results})
}

func (s *Server) handleGetProbeRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "run_id")
	results, ok := s.runs.get(runID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "run not found (results are also queryable per consumer)")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run_id": runID, "results": results})
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
	v, err := s.deps.Store.LatestVersion(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "no_snapshot", "take a snapshot first")
		return
	}
	if s.deps.Libs != nil {
		for i := range v.Consumers {
			s.deps.Libs.Enrich(&v.Consumers[i])
		}
	}
	history := map[string][]model.ProbeResult{}
	for _, c := range v.Consumers {
		if h, herr := s.deps.Store.ProbeHistory(ctx, c.StableID, 50); herr == nil && len(h) > 0 {
			history[c.StableID] = h
		}
	}
	proposal := blastradius.ChangeProposal{
		Kind: req.Proposal.Kind, IssuerID: req.Proposal.IssuerID, KID: req.Proposal.KID,
		ClaimName: req.Proposal.ClaimName, NewIssuerURL: req.Proposal.NewIssuerURL, Algorithm: req.Proposal.Algorithm,
	}
	res, err := blastradius.Resolve(v, proposal, history, time.Now())
	if err != nil {
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

func (s *Server) firstIssuer() (issuerregistry.CanaryIssuer, bool) {
	names := s.deps.Issuers.Names()
	if len(names) == 0 {
		return nil, false
	}
	return s.deps.Issuers.Get(names[0])
}

func mintFuncOf(iss issuerregistry.CanaryIssuer) probe.MintFunc {
	return func(kid string, claims map[string]any) (string, error) {
		return iss.MintToken(context.Background(), kid, claims)
	}
}

func filterByStableID(cs []model.Consumer, ids []string) []model.Consumer {
	want := map[string]struct{}{}
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var out []model.Consumer
	for _, c := range cs {
		if _, ok := want[c.StableID]; ok {
			out = append(out, c)
		}
	}
	return out
}

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
