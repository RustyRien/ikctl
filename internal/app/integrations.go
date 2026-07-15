package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/rivo/tview"
)

func (a *App) openIntegrationOverview(id string, name string) {
	a.stopLiveLogStream()
	a.rememberCurrentDetailState()

	title := fmt.Sprintf("Integration: %s", name)
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.activeSourceCodeDetail = nil
	a.activeSourceCodeVersionDetail = nil
	a.activeSecretDetail = nil
	a.activeIntegrationDetail = &entityDetailSelection{ID: id, Name: name, Kind: "integrations"}
	a.activeStorageDetail = nil
	a.activeWorkerDetail = nil
	a.activeWorkspaceDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading integration overview...")
	a.ui.SetIntegrationOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Integration(ctx, id)
		var primitive tview.Primitive
		var jumpActions map[rune]func()
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load integration overview.\n\n%v", err))
		} else if full != nil {
			a.activeIntegrationDetail = &entityDetailSelection{ID: full.ID, Name: full.Name, Kind: "integrations"}
			primitive = integrationOverviewView(*full)
			jumpActions = a.integrationOverviewJumpActions(*full)
		} else {
			a.activeIntegrationDetail = nil
			a.activeWorkerDetail = nil
			a.activeWorkspaceDetail = nil
			primitive = errorView("Integration not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.clearOverviewJumpState()
			for key, action := range jumpActions {
				a.setOverviewJumpAction(key, action)
			}
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetIntegrationOverviewHotkeys()
		})
	}()
}

func (a *App) integrationOverviewJumpActions(integration client.Integration) map[rune]func() {
	if integration.ID == "" {
		return nil
	}
	return map[rune]func(){
		'r': func() {
			a.openIntegrationResources(integration)
		},
	}
}

func (a *App) openIntegrationResources(integration client.Integration) {
	title := fmt.Sprintf("Integration Resources: %s", valueOr(integration.Name, integration.ID))
	a.overviewJumpSelector = nil
	a.ui.OpenOverlay(title, "Loading integration resources...")

	go func(integration client.Integration) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Resources(ctx, map[string]any{"integration_ids__any": []string{integration.ID}}, []string{"updated_at", "DESC"}, []int{0, 200})
		var primitive tview.Primitive
		var selector *overviewJumpSelector
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load integration resources.\n\n%v", err))
		} else if len(result.Items) == 0 {
			primitive = errorView("No resources found for this integration")
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
	}(integration)
}

func integrationOverviewView(integration client.Integration) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", integration.Name},
		{"ID", integration.ID},
		{"Provider", blankDash(integration.IntegrationProvider)},
		{"Type", blankDash(integration.IntegrationType)},
		{"Created", integration.CreatedAt.Format(time.RFC3339)},
		{"Updated", integration.UpdatedAt.Format(time.RFC3339)},
	})

	meta := kvTable("Usage", [][2]string{
		{"Entity", "integration"},
		{"Provider / Type", strings.TrimSpace(blankDash(integration.IntegrationProvider) + " / " + blankDash(integration.IntegrationType))},
	})

	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(integration.Description, "-"))

	details := tview.NewTextView()
	details.SetBorder(true)
	details.SetTitle("Details")
	details.SetWrap(true)
	details.SetDynamicColors(true)
	details.SetText(strings.Join([]string{
		"[::b]Provider[::-] " + blankDash(integration.IntegrationProvider),
		"[::b]Type[::-] " + blankDash(integration.IntegrationType),
		"[::b]Created[::-] " + integration.CreatedAt.Format(time.RFC3339),
		"[::b]Updated[::-] " + integration.UpdatedAt.Format(time.RFC3339),
	}, "\n"))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(meta, 0, 1, false), 8, 0, true)
	root.AddItem(tview.NewFlex().
		AddItem(description, 0, 2, false).
		AddItem(details, 0, 1, false), 0, 1, false)
	root.AddItem(overviewFooter(integrationOverviewHint(integration)), 1, 0, false)
	return root
}

func integrationOverviewHint(integration client.Integration) string {
	hints := []string{"y yaml", "A actions", "l logs", "a audit", "r resources", "E edit", "Esc/q close"}
	if integration.ID == "" {
		hints = []string{"y yaml", "A actions", "l logs", "a audit", "E edit", "Esc/q close"}
	}
	return strings.Join(hints, "  ")
}
