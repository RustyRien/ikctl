package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/rivo/tview"
)

func (a *App) openSourceCodeOverview(id string, name string) {
	a.stopLiveLogStream()

	title := fmt.Sprintf("Source Code: %s", valueOr(name, id))
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.activeSourceCodeDetail = &entityDetailSelection{ID: id, Name: name, Kind: "source_codes"}
	a.activeIntegrationDetail = nil
	a.activeStorageDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading source code overview...")
	a.ui.SetSourceCodeOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.SourceCode(ctx, id)
		var primitive tview.Primitive
		var jumpActions map[rune]func()
		if err != nil {
			a.activeSourceCodeDetail = nil
			primitive = errorView(fmt.Sprintf("Failed to load source code overview.\n\n%v", err))
		} else if full != nil {
			a.activeSourceCodeDetail = &entityDetailSelection{ID: full.ID, Name: full.DisplayName(), Kind: "source_codes"}
			primitive = sourceCodeOverviewView(*full)
			jumpActions = a.sourceCodeOverviewJumpActions(*full)
		} else {
			a.activeSourceCodeDetail = nil
			a.activeIntegrationDetail = nil
			a.activeStorageDetail = nil
			primitive = errorView("Source code not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.clearOverviewJumpState()
			for key, action := range jumpActions {
				a.setOverviewJumpAction(key, action)
			}
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetSourceCodeOverviewHotkeys()
		})
	}()
}

func (a *App) sourceCodeOverviewJumpActions(sourceCode client.SourceCode) map[rune]func() {
	if sourceCode.ID == "" {
		return nil
	}
	return map[rune]func(){
		'r': func() {
			a.openSourceCodeResources(sourceCode)
		},
	}
}

func (a *App) openSourceCodeResources(sourceCode client.SourceCode) {
	title := fmt.Sprintf("Source Code Resources: %s", valueOr(sourceCode.DisplayName(), sourceCode.ID))
	a.overviewJumpSelector = nil
	a.ui.OpenOverlay(title, "Loading source code resources...")

	go func(sourceCode client.SourceCode) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Resources(ctx, map[string]any{"source_code_id": sourceCode.ID}, []string{"updated_at", "DESC"}, []int{0, 200})
		if err != nil {
			result, err = a.client.Resources(ctx, map[string]any{"source_code_version.source_code_id": sourceCode.ID}, []string{"updated_at", "DESC"}, []int{0, 200})
		}
		var primitive tview.Primitive
		var selector *overviewJumpSelector
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load source code resources.\n\n%v", err))
		} else if len(result.Items) == 0 {
			primitive = errorView("No resources found for this source code")
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
	}(sourceCode)
}

func sourceCodeOverviewView(sourceCode client.SourceCode) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", valueOr(sourceCode.DisplayName(), "-")},
		{"ID", sourceCode.ID},
		{"Identifier", blankDash(sourceCode.Identifier)},
		{"Provider", blankDash(sourceCode.SourceCodeProvider)},
		{"Language", blankDash(sourceCode.SourceCodeLanguage)},
		{"Status", blankDash(sourceCode.Status)},
		{"Entity", blankDash(sourceCode.EntityName)},
		{"Revision", fmt.Sprintf("%d", sourceCode.RevisionNumber)},
		{"Created", sourceCode.CreatedAt.Format(time.RFC3339)},
		{"Updated", sourceCode.UpdatedAt.Format(time.RFC3339)},
	})

	repository := kvTable("Repository", [][2]string{
		{"URL", blankDash(sourceCode.SourceCodeURL)},
		{"Identifier", blankDash(sourceCode.Identifier)},
		{"Provider / Language", strings.TrimSpace(blankDash(sourceCode.SourceCodeProvider) + " / " + blankDash(sourceCode.SourceCodeLanguage))},
		{"Integration", sourceCodeIntegrationName(sourceCode.Integration)},
		{"Integration ID", blankDash(sourceCode.IntegrationID)},
		{"Creator", sourceCodeCreatorName(sourceCode.Creator)},
		{"Labels", strings.Join(orSlice(sourceCode.Labels, []string{"-"}), ", ")},
	})

	refs := kvTable("Refs", [][2]string{
		{"Tags", fmt.Sprintf("%d", len(sourceCode.GitTags))},
		{"Branches", fmt.Sprintf("%d", len(sourceCode.GitBranches))},
		{"Folders", fmt.Sprintf("%d", sourceCodeFolderCount(sourceCode.GitFoldersMap))},
	})

	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(sourceCode.Description, "-"))

	tags := simpleList("Tags", sourceCodeRefLines(sourceCode.GitTags, sourceCode.GitTagMessages))
	branches := simpleList("Branches", sourceCodeRefLines(sourceCode.GitBranches, sourceCode.GitBranchMessages))
	folders := simpleList("Folders", sourceCodeFolderLines(sourceCode.GitFoldersMap))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(repository, 0, 1, false).
		AddItem(refs, 0, 1, false), 10, 0, true)
	root.AddItem(tview.NewFlex().
		AddItem(description, 0, 2, false).
		AddItem(folders, 0, 2, false), 8, 0, false)
	root.AddItem(tview.NewFlex().
		AddItem(tags, 0, 1, false).
		AddItem(branches, 0, 1, false), 0, 1, false)
	root.AddItem(overviewFooter(sourceCodeOverviewHint(sourceCode)), 1, 0, false)
	return root
}

func sourceCodeOverviewHint(sourceCode client.SourceCode) string {
	hints := []string{"y yaml", "A actions", "l logs", "a audit", "r resources", "E edit", "Esc/q close"}
	if sourceCode.ID == "" {
		hints = []string{"y yaml", "A actions", "l logs", "a audit", "E edit", "Esc/q close"}
	}
	return strings.Join(hints, "  ")
}

func sourceCodeCreatorName(creator *client.Creator) string {
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

func sourceCodeIntegrationName(integration *client.Integration) string {
	if integration == nil || integration.Name == "" {
		return "-"
	}
	return integration.Name
}

func sourceCodeRefLines(refs []string, messages map[string]string) []string {
	lines := make([]string, 0, len(refs))
	for _, ref := range refs {
		message := strings.TrimSpace(messages[ref])
		if message == "" {
			lines = append(lines, ref)
			continue
		}
		lines = append(lines, fmt.Sprintf("%s — %s", ref, message))
	}
	return lines
}

func sourceCodeFolderLines(refFolders []client.RefFolders) []string {
	lines := make([]string, 0, len(refFolders))
	for _, refFolder := range refFolders {
		folders := strings.Join(orSlice(refFolder.Folders, []string{"-"}), ", ")
		lines = append(lines, fmt.Sprintf("%s: %s", valueOr(refFolder.Ref, "-"), folders))
	}
	return lines
}

func sourceCodeFolderCount(refFolders []client.RefFolders) int {
	count := 0
	for _, refFolder := range refFolders {
		count += len(refFolder.Folders)
	}
	return count
}
