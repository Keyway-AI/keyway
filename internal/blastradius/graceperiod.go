package blastradius

import "time"

// Grace period bounds (PRD §10.3).
const (
	graceFloor   = 1 * time.Hour
	graceCeiling = 30 * 24 * time.Hour
	graceMargin  = 1.5
)

// GracePeriod applies the PRD §10.3 formula to a set of per-consumer refresh
// windows: RecommendedGracePeriod = max(window) * 1.5, floored at 1h and capped
// at 30d. Consumers with an unknown window must be excluded by the caller and
// surfaced under Unknown so the result can be flagged as a lower bound.
func GracePeriod(windows []time.Duration) time.Duration {
	var maxW time.Duration
	for _, w := range windows {
		if w > maxW {
			maxW = w
		}
	}
	rec := time.Duration(float64(maxW) * graceMargin)
	if rec < graceFloor {
		rec = graceFloor
	}
	if rec > graceCeiling {
		rec = graceCeiling
	}
	return rec
}
