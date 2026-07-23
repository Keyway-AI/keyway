package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// apiClient is a minimal authenticated client for the Keyway HTTP API, used by
// commands (canary) that operate on the running daemon's in-memory state.
type apiClient struct {
	base   string
	token  string
	client *http.Client
}

func newAPIClient(cmd *cobra.Command) *apiClient {
	base, _ := cmd.Flags().GetString("api")
	if base == "" {
		base = "http://localhost:8080"
	}
	return &apiClient{
		base:   base,
		token:  os.Getenv("KEYWAY_API_TOKEN"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *apiClient) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach Keyway API at %s (is `keyway serve` running?): %w", a.base, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("API %s %s -> %d: %s", method, path, resp.StatusCode, string(data))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
