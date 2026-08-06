package cloud

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
)

// Config is the cloud server's runtime configuration, sourced from the
// environment so it deploys anywhere without code changes.
type Config struct {
	Addr           string   // listen address, default :8090
	BaseURL        string   // public base URL, used to build the OAuth redirect
	FrontendURL    string   // where to send the user after login (the web app)
	SessionSecret  []byte   // HMAC key for session cookies
	GitHubClientID string   // GitHub OAuth app client id
	GitHubSecret   string   // GitHub OAuth app client secret
	AllowedOrigins []string // CORS allowlist (the frontend origins)
	DevLogin       bool     // enable a passwordless dev login (local only)
	SecureCookies  bool     // set the Secure flag on cookies (true in production/HTTPS)
}

// ConfigFromEnv builds a Config from environment variables:
//
//	KEYWAY_CLOUD_ADDR              (:8090)
//	KEYWAY_CLOUD_BASE_URL          (http://localhost:8090)
//	KEYWAY_CLOUD_FRONTEND_URL      (http://localhost:5173)
//	KEYWAY_CLOUD_SESSION_SECRET    (random if unset — sessions won't survive restart)
//	KEYWAY_CLOUD_ALLOWED_ORIGINS   (comma-separated; defaults to FRONTEND_URL)
//	GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET
//	KEYWAY_CLOUD_DEV_LOGIN=1       (enable local passwordless login)
func ConfigFromEnv() Config {
	c := Config{
		Addr:           envOr("KEYWAY_CLOUD_ADDR", ":8090"),
		BaseURL:        strings.TrimRight(envOr("KEYWAY_CLOUD_BASE_URL", "http://localhost:8090"), "/"),
		FrontendURL:    strings.TrimRight(envOr("KEYWAY_CLOUD_FRONTEND_URL", "http://localhost:5173"), "/"),
		GitHubClientID: os.Getenv("GITHUB_CLIENT_ID"),
		GitHubSecret:   os.Getenv("GITHUB_CLIENT_SECRET"),
		DevLogin:       os.Getenv("KEYWAY_CLOUD_DEV_LOGIN") == "1",
		SecureCookies:  strings.HasPrefix(envOr("KEYWAY_CLOUD_BASE_URL", ""), "https://"),
	}
	if s := os.Getenv("KEYWAY_CLOUD_SESSION_SECRET"); s != "" {
		c.SessionSecret = []byte(s)
	} else {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		c.SessionSecret = []byte(hex.EncodeToString(b))
	}
	if o := os.Getenv("KEYWAY_CLOUD_ALLOWED_ORIGINS"); o != "" {
		for _, part := range strings.Split(o, ",") {
			if p := strings.TrimSpace(part); p != "" {
				c.AllowedOrigins = append(c.AllowedOrigins, p)
			}
		}
	} else {
		c.AllowedOrigins = []string{c.FrontendURL}
	}
	return c
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
