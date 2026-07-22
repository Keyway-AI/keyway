package blastradius

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGracePeriod(t *testing.T) {
	cases := []struct {
		name    string
		windows []time.Duration
		want    time.Duration
	}{
		{"empty floors to 1h", nil, graceFloor},
		{"small floors to 1h", []time.Duration{10 * time.Minute}, graceFloor},
		{"max times 1.5", []time.Duration{48 * time.Hour, 2 * time.Hour}, time.Duration(float64(48*time.Hour) * 1.5)},
		{"caps at 30d", []time.Duration{60 * 24 * time.Hour}, graceCeiling},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, GracePeriod(tc.windows))
		})
	}
}
