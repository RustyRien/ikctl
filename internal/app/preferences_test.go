package app

import (
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/model"
)

func TestNewWithClientAppliesSavedViewPreferences(t *testing.T) {
	cfg := config.Config{
		Endpoint:         "https://example.com",
		RefreshInterval:  2 * time.Second,
		RefreshSeconds:   2,
		AutoRefresh:      true,
		DefaultSortOrder: "desc",
		ShowBreadcrumbs:  true,
		View: config.ViewConfig{
			Resources: config.ResourceViewConfig{
				Columns:                 []string{"name", "status", "age"},
				StorageFilter:           config.FilterRef{ID: "st-1", Name: "terraform-state"},
				SecretFilter:            config.FilterRef{ID: "sec-1", Name: "prod-aws-creds"},
				TemplateFilter:          config.FilterRef{ID: "tpl-1", Name: "base-template"},
				SourceCodeVersionFilter: config.FilterRef{ID: "scv-1", Name: "modules/redis:v1.2.3"},
				IntegrationFilter:       config.FilterRef{ID: "int-1", Name: "aws"},
				HideDestroyed:           true,
			},
			Templates: config.TemplateViewConfig{
				Columns: []string{"name", "updatedAt", "age"},
			},
		},
	}

	a := NewWithClient(cfg, BuildInfo{}, "resources", &client.Client{})

	if !a.visibleResourceColumns["name"] || !a.visibleResourceColumns["status"] || !a.visibleResourceColumns["age"] {
		t.Fatalf("resource columns not applied: %#v", a.visibleResourceColumns)
	}
	if a.visibleResourceColumns["template"] {
		t.Fatalf("unexpected default resource column still enabled: %#v", a.visibleResourceColumns)
	}
	if !a.visibleTemplateColumns["name"] || !a.visibleTemplateColumns["updatedAt"] || !a.visibleTemplateColumns["age"] {
		t.Fatalf("template columns not applied: %#v", a.visibleTemplateColumns)
	}
	if a.resourceStorageFilter == nil || a.resourceStorageFilter.ID != "st-1" {
		t.Fatalf("storage filter not applied: %#v", a.resourceStorageFilter)
	}
	if a.resourceSecretFilter == nil || a.resourceSecretFilter.ID != "sec-1" {
		t.Fatalf("secret filter not applied: %#v", a.resourceSecretFilter)
	}
	if a.resourceTemplateFilter == nil || a.resourceTemplateFilter.ID != "tpl-1" {
		t.Fatalf("template filter not applied: %#v", a.resourceTemplateFilter)
	}
	if a.resourceSourceCodeVersionFilter == nil || a.resourceSourceCodeVersionFilter.ID != "scv-1" {
		t.Fatalf("source code version filter not applied: %#v", a.resourceSourceCodeVersionFilter)
	}
	if a.resourceIntegrationFilter == nil || a.resourceIntegrationFilter.ID != "int-1" {
		t.Fatalf("integration filter not applied: %#v", a.resourceIntegrationFilter)
	}
	if !a.hideDestroyedResources {
		t.Fatalf("hide destroyed preference not applied")
	}

	resourceFilter := a.models[model.EntityResources].Filter()
	if resourceFilter["storage_id"] != "st-1" {
		t.Fatalf("resource model storage filter = %#v", resourceFilter)
	}
	if got, ok := resourceFilter["secret_ids__any"].([]string); !ok || len(got) != 1 || got[0] != "sec-1" {
		t.Fatalf("resource model secret filter = %#v", resourceFilter)
	}
	if resourceFilter["template_id"] != "tpl-1" {
		t.Fatalf("resource model template filter = %#v", resourceFilter)
	}
	if resourceFilter["source_code_version_id"] != "scv-1" {
		t.Fatalf("resource model source code version filter = %#v", resourceFilter)
	}
	if got, ok := resourceFilter["integration_ids__any"].([]string); !ok || len(got) != 1 || got[0] != "int-1" {
		t.Fatalf("resource model integration filter = %#v", resourceFilter)
	}
	if got, ok := resourceFilter["state__in"].([]string); !ok || len(got) != 3 {
		t.Fatalf("resource model destroyed filter = %#v", resourceFilter)
	}
}
