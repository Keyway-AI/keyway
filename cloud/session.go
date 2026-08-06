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

// Sessions are stateless: a signed cookie "b64(userID).exp|sig" carries the
// authenticated user id, HMAC-signed with the server secret. No server-side
// session store is needed.

const sessionTTL = 30 * 24 * time.Hour

func signSession(secret []byte, userID string) string {
	uid := base64.RawURLEncoding.EncodeToString([]byte(userID))
	payload := uid + "." + strconv.FormatInt(time.Now().Add(sessionTTL).Unix(), 10)
	return payload + "|" + sign(secret, payload)
}

func verifySession(secret []byte, value string) (string, bool) {
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
