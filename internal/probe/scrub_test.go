package probe

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestScrubTokens ensures reflected JWTs never survive into stored responses
// (defence-in-depth for PRD OPEN-4).
func TestScrubTokens(t *testing.T) {
	jwt := "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJrZXl3YXkifQ.c2lnbmF0dXJl"
	body := "invalid token: " + jwt + " rejected"
	got := scrubTokens(body)
	assert.NotContains(t, got, jwt)
	assert.Contains(t, got, "[REDACTED-JWT]")
	assert.Contains(t, got, "rejected", "surrounding text is preserved")
}

func TestScrubTokensLeavesNormalText(t *testing.T) {
	body := `{"error":"unauthorized","code":401}`
	assert.Equal(t, body, scrubTokens(body))
	assert.False(t, strings.Contains(scrubTokens(body), "REDACTED"))
}
