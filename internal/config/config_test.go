package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadPrecedence(t *testing.T) {
	t.Setenv("IK_ENDPOINT", "https://env.example")
	t.Setenv("IK_TOKEN", "env-token")
	t.Setenv("IK_REFRESH_SECONDS", "4")
	t.Setenv("IK_INSECURE_SKIP_TLS_VERIFY", "true")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte("endpoint: https://file.example\ntoken: file-token\nrefresh_seconds: 3\ninsecure_skip_tls_verify: false\n"), 0o644)
	if err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := Load(FlagOverrides{
		ConfigPath:         path,
		Endpoint:           "https://flag.example",
		Token:              "flag-token",
		RefreshSeconds:     5,
		InsecureSkipVerify: false,
		HasInsecureFlag:    true,
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Endpoint != "https://flag.example" {
		t.Fatalf("endpoint = %q", cfg.Endpoint)
	}
	if cfg.Token != "flag-token" {
		t.Fatalf("token = %q", cfg.Token)
	}
	if cfg.RefreshInterval != 5*time.Second {
		t.Fatalf("refresh = %v", cfg.RefreshInterval)
	}
	if cfg.InsecureSkipVerify {
		t.Fatalf("expected insecure skip verify false")
	}
}

func TestLoadDefaultsWithoutFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	cfg, err := Load(FlagOverrides{ConfigPath: path})
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	if cfg.Endpoint != defaultEndpoint {
		t.Fatalf("endpoint = %q", cfg.Endpoint)
	}
	if cfg.RefreshInterval != defaultRefreshInterval {
		t.Fatalf("refresh = %v", cfg.RefreshInterval)
	}
}
