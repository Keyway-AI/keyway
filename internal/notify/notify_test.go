package notify

import (
	"testing"

	"github.com/architsharma/keyway/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestShouldNotify(t *testing.T) {
	cases := []struct {
		name  string
		class model.ChangeClass
		sev   model.Severity
		want  bool
	}{
		{"critical pages", model.ChangeWidened, model.SeverityCritical, true},
		{"high pages", model.ChangeNarrowed, model.SeverityHigh, true},
		{"medium pages", model.ChangeWidened, model.SeverityMedium, true},
		{"low does not page", model.ChangeNarrowed, model.SeverityLow, false},
		{"unknown never pages regardless of severity", model.ChangeUnknown, model.SeverityCritical, false},
		{"info does not page", model.ChangeNeutral, model.SeverityInfo, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev := model.ChangeEvent{Class: tc.class, Severity: tc.sev}
			assert.Equal(t, tc.want, ShouldNotify(ev))
		})
	}
}
