package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nometria/keyway/internal/model"
)

// TestBuildDerivesGraph verifies issuers and edges are synthesized from the
// consumers' trusted issuers (KI-07), and that the hash is deterministic.
func TestBuildDerivesGraph(t *testing.T) {
	consumers := []model.Consumer{
		{ID: "c1", StableID: "k8s://c/ns/a", Name: "a", Expects: model.Expectations{Issuers: []string{"https://kc/realms/main"}, Audiences: []string{"a"}}},
		{ID: "c2", StableID: "k8s://c/ns/b", Name: "b", Expects: model.Expectations{Issuers: []string{"https://kc/realms/main", "https://other"}, Audiences: []string{"b"}}},
	}
	v1 := Build(BuildInput{Consumers: consumers})

	// Two distinct issuer URLs -> two synthesized issuers.
	require.Len(t, v1.Issuers, 2)
	// c1 trusts 1 issuer, c2 trusts 2 -> 3 edges.
	require.Len(t, v1.Edges, 3)

	// Every edge references a real issuer and consumer.
	issuerIDs := map[string]bool{}
	for _, is := range v1.Issuers {
		issuerIDs[is.ID] = true
	}
	for _, e := range v1.Edges {
		assert.True(t, issuerIDs[e.IssuerID], "edge issuer must exist")
	}

	// Deterministic hash despite fresh volatile issuer IDs.
	v2 := Build(BuildInput{Consumers: consumers})
	assert.Equal(t, v1.Hash, v2.Hash)
}

func TestBuildExplicitEdgesPreserved(t *testing.T) {
	in := BuildInput{
		Consumers: []model.Consumer{{ID: "c1", StableID: "s1", Expects: model.Expectations{Issuers: []string{"https://x"}}}},
		Edges:     []model.Edge{{IssuerID: "i1", ConsumerID: "c1"}},
	}
	v := Build(in)
	assert.Len(t, v.Edges, 1, "supplied edges are used as-is")
}
