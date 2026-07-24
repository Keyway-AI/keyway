package contract

import (
	"time"

	"github.com/google/uuid"
	"github.com/nometria/keyway/internal/model"
)

// BuildInput is the raw material for assembling a contract version.
type BuildInput struct {
	Issuers     []model.Issuer
	Consumers   []model.Consumer
	Edges       []model.Edge
	TriggerKind string // scheduled|deploy|commit|manual
	TriggerRef  string
	Now         time.Time
}

// Build assembles a ContractVersion from discovery output and computes its
// canonical hash. It assigns a fresh volatile ID and CreatedAt but leaves the
// baseline decision to Snapshot (version.go).
//
// When no explicit edges are supplied (the discovery path), Build derives the
// issuer→consumer graph: it synthesises a minimal Issuer record for every issuer
// URL that consumers trust but that no supplied Issuer covers, and one Edge per
// (issuer, consumer) trust relationship. The canonical hash keys edges by issuer
// URL + consumer StableID, so the derived graph hashes deterministically.
func Build(in BuildInput) model.ContractVersion {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	issuers, edges := deriveGraph(in.Issuers, in.Consumers, in.Edges)
	v := model.ContractVersion{
		ID:          uuid.NewString(),
		CreatedAt:   now,
		Issuers:     issuers,
		Consumers:   in.Consumers,
		Edges:       edges,
		TriggerKind: in.TriggerKind,
		TriggerRef:  in.TriggerRef,
	}
	v.Hash = Hash(v)
	return v
}

// deriveGraph returns the issuer set (supplied + synthesised) and the edges. If
// explicit edges are supplied they are used as-is.
func deriveGraph(supplied []model.Issuer, consumers []model.Consumer, edges []model.Edge) ([]model.Issuer, []model.Edge) {
	issuers := append([]model.Issuer{}, supplied...)
	byURL := map[string]string{} // issuer URL -> issuer ID
	for _, is := range issuers {
		byURL[is.IssuerURL] = is.ID
	}
	ensure := func(url string) string {
		if id, ok := byURL[url]; ok {
			return id
		}
		id := uuid.NewString()
		issuers = append(issuers, model.Issuer{ID: id, Name: url, Type: model.IssuerGenericOIDC, IssuerURL: url})
		byURL[url] = id
		return id
	}

	if len(edges) > 0 {
		return issuers, edges
	}
	for _, c := range consumers {
		for _, url := range c.Expects.Issuers {
			if url == "" {
				continue
			}
			edges = append(edges, model.Edge{
				IssuerID:    ensure(url),
				ConsumerID:  c.ID,
				Expects:     c.Expects,
				VerifyState: model.EdgeUnverified,
			})
		}
	}
	return issuers, edges
}
