package keystore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nometria/keyway/internal/issuer/localkeys"
)

// PostgresStore persists each issuer's key set as an AES-256-GCM ciphertext row
// in keyway_operated_keys. Unlike FileStore, the material is shared across every
// replica pointed at the same database, so canary state is both durable across
// restarts and consistent across a multi-replica deployment. Private keys are
// never written in plaintext; the root key comes from a secret manager
// (see KeySource / ResolveKey).
type PostgresStore struct {
	pool *pgxpool.Pool
	gcm  cipher.AEAD
}

// NewPostgresStore builds a Postgres-backed key store over an existing pool.
// encKey must be exactly 32 bytes (AES-256); use ResolveKey to source it from a
// secret manager.
func NewPostgresStore(pool *pgxpool.Pool, encKey []byte) (*PostgresStore, error) {
	if len(encKey) != 32 {
		return nil, fmt.Errorf("keystore: encryption key must be 32 bytes, got %d", len(encKey))
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("keystore: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("keystore: new gcm: %w", err)
	}
	return &PostgresStore{pool: pool, gcm: gcm}, nil
}

var _ Store = (*PostgresStore)(nil)

// Load returns the persisted keys for an issuer, or nil if none exist.
func (s *PostgresStore) Load(issuer string) ([]localkeys.PersistedKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var ct []byte
	err := s.pool.QueryRow(ctx, `SELECT data FROM keyway_operated_keys WHERE issuer = $1`, issuer).Scan(&ct)
	if err != nil {
		if pgxNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("keystore: load: %w", err)
	}
	pt, err := s.decrypt(ct)
	if err != nil {
		return nil, err
	}
	var keys []localkeys.PersistedKey
	if err := json.Unmarshal(pt, &keys); err != nil {
		return nil, fmt.Errorf("keystore: unmarshal: %w", err)
	}
	return keys, nil
}

// Save encrypts and upserts an issuer's keys.
func (s *PostgresStore) Save(issuer string, keys []localkeys.PersistedKey) error {
	pt, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("keystore: marshal: %w", err)
	}
	ct, err := s.encrypt(pt)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = s.pool.Exec(ctx,
		`INSERT INTO keyway_operated_keys (issuer, data, updated_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (issuer) DO UPDATE SET data = EXCLUDED.data, updated_at = now()`,
		issuer, ct,
	)
	if err != nil {
		return fmt.Errorf("keystore: save: %w", err)
	}
	return nil
}

func (s *PostgresStore) encrypt(pt []byte) ([]byte, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("keystore: nonce: %w", err)
	}
	return s.gcm.Seal(nonce, nonce, pt, nil), nil
}

func (s *PostgresStore) decrypt(ct []byte) ([]byte, error) {
	ns := s.gcm.NonceSize()
	if len(ct) < ns {
		return nil, fmt.Errorf("keystore: ciphertext too short")
	}
	pt, err := s.gcm.Open(nil, ct[:ns], ct[ns:], nil)
	if err != nil {
		return nil, fmt.Errorf("keystore: decrypt (wrong key or corrupt row): %w", err)
	}
	return pt, nil
}

func pgxNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
