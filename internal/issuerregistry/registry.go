// Package issuerregistry builds and holds live issuer adapters, the runtime home
// of Keyway-operated signing keys and canary state.
package issuerregistry

import (
	"fmt"
	"os"
	"sync"

	"github.com/Keyway-AI/keyway/internal/issuer"
	"github.com/Keyway-AI/keyway/internal/issuer/generic"
	"github.com/Keyway-AI/keyway/internal/issuer/k8ssa"
	"github.com/Keyway-AI/keyway/internal/issuer/keycloak"
	"github.com/Keyway-AI/keyway/internal/issuer/localkeys"
	"github.com/Keyway-AI/keyway/internal/keystore"
	"github.com/Keyway-AI/keyway/internal/model"
)

// CanaryIssuer is an issuer.Adapter that also exposes its locally-operated key
// set, so the API can drive the announce/promote/retire canary lifecycle.
type CanaryIssuer interface {
	issuer.Adapter
	KeySet() *localkeys.KeySet
}

// Spec describes an issuer to register.
type Spec struct {
	Name         string
	Type         model.IssuerType
	URL          string
	AdminCredEnv string
}

// Registry holds live issuer adapters by name. It is the runtime home of the
// Keyway-operated signing keys (canary state lives here, not in Postgres — the
// daemon owns it).
type Registry struct {
	mu     sync.RWMutex
	byName map[string]CanaryIssuer
}

// NewRegistry builds a registry from specs (no persistence: canary state lives
// only in memory and is lost on restart).
func NewRegistry(specs []Spec) (*Registry, error) {
	return NewRegistryWithStore(specs, nil)
}

// NewRegistryWithStore builds a registry and, when store is non-nil, makes each
// issuer's operated key set durable (KI-09): persisted keys are loaded on
// startup and every announce/promote/retire is written back through the store.
// Canary state therefore survives a `keyway serve` restart.
func NewRegistryWithStore(specs []Spec, store keystore.Store) (*Registry, error) {
	r := &Registry{byName: map[string]CanaryIssuer{}}
	for _, s := range specs {
		a, err := build(s)
		if err != nil {
			return nil, err
		}
		if store != nil {
			if err := bindStore(a, s.Name, store); err != nil {
				return nil, err
			}
		}
		r.byName[s.Name] = a
	}
	return r, nil
}

// bindStore loads any persisted keys for an issuer and installs a persist-on-
// change hook. Persistence failures on mutation are logged (best-effort) rather
// than failing the operation, since the in-memory state is still correct.
func bindStore(a CanaryIssuer, name string, store keystore.Store) error {
	ks := a.KeySet()
	persisted, err := store.Load(name)
	if err != nil {
		return fmt.Errorf("issuerregistry: load persisted keys for %q: %w", name, err)
	}
	if err := ks.Import(persisted); err != nil {
		return fmt.Errorf("issuerregistry: import persisted keys for %q: %w", name, err)
	}
	ks.SetOnChange(func(keys []localkeys.PersistedKey) {
		if err := store.Save(name, keys); err != nil {
			fmt.Fprintf(os.Stderr, "issuerregistry: persist keys for %q: %v\n", name, err)
		}
	})
	return nil
}

func build(s Spec) (CanaryIssuer, error) {
	switch s.Type {
	case model.IssuerKeycloak:
		return keycloak.New(keycloak.Options{Name: s.Name, RealmURL: s.URL, AdminCredEnv: s.AdminCredEnv})
	case model.IssuerK8sSA:
		return k8ssa.New(k8ssa.Options{Name: s.Name, IssuerURL: s.URL})
	case model.IssuerGenericOIDC, "":
		return generic.New(generic.Options{Name: s.Name, IssuerURL: s.URL})
	default:
		return nil, fmt.Errorf("issuer: unsupported type %q", s.Type)
	}
}

// Get returns the issuer registered under name.
func (r *Registry) Get(name string) (CanaryIssuer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.byName[name]
	return a, ok
}

// Names lists registered issuer names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byName))
	for n := range r.byName {
		out = append(out, n)
	}
	return out
}

// Len returns the number of registered issuers.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byName)
}
