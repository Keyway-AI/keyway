package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nometria/keyway/internal/model"
)

// Webhook posts change events as JSON to an arbitrary HTTP endpoint.
type Webhook struct {
	url    string
	client *http.Client
}

// NewWebhook constructs a webhook notifier.
func NewWebhook(url string) *Webhook {
	return &Webhook{url: url, client: &http.Client{Timeout: 10 * time.Second}}
}

var _ Notifier = (*Webhook)(nil)

// Notify POSTs all events (not just notifiable ones — the receiver decides) as a
// JSON array.
func (wh *Webhook) Notify(ctx context.Context, events []model.ChangeEvent) error {
	if wh.url == "" {
		return nil
	}
	if len(events) == 0 {
		return nil
	}
	body, err := json.Marshal(map[string]any{"events": events})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := wh.client.Do(req)
	if err != nil {
		return fmt.Errorf("notify: webhook post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("notify: webhook returned %d", resp.StatusCode)
	}
	return nil
}
