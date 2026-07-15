package edit

import (
	"strings"
	"testing"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestResourceInputFromYAML(t *testing.T) {
	input, err := ResourceInputFromYAML([]byte(`name: redis
source_code_version:
  id: scv1
storage:
  id: st1
workspace:
  id: w1
integration_ids:
  - id: i1
secret_ids:
  - id: s1
labels:
  - prod
dependency_tags:
  - name: env
    value: prod
`))
	if err != nil {
		t.Fatalf("ResourceInputFromYAML: %v", err)
	}
	if input["sourceCodeVersionId"] != "scv1" || input["storageId"] != "st1" || input["workspaceId"] != "w1" {
		t.Fatalf("unexpected id fields: %#v", input)
	}
	ids, _ := input["integrationIds"].([]string)
	if len(ids) != 1 || ids[0] != "i1" {
		t.Fatalf("integration ids = %#v", input["integrationIds"])
	}
}

func TestTemplateInputFromYAML(t *testing.T) {
	input, err := TemplateInputFromYAML([]byte(`name: tpl
template: resource {}
cloud_resource_types:
  - redis
parents:
  - id: p1
children:
  - c1
configuration:
  tier: backend
`))
	if err != nil {
		t.Fatalf("TemplateInputFromYAML: %v", err)
	}
	if input["name"] != "tpl" {
		t.Fatalf("name = %#v", input["name"])
	}
	parents, _ := input["parents"].([]string)
	if len(parents) != 1 || parents[0] != "p1" {
		t.Fatalf("parents = %#v", input["parents"])
	}
	if _, ok := input["template"]; ok {
		t.Fatalf("template should not be included in update input: %#v", input)
	}
}

func TestIntegrationInputFromYAML(t *testing.T) {
	input, err := IntegrationInputFromYAML([]byte(`name: aws
labels:
  - prod
configuration:
  role: admin
`))
	if err != nil {
		t.Fatalf("IntegrationInputFromYAML: %v", err)
	}
	if input["name"] != "aws" {
		t.Fatalf("name = %#v", input["name"])
	}
}

func TestWorkspaceInputFromYAML(t *testing.T) {
	input, err := WorkspaceInputFromYAML([]byte(`name: platform
description: Team workspace
labels:
  - prod
configuration:
  ignored: true
`))
	if err != nil {
		t.Fatalf("WorkspaceInputFromYAML: %v", err)
	}
	if input["name"] != "platform" || input["description"] != "Team workspace" {
		t.Fatalf("unexpected values: %#v", input)
	}
	if _, ok := input["configuration"]; ok {
		t.Fatalf("did not expect configuration in input: %#v", input)
	}
}

func TestStorageInputFromYAML(t *testing.T) {
	input, err := StorageInputFromYAML([]byte(`name: tf-state
description: Terraform state bucket
labels:
  - prod
configuration:
  aws_bucket_name: team-state
`))
	if err != nil {
		t.Fatalf("StorageInputFromYAML: %v", err)
	}
	if input["description"] != "Terraform state bucket" {
		t.Fatalf("unexpected values: %#v", input)
	}
	if _, ok := input["name"]; ok {
		t.Fatalf("did not expect name in input: %#v", input)
	}
	if _, ok := input["configuration"]; ok {
		t.Fatalf("did not expect configuration in input: %#v", input)
	}
}

func TestSourceCodeInputFromYAML(t *testing.T) {
	input, err := SourceCodeInputFromYAML([]byte(`description: Main infrastructure repo
integration:
  id: i1
labels:
  - platform
source_code_url: https://github.com/acme/infrastructure.git
`))
	if err != nil {
		t.Fatalf("SourceCodeInputFromYAML: %v", err)
	}
	if input["description"] != "Main infrastructure repo" || input["integrationId"] != "i1" {
		t.Fatalf("unexpected values: %#v", input)
	}
	if _, ok := input["sourceCodeUrl"]; ok {
		t.Fatalf("did not expect sourceCodeUrl in input: %#v", input)
	}
}

func TestResourceYAMLIncludesOnlyEditableFields(t *testing.T) {
	data, err := ResourceYAML(client.Resource{
		ID:          "r1",
		Name:        "redis",
		Description: "cache",
		State:       "ready",
		Status:      "ready",
		Labels:      []string{"prod"},
		Storage:     &client.Storage{ID: "st1", Name: "terraform-state"},
		Workspace:   &client.Workspace{ID: "w1", Name: "platform"},
		SourceCodeVersion: &client.SourceCodeVersion{
			ID:         "scv1",
			Identifier: "modules/redis:v1",
		},
		Integrations: []client.Integration{{ID: "i1", Name: "aws"}},
		Secrets:      []client.Secret{{ID: "s1", Name: "secret"}},
		StoragePath:  "ik/state/redis.tfstate",
		Variables:    []map[string]any{{"name": "replicas", "value": 1}},
	})
	if err != nil {
		t.Fatalf("ResourceYAML: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"name: redis",
		"description: cache",
		"source_code_version_id: scv1",
		"integration_ids:",
		"- i1",
		"secret_ids:",
		"- s1",
		"storage_id: st1",
		"storage_path: ik/state/redis.tfstate",
		"workspace_id: w1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected YAML to contain %q, got:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"\nstate:", "\nstatus:", "\ncreated_at:", "\nupdated_at:", "\nentity_name:", "\ntemplate:", "\nid:"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("did not expect YAML to contain %q, got:\n%s", unwanted, text)
		}
	}
}

func TestTemplateYAMLIncludesOnlyEditableFields(t *testing.T) {
	data, err := TemplateYAML(client.Template{
		ID:                 "t1",
		Name:               "aws_redis",
		Description:        "Redis template",
		Documentation:      "https://docs",
		Template:           "resource {}",
		CloudResourceTypes: []string{"redis"},
		Labels:             []string{"cache"},
		Configuration:      map[string]any{"tier": "backend"},
		Parents:            []client.TemplateReference{{ID: "p1", Name: "base"}},
		Children:           []client.TemplateReference{{ID: "c1", Name: "child"}},
		Status:             "ready",
	})
	if err != nil {
		t.Fatalf("TemplateYAML: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"name: aws_redis",
		"description: Redis template",
		"documentation: https://docs",
		"parents:",
		"- p1",
		"children:",
		"- c1",
		"cloud_resource_types:",
		"configuration:",
		"labels:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected YAML to contain %q, got:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"\ntemplate:", "\nstatus:", "\ncreated_at:", "\nupdated_at:", "\nentity_name:", "\nrevision_number:", "\nid:"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("did not expect YAML to contain %q, got:\n%s", unwanted, text)
		}
	}
}

func TestIntegrationYAMLIncludesOnlyEditableFields(t *testing.T) {
	data, err := IntegrationYAML(client.Integration{
		ID:                  "i1",
		Name:                "aws-prod",
		Description:         "AWS",
		Labels:              []string{"prod"},
		Configuration:       map[string]any{"role": "admin"},
		Status:              "ready",
		IntegrationProvider: "aws",
		IntegrationType:     "cloud",
	})
	if err != nil {
		t.Fatalf("IntegrationYAML: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"name: aws-prod",
		"description: AWS",
		"labels:",
		"- prod",
		"configuration:",
		"role: admin",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected YAML to contain %q, got:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"\nstatus:", "\ncreated_at:", "\nupdated_at:", "\nintegration_provider:", "\nintegration_type:", "\nentity_name:", "\nid:"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("did not expect YAML to contain %q, got:\n%s", unwanted, text)
		}
	}
}

func TestWorkspaceYAMLIncludesOnlyEditableFields(t *testing.T) {
	data, err := WorkspaceYAML(client.Workspace{
		ID:                "w1",
		Name:              "platform",
		Description:       "Team workspace",
		Labels:            []string{"prod"},
		WorkspaceProvider: "github",
		Status:            "ready",
		ResourcesCount:    3,
	})
	if err != nil {
		t.Fatalf("WorkspaceYAML: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"name: platform",
		"description: Team workspace",
		"labels:",
		"- prod",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected YAML to contain %q, got:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"\nworkspace_provider:", "\nstatus:", "\nresources_count:", "\nid:"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("did not expect YAML to contain %q, got:\n%s", unwanted, text)
		}
	}
}

func TestStorageYAMLIncludesOnlyEditableFields(t *testing.T) {
	data, err := StorageYAML(client.Storage{
		ID:            "st1",
		Name:          "terraform-state",
		Description:   "Primary state bucket",
		Labels:        []string{"prod"},
		Configuration: map[string]any{"aws_bucket_name": "tf-state"},
		State:         "ready",
		Status:        "ready",
		StorageType:   "tofu",
	})
	if err != nil {
		t.Fatalf("StorageYAML: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"description: Primary state bucket",
		"labels:",
		"- prod",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected YAML to contain %q, got:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"\nname:", "\nconfiguration:", "\nstate:", "\nstatus:", "\nstorage_type:", "\ncreated_at:", "\nupdated_at:", "\nid:"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("did not expect YAML to contain %q, got:\n%s", unwanted, text)
		}
	}
}

func TestSourceCodeYAMLIncludesOnlyEditableFields(t *testing.T) {
	data, err := SourceCodeYAML(client.SourceCode{
		ID:                 "sc1",
		Identifier:         "github.com/acme/infrastructure",
		Description:        "Main infrastructure repo",
		SourceCodeURL:      "https://github.com/acme/infrastructure.git",
		SourceCodeProvider: "github",
		SourceCodeLanguage: "opentofu",
		IntegrationID:      "i1",
		Labels:             []string{"platform"},
		Status:             "ready",
	})
	if err != nil {
		t.Fatalf("SourceCodeYAML: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"description: Main infrastructure repo",
		"integration_id: i1",
		"labels:",
		"- platform",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected YAML to contain %q, got:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"\nidentifier:", "\nsource_code_url:", "\nsource_code_provider:", "\nsource_code_language:", "\nstatus:", "\nid:"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("did not expect YAML to contain %q, got:\n%s", unwanted, text)
		}
	}
}

func TestDebugRoundTrip(t *testing.T) {
	r := client.Resource{
		Name:        "redis",
		Description: "cache",
		Variables:   []map[string]any{{"name": "replicas", "value": "3"}},
		Labels:      []string{"prod"},
	}
	yamlData, err := ResourceYAML(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("YAML:\n%s", yamlData)

	input, err := ResourceInputFromYAML(yamlData)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Input: %#v", input)
}
