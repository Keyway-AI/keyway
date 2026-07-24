// Package oidcclient discovers consumers from an OIDC provider's client registry
// (Keycloak). Each registered client that validates tokens — identified by its
// audience/protocol mappers — becomes a Consumer. Confidence 0.9: declarative,
// but the registry lists intent rather than a running, probeable endpoint.
package oidcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/model"
)

// Options configures the Keycloak client-registry discoverer.
type Options struct {
	// RealmURL is the issuer URL, e.g. https://kc.example.com/realms/main.
	RealmURL string
	// AdminUser / AdminPassword authenticate to the admin API (admin-cli grant).
	AdminUser     string
	AdminPassword string
	HTTPClient    *http.Client
	Now           func() time.Time
}

// Discoverer reads registered clients from a Keycloak realm.
type Discoverer struct {
	base   string // scheme://host
	realm  string
	user   string
	pass   string
	client *http.Client
	now    func() time.Time
}

// New constructs an OIDC client-registry discoverer. It returns an error if the
// realm URL is not a Keycloak-style .../realms/{realm} URL.
func New(opts Options) (*Discoverer, error) {
	base, realm, err := splitRealmURL(opts.RealmURL)
	if err != nil {
		return nil, err
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Discoverer{
		base: base, realm: realm,
		user: opts.AdminUser, pass: opts.AdminPassword,
		client: client, now: now,
	}, nil
}

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "oidcclient" }

// Discover authenticates to the admin API, lists clients, and maps the ones that
// validate tokens to consumers.
func (d *Discoverer) Discover(ctx context.Context, _ discovery.Scope) ([]model.Consumer, error) {
	token, err := d.adminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("oidcclient: admin auth: %w", err)
	}
	clients, err := d.listClients(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("oidcclient: list clients: %w", err)
	}
	issuerURL := d.base + "/realms/" + d.realm
	var out []model.Consumer
	for _, c := range clients {
		if !validatesTokens(c) {
			continue
		}
		out = append(out, d.toConsumer(c, issuerURL))
	}
	return out, nil
}

// --- Keycloak admin API ------------------------------------------------------

type kcClient struct {
	ID              string `json:"id"`
	ClientID        string `json:"clientId"`
	Protocol        string `json:"protocol"`
	BearerOnly      bool   `json:"bearerOnly"`
	PublicClient    bool   `json:"publicClient"`
	Enabled         bool   `json:"enabled"`
	ProtocolMappers []struct {
		ProtocolMapper string            `json:"protocolMapper"`
		Config         map[string]string `json:"config"`
	} `json:"protocolMappers"`
}

func (d *Discoverer) adminToken(ctx context.Context) (string, error) {
	form := url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"username":   {d.user},
		"password":   {d.pass},
	}
	tokenURL := d.base + "/realms/master/protocol/openid-connect/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}
	return tok.AccessToken, nil
}

func (d *Discoverer) listClients(ctx context.Context, token string) ([]kcClient, error) {
	clientsURL := d.base + "/admin/realms/" + d.realm + "/clients"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clients endpoint returned %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	var clients []kcClient
	if err := json.Unmarshal(body, &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

// --- mapping -----------------------------------------------------------------

// validatesTokens reports whether a client is one that verifies incoming tokens.
// Bearer-only and confidential (non-public) OIDC clients validate; public SPA
// clients typically only obtain tokens, so they are marked but not probeable.
func validatesTokens(c kcClient) bool {
	if c.Protocol != "" && c.Protocol != "openid-connect" {
		return false
	}
	if !c.Enabled {
		return false
	}
	// A well-known internal client (admin-cli, account, security-admin-console,
	// broker, realm-management) is Keycloak plumbing, not a customer consumer.
	switch c.ClientID {
	case "admin-cli", "account", "account-console", "security-admin-console", "broker", "realm-management":
		return false
	}
	return true
}

func (d *Discoverer) toConsumer(c kcClient, issuerURL string) model.Consumer {
	stableID := "oidc://" + d.realm + "/" + c.ClientID

	audiences := audienceMappers(c)
	// The clientId is itself the conventional audience for a confidential client.
	if !c.PublicClient {
		audiences = appendUnique(audiences, c.ClientID)
	}

	kind := model.ConsumerService
	probeable := true
	if c.PublicClient {
		// SPA/mobile-style clients obtain tokens but are not a probeable service.
		kind = model.ConsumerClient
		probeable = false
	}

	source := "oidcclient:" + d.realm + "/" + c.ClientID
	prov := model.ProvenanceRecord{Source: source, Locator: issuerURL, ObservedAt: d.now(), Confidence: 0.9}

	return model.Consumer{
		ID:        stableID,
		StableID:  stableID,
		Kind:      kind,
		Name:      c.ClientID,
		Namespace: d.realm,
		Expects: model.Expectations{
			Issuers:   []string{issuerURL},
			Audiences: audiences,
		},
		JWKSBehavior: model.JWKSBehavior{Source: model.SrcConfig},
		Provenance: map[string][]model.ProvenanceRecord{
			"expects.issuers":   {prov},
			"expects.audiences": {prov},
		},
		Confidence: map[string]float64{"overall": 0.9},
		Probeable:  probeable,
	}
}

// audienceMappers extracts audiences declared by the client's audience mappers.
func audienceMappers(c kcClient) []string {
	var out []string
	for _, m := range c.ProtocolMappers {
		if m.ProtocolMapper != "oidc-audience-mapper" {
			continue
		}
		if a := m.Config["included.client.audience"]; a != "" {
			out = appendUnique(out, a)
		}
		if a := m.Config["included.custom.audience"]; a != "" {
			out = appendUnique(out, a)
		}
	}
	return out
}

// splitRealmURL parses https://host/realms/{realm} into base + realm.
func splitRealmURL(realmURL string) (base, realm string, err error) {
	u, perr := url.Parse(realmURL)
	if perr != nil || u.Host == "" {
		return "", "", fmt.Errorf("oidcclient: invalid realm URL %q", realmURL)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[len(parts)-2] != "realms" {
		return "", "", fmt.Errorf("oidcclient: not a Keycloak realm URL (want .../realms/{realm}): %q", realmURL)
	}
	realm = parts[len(parts)-1]
	base = u.Scheme + "://" + u.Host
	return base, realm, nil
}

func appendUnique(ss []string, v string) []string {
	for _, s := range ss {
		if s == v {
			return ss
		}
	}
	return append(ss, v)
}
