package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/architsharma/keyway/internal/model"
	"github.com/spf13/cobra"
)

// The canary commands operate on the running daemon's in-memory issuer keys via
// the HTTP API (canary state is owned by `keyway serve`, not Postgres).
func newCanaryCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "canary", Short: "Manage canary keys (announce without signing)"}
	cmd.PersistentFlags().String("api", "http://localhost:8080", "Keyway API base URL")

	start := &cobra.Command{
		Use:   "start",
		Short: "Announce a key in JWKS without using it to sign",
		RunE: func(cmd *cobra.Command, _ []string) error {
			issuer, _ := cmd.Flags().GetString("issuer")
			alg, _ := cmd.Flags().GetString("alg")
			var key model.Key
			if err := newAPIClient(cmd).do(context.Background(), "POST", "/v1/canary/announce",
				map[string]any{"issuer_id": issuer, "alg": alg}, &key); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Announced canary key %s (%s), status=%s — not yet used for signing.\n", key.KID, key.Alg, key.Status)
			return nil
		},
	}
	start.Flags().String("issuer", "", "issuer name")
	start.Flags().String("alg", "RS256", "key algorithm")

	status := &cobra.Command{
		Use:   "status",
		Short: "Show which key is announced/active",
		RunE: func(cmd *cobra.Command, _ []string) error {
			issuer, _ := cmd.Flags().GetString("issuer")
			var out map[string]any
			if err := newAPIClient(cmd).do(context.Background(), "GET", "/v1/canary/status?issuer_id="+issuer, nil, &out); err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	status.Flags().String("issuer", "", "issuer name")

	promote := &cobra.Command{
		Use:   "promote",
		Short: "Promote an announced key to active signing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			issuer, _ := cmd.Flags().GetString("issuer")
			kid, _ := cmd.Flags().GetString("kid")
			if err := newAPIClient(cmd).do(context.Background(), "POST", "/v1/canary/promote",
				map[string]any{"issuer_id": issuer, "kid": kid}, nil); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Promoted %s to active on %s.\n", kid, issuer)
			return nil
		},
	}
	promote.Flags().String("issuer", "", "issuer name")
	promote.Flags().String("kid", "", "key ID to promote")

	cmd.AddCommand(start, status, promote)
	return cmd
}
