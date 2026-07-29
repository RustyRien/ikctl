package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/electrolux-oss/ik-tui/internal/config"
)

func TestLoginStoresProvidedTokenInConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	flags = config.FlagOverrides{}

	cmd := rootCmd
	cmd.SetArgs([]string{"--config", configPath, "--endpoint", "https://example.com", "--token", "token-123", "login"})
	t.Cleanup(func() {
		cmd.SetArgs(nil)
		flags = config.FlagOverrides{}
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute login: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if string(data) == "" {
		t.Fatal("expected config file to be written")
	}

	cfg, err := config.Load(config.FlagOverrides{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Token != "token-123" {
		t.Fatalf("token = %q", cfg.Token)
	}
	if cfg.Endpoint != "https://example.com" {
		t.Fatalf("endpoint = %q", cfg.Endpoint)
	}
}
