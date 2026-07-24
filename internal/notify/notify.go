// Package notify delivers change events to external sinks (Slack, webhooks).
// unknown-class changes never page (PRD §9.2); only medium+ severity notifies.
package notify

import (
	"context"

	"github.com/nometria/keyway/internal/model"
)

// Notifier delivers a batch of change events.
type Notifier interface {
	Notify(ctx context.Context, events []model.ChangeEvent) error
}

// ShouldNotify reports whether an event warrants paging. unknown-class and info
// severity are report-only.
func ShouldNotify(ev model.ChangeEvent) bool {
	if ev.Class == model.ChangeUnknown {
		return false
	}
	switch ev.Severity {
	case model.SeverityCritical, model.SeverityHigh, model.SeverityMedium:
		return true
	default:
		return false
	}
}
