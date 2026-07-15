package app

import (
	"testing"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestResourceFiltersIncludeStorageFilter(t *testing.T) {
	a := &App{
		resourceStorageFilter:           &client.Storage{ID: "st-1", Name: "terraform-state"},
		resourceSecretFilter:            &client.Secret{ID: "sec-1", Name: "prod-aws-creds"},
		resourceTemplateFilter:          &client.Template{ID: "tpl-1", Name: "base-template"},
		resourceSourceCodeVersionFilter: &client.SourceCodeVersion{ID: "scv-1", Identifier: "modules/redis:v1.2.3"},
		resourceIntegrationFilter:       &client.Integration{ID: "int-1", Name: "aws"},
		hideDestroyedResources:          true,
	}

	filter := a.resourceFilters()
	if filter["storage_id"] != "st-1" {
		t.Fatalf("storage filter = %#v", filter["storage_id"])
	}
	if got, ok := filter["secret_ids__any"].([]string); !ok || len(got) != 1 || got[0] != "sec-1" {
		t.Fatalf("secret filter = %#v", filter["secret_ids__any"])
	}
	if filter["template_id"] != "tpl-1" {
		t.Fatalf("template filter = %#v", filter["template_id"])
	}
	if filter["source_code_version_id"] != "scv-1" {
		t.Fatalf("source code version filter = %#v", filter["source_code_version_id"])
	}
}

func TestFilterSourceCodeVersionsMatchesIdentifierTemplateAndVersion(t *testing.T) {
	versions := []client.SourceCodeVersion{
		{ID: "scv-1", Identifier: "modules/redis:v1.2.3", SourceCodeVersion: "v1.2.3", SourceCodeFolder: "modules/redis", Template: &client.Template{Name: "aws_redis"}},
		{ID: "scv-2", Identifier: "modules/cache:main", SourceCodeBranch: "main", SourceCodeFolder: "modules/cache", Template: &client.Template{Name: "gcp_cache"}},
	}

	byIdentifier := filterSourceCodeVersions(versions, "redis")
	if len(byIdentifier) != 1 || byIdentifier[0].ID != "scv-1" {
		t.Fatalf("filter by identifier = %#v", byIdentifier)
	}

	byTemplate := filterSourceCodeVersions(versions, "gcp_cache")
	if len(byTemplate) != 1 || byTemplate[0].ID != "scv-2" {
		t.Fatalf("filter by template = %#v", byTemplate)
	}

	byVersion := filterSourceCodeVersions(versions, "v1.2.3")
	if len(byVersion) != 1 || byVersion[0].ID != "scv-1" {
		t.Fatalf("filter by version = %#v", byVersion)
	}

	byBranch := filterSourceCodeVersions(versions, "main")
	if len(byBranch) != 1 || byBranch[0].ID != "scv-2" {
		t.Fatalf("filter by branch = %#v", byBranch)
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

func TestFilterSecretsMatchesNameProviderAndType(t *testing.T) {
	secrets := []client.Secret{
		{ID: "sec-1", Name: "prod-aws-creds", SecretProvider: "aws", SecretType: "cloud_credentials", Status: "available"},
		{ID: "sec-2", Name: "vault-app", SecretProvider: "vault", SecretType: "kv", Status: "available"},
	}

	byName := filterSecrets(secrets, "prod")
	if len(byName) != 1 || byName[0].ID != "sec-1" {
		t.Fatalf("filter by name = %#v", byName)
	}

	byProvider := filterSecrets(secrets, "vault")
	if len(byProvider) != 1 || byProvider[0].ID != "sec-2" {
		t.Fatalf("filter by provider = %#v", byProvider)
	}

	byType := filterSecrets(secrets, "kv")
	if len(byType) != 1 || byType[0].ID != "sec-2" {
		t.Fatalf("filter by type = %#v", byType)
	}
}
