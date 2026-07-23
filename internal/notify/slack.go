package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/architsharma/keyway/internal/model"
)

// Slack posts change events to a Slack incoming webhook.
type Slack struct {
	webhookURL string
	client     *http.Client
}

// NewSlack constructs a Slack notifier.
func NewSlack(webhookURL string) *Slack {
	return &Slack{webhookURL: webhookURL, client: &http.Client{Timeout: 10 * time.Second}}
}

var _ Notifier = (*Slack)(nil)

// Notify posts notifiable events (medium+ severity, never unknown) to Slack as a
// Block Kit message grouped by severity.
func (s *Slack) Notify(ctx context.Context, events []model.ChangeEvent) error {
	if s.webhookURL == "" {
		return nil // disabled
	}
	notifiable := filterNotifiable(events)
	if len(notifiable) == 0 {
		return nil
	}
	payload := map[string]any{
		"blocks": slackBlocks(notifiable),
		"text":   fmt.Sprintf("Keyway: %d contract change(s) need attention", len(notifiable)),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("notify: slack post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("notify: slack returned %d", resp.StatusCode)
	}
	return nil
}

func slackBlocks(events []model.ChangeEvent) []map[string]any {
	blocks := []map[string]any{{
		"type": "header",
		"text": map[string]any{"type": "plain_text", "text": "🔑 Keyway contract changes"},
	}}
	for _, e := range events {
		blocks = append(blocks, map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*%s* — `%s` %s → *%s* (%v → %v)",
					severityEmoji(e.Severity), e.ConsumerID, e.Field, e.Class, e.OldValue, e.NewValue),
			},
		})
	}
	return blocks
}

func severityEmoji(s model.Severity) string {
	switch s {
	case model.SeverityCritical:
		return "🔴 critical"
	case model.SeverityHigh:
		return "🟠 high"
	case model.SeverityMedium:
		return "🟡 medium"
	default:
		return string(s)
	}
}

func filterNotifiable(events []model.ChangeEvent) []model.ChangeEvent {
	var out []model.ChangeEvent
	for _, e := range events {
		if ShouldNotify(e) {
			out = append(out, e)
		}
	}
	return out
}
