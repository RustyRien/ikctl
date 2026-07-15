package app

import (
	"context"
	"strings"
	"testing"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

func TestEntityActionPromptForRowSupportsTemplatesAndIntegrations(t *testing.T) {
	a := &App{client: &client.Client{}}

	templatePrompt, ok := a.entityActionPromptForRow(tabledata.Row{Raw: client.Template{ID: "t1", Name: "tpl"}}, "enable")
	if !ok || templatePrompt == nil {
		t.Fatal("expected template enable prompt")
	}
	if templatePrompt.Kind != "template" || templatePrompt.Verb != "enable" || templatePrompt.ID != "t1" {
		t.Fatalf("unexpected template prompt: %#v", templatePrompt)
	}

	integrationPrompt, ok := a.entityActionPromptForRow(tabledata.Row{Raw: client.Integration{ID: "i1", Name: "aws"}}, "delete")
	if !ok || integrationPrompt == nil {
		t.Fatal("expected integration delete prompt")
	}
	if integrationPrompt.Kind != "integration" || integrationPrompt.Verb != "delete" || integrationPrompt.ID != "i1" {
		t.Fatalf("unexpected integration prompt: %#v", integrationPrompt)
	}

	resourcePrompt, ok := a.entityActionPromptForRow(tabledata.Row{Raw: client.Resource{ID: "r1", Name: "res"}}, "disable")
	if ok || resourcePrompt != nil {
		t.Fatalf("unexpected resource action prompt: %#v", resourcePrompt)
	}
}

func TestTitleCase(t *testing.T) {
	if got := titleCase("enable"); got != "Enable" {
		t.Fatalf("titleCase(enable) = %q", got)
	}
	if got := titleCase(""); got != "" {
		t.Fatalf("titleCase(empty) = %q", got)
	}
}

func TestActionPromptFactoriesReturnAction(t *testing.T) {
	a := &App{client: &client.Client{}}
	for _, tc := range []struct {
		name string
		mk   func() (*entityActionPrompt, bool)
	}{
		{name: "template disable", mk: func() (*entityActionPrompt, bool) {
			return a.templateActionPrompt(client.Template{ID: "t1", Name: "tpl"}, "disable")
		}},
		{name: "integration enable", mk: func() (*entityActionPrompt, bool) {
			return a.integrationActionPrompt(client.Integration{ID: "i1", Name: "aws"}, "enable")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prompt, ok := tc.mk()
			if !ok || prompt == nil || prompt.Action == nil {
				t.Fatalf("invalid prompt: %#v ok=%v", prompt, ok)
			}
			_ = prompt.Action
		})
	}

	if prompt, ok := a.templateActionPrompt(client.Template{ID: "t1"}, "bogus"); ok || prompt != nil {
		t.Fatalf("unexpected prompt for bogus verb: %#v", prompt)
	}

	_ = context.Background()
}

func TestYAMLDetailForRowSupportsEntities(t *testing.T) {
	tests := []struct {
		name      string
		row       tabledata.Row
		wantTitle string
	}{
		{name: "resource", row: tabledata.Row{Raw: client.Resource{ID: "r1", Name: "redis"}}, wantTitle: "YAML: Resource redis"},
		{name: "template", row: tabledata.Row{Raw: client.Template{ID: "t1", Name: "base"}}, wantTitle: "YAML: Template base"},
		{name: "integration", row: tabledata.Row{Raw: client.Integration{ID: "i1", Name: "aws"}}, wantTitle: "YAML: Integration aws"},
	}

	a := &App{client: &client.Client{}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			title, entityID, fetch, ok := a.yamlDetailForRow(tc.row)
			if !ok {
				t.Fatal("expected yaml support")
			}
			if title != tc.wantTitle {
				t.Fatalf("title = %q, want %q", title, tc.wantTitle)
			}
			if entityID == "" {
				t.Fatal("expected entity id")
			}
			if fetch == nil {
				t.Fatal("expected fetch function")
			}
		})
	}

	if _, _, _, ok := a.yamlDetailForRow(tabledata.Row{}); ok {
		t.Fatal("expected unsupported row to be rejected")
	}
}

func TestColorizeYAMLColorsKeys(t *testing.T) {
	input := "name: redis\nmetadata:\n  labels:\n    app: cache\n  list:\n    - item: value\n[brackets]: kept"
	got := colorizeYAML(input, true)

	for _, want := range []string{
		"[steelblue::b]name[white::-]: [papayawhip::]redis",
		"[steelblue::b]metadata[white::-]:",
		"  [steelblue::b]labels[white::-]:",
		"    [steelblue::b]app[white::-]: [papayawhip::]cache",
		"    - [steelblue::b]item[white::-]: [papayawhip::]value",
		"[papayawhip::][brackets[]: kept",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("colored yaml missing %q in %q", want, got)
		}
	}
}

func TestColorizeYAMLWithoutColorsEscapesMarkup(t *testing.T) {
	input := "name: redis\n[raw]: value"
	got := colorizeYAML(input, false)
	if got != "name: redis\n[raw]: value" {
		t.Fatalf("colorizeYAML without colors = %q", got)
	}
}

func TestColorizeYAMLSearchRegionsMatchK9sBehavior(t *testing.T) {
	input := "name: <<<\"search_0\">>>redis<<<\"\">>>"
	got := colorizeYAML(input, true)
	want := "[steelblue::b]name[white::-]: [papayawhip::][\"search_0\"]redis[\"\"]"
	if got != want {
		t.Fatalf("colorizeYAML search regions = %q, want %q", got, want)
	}
}

func TestTemplateOverviewHintIncludesResourcesAndAudit(t *testing.T) {
	hint := templateOverviewHint(client.Template{ID: "t1", Name: "aws_redis"})
	for _, want := range []string{"y yaml", "l logs", "a audit", "r resources", "t tree view"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("template overview hint missing %q in %q", want, hint)
		}
	}
}

func TestTemplateResourceJumpOptions(t *testing.T) {
	options := templateResourceJumpOptions([]client.Resource{{ID: "r1", Name: "redis-prod", State: "provisioned", Status: "ready"}})
	if len(options) != 1 {
		t.Fatalf("options len = %d", len(options))
	}
	if options[0].Label != "redis-prod" || options[0].Description != "ready" {
		t.Fatalf("option = %#v", options[0])
	}
}

func TestIntegrationOverviewHintIncludesResourcesAndAudit(t *testing.T) {
	hint := integrationOverviewHint(client.Integration{ID: "i1", Name: "aws-prod"})
	for _, want := range []string{"y yaml", "l logs", "a audit", "r resources"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("integration overview hint missing %q in %q", want, hint)
		}
	}
}
