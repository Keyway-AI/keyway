// Package config loads and validates Keyway runtime configuration from a YAML
// file and/or environment variables.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the on-disk / environment configuration for the CLI and runner.
type Config struct {
	DBURL          string          `yaml:"db_url"`
	APIAddr        string          `yaml:"api_addr"`
	APIToken       string          `yaml:"api_token"`
	ProbeAllowlist []string        `yaml:"probe_allowlist"`
	Slack          SlackConfig     `yaml:"slack"`
	Issuers        []IssuerConfig  `yaml:"issuers"`
	Discovery      DiscoveryConfig `yaml:"discovery"`
}

// SlackConfig configures the optional Slack notifier.
type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
}

// IssuerConfig registers an issuer without embedding secrets — credentials are
// referenced by environment variable name only.
type IssuerConfig struct {
	Name               string `yaml:"name"`
	Type               string `yaml:"type"`
	URL                string `yaml:"url"`
	AdminCredentialEnv string `yaml:"admin_credential_env"`
}

// DiscoveryConfig bounds default discovery scope.
type DiscoveryConfig struct {
	KubeContext string   `yaml:"kube_context"`
	Namespaces  []string `yaml:"namespaces"`
	ConfigPaths []string `yaml:"config_paths"`
}

// Default returns a config seeded from environment variables.
func Default() Config {
	return Config{
		DBURL:    os.Getenv("KEYWAY_DB_URL"),
		APIAddr:  envOr("KEYWAY_API_ADDR", ":8080"),
		APIToken: os.Getenv("KEYWAY_API_TOKEN"),
		Slack:    SlackConfig{WebhookURL: os.Getenv("KEYWAY_SLACK_WEBHOOK_URL")},
	}
}

// Scaffold returns a config suitable for writing a fresh file: sane non-secret
// defaults only. It deliberately does NOT read secrets (API token, Slack
// webhook, DB URL) from the environment, so `keyway init` never captures an
// ambient secret into a plaintext file — those stay in the environment.
func Scaffold() Config {
	return Config{APIAddr: ":8080"}
}

// LoadFile reads a YAML config file WITHOUT overlaying environment variables.
// Read-modify-write commands (e.g. `issuer add`) use this so a marshal-back only
// ever persists what the file already contained — an ambient KEYWAY_API_TOKEN or
// KEYWAY_SLACK_WEBHOOK_URL is never written to disk as a side effect. A missing
// path returns an empty config with no error.
func LoadFile(path string) (Config, error) {
	var cfg Config
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Load reads a YAML config file, overlaying environment defaults for empty
// fields. A missing path returns the environment defaults with no error.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.DBURL == "" {
		cfg.DBURL = os.Getenv("KEYWAY_DB_URL")
	}
	return cfg, nil
}

// Validate checks required fields for the given operation.
func (c Config) Validate() error {
	if c.DBURL == "" {
		return fmt.Errorf("db_url is required (set KEYWAY_DB_URL or config db_url)")
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
