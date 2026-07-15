package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/rivo/tview"
)

func (a *App) openStorageOverview(id string, name string) {
	a.stopLiveLogStream()

	title := fmt.Sprintf("Storage: %s", name)
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.activeSourceCodeDetail = nil
	a.activeSourceCodeVersionDetail = nil
	a.activeIntegrationDetail = nil
	a.activeStorageDetail = &entityDetailSelection{ID: id, Name: name, Kind: "storages"}
	a.activeWorkerDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading storage overview...")
	a.ui.SetStorageOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Storage(ctx, id)
		var primitive tview.Primitive
		var jumpActions map[rune]func()
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load storage overview.\n\n%v", err))
		} else if full != nil {
			a.activeStorageDetail = &entityDetailSelection{ID: full.ID, Name: full.Name, Kind: "storages"}
			primitive = storageOverviewView(*full)
			jumpActions = a.storageOverviewJumpActions(*full)
		} else {
			a.activeStorageDetail = nil
			a.activeWorkerDetail = nil
			primitive = errorView("Storage not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.clearOverviewJumpState()
			for key, action := range jumpActions {
				a.setOverviewJumpAction(key, action)
			}
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetStorageOverviewHotkeys()
		})
	}()
}

func (a *App) storageOverviewJumpActions(storage client.Storage) map[rune]func() {
	if storage.ID == "" {
		return nil
	}
	return map[rune]func(){
		'r': func() {
			a.openStorageResources(storage)
		},
	}
}

func (a *App) openStorageResources(storage client.Storage) {
	title := fmt.Sprintf("Storage Resources: %s", valueOr(storage.Name, storage.ID))
	a.overviewJumpSelector = nil
	a.ui.OpenOverlay(title, "Loading storage resources...")

	go func(storage client.Storage) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Resources(ctx, map[string]any{"storage_id": storage.ID}, []string{"updated_at", "DESC"}, []int{0, 200})
		var primitive tview.Primitive
		var selector *overviewJumpSelector
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load storage resources.\n\n%v", err))
		} else if len(result.Items) == 0 {
			primitive = errorView("No resources found for this storage")
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
	}(storage)
}

func storageOverviewView(storage client.Storage) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", storage.Name},
		{"ID", storage.ID},
		{"Type", blankDash(storage.StorageType)},
		{"Provider", blankDash(storage.StorageProvider)},
		{"State", blankDash(storage.State)},
		{"Status", blankDash(storage.Status)},
		{"Created", storage.CreatedAt.Format(time.RFC3339)},
		{"Updated", storage.UpdatedAt.Format(time.RFC3339)},
	})

	usage := kvTable("Usage", [][2]string{
		{"Resources", fmt.Sprintf("%d", storage.ResourcesCount)},
		{"Executors", fmt.Sprintf("%d", storage.ExecutorsCount)},
		{"Revision", fmt.Sprintf("%d", storage.RevisionNumber)},
		{"Integration", storageIntegrationName(storage.Integration)},
		{"Creator", storageCreatorName(storage.Creator)},
	})

	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(storage.Description, "-"))

	configuration := tview.NewTextView()
	configuration.SetBorder(true)
	configuration.SetTitle("Configuration")
	configuration.SetWrap(true)
	configuration.SetDynamicColors(true)
	configuration.SetText(storageConfigurationText(storage))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(usage, 0, 1, false), 10, 0, true)
	root.AddItem(tview.NewFlex().
		AddItem(description, 0, 2, false).
		AddItem(configuration, 0, 3, false), 0, 1, false)
	root.AddItem(overviewFooter(storageOverviewHint(storage)), 1, 0, false)
	return root
}

func storageCreatorName(creator *client.Creator) string {
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

func storageIntegrationName(integration *client.Integration) string {
	if integration == nil || integration.Name == "" {
		return "-"
	}
	return integration.Name
}

func storageConfigurationText(storage client.Storage) string {
	lines := []string{
		"[::b]Type[::-] " + blankDash(storage.StorageType),
		"[::b]Provider[::-] " + blankDash(storage.StorageProvider),
	}
	if storage.Integration != nil && storage.Integration.Name != "" {
		lines = append(lines, "[::b]Integration[::-] "+storage.Integration.Name)
	}
	if len(storage.Labels) > 0 {
		lines = append(lines, "[::b]Labels[::-] "+strings.Join(storage.Labels, ", "))
	}
	for _, line := range mapLines(storage.Configuration) {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func mapLines(values map[string]any) []string {
	if len(values) == 0 {
		return []string{"[::b]Configuration[::-] -"}
	}
	lines := make([]string, 0, len(values))
	for key, value := range values {
		lines = append(lines, fmt.Sprintf("[::b]%s[::-] %v", key, value))
	}
	return lines
}

func storageOverviewHint(storage client.Storage) string {
	hints := []string{"y yaml", "l logs", "a audit", "r resources", "E edit", "Esc/q close"}
	if storage.ID == "" {
		hints = []string{"y yaml", "l logs", "a audit", "E edit", "Esc/q close"}
	}
	return strings.Join(hints, "  ")
}
