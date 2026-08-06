package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Keyway-AI/keyway/internal/model"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name       string
		field      string
		op         FieldOp
		value      any
		confidence float64
		wantClass  model.ChangeClass
		wantSev    model.Severity
	}{
		{"audience added", FieldAudiences, OpAdd, "api-b", 1.0, model.ChangeWidened, model.SeverityMedium},
		{"audience removed", FieldAudiences, OpRemove, "api-b", 1.0, model.ChangeNarrowed, model.SeverityLow},
		{"issuer added", FieldIssuers, OpAdd, "https://x", 1.0, model.ChangeWidened, model.SeverityHigh},
		{"algorithm added", FieldAlgorithms, OpAdd, "ES256", 1.0, model.ChangeWidened, model.SeverityMedium},
		{"alg none added is critical", FieldAlgorithms, OpAdd, "none", 1.0, model.ChangeWidened, model.SeverityCritical},
		{"required claim added narrows", FieldRequiredClaims, OpAdd, "dept", 1.0, model.ChangeNarrowed, model.SeverityLow},
		{"required claim removed is critical", FieldRequiredClaims, OpRemove, "dept", 1.0, model.ChangeWidened, model.SeverityCritical},
		{"clock skew increase widens", FieldClockSkew, OpIncrease, 120, 1.0, model.ChangeWidened, model.SeverityMedium},
		{"clock skew decrease narrows", FieldClockSkew, OpDecrease, 30, 1.0, model.ChangeNarrowed, model.SeverityLow},
		{"refresh unknown kid off is high", FieldRefreshUnknown, OpTrueToFalse, false, 1.0, model.ChangeNarrowed, model.SeverityHigh},
		{"cache ttl increase", FieldCacheTTL, OpIncrease, 3600, 1.0, model.ChangeNarrowed, model.SeverityMedium},
		{"consumer added neutral", FieldConsumer, OpConsumerAdd, nil, 1.0, model.ChangeNeutral, model.SeverityLow},
		{"low confidence is unknown", FieldAudiences, OpAdd, "api-b", 0.4, model.ChangeUnknown, model.SeverityInfo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.field, tc.op, tc.value, tc.confidence)
			assert.Equal(t, tc.wantClass, got.Class)
			assert.Equal(t, tc.wantSev, got.Severity)
		})
	}
}
