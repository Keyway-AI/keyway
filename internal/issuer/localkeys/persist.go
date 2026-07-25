package localkeys

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/nometria/keyway/internal/model"
)

// PersistedKey is the serializable form of a managed key, including its private
// material (PKCS#8 PEM). It exists so canary key state can survive a daemon
// restart (KI-09). Because it carries a private key, a PersistedKey must only
// ever be written through an encrypting KeyStore — never to a plaintext sink.
type PersistedKey struct {
	KID               string     `json:"kid"`
	Alg               string     `json:"alg"`
	Status            string     `json:"status"`
	PrivatePEM        string     `json:"private_pem"`
	FirstSeenInJWKS   time.Time  `json:"first_seen_in_jwks"`
	InSigningUseSince *time.Time `json:"in_signing_use_since,omitempty"`
	RetiredAt         *time.Time `json:"retired_at,omitempty"`
}

// SetOnChange registers a callback invoked (with the full key set) after every
// lifecycle mutation, so a KeyStore can persist the new state. Set it once at
// construction, before the key set is used concurrently.
func (ks *KeySet) SetOnChange(fn func([]PersistedKey)) { ks.onChange = fn }

// notify pushes the current state to the onChange sink, if one is set. It must be
// called with ks.mu unlocked (Export takes the lock).
func (ks *KeySet) notify() {
	if ks.onChange == nil {
		return
	}
	ks.onChange(ks.Export())
}

// Export returns a serializable snapshot of every managed key.
func (ks *KeySet) Export() []PersistedKey {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	out := make([]PersistedKey, 0, len(ks.keys))
	for _, k := range ks.keys {
		privPEM, err := privateKeyPEM(k.priv)
		if err != nil {
			continue
		}
		out = append(out, PersistedKey{
			KID:               k.meta.KID,
			Alg:               k.meta.Alg,
			Status:            string(k.meta.Status),
			PrivatePEM:        privPEM,
			FirstSeenInJWKS:   k.meta.FirstSeenInJWKS,
			InSigningUseSince: k.meta.InSigningUseSince,
			RetiredAt:         k.meta.RetiredAt,
		})
	}
	return out
}

// Import loads persisted keys into the set. Keys whose KID is already present are
// skipped, so Import is idempotent. It does not fire the onChange callback (a
// load is not a change to persist).
func (ks *KeySet) Import(keys []PersistedKey) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	for _, pk := range keys {
		if ks.byKIDLocked(pk.KID) != nil {
			continue
		}
		priv, err := parsePrivateKeyPEM(pk.PrivatePEM)
		if err != nil {
			return fmt.Errorf("localkeys: import kid %q: %w", pk.KID, err)
		}
		pubPEM, err := publicKeyPEM(&priv.PublicKey)
		if err != nil {
			return err
		}
		ks.keys = append(ks.keys, &managed{priv: priv, meta: model.Key{
			KID:               pk.KID,
			Alg:               pk.Alg,
			Use:               "sig",
			PublicKeyPEM:      pubPEM,
			Status:            model.KeyStatus(pk.Status),
			FirstSeenInJWKS:   pk.FirstSeenInJWKS,
			InSigningUseSince: pk.InSigningUseSince,
			RetiredAt:         pk.RetiredAt,
		}})
	}
	return nil
}

func privateKeyPEM(priv *rsa.PrivateKey) (string, error) {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", fmt.Errorf("localkeys: marshal private key: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}

func parsePrivateKeyPEM(s string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		return nil, fmt.Errorf("localkeys: invalid private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("localkeys: parse private key: %w", err)
	}
	rk, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("localkeys: persisted key is not RSA")
	}
	return rk, nil
}
