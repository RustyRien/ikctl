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
				Columns:           []string{"name", "status", "age"},
				TemplateFilter:    config.FilterRef{ID: "tpl-1", Name: "base-template"},
				IntegrationFilter: config.FilterRef{ID: "int-1", Name: "aws"},
				HideDestroyed:     true,
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
	if a.resourceTemplateFilter == nil || a.resourceTemplateFilter.ID != "tpl-1" {
		t.Fatalf("template filter not applied: %#v", a.resourceTemplateFilter)
	}
	if a.resourceIntegrationFilter == nil || a.resourceIntegrationFilter.ID != "int-1" {
		t.Fatalf("integration filter not applied: %#v", a.resourceIntegrationFilter)
	}
	if !a.hideDestroyedResources {
		t.Fatalf("hide destroyed preference not applied")
	}

	resourceFilter := a.models[model.EntityResources].Filter()
	if resourceFilter["template_id"] != "tpl-1" {
		t.Fatalf("resource model template filter = %#v", resourceFilter)
	}
	if got, ok := resourceFilter["integration_ids__any"].([]string); !ok || len(got) != 1 || got[0] != "int-1" {
		t.Fatalf("resource model integration filter = %#v", resourceFilter)
	}
	if got, ok := resourceFilter["state__in"].([]string); !ok || len(got) != 3 {
		t.Fatalf("resource model destroyed filter = %#v", resourceFilter)
	}
}
