package app

import (
	"context"
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
		{name: "template disable", mk: func() (*entityActionPrompt, bool) { return a.templateActionPrompt(client.Template{ID: "t1", Name: "tpl"}, "disable") }},
		{name: "integration enable", mk: func() (*entityActionPrompt, bool) { return a.integrationActionPrompt(client.Integration{ID: "i1", Name: "aws"}, "enable") }},
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
