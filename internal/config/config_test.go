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
	t.Setenv("IK_NO_COLORS", "true")
	t.Setenv("IK_AUTO_REFRESH", "false")
	t.Setenv("IK_DEFAULT_SORT_ORDER", "asc")
	t.Setenv("IK_SHOW_BREADCRUMBS", "false")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte("endpoint: https://file.example\ntoken: file-token\nrefresh_seconds: 3\nauto_refresh: true\ndefault_sort_order: desc\nshow_breadcrumbs: true\ninsecure_skip_tls_verify: false\nno_colors: false\n"), 0o644)
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
		NoColors:           false,
		HasNoColorsFlag:    true,
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
	if cfg.NoColors {
		t.Fatalf("expected no colors false")
	}
	if cfg.AutoRefresh {
		t.Fatalf("expected auto refresh false")
	}
	if cfg.DefaultSortOrder != "asc" {
		t.Fatalf("default sort order = %q", cfg.DefaultSortOrder)
	}
	if cfg.ShowBreadcrumbs {
		t.Fatalf("expected show breadcrumbs false")
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
	if !cfg.AutoRefresh {
		t.Fatalf("expected auto refresh true")
	}
	if cfg.DefaultSortOrder != defaultSortOrder {
		t.Fatalf("default sort order = %q", cfg.DefaultSortOrder)
	}
	if !cfg.ShowBreadcrumbs {
		t.Fatalf("expected show breadcrumbs true")
	}
}

func TestSavePersistsSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Config{
		Endpoint:         "https://example.com",
		RefreshInterval:  2 * time.Second,
		RefreshSeconds:   2,
		AutoRefresh:      false,
		DefaultSortOrder: "asc",
		ShowBreadcrumbs:  false,
		ConfigPath:       path,
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	reloaded, err := Load(FlagOverrides{ConfigPath: path})
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.AutoRefresh {
		t.Fatalf("expected auto refresh false after reload")
	}
	if reloaded.DefaultSortOrder != "asc" {
		t.Fatalf("expected default sort order asc after reload, got %q", reloaded.DefaultSortOrder)
	}
	if reloaded.ShowBreadcrumbs {
		t.Fatalf("expected show breadcrumbs false after reload")
	}
}

func TestSavePersistsTUIViewPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Config{
		Endpoint:         "https://example.com",
		RefreshInterval:  2 * time.Second,
		RefreshSeconds:   2,
		AutoRefresh:      true,
		DefaultSortOrder: "desc",
		ShowBreadcrumbs:  true,
		ConfigPath:       path,
		View: ViewConfig{
			Resources: ResourceViewConfig{
				Columns:                 []string{"name", "status", "age"},
				StorageFilter:           FilterRef{ID: "st-1", Name: "terraform-state"},
				TemplateFilter:          FilterRef{ID: "tpl-1", Name: "base-template"},
				SourceCodeVersionFilter: FilterRef{ID: "scv-1", Name: "modules/redis:v1.2.3"},
				IntegrationFilter:       FilterRef{ID: "int-1", Name: "aws"},
				HideDestroyed:           true,
			},
			Templates: TemplateViewConfig{
				Columns: []string{"name", "updatedAt", "age"},
			},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	reloaded, err := Load(FlagOverrides{ConfigPath: path})
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if got := reloaded.View.Resources.Columns; len(got) != 3 || got[0] != "name" || got[1] != "status" || got[2] != "age" {
		t.Fatalf("unexpected resource columns: %#v", got)
	}
	if reloaded.View.Resources.StorageFilter.ID != "st-1" || reloaded.View.Resources.StorageFilter.Name != "terraform-state" {
		t.Fatalf("unexpected storage filter: %#v", reloaded.View.Resources.StorageFilter)
	}
	if reloaded.View.Resources.TemplateFilter.ID != "tpl-1" || reloaded.View.Resources.TemplateFilter.Name != "base-template" {
		t.Fatalf("unexpected template filter: %#v", reloaded.View.Resources.TemplateFilter)
	}
	if reloaded.View.Resources.SourceCodeVersionFilter.ID != "scv-1" || reloaded.View.Resources.SourceCodeVersionFilter.Name != "modules/redis:v1.2.3" {
		t.Fatalf("unexpected source code version filter: %#v", reloaded.View.Resources.SourceCodeVersionFilter)
	}
	if reloaded.View.Resources.IntegrationFilter.ID != "int-1" || reloaded.View.Resources.IntegrationFilter.Name != "aws" {
		t.Fatalf("unexpected integration filter: %#v", reloaded.View.Resources.IntegrationFilter)
	}
	if !reloaded.View.Resources.HideDestroyed {
		t.Fatalf("expected hide destroyed true after reload")
	}
	if got := reloaded.View.Templates.Columns; len(got) != 3 || got[0] != "name" || got[1] != "updatedAt" || got[2] != "age" {
		t.Fatalf("unexpected template columns: %#v", got)
	}
}
