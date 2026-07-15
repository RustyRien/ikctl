package app

import (
	"fmt"
	"strings"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/model"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) openOverview(row tabledata.Row) {
	a.stopLiveLogStream()

	switch value := row.Raw.(type) {
	case client.Resource:
		a.openResourceOverview(value)
	case client.Template:
		a.openTemplateOverview(value.ID, value.Name)
	case client.Integration:
		a.openIntegrationOverview(value.ID, value.Name)
	}
}

func (a *App) handleNav(key rune) {
	a.stopLiveLogStream()

	var next model.EntityKind
	switch key {
	case 'r':
		next = model.EntityResources
	case 't':
		next = model.EntityTemplates
	case 'i':
		next = model.EntityIntegrations
	default:
		return
	}
	if _, ok := a.models[next]; !ok {
		return
	}
	if next == a.activeKind {
		return
	}

	a.activeKind = next
	a.clearOverviewJumpState()
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.entitySelectorTable = nil
	a.settingsTable = nil
	a.templateFilterAllRows = nil
	a.templateFilterRows = nil
	a.templateFilterTable = nil
	a.templateFilterQuery = ""
	a.templateFilterMode = false
	a.integrationFilterAllRows = nil
	a.integrationFilterRows = nil
	a.integrationFilterTable = nil
	a.integrationFilterQuery = ""
	a.integrationFilterMode = false
	a.resourceColumnsTable = nil
	a.templateColumnsTable = nil
	a.activeTemplateDetail = nil
	a.activeIntegrationDetail = nil
	a.pendingEntityAction = nil
	a.overviewTree = nil
	a.ui.CloseOverlay()
	a.ui.ResetHeaderHotkeys()
	a.renderCurrentModel()
	a.requestRefresh()
}

func (a *App) handleSort(column int, asc bool) {
	if a.activeKind == model.EntityResources {
		headers, _ := a.projectResourceList(render.ResourceListHeaders(), nil)
		if column < 0 || column >= len(headers) {
			return
		}
		if !a.currentModel().SetSortField(headers[column].SortField, asc) {
			return
		}
		a.requestRefresh()
		return
	}
	if a.activeKind == model.EntityTemplates {
		headers, _ := a.projectTemplateList(render.TemplateListHeaders(), nil)
		if column < 0 || column >= len(headers) {
			return
		}
		if !a.currentModel().SetSortField(headers[column].SortField, asc) {
			return
		}
		a.requestRefresh()
		return
	}
	if !a.currentModel().SetSortByColumn(column, asc) {
		return
	}
	a.requestRefresh()
}

func (a *App) openColumns() {
	switch a.activeKind {
	case model.EntityResources:
		a.openResourceColumns()
	case model.EntityTemplates:
		a.openTemplateColumns()
	}
}

func (a *App) openEntitySelector() {
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.settingsTable = nil
	a.templateFilterAllRows = nil
	a.templateFilterRows = nil
	a.templateFilterTable = nil
	a.templateFilterQuery = ""
	a.templateFilterMode = false
	a.integrationFilterAllRows = nil
	a.integrationFilterRows = nil
	a.integrationFilterTable = nil
	a.integrationFilterQuery = ""
	a.integrationFilterMode = false
	a.resourceColumnsTable = nil

	primitive, table := entitySelectorView(a.activeKind)
	a.entitySelectorTable = table
	a.ui.OpenOverlayPrimitive("Entity Selector", primitive)
}

func (a *App) applySelectedEntity() {
	if a.entitySelectorTable == nil {
		return
	}

	selectedRow, _ := a.entitySelectorTable.GetSelection()
	if selectedRow <= 0 {
		return
	}

	var key rune
	switch selectedRow {
	case 1:
		key = 'r'
	case 2:
		key = 't'
	case 3:
		key = 'i'
	default:
		return
	}

	a.entitySelectorTable = nil
	a.ui.CloseOverlay()
	a.handleNav(key)
}

func entitySelectorView(activeKind model.EntityKind) (tview.Primitive, *tview.Table) {
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 0)
	table.SetBorder(true)
	table.SetTitle("Entities")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetBorderColor(tcell.ColorCadetBlue)

	headers := []string{"ENTITY", "KEY", "ACTIVE"}
	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorCadetBlue).
			SetSelectable(false).
			SetExpansion(1))
	}

	items := []struct {
		kind  model.EntityKind
		label string
		key   string
	}{
		{kind: model.EntityResources, label: "Resources", key: "r"},
		{kind: model.EntityTemplates, label: "Templates", key: "t"},
		{kind: model.EntityIntegrations, label: "Integrations", key: "i"},
	}

	selectedRow := 1
	for rowIndex, item := range items {
		active := ""
		if item.kind == activeKind {
			active = "yes"
			selectedRow = rowIndex + 1
		}
		fields := []string{item.label, item.key, active}
		for col, field := range fields {
			table.SetCell(rowIndex+1, col, tview.NewTableCell(field).
				SetTextColor(tcell.ColorLightSteelBlue).
				SetExpansion(1))
		}
	}

	table.Select(selectedRow, 0)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(table, 0, 1, true)
	root.AddItem(overviewFooter("Enter apply  r/t/i quick switch  Esc/q close"), 1, 0, false)
	return root, table
}

func entityTitle(kind model.EntityKind) string {
	switch kind {
	case model.EntityTemplates:
		return "Templates"
	case model.EntityIntegrations:
		return "Integrations"
	default:
		return "Resources"
	}
}

func (a *App) currentEntityTitle() string {
	title := entityTitle(a.activeKind)
	if a.activeKind != model.EntityResources {
		return title
	}

	filters := make([]string, 0, 2)
	if a.resourceTemplateFilter != nil {
		filters = append(filters, fmt.Sprintf("template: %s", a.resourceTemplateFilter.Name))
	}
	if a.resourceIntegrationFilter != nil {
		filters = append(filters, fmt.Sprintf("integration: %s", a.resourceIntegrationFilter.Name))
	}
	if a.hideDestroyedResources {
		filters = append(filters, "hide destroyed")
	}
	if len(filters) == 0 {
		return title
	}
	return fmt.Sprintf("%s [%s]", title, strings.Join(filters, ", "))
}

func entityEmptyLabel(kind model.EntityKind) string {
	switch kind {
	case model.EntityTemplates:
		return "No templates"
	case model.EntityIntegrations:
		return "No integrations"
	default:
		return "No resources"
	}
}
