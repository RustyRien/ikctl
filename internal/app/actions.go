package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

var resourceActionDescriptions = map[string]string{
	"approve":                "Approve the pending resource changes.",
	"reject":                 "Reject the pending resource changes.",
	"destroy":                "Permanently destroy the resource infrastructure and data.",
	"delete":                 "Delete the resource record.",
	"disable":                "Disable the resource.",
	"enable":                 "Enable the resource.",
	"execute":                "Apply the pending resource changes.",
	"retry":                  "Retry the last failed or interrupted execution.",
	"recreate":               "Recreate the resource.",
	"sync":                   "Synchronize the resource state from the backend.",
	"dryrun":                 "Create an execution plan without applying changes.",
	"dryrun_with_temp_state": "Create a plan using the temporary state.",
	"cascade_destroy":        "Create a cascade destroy workflow for the resource tree.",
}

var resourceActionPriority = map[string]int{
	"execute":                10,
	"dryrun":                 20,
	"dryrun_with_temp_state": 30,
	"retry":                  40,
	"sync":                   50,
	"recreate":               60,
	"enable":                 70,
	"disable":                80,
	"destroy":                90,
	"cascade_destroy":        100,
	"delete":                 110,
	"approve":                120,
	"reject":                 130,
}

var entityActionDescriptions = map[string]map[string]string{
	"resource": resourceActionDescriptions,
	"template": {
		"enable":  "Enable the template.",
		"disable": "Disable the template.",
		"delete":  "Delete the template record.",
	},
	"source_code": {
		"enable":  "Enable the source code repository.",
		"disable": "Disable the source code repository.",
		"sync":    "Synchronize the source code repository with the remote provider.",
		"delete":  "Delete the source code repository record.",
	},
	"source_code_version": {
		"enable":  "Enable the source code version.",
		"disable": "Disable the source code version.",
		"sync":    "Synchronize the source code version.",
		"delete":  "Delete the source code version record.",
	},
	"integration": {
		"enable":  "Enable the integration.",
		"disable": "Disable the integration.",
		"delete":  "Delete the integration record.",
	},
	"storage": {},
	"worker":  {},
}

var entityActionPriority = map[string]map[string]int{
	"resource":            resourceActionPriority,
	"template":            {"enable": 10, "disable": 20, "delete": 30},
	"source_code":         {"enable": 10, "disable": 20, "sync": 30, "delete": 40},
	"source_code_version": {"enable": 10, "disable": 20, "sync": 30, "delete": 40},
	"integration":         {"enable": 10, "disable": 20, "delete": 30},
	"storage":             {},
	"worker":              {},
}

var entityActionKindLabel = map[string]string{
	"resource":            "resource",
	"template":            "template",
	"source_code":         "source code",
	"source_code_version": "source code version",
	"integration":         "integration",
	"storage":             "storage",
	"worker":              "worker",
}

func titleCase(value string) string {
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.TrimSpace(value)
	runes := []rune(value)
	first := runes[0]
	if first >= 'a' && first <= 'z' {
		runes[0] = first - ('a' - 'A')
	}
	return string(runes)
}

func actionLabel(value string) string {
	return titleCase(strings.ReplaceAll(value, "_", " "))
}

func (a *App) openEntityActionPrompt(row tabledata.Row, verb string) {
	prompt, ok := a.entityActionPromptForRow(row, verb)
	if !ok {
		a.ui.OpenOverlayPrimitive("Action", errorView(fmt.Sprintf("%s is not supported for the selected item", actionLabel(verb))))
		return
	}
	a.pendingEntityAction = prompt
	a.ui.OpenOverlayPrimitive("Confirm Action", entityActionPromptView(prompt))
}

func (a *App) entityActionPromptForRow(row tabledata.Row, verb string) (*entityActionPrompt, bool) {
	switch value := row.Raw.(type) {
	case client.Template:
		return a.templateActionPrompt(value, verb)
	case client.SourceCode:
		return a.sourceCodeActionPrompt(value, verb)
	case client.SourceCodeVersion:
		return a.sourceCodeVersionActionPrompt(value, verb)
	case client.Integration:
		return a.integrationActionPrompt(value, verb)
	case client.Resource:
		return a.resourceActionPrompt(value, verb)
	default:
		return nil, false
	}
}

func (a *App) openEntityActionMenu(row tabledata.Row) {
	switch value := row.Raw.(type) {
	case client.Resource:
		a.openTypedEntityActionMenu("resource", value.ID, value.Name, func(ctx context.Context, id string) ([]string, error) {
			full, err := a.client.Resource(ctx, id)
			if err != nil || full == nil {
				return nil, err
			}
			actions, err := a.client.ResourceActions(ctx, full.ID)
			if err == nil {
				full.Actions = actions
			}
			return full.Actions, err
		}, func(action string) (*entityActionPrompt, bool) {
			return a.resourceActionPrompt(value, action)
		})
	case client.Template:
		a.openTypedEntityActionMenu("template", value.ID, value.Name, func(context.Context, string) ([]string, error) {
			return []string{"enable", "disable", "delete"}, nil
		}, func(action string) (*entityActionPrompt, bool) {
			return a.templateActionPrompt(value, action)
		})
	case client.SourceCode:
		a.openTypedEntityActionMenu("source_code", value.ID, valueOr(value.DisplayName(), value.ID), func(context.Context, string) ([]string, error) {
			return []string{"enable", "disable", "sync", "delete"}, nil
		}, func(action string) (*entityActionPrompt, bool) {
			return a.sourceCodeActionPrompt(value, action)
		})
	case client.SourceCodeVersion:
		a.openTypedEntityActionMenu("source_code_version", value.ID, valueOr(value.GetName(), value.ID), func(context.Context, string) ([]string, error) {
			return []string{"enable", "disable", "sync", "delete"}, nil
		}, func(action string) (*entityActionPrompt, bool) {
			return a.sourceCodeVersionActionPrompt(value, action)
		})
	case client.Integration:
		a.openTypedEntityActionMenu("integration", value.ID, value.Name, func(context.Context, string) ([]string, error) {
			return []string{"enable", "disable", "delete"}, nil
		}, func(action string) (*entityActionPrompt, bool) {
			return a.integrationActionPrompt(value, action)
		})
	case client.Storage:
		a.ui.OpenOverlayPrimitive("Actions", errorView("Actions are not supported for storages"))
	case client.Worker:
		a.ui.OpenOverlayPrimitive("Actions", errorView("Actions are not supported for workers"))
	default:
		a.ui.OpenOverlayPrimitive("Actions", errorView("Actions are not supported for the selected item"))
	}
}

func (a *App) openTypedEntityActionMenu(kind string, id string, name string, loadActions func(context.Context, string) ([]string, error), promptForAction func(string) (*entityActionPrompt, bool)) {
	a.overviewJumpSelector = nil
	title := fmt.Sprintf("%s Actions", titleCase(kind))
	a.ui.OpenOverlay(title, fmt.Sprintf("Loading %s actions...", kind))

	go func(kind string, id string, name string) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		actions, err := loadActions(ctx, id)
		if err != nil {
			a.ui.Application().QueueUpdateDraw(func() {
				a.ui.OpenOverlayPrimitive(title, errorView(fmt.Sprintf("Failed to load %s actions.\n\n%v", kind, err)))
			})
			return
		}

		primitive, selector, openErr := a.entityActionSelectionView(kind, name, id, actions, promptForAction)
		a.ui.Application().QueueUpdateDraw(func() {
			if openErr != nil {
				a.ui.OpenOverlayPrimitive(title, errorView(openErr.Error()))
				return
			}
			a.overviewJumpSelector = selector
			a.ui.OpenOverlayPrimitive(title, primitive)
		})
	}(kind, id, name)
}

func (a *App) entityActionSelectionView(kind string, name string, id string, actions []string, promptForAction func(string) (*entityActionPrompt, bool)) (tview.Primitive, *overviewJumpSelector, error) {
	actions = entityMenuActions(kind, actions)
	if len(actions) == 0 {
		return nil, nil, fmt.Errorf("No %s actions available for %s", kind, valueOr(name, id))
	}

	options := make([]overviewJumpOption, 0, len(actions))
	for _, action := range actions {
		options = append(options, overviewJumpOption{
			Label:       actionLabel(action),
			Description: entityActionDescription(kind, action),
			Value:       action,
		})
	}

	primitive, table := overviewJumpSelectionView(options)
	selector := &overviewJumpSelector{
		title:   fmt.Sprintf("%s Actions", titleCase(kind)),
		options: options,
		table:   table,
		onSelect: func(option overviewJumpOption) {
			action, _ := option.Value.(string)
			if action == "" {
				return
			}
			prompt, ok := promptForAction(action)
			if !ok || prompt == nil {
				a.ui.OpenOverlayPrimitive("Action", errorView(fmt.Sprintf("%s is not supported for the selected item", actionLabel(action))))
				return
			}
			a.pendingEntityAction = prompt
			a.ui.OpenOverlayPrimitive("Confirm Action", entityActionPromptView(prompt))
		},
	}
	return primitive, selector, nil
}

func entityMenuActions(kind string, actions []string) []string {
	filtered := make([]string, 0, len(actions))
	for _, action := range actions {
		normalized := strings.TrimSpace(strings.ToLower(action))
		if normalized == "" || normalized == "edit" || (kind == "resource" && normalized == "has_temporary_state") {
			continue
		}
		filtered = append(filtered, normalized)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left, right := filtered[i], filtered[j]
		priorities := entityActionPriority[kind]
		lp, lok := priorities[left]
		rp, rok := priorities[right]
		switch {
		case lok && rok && lp != rp:
			return lp < rp
		case lok != rok:
			return lok
		default:
			return left < right
		}
	})
	return filtered
}

func resourceMenuActions(actions []string) []string {
	return entityMenuActions("resource", actions)
}

func entityActionDescription(kind string, action string) string {
	if descriptions := entityActionDescriptions[kind]; descriptions != nil {
		if description, ok := descriptions[action]; ok {
			return description
		}
	}
	if kind == "resource" {
		return resourceActionDescription(action)
	}
	return "Send this action request."
}

func resourceActionDescription(action string) string {
	if description, ok := resourceActionDescriptions[action]; ok {
		return description
	}
	return "Send this resource action request."
}

func (a *App) resourceActionPrompt(value client.Resource, verb string) (*entityActionPrompt, bool) {
	var action func(context.Context) (string, error)
	switch verb {
	case "delete":
		action = func(ctx context.Context) (string, error) { return "deleted", a.client.DeleteResource(ctx, value.ID) }
	default:
		action = func(ctx context.Context) (string, error) {
			return "updated", a.client.ResourceAction(ctx, value.ID, verb)
		}
	}
	return &entityActionPrompt{Verb: verb, Kind: "resource", ID: value.ID, Name: value.Name, Action: action}, true
}

func (a *App) templateActionPrompt(value client.Template, verb string) (*entityActionPrompt, bool) {
	var action func(context.Context) (string, error)
	switch verb {
	case "enable":
		action = func(ctx context.Context) (string, error) { return "updated", a.client.EnableTemplate(ctx, value.ID) }
	case "disable":
		action = func(ctx context.Context) (string, error) { return "updated", a.client.DisableTemplate(ctx, value.ID) }
	case "delete":
		action = func(ctx context.Context) (string, error) { return "deleted", a.client.DeleteTemplate(ctx, value.ID) }
	default:
		return nil, false
	}
	return &entityActionPrompt{Verb: verb, Kind: "template", ID: value.ID, Name: value.Name, Action: action}, true
}

func (a *App) sourceCodeActionPrompt(value client.SourceCode, verb string) (*entityActionPrompt, bool) {
	var action func(context.Context) (string, error)
	switch verb {
	case "enable":
		action = func(ctx context.Context) (string, error) { return "updated", a.client.EnableSourceCode(ctx, value.ID) }
	case "disable":
		action = func(ctx context.Context) (string, error) { return "updated", a.client.DisableSourceCode(ctx, value.ID) }
	case "sync":
		action = func(ctx context.Context) (string, error) { return "synced", a.client.SyncSourceCode(ctx, value.ID) }
	case "delete":
		action = func(ctx context.Context) (string, error) { return "deleted", a.client.DeleteSourceCode(ctx, value.ID) }
	default:
		return nil, false
	}
	return &entityActionPrompt{Verb: verb, Kind: "source_code", ID: value.ID, Name: valueOr(value.DisplayName(), value.ID), Action: action}, true
}

func (a *App) sourceCodeVersionActionPrompt(value client.SourceCodeVersion, verb string) (*entityActionPrompt, bool) {
	var action func(context.Context) (string, error)
	switch verb {
	case "enable":
		action = func(ctx context.Context) (string, error) {
			return "updated", a.client.EnableSourceCodeVersion(ctx, value.ID)
		}
	case "disable":
		action = func(ctx context.Context) (string, error) {
			return "updated", a.client.DisableSourceCodeVersion(ctx, value.ID)
		}
	case "sync":
		action = func(ctx context.Context) (string, error) {
			return "synced", a.client.SyncSourceCodeVersion(ctx, value.ID)
		}
	case "delete":
		action = func(ctx context.Context) (string, error) {
			return "deleted", a.client.DeleteSourceCodeVersion(ctx, value.ID)
		}
	default:
		return nil, false
	}
	return &entityActionPrompt{Verb: verb, Kind: "source_code_version", ID: value.ID, Name: valueOr(value.GetName(), value.ID), Action: action}, true
}

func (a *App) integrationActionPrompt(value client.Integration, verb string) (*entityActionPrompt, bool) {
	var action func(context.Context) (string, error)
	switch verb {
	case "enable":
		action = func(ctx context.Context) (string, error) { return "updated", a.client.EnableIntegration(ctx, value.ID) }
	case "disable":
		action = func(ctx context.Context) (string, error) {
			return "updated", a.client.DisableIntegration(ctx, value.ID)
		}
	case "delete":
		action = func(ctx context.Context) (string, error) { return "deleted", a.client.DeleteIntegration(ctx, value.ID) }
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
	title := fmt.Sprintf("%s %s", actionLabel(prompt.Verb), titleCase(prompt.Kind))
	kindLabel := entityActionKindTitle(prompt.Kind)
	a.ui.OpenOverlay(title, fmt.Sprintf("Sending %s request for %s %s (%s)...", prompt.Verb, kindLabel, valueOr(prompt.Name, prompt.ID), prompt.ID))

	go func(prompt *entityActionPrompt) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		resultVerb, err := prompt.Action(ctx)
		if resultVerb == "" {
			resultVerb = prompt.Verb
		}
		a.ui.Application().QueueUpdateDraw(func() {
			if err != nil {
				a.ui.OpenOverlayPrimitive(title, errorView(fmt.Sprintf("Failed to %s %s %s (%s).\n\n%v", prompt.Verb, kindLabel, valueOr(prompt.Name, prompt.ID), prompt.ID, err)))
				return
			}
			a.ui.OpenOverlayPrimitive(title, errorView(fmt.Sprintf("%s request sent for %s %s (%s).\n\nRefreshing list...\n\nPress Esc or q to close", actionLabel(resultVerb), kindLabel, valueOr(prompt.Name, prompt.ID), prompt.ID)))
			a.requestRefresh()
		})
	}(prompt)
}

func entityActionPromptView(prompt *entityActionPrompt) tview.Primitive {
	message := fmt.Sprintf("%s %s %s (%s)?\n\nEnter/y confirm\nEsc/q cancel", actionLabel(prompt.Verb), entityActionKindTitle(prompt.Kind), valueOr(prompt.Name, prompt.ID), prompt.ID)
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetScrollable(true)
	view.SetWrap(true)
	view.SetText(message)
	return view
}

func entityActionKindTitle(kind string) string {
	if label, ok := entityActionKindLabel[kind]; ok {
		return label
	}
	return strings.ToLower(titleCase(kind))
}
