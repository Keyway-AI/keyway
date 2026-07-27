// Package diff compares two contract versions and classifies each field change.
package diff

import "github.com/nometria/keyway/internal/model"

// FieldOp is the atomic operation the diff walker detected on a field.
type FieldOp string

const (
	OpAdd         FieldOp = "add"          // value added to a set-valued field
	OpRemove      FieldOp = "remove"       // value removed from a set-valued field
	OpIncrease    FieldOp = "increase"     // numeric field went up
	OpDecrease    FieldOp = "decrease"     // numeric field went down
	OpTrueToFalse FieldOp = "true_false"   // boolean flipped to false
	OpFalseToTrue FieldOp = "false_true"   // boolean flipped to true
	OpConsumerAdd FieldOp = "consumer_add" // whole consumer appeared
	OpConsumerDel FieldOp = "consumer_del" // whole consumer disappeared
)

// Field path constants matching PRD §9.2.
const (
	FieldAudiences      = "expects.audiences"
	FieldIssuers        = "expects.issuers"
	FieldAlgorithms     = "expects.algorithms"
	FieldRequiredClaims = "expects.required_claims"
	FieldClockSkew      = "expects.clock_skew_sec"
	FieldRefreshUnknown = "jwks_behavior.refreshes_on_unknown_kid"
	FieldCacheTTL       = "jwks_behavior.cache_ttl_sec"
	FieldConsumer       = "consumer"
)

// confidenceFloor: below this, any change is downgraded to unknown/info and
// never pages (PRD §9.2 final row).
const confidenceFloor = 0.6

// Classification is the result of classifying one field change.
type Classification struct {
	Class    model.ChangeClass
	Severity model.Severity
}

// Classify applies the exact classification and severity tables from PRD
// §9.2/§9.3 to a single atomic field change. `value` carries the added/removed
// element where it matters (notably `algorithms: none`).
func Classify(field string, op FieldOp, value any, confidence float64) Classification {
	// Confidence gate overrides everything (§9.2): unknown never pages.
	if confidence < confidenceFloor {
		return Classification{Class: model.ChangeUnknown, Severity: model.SeverityInfo}
	}

	switch field {
	case FieldAudiences:
		if op == OpAdd {
			return Classification{model.ChangeWidened, model.SeverityMedium}
		}
		return Classification{model.ChangeNarrowed, model.SeverityLow}

	case FieldIssuers:
		if op == OpAdd {
			return Classification{model.ChangeWidened, model.SeverityHigh}
		}
		return Classification{model.ChangeNarrowed, model.SeverityLow}

	case FieldAlgorithms:
		if op == OpAdd {
			if s, ok := value.(string); ok && s == "none" {
				return Classification{model.ChangeWidened, model.SeverityCritical}
			}
			return Classification{model.ChangeWidened, model.SeverityMedium}
		}
		return Classification{model.ChangeNarrowed, model.SeverityLow}

	case FieldRequiredClaims:
		if op == OpAdd {
			// Requiring a new claim narrows what is accepted.
			return Classification{model.ChangeNarrowed, model.SeverityLow}
		}
		// Removing a required claim widens acceptance — critical.
		return Classification{model.ChangeWidened, model.SeverityCritical}

	case FieldClockSkew:
		if op == OpIncrease {
			return Classification{model.ChangeWidened, model.SeverityMedium}
		}
		return Classification{model.ChangeNarrowed, model.SeverityLow}

	case FieldRefreshUnknown:
		if op == OpTrueToFalse {
			// Stops refreshing on unknown kid — the OpenFGA-class outage cause.
			return Classification{model.ChangeNarrowed, model.SeverityHigh}
		}
		return Classification{model.ChangeWidened, model.SeverityLow}

	case FieldCacheTTL:
		if op == OpIncrease {
			// Longer cache raises the required grace period.
			return Classification{model.ChangeNarrowed, model.SeverityMedium}
		}
		return Classification{model.ChangeNarrowed, model.SeverityLow}

	case FieldConsumer:
		// consumer added/removed are neutral, low severity.
		return Classification{model.ChangeNeutral, model.SeverityLow}
	}

	// Unrecognized field: report but do not page.
	return Classification{model.ChangeUnknown, model.SeverityInfo}
}
