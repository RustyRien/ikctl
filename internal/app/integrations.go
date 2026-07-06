package app

import (
	"context"
	"fmt"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/rivo/tview"
)

func (a *App) openIntegrationOverview(id string, name string) {
	title := fmt.Sprintf("Integration: %s", name)
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading integration overview...")
	a.ui.SetDetailHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Integration(ctx, id)
		var primitive tview.Primitive
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load integration overview.\n\n%v", err))
		} else if full != nil {
			primitive = integrationOverviewView(*full)
		} else {
			primitive = errorView("Integration not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetDetailHotkeys()
		})
	}()
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

	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(integration.Description, "-"))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(summary, 8, 0, true)
	root.AddItem(description, 0, 1, false)
	root.AddItem(overviewFooter("Esc/q close"), 1, 0, false)
	return root
}
