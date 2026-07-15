package app

import (
	"context"
	"strings"
	"testing"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

func TestEntityActionPromptForRowSupportsEntities(t *testing.T) {
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
	if !ok || resourcePrompt == nil {
		t.Fatal("expected resource disable prompt")
	}
	if resourcePrompt.Kind != "resource" || resourcePrompt.Verb != "disable" || resourcePrompt.ID != "r1" {
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
		{name: "resource execute", mk: func() (*entityActionPrompt, bool) {
			return a.resourceActionPrompt(client.Resource{ID: "r1", Name: "redis"}, "execute")
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

func TestResourceMenuActionsFiltersAndSorts(t *testing.T) {
	got := resourceMenuActions([]string{" delete ", "has_temporary_state", "retry", "edit", "execute"})
	want := []string{"execute", "retry", "delete"}
	if len(got) != len(want) {
		t.Fatalf("len(actions) = %d want %d (%#v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("actions[%d] = %q want %q (%#v)", i, got[i], want[i], got)
		}
	}
}

func TestEntityMenuActionsFiltersEditForTemplatesAndIntegrations(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind string
		in   []string
		want []string
	}{
		{name: "template", kind: "template", in: []string{"enable", "edit", "delete"}, want: []string{"enable", "delete"}},
		{name: "integration", kind: "integration", in: []string{"disable", "edit", "delete"}, want: []string{"disable", "delete"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := entityMenuActions(tc.kind, tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len(actions) = %d want %d (%#v)", len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("actions[%d] = %q want %q (%#v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

func TestResourceActionDescriptionUsesFrontendStyleMappings(t *testing.T) {
	if got := resourceActionDescription("cascade_destroy"); !strings.Contains(got, "cascade destroy workflow") {
		t.Fatalf("description = %q", got)
	}
	if got := resourceActionDescription("unknown_action"); got != "Send this resource action request." {
		t.Fatalf("fallback description = %q", got)
	}
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
	for _, want := range []string{"y yaml", "A actions", "l logs", "a audit", "r resources", "t tree view", "E edit"} {
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
	for _, want := range []string{"y yaml", "A actions", "l logs", "a audit", "r resources", "E edit"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("integration overview hint missing %q in %q", want, hint)
		}
	}
}

func TestResourceTemplateHintIncludesEdit(t *testing.T) {
	hint := resourceTemplateHint(client.Resource{ID: "r1", Name: "redis", Template: &client.Template{ID: "t1"}})
	for _, want := range []string{"y yaml", "E edit", "t template", "T tree"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("resource hint missing %q in %q", want, hint)
		}
	}
}

func TestResourceTemplateHintIncludesDeleteWhenAvailable(t *testing.T) {
	hint := resourceTemplateHint(client.Resource{ID: "r1", Name: "redis", Actions: []string{"delete"}})
	if !strings.Contains(hint, "D delete") {
		t.Fatalf("resource hint missing delete in %q", hint)
	}
}

func TestResourceTemplateHintIncludesActionsWhenAvailable(t *testing.T) {
	hint := resourceTemplateHint(client.Resource{ID: "r1", Name: "redis", Actions: []string{"execute"}})
	if !strings.Contains(hint, "A actions") {
		t.Fatalf("resource hint missing actions in %q", hint)
	}
}

func TestResourceTemplateHintIncludesReviewWhenApprovalAvailable(t *testing.T) {
	hint := resourceTemplateHint(client.Resource{ID: "r1", Name: "redis", Actions: []string{"approve"}})
	if !strings.Contains(hint, "R review") {
		t.Fatalf("resource hint missing review in %q", hint)
	}
}

func TestBuildResourceReviewDiffUsesTemporaryStateSubset(t *testing.T) {
	resource := client.Resource{
		ID:               "r1",
		Name:             "redis-prod",
		Description:      "Managed Redis",
		Labels:           []string{"prod"},
		Variables:        []map[string]any{{"name": "size", "value": "small"}},
		DependencyTags:   []map[string]any{{"name": "env", "value": "prod"}},
		DependencyConfig: []map[string]any{{"name": "region", "value": "eu-west-1"}},
		Integrations:     []client.Integration{{ID: "i1"}},
		Secrets:          []client.Secret{{ID: "s1"}},
		Storage:          &client.Storage{ID: "st1"},
		Workspace:        &client.Workspace{ID: "w1"},
	}
	tempState := &client.ResourceTempState{Value: map[string]any{
		"description":     "Managed Redis updated",
		"integration_ids": []any{"i1"},
		"variables":       []any{map[string]any{"name": "size", "value": "large"}},
	}}
	diff, hasValue, err := buildResourceReviewDiff(resource, tempState)
	if err != nil {
		t.Fatalf("buildResourceReviewDiff: %v", err)
	}
	if !hasValue {
		t.Fatal("expected diff content")
	}
	for _, want := range []string{"-description: Managed Redis", "+description: Managed Redis updated", "-      value: small", "+      value: large"} {
		if !strings.Contains(diff, want) {
			t.Fatalf("diff missing %q in %q", want, diff)
		}
	}
	for _, want := range []string{"--- Current State", "+++ Temporary State", "@@"} {
		if !strings.Contains(diff, want) {
			t.Fatalf("diff header missing %q in %q", want, diff)
		}
	}
}

func TestResourceReviewNoticeViewShowsApprovalPendingMessage(t *testing.T) {
	view := resourceReviewNoticeView(client.Resource{Status: "approval_pending"})
	if view == nil {
		t.Fatal("expected review notice view")
	}
	textView, ok := view.(*tview.TextView)
	if !ok {
		t.Fatalf("expected text view, got %T", view)
	}
	if text := textView.GetText(false); !strings.Contains(text, "Approval pending") || !strings.Contains(text, "Press [white::b]R") {
		t.Fatalf("unexpected notice text %q", text)
	}
}

func TestResourceReviewNoticeViewNilWithoutReview(t *testing.T) {
	if view := resourceReviewNoticeView(client.Resource{}); view != nil {
		t.Fatalf("expected nil notice, got %T", view)
	}
}

func TestColorizeReviewDiffColorsGitStyleLines(t *testing.T) {
	input := "--- Current State\n+++ Temporary State\n@@\n unchanged\n-old\n+new"
	got := colorizeReviewDiff(input, true)
	for _, want := range []string{
		"[deepskyblue::b]--- Current State[-:-:-]",
		"[deepskyblue::b]+++ Temporary State[-:-:-]",
		"[mediumpurple::b]@@[-:-:-]",
		"[red::b]-old[-:-:-]",
		"[green::b]+new[-:-:-]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("colored diff missing %q in %q", want, got)
		}
	}
}

func TestColorizeReviewDiffDisabledKeepsPlainText(t *testing.T) {
	input := "+new\n-old"
	if got := colorizeReviewDiff(input, false); got != input {
		t.Fatalf("plain diff = %q", got)
	}
}
