package realworld

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealWorldCasesDetected runs every documented incident reproduction and
// asserts Keyway flags the risk. This is the CI guarantee that the tool still
// catches the real-world classes it claims to.
func TestRealWorldCasesDetected(t *testing.T) {
	cases := Cases()
	require.NotEmpty(t, cases)

	for _, c := range cases {
		t.Run(c.ID+" "+c.Title, func(t *testing.T) {
			res := c.Run()
			assert.Truef(t, res.Detected,
				"Keyway must detect %s (%s): %s", c.Reference, c.Mechanism, res.Verdict)
			t.Logf("%s [%s] → %s", c.ID, c.Reference, res.Verdict)
		})
	}
}

// TestEveryCaseHasCitation guards against un-sourced cases sneaking in.
func TestEveryCaseHasCitation(t *testing.T) {
	for _, c := range Cases() {
		assert.NotEmpty(t, c.Reference, "%s needs a reference", c.ID)
		assert.Contains(t, c.Source, "http", "%s needs a source URL", c.ID)
	}
}
