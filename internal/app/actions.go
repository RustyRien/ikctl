package app

import (
	"context"
	"fmt"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

func titleCase(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	first := runes[0]
	if first >= 'a' && first <= 'z' {
		runes[0] = first - ('a' - 'A')
	}
	return string(runes)
}

func (a *App) openEntityActionPrompt(row tabledata.Row, verb string) {
	prompt, ok := a.entityActionPromptForRow(row, verb)
	if !ok {
		a.ui.OpenOverlayPrimitive("Action", errorView(fmt.Sprintf("%s is not supported for the selected item", titleCase(verb))))
		return
	}
	a.pendingEntityAction = prompt
	a.ui.OpenOverlayPrimitive("Confirm Action", entityActionPromptView(prompt))
}

func (a *App) entityActionPromptForRow(row tabledata.Row, verb string) (*entityActionPrompt, bool) {
	switch value := row.Raw.(type) {
	case client.Template:
		return a.templateActionPrompt(value, verb)
	case client.Integration:
		return a.integrationActionPrompt(value, verb)
	default:
		return nil, false
	}
}

func (a *App) templateActionPrompt(value client.Template, verb string) (*entityActionPrompt, bool) {
	var action func(context.Context) error
	switch verb {
	case "enable":
		action = func(ctx context.Context) error { return a.client.EnableTemplate(ctx, value.ID) }
	case "disable":
		action = func(ctx context.Context) error { return a.client.DisableTemplate(ctx, value.ID) }
	case "delete":
		action = func(ctx context.Context) error { return a.client.DeleteTemplate(ctx, value.ID) }
	default:
		return nil, false
	}
	return &entityActionPrompt{Verb: verb, Kind: "template", ID: value.ID, Name: value.Name, Action: action}, true
}

func (a *App) integrationActionPrompt(value client.Integration, verb string) (*entityActionPrompt, bool) {
	var action func(context.Context) error
	switch verb {
	case "enable":
		action = func(ctx context.Context) error { return a.client.EnableIntegration(ctx, value.ID) }
	case "disable":
		action = func(ctx context.Context) error { return a.client.DisableIntegration(ctx, value.ID) }
	case "delete":
		action = func(ctx context.Context) error { return a.client.DeleteIntegration(ctx, value.ID) }
	default:
		return nil, false
	}
	return &entityActionPrompt{Verb: verb, Kind: "integration", ID: value.ID, Name: value.Name, Action: action}, true
}

func (a *App) confirmPendingEntityAction() {
	prompt := a.pendingEntityAction
	if prompt == nil || prompt.Action == nil {
		return
	}
	a.pendingEntityAction = nil
	title := fmt.Sprintf("%s %s", titleCase(prompt.Verb), titleCase(prompt.Kind))
	a.ui.OpenOverlay(title, fmt.Sprintf("Sending %s request for %s %s (%s)...", prompt.Verb, prompt.Kind, valueOr(prompt.Name, prompt.ID), prompt.ID))

	go func(prompt *entityActionPrompt) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		err := prompt.Action(ctx)
		a.ui.Application().QueueUpdateDraw(func() {
			if err != nil {
				a.ui.OpenOverlayPrimitive(title, errorView(fmt.Sprintf("Failed to %s %s %s (%s).\n\n%v", prompt.Verb, prompt.Kind, valueOr(prompt.Name, prompt.ID), prompt.ID, err)))
				return
			}
			a.ui.OpenOverlayPrimitive(title, errorView(fmt.Sprintf("%s request sent for %s %s (%s).\n\nRefreshing list...\n\nPress Esc or q to close", titleCase(prompt.Verb), prompt.Kind, valueOr(prompt.Name, prompt.ID), prompt.ID)))
			a.requestRefresh()
		})
	}(prompt)
}

func entityActionPromptView(prompt *entityActionPrompt) tview.Primitive {
	message := fmt.Sprintf("%s %s %s (%s)?\n\nEnter/y confirm\nEsc/q cancel", titleCase(prompt.Verb), prompt.Kind, valueOr(prompt.Name, prompt.ID), prompt.ID)
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetScrollable(true)
	view.SetWrap(true)
	view.SetText(message)
	return view
}
