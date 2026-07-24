package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nometria/keyway/internal/config"
	"github.com/nometria/keyway/internal/diff"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/store/postgres"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "keyway.yaml"

func configPath(cmd *cobra.Command) string {
	if p, _ := cmd.Flags().GetString("config"); p != "" {
		return p
	}
	if p := os.Getenv("KEYWAY_CONFIG"); p != "" {
		return p
	}
	return defaultConfigPath
}

// --- init -------------------------------------------------------------------

func runInit(cmd *cobra.Command) error {
	path := configPath(cmd)
	out := cmd.OutOrStdout()

	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(out, "Config already exists at %s (leaving it untouched).\n", path)
	} else {
		cfg := config.Default()
		if cfg.DBURL == "" {
			cfg.DBURL = "postgres://keyway:keyway@localhost:5432/keyway?sslmode=disable"
		}
		b, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, b, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Fprintf(out, "Wrote %s\n", path)
	}

	// Test connectivity if a DB is configured.
	if dsn := dbURL(cmd); dsn != "" {
		st, err := postgres.Open(context.Background(), dsn)
		if err != nil {
			fmt.Fprintf(out, "⚠ database not reachable: %v\n", err)
			return nil
		}
		st.Close()
		fmt.Fprintln(out, "✓ database reachable")
	} else {
		fmt.Fprintln(out, "No database configured yet (set KEYWAY_DB_URL or db_url in config).")
	}
	return nil
}

// --- issuer add/list --------------------------------------------------------

func runIssuerAdd(cmd *cobra.Command) error {
	path := configPath(cmd)
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	name, _ := cmd.Flags().GetString("name")
	typ, _ := cmd.Flags().GetString("type")
	url, _ := cmd.Flags().GetString("url")
	credEnv, _ := cmd.Flags().GetString("admin-credential-env")
	if name == "" {
		name = typ // default the name to the type
	}

	for _, ic := range cfg.Issuers {
		if ic.Name == name {
			return fmt.Errorf("an issuer named %q already exists", name)
		}
	}
	cfg.Issuers = append(cfg.Issuers, config.IssuerConfig{
		Name: name, Type: typ, URL: url, AdminCredentialEnv: credEnv,
	})
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Registered issuer %q (%s) in %s\n", name, typ, path)
	return nil
}

func runIssuerList(cmd *cobra.Command) error {
	cfg, err := config.Load(configPath(cmd))
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if len(cfg.Issuers) == 0 {
		fmt.Fprintln(out, "No issuers registered. Add one with `keyway issuer add`.")
		return nil
	}
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tURL\tCREDENTIAL ENV")
	for _, ic := range cfg.Issuers {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", ic.Name, ic.Type, ic.URL, ic.AdminCredentialEnv)
	}
	return tw.Flush()
}

// --- diff -------------------------------------------------------------------

func runDiff(cmd *cobra.Command) error {
	dsn := dbURL(cmd)
	if dsn == "" {
		return fmt.Errorf("no database configured (set --db or KEYWAY_DB_URL)")
	}
	ctx := context.Background()
	st, err := postgres.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()

	from, err := resolveVersion(ctx, st, cmdString(cmd, "from"), true)
	if err != nil {
		return fmt.Errorf("resolve --from: %w", err)
	}
	to, err := resolveVersion(ctx, st, cmdString(cmd, "to"), false)
	if err != nil {
		return fmt.Errorf("resolve --to: %w", err)
	}

	events := diff.Compute(from, to)
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Diff %s → %s: %d change event(s)\n\n", short(from.Hash), short(to.Hash), len(events))
	if len(events) == 0 {
		fmt.Fprintln(out, "No changes.")
		return nil
	}
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "CONSUMER\tFIELD\tCLASS\tSEVERITY\tOLD → NEW")
	for _, e := range events {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%v → %v\n", short(e.ConsumerID), e.Field, e.Class, e.Severity, e.OldValue, e.NewValue)
	}
	return tw.Flush()
}

func resolveVersion(ctx context.Context, st *postgres.Store, id string, baselineDefault bool) (model.ContractVersion, error) {
	if id != "" {
		return st.GetContractVersion(ctx, id)
	}
	if baselineDefault {
		return st.BaselineVersion(ctx)
	}
	return st.LatestVersion(ctx)
}

func cmdString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
