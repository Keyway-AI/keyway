package cloud

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Sessions are stateless: a signed token "b64(userID).exp|sig" carries the
// authenticated user id, HMAC-signed with the server secret. The same signed form
// backs both the browser session cookie (short TTL) and long-lived CI/CLI tokens
// (prefixed "kwci_", year TTL) — so no server-side session store is needed.
//
// Trade-off: statelessness means an individual CI token can't be revoked without
// rotating KEYWAY_CLOUD_SESSION_SECRET (which invalidates all tokens at once).

const (
	sessionTTL = 30 * 24 * time.Hour
	// ciTokenTTL is the lifetime of a CLI/CI token minted via POST /v1/tokens.
	ciTokenTTL = 365 * 24 * time.Hour
	// ciTokenPrefix marks a token as a CI/CLI credential (cosmetic; the middleware
	// accepts the raw signed value too).
	ciTokenPrefix = "kwci_"
)

func signToken(secret []byte, userID string, ttl time.Duration) string {
	uid := base64.RawURLEncoding.EncodeToString([]byte(userID))
	payload := uid + "." + strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
	return payload + "|" + sign(secret, payload)
}

func signSession(secret []byte, userID string) string {
	return signToken(secret, userID, sessionTTL)
}

// mintCIToken produces a long-lived bearer token for the CLI and GitHub Action.
func mintCIToken(secret []byte, userID string) string {
	return ciTokenPrefix + signToken(secret, userID, ciTokenTTL)
}

// verifyToken validates a session cookie value or a bearer token (with or without
// the CI prefix), returning the authenticated user id.
func verifyToken(secret []byte, value string) (string, bool) {
	value = strings.TrimPrefix(value, ciTokenPrefix)
	payload, sig, ok := strings.Cut(value, "|")
	if !ok {
		return "", false
	}
	if !hmac.Equal([]byte(sign(secret, payload)), []byte(sig)) {
		return "", false
	}
	uidB64, expStr, ok := strings.Cut(payload, ".")
	if !ok {
		return "", false
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", false
	}
	uid, err := base64.RawURLEncoding.DecodeString(uidB64)
	if err != nil {
		return "", false
	}
	return string(uid), true
}

func sign(secret []byte, msg string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func userID(provider string, id int64) string { return fmt.Sprintf("%s:%d", provider, id) }
