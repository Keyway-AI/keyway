package contract

import (
	"testing"
	"time"

	"github.com/nometria/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleVersion builds a small but representative contract graph.
func sampleVersion() model.ContractVersion {
	ttl := 300
	refresh := false
	return model.ContractVersion{
		ID:        "vol-uuid-1",
		Hash:      "will-be-recomputed",
		CreatedAt: time.Now(),
		Issuers: []model.Issuer{{
			ID:                 "iss-uuid-1",
			Name:               "keycloak-main",
			Type:               model.IssuerKeycloak,
			IssuerURL:          "https://kc/realms/main",
			JWKSURI:            "https://kc/realms/main/protocol/openid-connect/certs",
			ControlsPrivateKey: true,
			Keys: []model.Key{{
				KID: "rsa-2026-01", Alg: "RS256", Use: "sig",
				Status: model.KeyActive, FirstSeenInJWKS: time.Now(),
			}},
			ClaimSchema: []model.ClaimObs{{Name: "sub", PresenceRate: 1.0, FirstSeen: time.Now()}},
		}},
		Consumers: []model.Consumer{{
			ID:       "con-uuid-1",
			StableID: "k8s://prod/default/api-a",
			Kind:     model.ConsumerService,
			Name:     "api-a",
			Expects: model.Expectations{
				Issuers:    []string{"https://kc/realms/main"},
				Audiences:  []string{"api-a", "api-shared"},
				Algorithms: []string{"RS256"},
			},
			JWKSBehavior: model.JWKSBehavior{
				CacheTTLSec: &ttl, RefreshesOnUnknownKID: &refresh, Source: model.SrcConfig,
			},
			Probeable: true,
		}},
		Edges: []model.Edge{{
			IssuerID:   "iss-uuid-1",
			ConsumerID: "con-uuid-1",
			Expects:    model.Expectations{Audiences: []string{"api-a"}},
		}},
	}
}

// TestHashDeterministic is the first acceptance test (AC-1): two runs against an
// unchanged system must produce an identical hash.
func TestHashDeterministic(t *testing.T) {
	v1 := sampleVersion()
	v2 := sampleVersion()
	assert.Equal(t, Hash(v1), Hash(v2), "identical graphs must hash identically")
}

// TestHashIgnoresVolatileFields verifies that timestamps, IDs, and the stored
// hash field do not affect the canonical hash.
func TestHashIgnoresVolatileFields(t *testing.T) {
	base := sampleVersion()
	mut := sampleVersion()
	mut.ID = "different-uuid"
	mut.Hash = "stale"
	mut.CreatedAt = base.CreatedAt.Add(48 * time.Hour)
	mut.Issuers[0].ID = "other-iss-uuid"
	mut.Issuers[0].Keys[0].FirstSeenInJWKS = time.Now().Add(time.Hour)
	mut.Consumers[0].ID = "other-con-uuid"
	// Edges reference the mutated IDs; canonicalisation must resolve them to
	// stable keys so the hash is unaffected.
	mut.Edges[0].IssuerID = "other-iss-uuid"
	mut.Edges[0].ConsumerID = "other-con-uuid"
	mut.Edges[0].VerifyState = model.EdgeVerified
	now := time.Now()
	mut.Edges[0].LastVerified = &now

	assert.Equal(t, Hash(base), Hash(mut), "volatile fields must not change the hash")
}

// TestHashOrderIndependence verifies element order does not affect the hash.
func TestHashOrderIndependence(t *testing.T) {
	base := sampleVersion()
	// Reverse the audience slice order and add a second consumer, then reorder.
	reordered := sampleVersion()
	reordered.Consumers[0].Expects.Audiences = []string{"api-shared", "api-a"}
	assert.Equal(t, Hash(base), Hash(reordered), "inner slice order must not matter")
}

// TestHashDetectsRealChange verifies a genuine contract change alters the hash.
func TestHashDetectsRealChange(t *testing.T) {
	base := sampleVersion()
	changed := sampleVersion()
	changed.Consumers[0].Expects.Audiences = append(changed.Consumers[0].Expects.Audiences, "api-new")
	assert.NotEqual(t, Hash(base), Hash(changed), "adding an audience must change the hash")
}

func TestHashHexLength(t *testing.T) {
	h := Hash(sampleVersion())
	require.Len(t, h, 64, "sha256 hex is 64 chars")
}
