package app

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	editcore "github.com/electrolux-oss/ik-tui/internal/edit"
	"github.com/electrolux-oss/ik-tui/internal/printer"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

func (a *App) openResourceReview(row tabledata.Row) {
	resource, ok := row.Raw.(client.Resource)
	if !ok {
		return
	}
	a.resourceReview = &resourceReviewState{Resource: resource, Loading: true}
	a.ui.OpenOverlayPrimitive("Review Resource Changes", resourceReviewView(a.resourceReview, !a.config.NoColors))

	go func(resource client.Resource) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Resource(ctx, resource.ID)
		if err != nil {
			a.ui.Application().QueueUpdateDraw(func() {
				if a.resourceReview == nil || a.resourceReview.Resource.ID != resource.ID {
					return
				}
				a.resourceReview.Loading = false
				a.resourceReview.LoadErr = err
				a.ui.OpenOverlayPrimitive("Review Resource Changes", resourceReviewView(a.resourceReview, !a.config.NoColors))
			})
			return
		}
		if full != nil {
			resource = *full
		}

		actions, actionsErr := a.client.ResourceActions(ctx, resource.ID)
		if actionsErr != nil {
			err = actionsErr
		}

		var tempState *client.ResourceTempState
		if err == nil && resource.Status != "approval_pending" {
			tempState, err = a.client.ResourceTempState(ctx, resource.ID)
		}

		diffText, diffHasValue, diffErr := buildResourceReviewDiff(resource, tempState)
		if err == nil && diffErr != nil {
			err = diffErr
		}

		infoMessage := ""
		if resource.Status == "approval_pending" {
			infoMessage = "Resource is pending approval."
		} else if tempState == nil {
			infoMessage = "No temporary state found for this resource."
		}

		a.ui.Application().QueueUpdateDraw(func() {
			if a.resourceReview == nil || a.resourceReview.Resource.ID != resource.ID {
				return
			}
			a.resourceReview.Resource = resource
			a.resourceReview.Actions = actions
			a.resourceReview.TempState = tempState
			a.resourceReview.DiffText = diffText
			a.resourceReview.DiffHasValue = diffHasValue
			a.resourceReview.InfoMessage = infoMessage
			a.resourceReview.LoadErr = err
			a.resourceReview.Loading = false
			a.ui.OpenOverlayPrimitive("Review Resource Changes", resourceReviewView(a.resourceReview, !a.config.NoColors))
		})
	}(resource)
}

func buildResourceReviewDiff(resource client.Resource, tempState *client.ResourceTempState) (string, bool, error) {
	if tempState == nil || len(tempState.Value) == 0 {
		return "", false, nil
	}

	current := filterCurrentResourceState(resource, tempState.Value)
	left, err := yamlFromValue(current)
	if err != nil {
		return "", false, err
	}
	right, err := yamlFromValue(tempState.Value)
	if err != nil {
		return "", false, err
	}
	return renderUnifiedDiff("Current State", left, "Temporary State", right), true, nil
}

func filterCurrentResourceState(resource client.Resource, temp map[string]any) map[string]any {
	current := editcore.ResourceEditableState(resource)
	filtered := make(map[string]any, len(temp))
	for key := range temp {
		switch key {
		case "storage_id":
			filtered[key] = current[key]
		case "integration_ids", "secret_ids", "workspace_id", "source_code_version_id", "name", "description", "storage_path", "variables", "dependency_tags", "dependency_config", "labels":
			filtered[key] = current[key]
		default:
			if value, ok := current[key]; ok {
				filtered[key] = value
			}
		}
	}
	return filtered
}

func yamlFromValue(value any) (string, error) {
	var buf bytes.Buffer
	if err := printer.Print(&buf, "yaml", nil, nil, []any{value}); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func renderUnifiedDiff(leftTitle string, left string, rightTitle string, right string) string {
	leftLines := splitDiffLines(left)
	rightLines := splitDiffLines(right)
	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}

	var builder strings.Builder
	builder.WriteString("--- ")
	builder.WriteString(leftTitle)
	builder.WriteString("\n+++ ")
	builder.WriteString(rightTitle)
	builder.WriteString("\n@@\n")
	for i := 0; i < maxLen; i++ {
		var l, r string
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		switch {
		case l == r:
			builder.WriteString("  ")
			builder.WriteString(l)
		case l == "":
			builder.WriteString("+")
			builder.WriteString(r)
		case r == "":
			builder.WriteString("-")
			builder.WriteString(l)
		default:
			builder.WriteString("-")
			builder.WriteString(l)
			builder.WriteString("\n+")
			builder.WriteString(r)
		}
		builder.WriteString("\n")
	}
	return strings.TrimRight(builder.String(), "\n")
}

func splitDiffLines(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

func resourceReviewView(state *resourceReviewState, colorsEnabled bool) tview.Primitive {
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	text := tview.NewTextView()
	text.SetDynamicColors(true)
	text.SetScrollable(true)
	text.SetWrap(false)
	text.SetWordWrap(false)
	text.SetBorder(true)
	text.SetTitle("State Diff")

	var body strings.Builder
	body.WriteString(fmt.Sprintf("Resource: %s (%s)\nStatus: %s\n", valueOr(state.Resource.Name, state.Resource.ID), state.Resource.ID, blankDash(state.Resource.Status)))
	if len(state.Actions) > 0 {
		body.WriteString("Actions: ")
		body.WriteString(strings.Join(state.Actions, ", "))
		body.WriteString("\n")
	}
	if state.Loading {
		body.WriteString("\nLoading review data...")
	} else if state.LoadErr != nil {
		body.WriteString("\n[red]Failed to load review data[-]\n\n")
		body.WriteString(state.LoadErr.Error())
	} else {
		if state.InfoMessage != "" {
			body.WriteString("\n")
			body.WriteString(state.InfoMessage)
			body.WriteString("\n")
		}
		if state.ActionErr != nil {
			body.WriteString("\n[red]Action failed[-]\n")
			body.WriteString(state.ActionErr.Error())
			body.WriteString("\n")
		}
		if state.DiffHasValue {
			body.WriteString("\n")
			body.WriteString(colorizeReviewDiff(state.DiffText, colorsEnabled))
		} else if state.TempState == nil {
			body.WriteString("\nNo temporary state diff available.")
		}
	}

	text.SetText(body.String())
	footer := "Esc/q close"
	if hasAction(state.Actions, "approve") {
		footer = "a approve  r reject  " + footer
	}
	root.AddItem(text, 0, 1, true)
	root.AddItem(overviewFooter(footer), 1, 0, false)
	return root
}

func (a *App) performResourceReviewAction(action string) {
	if a.resourceReview == nil || a.resourceReview.Loading {
		return
	}
	if action != "approve" && action != "reject" {
		return
	}
	if !hasAction(a.resourceReview.Actions, "approve") {
		return
	}

	state := a.resourceReview
	state.ActionErr = nil
	state.InfoMessage = ""
	if action == "approve" {
		state.Approving = true
	} else {
		state.Rejecting = true
	}
	a.ui.OpenOverlayPrimitive("Review Resource Changes", resourceReviewView(state, !a.config.NoColors))

	go func(resourceID string, action string) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		var err error
		switch action {
		case "approve":
			err = a.client.ApproveResource(ctx, resourceID)
		case "reject":
			err = a.client.RejectResource(ctx, resourceID)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			if a.resourceReview == nil || a.resourceReview.Resource.ID != resourceID {
				return
			}
			a.resourceReview.Approving = false
			a.resourceReview.Rejecting = false
			if err != nil {
				a.resourceReview.ActionErr = err
				a.ui.OpenOverlayPrimitive("Review Resource Changes", resourceReviewView(a.resourceReview, !a.config.NoColors))
				return
			}
			a.resourceReview = nil
			a.ui.CloseOverlay()
			a.requestRefresh()
			a.openResourceOverview(client.Resource{ID: resourceID, Name: valueOr(state.Resource.Name, resourceID)})
		})
	}(state.Resource.ID, action)
}

func colorizeReviewDiff(text string, colorsEnabled bool) string {
	if text == "" || !colorsEnabled {
		return text
	}

	lines := strings.Split(tview.Escape(text), "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++"):
			lines[i] = "[deepskyblue::b]" + line + "[-:-:-]"
		case strings.HasPrefix(line, "---"):
			lines[i] = "[deepskyblue::b]" + line + "[-:-:-]"
		case strings.HasPrefix(line, "@@"):
			lines[i] = "[mediumpurple::b]" + line + "[-:-:-]"
		case strings.HasPrefix(line, "+"):
			lines[i] = "[green::b]" + line + "[-:-:-]"
		case strings.HasPrefix(line, "-"):
			lines[i] = "[red::b]" + line + "[-:-:-]"
		default:
			lines[i] = line
		}
	}
	return strings.Join(lines, "\n")
}

func (a *App) clearResourceReview() {
	a.resourceReview = nil
}
