package app

import (
	"testing"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

func TestRestoreDetailStateRestoresOverviewJumpActions(t *testing.T) {
	a := &App{
		overviewJumpActions: map[rune]func(){
			'w': func() {},
			'T': func() {},
		},
		overviewJumpSelector:    &overviewJumpSelector{title: "selector"},
		auditLogRows:            []tabledata.Row{{ID: "a1"}},
		auditLogTable:           tview.NewTable(),
		overviewTree:            &overviewTreeSelection{},
		entitySelectorTable:     tview.NewTable(),
		activeStorageDetail:     &entityDetailSelection{ID: "st1", Name: "state", Kind: "storages"},
		activeWorkspaceDetail:   &entityDetailSelection{ID: "w1", Name: "platform", Kind: "workspaces"},
		activeIntegrationDetail: &entityDetailSelection{ID: "i1", Name: "aws", Kind: "integrations"},
	}

	snapshot := a.captureDetailState()
	a.clearOverviewJumpState()
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.overviewTree = nil
	a.activeStorageDetail = nil
	a.activeWorkspaceDetail = nil
	a.activeIntegrationDetail = nil

	a.restoreDetailState(snapshot)

	if a.overviewJumpActions == nil || a.overviewJumpActions['w'] == nil || a.overviewJumpActions['T'] == nil {
		t.Fatal("expected overview jump actions restored")
	}
	if a.overviewJumpSelector == nil {
		t.Fatal("expected overview jump selector restored")
	}
	if len(a.auditLogRows) != 1 || a.auditLogTable == nil {
		t.Fatal("expected audit state restored")
	}
	if a.entitySelectorTable != nil {
		t.Fatal("expected entity selector state cleared on detail restore")
	}
	if a.activeStorageDetail == nil || a.activeWorkspaceDetail == nil || a.activeIntegrationDetail == nil {
		t.Fatal("expected active detail selections restored")
	}
}

func TestResourceOverviewJumpActionsIncludeWorkspaceOnW(t *testing.T) {
	a := &App{}
	resource := client.Resource{
		ID:        "r1",
		Name:      "redis",
		Workspace: &client.Workspace{ID: "w1", Name: "platform"},
	}

	actions := a.resourceOverviewJumpActions(resource)
	if actions == nil {
		t.Fatal("expected jump actions")
	}
	if actions['w'] == nil {
		t.Fatal("expected workspace jump action on 'w'")
	}
	if actions['p'] != nil {
		t.Fatal("did not expect workspace jump action on 'p'")
	}
}

func TestResourceOverviewLinksIncludeWorkspace(t *testing.T) {
	links := resourceOverviewLinks(client.Resource{
		Workspace: &client.Workspace{ID: "w1", Name: "platform"},
	})
	if len(links) != 1 {
		t.Fatalf("links len = %d, want 1", len(links))
	}
	if links[0] != "w Workspace: platform" {
		t.Fatalf("workspace link = %q", links[0])
	}
}

func TestResourceFiltersIncludeStorageFilter(t *testing.T) {
	a := &App{
		resourceStorageFilter:           &client.Storage{ID: "st-1", Name: "terraform-state"},
		resourceWorkspaceFilter:         &client.Workspace{ID: "ws-1", Name: "platform"},
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
	if filter["workspace_id"] != "ws-1" {
		t.Fatalf("workspace filter = %#v", filter["workspace_id"])
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

func TestFilterWorkspacesMatchesNameProviderAndStatus(t *testing.T) {
	workspaces := []client.Workspace{
		{ID: "ws-1", Name: "platform", WorkspaceProvider: "github", Status: "ready", Description: "Main workspace"},
		{ID: "ws-2", Name: "delivery", WorkspaceProvider: "bitbucket", Status: "pending", Description: "Delivery repos"},
	}

	byName := filterWorkspaces(workspaces, "platform")
	if len(byName) != 1 || byName[0].ID != "ws-1" {
		t.Fatalf("filter by name = %#v", byName)
	}

	byProvider := filterWorkspaces(workspaces, "bitbucket")
	if len(byProvider) != 1 || byProvider[0].ID != "ws-2" {
		t.Fatalf("filter by provider = %#v", byProvider)
	}

	byStatus := filterWorkspaces(workspaces, "ready")
	if len(byStatus) != 1 || byStatus[0].ID != "ws-1" {
		t.Fatalf("filter by status = %#v", byStatus)
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
