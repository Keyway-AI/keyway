package attribution

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nometria/keyway/internal/model"
)

// KeycloakAdminAttributor binds an IdP-side change (a key rotation, or an
// issuer/audience/claim change on an OIDC-registry consumer) to the Keycloak
// admin action that caused it, by querying the realm's admin-events log
// (PRD §16 OPEN-5). It is the fallback in the attribution Chain: file-based
// sources (git, deploy) are more specific and run first, so this attributor
// deliberately only claims changes that look IdP-side.
type KeycloakAdminAttributor struct {
	base   string // scheme://host
	realm  string
	user   string
	pass   string
	client *http.Client
	// window bounds how far before a change we look for a causing admin event.
	window time.Duration
	now    func() time.Time
}

// KeycloakOptions configures the admin-events attributor.
type KeycloakOptions struct {
	// RealmURL is the issuer URL, e.g. https://kc.example.com/realms/main.
	RealmURL      string
	AdminUser     string
	AdminPassword string
	HTTPClient    *http.Client
	// Window bounds how far before a detected change an admin event may sit and
	// still be considered its cause (default 24h).
	Window time.Duration
	Now    func() time.Time
}

// NewKeycloakAdmin constructs an admin-events attributor. It returns an error if
// the realm URL is not a Keycloak-style .../realms/{realm} URL.
func NewKeycloakAdmin(opts KeycloakOptions) (*KeycloakAdminAttributor, error) {
	base, realm, err := splitRealmURL(opts.RealmURL)
	if err != nil {
		return nil, err
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
			// Fail closed on redirects: admin creds and audit data must not be
			// followed to an attacker-chosen host (SSRF).
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	window := opts.Window
	if window == 0 {
		window = 24 * time.Hour
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &KeycloakAdminAttributor{
		base: base, realm: realm,
		user: opts.AdminUser, pass: opts.AdminPassword,
		client: client, window: window, now: now,
	}, nil
}

var _ Attributor = (*KeycloakAdminAttributor)(nil)

// adminEvent is the subset of a Keycloak admin-event record we use.
type adminEvent struct {
	Time          int64  `json:"time"` // epoch milliseconds
	OperationType string `json:"operationType"`
	ResourceType  string `json:"resourceType"`
	ResourcePath  string `json:"resourcePath"`
	AuthDetails   struct {
		UserID    string `json:"userId"`
		ClientID  string `json:"clientId"`
		IPAddress string `json:"ipAddress"`
	} `json:"authDetails"`
}

// Attribute returns the admin action that most likely caused the change. It only
// engages for IdP-side changes (see idpSide); otherwise it returns Unattributed
// so a more specific attributor in the chain can claim the change.
func (a *KeycloakAdminAttributor) Attribute(ctx context.Context, ev model.ChangeEvent) (*model.Attribution, error) {
	if !idpSide(ev) {
		return Unattributed(), nil
	}
	token, err := a.adminToken(ctx)
	if err != nil {
		return Unattributed(), nil
	}
	events, err := a.adminEvents(ctx, token)
	if err != nil || len(events) == 0 {
		return Unattributed(), nil
	}

	// The change was detected at ev.DetectedAt; the causing admin action is the
	// most recent event at or before it, within the window.
	ref := ev.DetectedAt
	if ref.IsZero() {
		ref = a.now()
	}
	var best *adminEvent
	var bestT time.Time
	for i := range events {
		et := time.UnixMilli(events[i].Time)
		if et.After(ref) {
			continue // happened after we detected the change; not the cause
		}
		if ref.Sub(et) > a.window {
			continue // too old to plausibly be the cause
		}
		if best == nil || et.After(bestT) {
			best = &events[i]
			bestT = et
		}
	}
	if best == nil {
		return Unattributed(), nil
	}

	actor := best.AuthDetails.UserID
	if actor == "" {
		actor = best.AuthDetails.ClientID
	}
	ref2 := strings.TrimSpace(best.OperationType + " " + best.ResourcePath)
	return &model.Attribution{
		Kind:       "idp_audit",
		Ref:        ref2,
		Actor:      actor,
		Timestamp:  bestT,
		Confidence: 0.7,
	}, nil
}

// idpSide reports whether a change originates on the identity provider (a key
// rotation, or an issuer/audience/claim change on an OIDC-registry consumer),
// as opposed to a mesh/manifest change that git/deploy attribution should own.
func idpSide(ev model.ChangeEvent) bool {
	if strings.HasPrefix(ev.Field, "jwks_behavior") {
		return true
	}
	for _, e := range ev.Evidence {
		if strings.HasPrefix(e, "oidcclient") || strings.Contains(e, "/realms/") {
			return true
		}
	}
	return false
}

// --- Keycloak admin API ------------------------------------------------------

func (a *KeycloakAdminAttributor) adminToken(ctx context.Context) (string, error) {
	form := url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"username":   {a.user},
		"password":   {a.pass},
	}
	tokenURL := a.base + "/realms/master/protocol/openid-connect/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.client.Do(req)
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

func (a *KeycloakAdminAttributor) adminEvents(ctx context.Context, token string) ([]adminEvent, error) {
	eventsURL := a.base + "/admin/realms/" + a.realm + "/admin-events?max=100"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eventsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("admin-events endpoint returned %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	var events []adminEvent
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// splitRealmURL parses https://host/realms/{realm} into base + realm, requiring
// https (admin credentials are sent to this host) except for localhost dev.
func splitRealmURL(realmURL string) (base, realm string, err error) {
	u, perr := url.Parse(realmURL)
	if perr != nil || u.Host == "" {
		return "", "", fmt.Errorf("attribution: invalid realm URL %q", realmURL)
	}
	if u.Scheme != "https" && !isLocalhost(u.Hostname()) {
		return "", "", fmt.Errorf("attribution: realm URL must be https (admin credentials are sent to it): %q", realmURL)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[len(parts)-2] != "realms" {
		return "", "", fmt.Errorf("attribution: not a Keycloak realm URL (want .../realms/{realm}): %q", realmURL)
	}
	realm = parts[len(parts)-1]
	base = u.Scheme + "://" + u.Host
	return base, realm, nil
}

func isLocalhost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
