package notify

import (
	"context"

	"github.com/architsharma/keyway/internal/model"
)

// Slack posts change events to a Slack incoming webhook.
type Slack struct {
	webhookURL string
}

// NewSlack constructs a Slack notifier.
func NewSlack(webhookURL string) *Slack { return &Slack{webhookURL: webhookURL} }

var _ Notifier = (*Slack)(nil)

// Notify posts notifiable events to Slack. TODO(M9): render a Block Kit message
// grouping events by consumer/severity; respect ShouldNotify.
func (s *Slack) Notify(ctx context.Context, events []model.ChangeEvent) error {
	_ = ctx
	_ = events
	if s.webhookURL == "" {
		return nil // disabled
	}
	return model.ErrUnsupported
}
