package app

import (
	"testing"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestResourceFiltersIncludeStorageFilter(t *testing.T) {
	a := &App{
		resourceStorageFilter:     &client.Storage{ID: "st-1", Name: "terraform-state"},
		resourceTemplateFilter:    &client.Template{ID: "tpl-1", Name: "base-template"},
		resourceIntegrationFilter: &client.Integration{ID: "int-1", Name: "aws"},
		hideDestroyedResources:    true,
	}

	filter := a.resourceFilters()
	if filter["storage_id"] != "st-1" {
		t.Fatalf("storage filter = %#v", filter["storage_id"])
	}
	if filter["template_id"] != "tpl-1" {
		t.Fatalf("template filter = %#v", filter["template_id"])
	}
}

func TestFilterStoragesMatchesNameProviderAndType(t *testing.T) {
	storages := []client.Storage{
		{ID: "st-1", Name: "terraform-state", StorageProvider: "aws", StorageType: "tofu"},
		{ID: "st-2", Name: "cache-state", StorageProvider: "gcp", StorageType: "tofu"},
	}

	byName := filterStorages(storages, "terraform")
	if len(byName) != 1 || byName[0].ID != "st-1" {
		t.Fatalf("filter by name = %#v", byName)
	}

	byProvider := filterStorages(storages, "gcp")
	if len(byProvider) != 1 || byProvider[0].ID != "st-2" {
		t.Fatalf("filter by provider = %#v", byProvider)
	}

	byType := filterStorages(storages, "tofu")
	if len(byType) != 2 {
		t.Fatalf("filter by type = %#v", byType)
	}
}
