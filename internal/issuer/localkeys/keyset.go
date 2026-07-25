// Package localkeys manages a set of locally-controlled signing keys and the
// JOSE operations Keyway performs with them: minting tokens, publishing a JWKS,
// and running the announce → active → retired key lifecycle used by the canary
// flow (PRD §4.1, §6, §10). It is the engine behind every issuer type where
// Keyway controls the private key.
package localkeys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	"github.com/nometria/keyway/internal/model"
)

// rsaBits is the modulus size for generated keys. 2048 is the floor for RS256.
const rsaBits = 2048

type managed struct {
	priv *rsa.PrivateKey
	meta model.Key
}

// KeySet is a concurrency-safe collection of managed signing keys.
type KeySet struct {
	mu        sync.Mutex
	keys      []*managed
	onChange  func([]PersistedKey) // fired after lifecycle mutations (persistence hook)
	persistMu sync.Mutex           // serializes Export+onChange so persists never race or reorder
}

// NewKeySet returns an empty key set.
func NewKeySet() *KeySet { return &KeySet{} }

// Generate creates a new RSA key in the given status and returns its public
// metadata. alg must be an RSA algorithm (RS256/RS384/RS512); it defaults to
// RS256 when empty.
func (ks *KeySet) Generate(alg string, status model.KeyStatus, now time.Time) (model.Key, error) {
	if alg == "" {
		alg = "RS256"
	}
	if alg != "RS256" && alg != "RS384" && alg != "RS512" {
		return model.Key{}, fmt.Errorf("localkeys: unsupported alg %q", alg)
	}
	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return model.Key{}, fmt.Errorf("localkeys: generate key: %w", err)
	}
	pemStr, err := publicKeyPEM(&priv.PublicKey)
	if err != nil {
		return model.Key{}, err
	}
	meta := model.Key{
		KID:             uuid.NewString(),
		Alg:             alg,
		Use:             "sig",
		PublicKeyPEM:    pemStr,
		Status:          status,
		FirstSeenInJWKS: now,
	}
	if status == model.KeyActive {
		t := now
		meta.InSigningUseSince = &t
	}
	ks.mu.Lock()
	ks.keys = append(ks.keys, &managed{priv: priv, meta: meta})
	ks.mu.Unlock()
	ks.notify()
	return meta, nil
}

// Sign produces a compact JWS (JWT) over claims, signed by the named key. When
// kid is empty the active key is used. Retired keys can still be used to sign —
// this is deliberate, since probe 10 needs to present a retired-key token.
func (ks *KeySet) Sign(kid string, claims map[string]any) (string, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	var target *managed
	if kid == "" {
		target = ks.activeLocked()
	} else {
		target = ks.byKIDLocked(kid)
	}
	if target == nil {
		return "", fmt.Errorf("localkeys: no key for kid %q", kid)
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.SignatureAlgorithm(target.meta.Alg), Key: target.priv},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", target.meta.KID),
	)
	if err != nil {
		return "", fmt.Errorf("localkeys: new signer: %w", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("localkeys: marshal claims: %w", err)
	}
	obj, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("localkeys: sign: %w", err)
	}
	return obj.CompactSerialize()
}

// Promote moves a key to active. Any previously-active key becomes retiring, so
// only one key signs at a time.
func (ks *KeySet) Promote(kid string, now time.Time) error {
	ks.mu.Lock()
	target := ks.byKIDLocked(kid)
	if target == nil {
		ks.mu.Unlock()
		return fmt.Errorf("localkeys: no key for kid %q", kid)
	}
	for _, k := range ks.keys {
		if k.meta.Status == model.KeyActive {
			k.meta.Status = model.KeyRetiring
		}
	}
	target.meta.Status = model.KeyActive
	t := now
	target.meta.InSigningUseSince = &t
	ks.mu.Unlock()
	ks.notify()
	return nil
}

// Retire marks a key retired.
func (ks *KeySet) Retire(kid string, now time.Time) error {
	ks.mu.Lock()
	target := ks.byKIDLocked(kid)
	if target == nil {
		ks.mu.Unlock()
		return fmt.Errorf("localkeys: no key for kid %q", kid)
	}
	target.meta.Status = model.KeyRetired
	t := now
	target.meta.RetiredAt = &t
	ks.mu.Unlock()
	ks.notify()
	return nil
}

// Keys returns a copy of all key metadata.
func (ks *KeySet) Keys() []model.Key {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	out := make([]model.Key, 0, len(ks.keys))
	for _, k := range ks.keys {
		out = append(out, k.meta)
	}
	return out
}

// ActiveKID returns the kid of the active key, or "".
func (ks *KeySet) ActiveKID() string { return ks.kidByStatus(model.KeyActive) }

// AnnouncedKID returns the kid of the announced (canary) key, or "".
func (ks *KeySet) AnnouncedKID() string { return ks.kidByStatus(model.KeyAnnounced) }

// RetiredKID returns the kid of a retired key, or "".
func (ks *KeySet) RetiredKID() string { return ks.kidByStatus(model.KeyRetired) }

// PublicKeyPEM returns the PEM-encoded public key for a kid (probe 7 needs the
// exact PEM bytes as an HMAC secret).
func (ks *KeySet) PublicKeyPEM(kid string) (string, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	k := ks.byKIDLocked(kid)
	if k == nil {
		return "", fmt.Errorf("localkeys: no key for kid %q", kid)
	}
	return k.meta.PublicKeyPEM, nil
}

// JWKS returns the public JSON Web Key Set for all non-retired keys, matching
// what an issuer would publish.
func (ks *KeySet) JWKS() jose.JSONWebKeySet {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	var set jose.JSONWebKeySet
	for _, k := range ks.keys {
		if k.meta.Status == model.KeyRetired {
			continue
		}
		set.Keys = append(set.Keys, jose.JSONWebKey{
			Key:       &k.priv.PublicKey,
			KeyID:     k.meta.KID,
			Algorithm: k.meta.Alg,
			Use:       "sig",
		})
	}
	return set
}

// --- locked helpers ---------------------------------------------------------

func (ks *KeySet) activeLocked() *managed {
	for _, k := range ks.keys {
		if k.meta.Status == model.KeyActive {
			return k
		}
	}
	return nil
}

func (ks *KeySet) byKIDLocked(kid string) *managed {
	for _, k := range ks.keys {
		if k.meta.KID == kid {
			return k
		}
	}
	return nil
}

func (ks *KeySet) kidByStatus(s model.KeyStatus) string {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	for _, k := range ks.keys {
		if k.meta.Status == s {
			return k.meta.KID
		}
	}
	return ""
}

func publicKeyPEM(pub *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("localkeys: marshal public key: %w", err)
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}
