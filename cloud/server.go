package cloud

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Keyway-AI/keyway/internal/agentauth"
	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/threats"
	"github.com/Keyway-AI/keyway/pkg/apitypes"
)

// Server is the Keyway Cloud HTTP API. It owns accounts, projects, and analysis
// history; the analysis itself delegates to the shared engine (see analyze.go).
type Server struct {
	cfg    Config
	store  Store
	gh     *GitHub
	tokens map[string]string // userID -> github token (in-memory; Postgres would encrypt-persist)
	tokmu  sync.Mutex
}

type ctxKey int

const userCtxKey ctxKey = 0

// NewServer wires the cloud server over a store.
func NewServer(cfg Config, store Store) *Server {
	return &Server{
		cfg:    cfg,
		store:  store,
		gh:     NewGitHub(cfg.GitHubClientID, cfg.GitHubSecret),
		tokens: map[string]string{},
	}
}

// Routes builds the HTTP handler.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.cors)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	// Public tools & auth.
	r.Get("/v1/auth/github/login", s.handleGitHubLogin)
	r.Get("/v1/auth/github/callback", s.handleGitHubCallback)
	r.Post("/v1/auth/logout", s.handleLogout)
	r.Get("/v1/config", s.handlePublicConfig)
	r.Post("/v1/agent/inspect", s.handleAgentInspect)
	r.Get("/v1/threats/coverage", s.handleThreatCoverage)
	if s.cfg.DevLogin {
		r.Post("/v1/auth/dev-login", s.handleDevLogin)
	}

	// Authenticated: accounts & projects.
	r.Group(func(r chi.Router) {
		r.Use(s.requireUser)
		r.Get("/v1/me", s.handleMe)
		r.Post("/v1/tokens", s.handleMintToken)
		r.Get("/v1/projects", s.handleListProjects)
		r.Post("/v1/projects", s.handleCreateProject)
		r.Get("/v1/projects/{id}", s.handleGetProject)
		r.Delete("/v1/projects/{id}", s.handleDeleteProject)
		r.Post("/v1/projects/{id}/analyze", s.handleAnalyze)
		r.Get("/v1/projects/{id}/analyses", s.handleListAnalyses)
		r.Get("/v1/analyses/{id}", s.handleGetAnalysis)
	})
	return r
}

// ─── auth ────────────────────────────────────────────────────────────────

func (s *Server) handlePublicConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"github_login": s.gh.Configured(),
		"dev_login":    s.cfg.DevLogin,
	})
}

func (s *Server) handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if !s.gh.Configured() {
		writeErr(w, http.StatusServiceUnavailable, "github oauth not configured (set GITHUB_CLIENT_ID/SECRET)")
		return
	}
	state := newID()
	http.SetCookie(w, s.cookie("keyway_oauth_state", state, 10*time.Minute))
	scope := "read:user user:email"
	if r.URL.Query().Get("repos") == "1" {
		scope += " repo"
	}
	http.Redirect(w, r, s.gh.AuthURL(s.cfg.BaseURL+"/v1/auth/github/callback", state, scope), http.StatusFound)
}

func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	sc, _ := r.Cookie("keyway_oauth_state")
	if code == "" || state == "" || sc == nil || sc.Value != state {
		writeErr(w, http.StatusBadRequest, "invalid oauth callback")
		return
	}
	token, err := s.gh.Exchange(r.Context(), code, s.cfg.BaseURL+"/v1/auth/github/callback")
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	u, err := s.gh.FetchUser(r.Context(), token)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if err := s.store.UpsertUser(r.Context(), u); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.setToken(u.ID, token)
	http.SetCookie(w, s.cookie("keyway_session", signSession(s.cfg.SessionSecret, u.ID), sessionTTL))
	http.Redirect(w, r, s.cfg.FrontendURL+"/app", http.StatusFound)
}

// handleDevLogin creates/uses a local demo account without GitHub (dev only).
func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	u := User{ID: "dev:local", Login: "dev", Name: "Local Dev", CreatedAt: time.Now().UTC()}
	if err := s.store.UpsertUser(r.Context(), u); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, s.cookie("keyway_session", signSession(s.cfg.SessionSecret, u.ID), sessionTTL))
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, s.cookie("keyway_session", "", -time.Hour))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.user(r))
}

// requireUser is auth middleware: it authenticates either the browser session
// cookie or an `Authorization: Bearer <token>` credential (used by the CLI and
// GitHub Action), loads the user, and 401s otherwise.
func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value, present := bearerToken(r)
		if !present {
			if c, err := r.Cookie("keyway_session"); err == nil {
				value = c.Value
				present = true
			}
		}
		if !present {
			writeErr(w, http.StatusUnauthorized, "sign in required (session cookie or Bearer token)")
			return
		}
		uid, ok := verifyToken(s.cfg.SessionSecret, value)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "session expired or invalid token")
			return
		}
		u, err := s.store.GetUser(r.Context(), uid)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unknown user")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, u)))
	})
}

// bearerToken extracts a bearer credential from the Authorization header.
func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:]), true
	}
	return "", false
}

// handleMintToken issues a long-lived CI/CLI token for the current user. The token
// is returned once and never stored server-side (it's a signed, stateless
// credential), so the UI must surface it immediately.
func (s *Server) handleMintToken(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusCreated, map[string]any{
		"token":      mintCIToken(s.cfg.SessionSecret, s.user(r).ID),
		"expires_at": time.Now().Add(ciTokenTTL).UTC(),
	})
}

func (s *Server) user(r *http.Request) User { u, _ := r.Context().Value(userCtxKey).(User); return u }

// ─── projects ────────────────────────────────────────────────────────────

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	ps, err := s.store.ListProjects(r.Context(), s.user(r).ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ps == nil {
		ps = []Project{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": ps})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Source Source `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if body.Source.Kind == "" {
		body.Source.Kind = SourceUpload
	}
	if body.Source.Kind == SourceGitHub && body.Source.Repo == "" {
		writeErr(w, http.StatusBadRequest, "github source requires a repo (owner/name)")
		return
	}
	p := Project{
		ID: newID(), OwnerID: s.user(r).ID, Name: body.Name,
		Source: body.Source, CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateProject(r.Context(), p); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// project loads a project and enforces tenant ownership (404 if not the owner's).
func (s *Server) project(r *http.Request) (Project, error) {
	p, err := s.store.GetProject(r.Context(), chi.URLParam(r, "id"))
	if err != nil || p.OwnerID != s.user(r).ID {
		return Project{}, ErrNotFound
	}
	return p, nil
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.project(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}
	resp := map[string]any{"project": p}
	if a, err := s.store.LatestAnalysis(r.Context(), p.ID); err == nil {
		resp["latest"] = a
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.project(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}
	if err := s.store.DeleteProject(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAnalyze runs the engine over the project's config: either manifests posted
// in the body (upload) or fetched from the connected GitHub repo, diffs against
// the previous analysis, and stores the result.
func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	p, err := s.project(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		Manifests map[string]string `json:"manifests"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	manifests := body.Manifests
	trigger, ref := "upload", "upload"
	if len(manifests) == 0 && p.Source.Kind == SourceGitHub {
		m, sha, ferr := s.gh.FetchManifests(r.Context(), p.Source.Repo, p.Source.Ref, p.Source.Path, s.getToken(s.user(r).ID))
		if ferr != nil {
			writeErr(w, http.StatusBadRequest, ferr.Error())
			return
		}
		manifests, trigger, ref = m, "sync", short(sha)
	}
	if len(manifests) == 0 {
		writeErr(w, http.StatusBadRequest, "no manifests to analyze (upload files or connect a repo)")
		return
	}

	var prevPtr *model.ContractVersion
	if prev, perr := s.store.LatestAnalysis(r.Context(), p.ID); perr == nil {
		v := prev.Version
		prevPtr = &v
	}

	version, changes, aerr := Analyze(r.Context(), manifests, prevPtr)
	if aerr != nil {
		writeErr(w, http.StatusBadRequest, aerr.Error())
		return
	}
	if changes == nil {
		changes = []model.ChangeEvent{}
	}
	a := Analysis{
		ID: newID(), ProjectID: p.ID, CreatedAt: time.Now().UTC(),
		TriggerKind: trigger, TriggerRef: ref, Hash: version.Hash, IsBaseline: prevPtr == nil,
		Version: version, Changes: changes,
		ConsumerCount: len(version.Consumers), IssuerCount: len(version.Issuers), ChangeCount: len(changes),
	}
	if err := s.store.SaveAnalysis(r.Context(), a); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleListAnalyses(w http.ResponseWriter, r *http.Request) {
	p, err := s.project(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}
	list, err := s.store.ListAnalyses(r.Context(), p.ID, 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]AnalysisSummary, 0, len(list))
	for _, a := range list {
		out = append(out, a.Summary())
	}
	writeJSON(w, http.StatusOK, map[string]any{"analyses": out})
}

func (s *Server) handleGetAnalysis(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetAnalysis(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "analysis not found")
		return
	}
	// Tenant check via the owning project.
	if p, perr := s.store.GetProject(r.Context(), a.ProjectID); perr != nil || p.OwnerID != s.user(r).ID {
		writeErr(w, http.StatusNotFound, "analysis not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// ─── public tools (reuse the engine) ───────────────────────────────────────

func (s *Server) handleAgentInspect(w http.ResponseWriter, r *http.Request) {
	var req apitypes.AgentInspectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeErr(w, http.StatusBadRequest, "token is required")
		return
	}
	findings, err := agentauth.Analyze(req.Token, agentauth.Policy{
		Audience: req.Audience, RequireDelegation: req.RequireDelegation,
		MaxLifetime:   time.Duration(req.MaxLifetimeSeconds) * time.Second,
		AllowedScopes: req.AllowedScopes, Now: time.Now(),
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if findings == nil {
		findings = []agentauth.Finding{}
	}
	writeJSON(w, http.StatusOK, apitypes.AgentInspectResponse{Findings: findings, Count: len(findings)})
}

func (s *Server) handleThreatCoverage(w http.ResponseWriter, _ *http.Request) {
	cat := threats.Catalog()
	rep := threats.Compute(cat)
	out := apitypes.ThreatCoverageResponse{Total: rep.Total, Covered: rep.Covered, Gaps: len(rep.Gaps), Percent: rep.Pct()}
	for _, d := range rep.Domains {
		out.Domains = append(out.Domains, apitypes.CoverageDomain{Domain: string(d.Domain), Covered: d.Covered, Total: d.Total, Percent: d.Pct()})
	}
	for _, c := range rep.Categories {
		out.Categories = append(out.Categories, apitypes.CoverageCategory{Category: string(c.Category), Covered: c.Covered, Total: c.Total})
	}
	for _, t := range cat {
		ct := apitypes.CoverageThreat{ID: t.ID, Domain: string(t.Domain), Category: string(t.Category), Severity: t.Severity, Title: t.Title, Invariant: t.Invariant, Detectors: make([]string, 0, len(t.Detections))}
		for _, src := range t.Sources {
			ct.Sources = append(ct.Sources, apitypes.CoverageSource{Ref: src.Ref, URL: src.URL})
		}
		for _, d := range t.Detections {
			ct.Detectors = append(ct.Detectors, fmt.Sprintf("%s:%s", d.Kind, d.ID))
		}
		out.Threats = append(out.Threats, ct)
	}
	writeJSON(w, http.StatusOK, out)
}

// ─── helpers ───────────────────────────────────────────────────────────────

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.cfg.AllowedOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cookie(name, value string, ttl time.Duration) *http.Cookie {
	sameSite := http.SameSiteLaxMode
	if s.cfg.SecureCookies {
		sameSite = http.SameSiteNoneMode // cross-site frontend→backend needs None+Secure
	}
	// #nosec G124 -- HttpOnly is always set; Secure + SameSite=None are enabled in
	// production (https BASE_URL). Secure is intentionally relaxed only for local
	// http dev, where a Secure cookie would never be stored and login would break.
	return &http.Cookie{
		Name: name, Value: value, Path: "/",
		HttpOnly: true, Secure: s.cfg.SecureCookies, SameSite: sameSite,
		Expires: time.Now().Add(ttl), MaxAge: int(ttl.Seconds()),
	}
}

func (s *Server) setToken(userID, token string) {
	s.tokmu.Lock()
	defer s.tokmu.Unlock()
	s.tokens[userID] = token
}

func (s *Server) getToken(userID string) string {
	s.tokmu.Lock()
	defer s.tokmu.Unlock()
	return s.tokens[userID]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
