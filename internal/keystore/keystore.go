// Package keystore persists Keyway-operated signing keys so canary state
// survives a daemon restart (KI-09). Because the persisted material includes
// private keys, the file-backed store encrypts every record with AES-256-GCM
// under an operator-supplied key — private keys are never written in plaintext.
package keystore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nometria/keyway/internal/issuer/localkeys"
)

// Store loads and saves the persisted key set for a named issuer.
type Store interface {
	Load(issuer string) ([]localkeys.PersistedKey, error)
	Save(issuer string, keys []localkeys.PersistedKey) error
}

// FileStore persists each issuer's keys to an encrypted file in a directory.
type FileStore struct {
	dir string
	gcm cipher.AEAD
	mu  sync.Mutex // serializes Save so concurrent writes to one issuer can't lose a rename
}

// NewFileStore opens (creating if needed) a directory-backed store. encKey must
// be exactly 32 bytes (AES-256); use KeyFromEnv to source it from the
// environment. The directory is created 0700.
func NewFileStore(dir string, encKey []byte) (*FileStore, error) {
	if len(encKey) != 32 {
		return nil, fmt.Errorf("keystore: encryption key must be 32 bytes, got %d", len(encKey))
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("keystore: create dir: %w", err)
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("keystore: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("keystore: new gcm: %w", err)
	}
	return &FileStore{dir: dir, gcm: gcm}, nil
}

var _ Store = (*FileStore)(nil)

// Load returns the persisted keys for an issuer, or an empty slice if none exist.
func (s *FileStore) Load(issuer string) ([]localkeys.PersistedKey, error) {
	ct, err := os.ReadFile(s.path(issuer))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("keystore: read: %w", err)
	}
	ns := s.gcm.NonceSize()
	if len(ct) < ns {
		return nil, fmt.Errorf("keystore: ciphertext too short")
	}
	nonce, enc := ct[:ns], ct[ns:]
	pt, err := s.gcm.Open(nil, nonce, enc, nil)
	if err != nil {
		return nil, fmt.Errorf("keystore: decrypt (wrong key or corrupt file): %w", err)
	}
	var keys []localkeys.PersistedKey
	if err := json.Unmarshal(pt, &keys); err != nil {
		return nil, fmt.Errorf("keystore: unmarshal: %w", err)
	}
	return keys, nil
}

// Save encrypts and writes an issuer's keys, replacing any prior file. The write
// is atomic (temp file + rename) and the file is 0600.
func (s *FileStore) Save(issuer string, keys []localkeys.PersistedKey) error {
	pt, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("keystore: marshal: %w", err)
	}
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("keystore: nonce: %w", err)
	}
	ct := s.gcm.Seal(nonce, nonce, pt, nil)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Unique temp file (not a fixed ".tmp") + fsync + atomic rename, so two
	// concurrent Saves can never move each other's temp file out from under them.
	final := s.path(issuer)
	f, err := os.CreateTemp(s.dir, sanitize(issuer)+".*.tmp")
	if err != nil {
		return fmt.Errorf("keystore: create temp: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op after a successful rename
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return fmt.Errorf("keystore: chmod: %w", err)
	}
	if _, err := f.Write(ct); err != nil {
		_ = f.Close()
		return fmt.Errorf("keystore: write: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("keystore: sync: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("keystore: close: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		return fmt.Errorf("keystore: rename: %w", err)
	}
	return nil
}

func (s *FileStore) path(issuer string) string {
	return filepath.Join(s.dir, sanitize(issuer)+".enc")
}

// sanitize turns an issuer name into a safe filename component.
func sanitize(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "issuer"
	}
	return b.String()
}

// KeySource describes where the 32-byte AES root key comes from. The key
// protects all persisted private key material, so it should itself live in a
// secret manager rather than the config file. Exactly one of the fields is used,
// in this precedence: Command > File > Env. A command lets an operator bridge to
// any secret manager without Keyway bundling its SDK (e.g. `vault kv get -field
// key secret/keyway` or `aws secretsmanager get-secret-value ...`).
type KeySource struct {
	Env     string // environment variable name (e.g. KEYWAY_KEY_ENCRYPTION_KEY)
	File    string // path to a file (e.g. a mounted Kubernetes/Vault secret)
	Command string // shell command whose stdout is the key
}

// ResolveKey returns the 32-byte AES key from the highest-precedence configured
// source. It fails closed: if a source is configured but cannot yield a 32-byte
// key, that is an error rather than a silent fallback to a weaker source.
func ResolveKey(src KeySource) ([]byte, error) {
	switch {
	case src.Command != "":
		return KeyFromCommand(src.Command)
	case src.File != "":
		return KeyFromFile(src.File)
	case src.Env != "":
		return KeyFromEnv(src.Env)
	default:
		return nil, fmt.Errorf("keystore: no key source configured (set an env var, file, or command)")
	}
}

// KeyFromEnv reads a 32-byte AES key from an environment variable. The value may
// be base64 (std or url), hex, or 32 raw bytes. It returns an error if the
// variable is unset or does not decode to exactly 32 bytes, so key persistence
// fails closed rather than writing private keys under a weak/absent key.
func KeyFromEnv(envVar string) ([]byte, error) {
	v := strings.TrimSpace(os.Getenv(envVar))
	if v == "" {
		return nil, fmt.Errorf("keystore: %s is not set (a 32-byte key is required to persist signing keys)", envVar)
	}
	return decodeKey(v, envVar)
}

// KeyFromFile reads the key from a file (e.g. a secret mounted by Kubernetes or
// Vault Agent). Whitespace is trimmed; the contents may be base64, hex, or raw.
func KeyFromFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("keystore: read key file: %w", err)
	}
	return decodeKey(strings.TrimSpace(string(b)), path)
}

// KeyFromCommand runs an operator-configured shell command and uses its stdout as
// the key. This is the secret-manager bridge: the command typically shells out to
// a secret-manager CLI. The command is trusted operator configuration (never
// derived from request data); it is run via `sh -c`.
func KeyFromCommand(command string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", command).Output()
	if err != nil {
		return nil, fmt.Errorf("keystore: key command failed: %w", err)
	}
	return decodeKey(strings.TrimSpace(string(out)), "key command output")
}

// decodeKey turns a string (base64 std/url, hex, or 32 raw bytes) into a 32-byte
// key, or returns a descriptive error naming the source.
func decodeKey(v, source string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(v); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.RawURLEncoding.DecodeString(v); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := hex.DecodeString(v); err == nil && len(b) == 32 {
		return b, nil
	}
	if len(v) == 32 {
		return []byte(v), nil
	}
	return nil, fmt.Errorf("keystore: %s must decode to 32 bytes (base64, hex, or raw)", source)
}
