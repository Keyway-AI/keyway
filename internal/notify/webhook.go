package notify

import (
	"context"

	"github.com/architsharma/keyway/internal/model"
)

// Webhook posts change events as JSON to an arbitrary HTTP endpoint.
type Webhook struct {
	url string
}

// NewWebhook constructs a webhook notifier.
func NewWebhook(url string) *Webhook { return &Webhook{url: url} }

var _ Notifier = (*Webhook)(nil)

// Notify POSTs the events as JSON. TODO(M9).
func (w *Webhook) Notify(ctx context.Context, events []model.ChangeEvent) error {
	_ = ctx
	_ = events
	if w.url == "" {
		return nil
	}
	return model.ErrUnsupported
}
