// Package oidc provides OIDC discovery and JWKS fetching over HTTP, shared by
// issuer adapters that read an issuer's published metadata.
package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/Keyway-AI/keyway/internal/model"
)

// DiscoveryDoc is the subset of the OIDC discovery document Keyway needs.
type DiscoveryDoc struct {
	Issuer                string   `json:"issuer"`
	JWKSURI               string   `json:"jwks_uri"`
	AuthorizationEndpoint string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string   `json:"token_endpoint,omitempty"`
	IDTokenSigningAlgs    []string `json:"id_token_signing_alg_values_supported,omitempty"`

	Raw json.RawMessage `json:"-"`
}

// DefaultClient is a timeout-bounded HTTP client for discovery. It does NOT
// follow redirects: a compromised/misconfigured issuer must not be able to
// redirect a discovery or JWKS fetch to an internal address (SSRF). Legitimate
// OIDC providers serve these documents directly.
func DefaultClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// Discover fetches issuerURL/.well-known/openid-configuration.
func Discover(ctx context.Context, issuerURL string, client *http.Client) (DiscoveryDoc, error) {
	if client == nil {
		client = DefaultClient()
	}
	url := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
	body, err := get(ctx, client, url)
	if err != nil {
		return DiscoveryDoc{}, fmt.Errorf("oidc: discover %s: %w", url, err)
	}
	var doc DiscoveryDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return DiscoveryDoc{}, fmt.Errorf("oidc: parse discovery: %w", err)
	}
	doc.Raw = body
	return doc, nil
}

// FetchJWKS fetches and parses a JWKS document.
func FetchJWKS(ctx context.Context, jwksURI string, client *http.Client) (jose.JSONWebKeySet, error) {
	if client == nil {
		client = DefaultClient()
	}
	body, err := get(ctx, client, jwksURI)
	if err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("oidc: fetch jwks %s: %w", jwksURI, err)
	}
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(body, &set); err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("oidc: parse jwks: %w", err)
	}
	return set, nil
}

// Keys converts a parsed JWKS into Keyway key metadata. Fetched keys are marked
// active by default (we cannot know their lifecycle state from JWKS alone; the
// issuer adapter refines status where it has more information).
func Keys(set jose.JSONWebKeySet, now time.Time) []model.Key {
	out := make([]model.Key, 0, len(set.Keys))
	for _, k := range set.Keys {
		out = append(out, model.Key{
			KID:             k.KeyID,
			Alg:             k.Algorithm,
			Use:             k.Use,
			Status:          model.KeyActive,
			FirstSeenInJWKS: now,
		})
	}
	return out
}

func get(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}
