package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/rivo/tview"
)

func (a *App) openSourceCodeVersionOverview(id string, name string) {
	a.stopLiveLogStream()

	title := fmt.Sprintf("Source Code Version: %s", valueOr(name, id))
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.activeSourceCodeDetail = nil
	a.activeSourceCodeVersionDetail = &entityDetailSelection{ID: id, Name: name, Kind: "source_code_versions"}
	a.activeIntegrationDetail = nil
	a.activeStorageDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading source code version overview...")
	a.ui.SetSourceCodeVersionOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.SourceCodeVersion(ctx, id)
		var primitive tview.Primitive
		var jumpActions map[rune]func()
		if err != nil {
			a.activeSourceCodeVersionDetail = nil
			primitive = errorView(fmt.Sprintf("Failed to load source code version overview.\n\n%v", err))
		} else if full != nil {
			a.activeSourceCodeVersionDetail = &entityDetailSelection{ID: full.ID, Name: full.GetName(), Kind: "source_code_versions"}
			primitive = sourceCodeVersionOverviewView(*full)
			jumpActions = a.sourceCodeVersionOverviewJumpActions(*full)
		} else {
			a.activeSourceCodeVersionDetail = nil
			primitive = errorView("Source code version not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.clearOverviewJumpState()
			for key, action := range jumpActions {
				a.setOverviewJumpAction(key, action)
			}
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetSourceCodeVersionOverviewHotkeys()
		})
	}()
}

func (a *App) sourceCodeVersionOverviewJumpActions(sourceCodeVersion client.SourceCodeVersion) map[rune]func() {
	if sourceCodeVersion.ID == "" {
		return nil
	}
	return map[rune]func(){
		'r': func() {
			a.openSourceCodeVersionResources(sourceCodeVersion)
		},
		't': func() {
			if sourceCodeVersion.Template != nil {
				a.openTemplateOverview(sourceCodeVersion.Template.ID, valueOr(sourceCodeVersion.Template.Name, sourceCodeVersion.Template.ID))
			}
		},
		'c': func() {
			if sourceCodeVersion.SourceCode != nil {
				a.openSourceCodeOverview(sourceCodeVersion.SourceCode.ID, valueOr(sourceCodeVersion.SourceCode.DisplayName(), sourceCodeVersion.SourceCode.ID))
			}
		},
	}
}

func (a *App) openSourceCodeVersionResources(sourceCodeVersion client.SourceCodeVersion) {
	title := fmt.Sprintf("Source Code Version Resources: %s", valueOr(sourceCodeVersion.GetName(), sourceCodeVersion.ID))
	a.overviewJumpSelector = nil
	a.ui.OpenOverlay(title, "Loading source code version resources...")

	go func(sourceCodeVersion client.SourceCodeVersion) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Resources(ctx, map[string]any{"source_code_version_id": sourceCodeVersion.ID}, []string{"updated_at", "DESC"}, []int{0, 200})
		var primitive tview.Primitive
		var selector *overviewJumpSelector
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load source code version resources.\n\n%v", err))
		} else if len(result.Items) == 0 {
			primitive = errorView("No resources found for this source code version")
		} else {
			options := templateResourceJumpOptions(result.Items)
			primitive, selector = templateResourceSelectionView(options, len(result.Items), result.Total)
			selector.onSelect = func(option overviewJumpOption) {
				resourceItem, ok := option.Value.(client.Resource)
				if !ok {
					return
				}
				a.openResourceOverview(client.Resource{ID: resourceItem.ID, Name: resourceItem.Name})
			}
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.overviewJumpSelector = selector
			a.ui.OpenOverlayPrimitive(title, primitive)
		})
	}(sourceCodeVersion)
}

func sourceCodeVersionOverviewView(sourceCodeVersion client.SourceCodeVersion) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", valueOr(sourceCodeVersion.GetName(), "-")},
		{"ID", sourceCodeVersion.ID},
		{"Status", blankDash(sourceCodeVersion.Status)},
		{"Revision", fmt.Sprintf("%d", sourceCodeVersion.RevisionNumber)},
		{"Resources", fmt.Sprintf("%d", sourceCodeVersion.ResourcesCount)},
		{"Created", sourceCodeVersion.CreatedAt.Format(time.RFC3339)},
		{"Updated", sourceCodeVersion.UpdatedAt.Format(time.RFC3339)},
		{"Creator", sourceCodeVersionCreatorName(sourceCodeVersion.Creator)},
	})

	refs := kvTable("Source", [][2]string{
		{"Template", sourceCodeVersionTemplateName(sourceCodeVersion.Template)},
		{"Source Code", appSourceCodeDisplayName(sourceCodeVersion.SourceCode)},
		{"Folder", blankDash(sourceCodeVersion.SourceCodeFolder)},
		{"Branch", blankDash(sourceCodeVersion.SourceCodeBranch)},
		{"Tag", blankDash(sourceCodeVersion.SourceCodeVersion)},
		{"Entity", blankDash(sourceCodeVersion.EntityName)},
		{"Labels", strings.Join(orSlice(sourceCodeVersion.Labels, []string{"-"}), ", ")},
	})

	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(sourceCodeVersion.Description, "-"))

	variables := mapListTable("Inputs", sourceCodeVersion.Variables)
	outputs := mapListTable("Outputs", sourceCodeVersion.Outputs)
	code := templateTextView("Code Snapshot", valueOr(strings.TrimSpace(sourceCodeVersion.CodeSnapshot), "-"), false)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(refs, 0, 1, false), 10, 0, true)
	root.AddItem(description, 5, 0, false)
	root.AddItem(tview.NewFlex().
		AddItem(variables, 0, 1, false).
		AddItem(outputs, 0, 1, false), 10, 0, false)
	root.AddItem(code, 0, 1, false)
	root.AddItem(overviewFooter(sourceCodeVersionOverviewHint(sourceCodeVersion)), 1, 0, false)
	return root
}

func sourceCodeVersionOverviewHint(sourceCodeVersion client.SourceCodeVersion) string {
	hints := []string{"y yaml", "A actions", "l logs", "a audit", "r resources", "t template", "c source code", "E edit", "Esc/q close"}
	if sourceCodeVersion.ID == "" {
		hints = []string{"y yaml", "A actions", "l logs", "a audit", "E edit", "Esc/q close"}
	}
	return strings.Join(hints, "  ")
}

func sourceCodeVersionCreatorName(creator *client.Creator) string {
	if creator == nil {
		return "-"
	}
	if creator.DisplayName != "" {
		return creator.DisplayName
	}
	if creator.Identifier != "" {
		return creator.Identifier
	}
	if creator.ID != "" {
		return creator.ID
	}
	return "-"
}

func sourceCodeVersionTemplateName(template *client.Template) string {
	if template == nil || strings.TrimSpace(template.Name) == "" {
		return "-"
	}
	return template.Name
}

func appSourceCodeDisplayName(sourceCode *client.SourceCode) string {
	if sourceCode == nil {
		return "-"
	}
	return blankDash(sourceCode.DisplayName())
}
