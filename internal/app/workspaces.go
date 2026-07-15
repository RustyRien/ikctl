package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/model"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

var workspaceColumnOptions = []resourceColumnOption{
	{Field: "name", Header: tabledata.Header{Title: "NAME", Key: "name", SortField: "name"}, Description: "Workspace name", DefaultOn: true},
	{Field: "workspaceProvider", Header: tabledata.Header{Title: "PROVIDER", Key: "workspaceProvider", SortField: "workspaceProvider"}, Description: "Workspace provider", DefaultOn: true},
	{Field: "status", Header: tabledata.Header{Title: "STATUS", Key: "status", SortField: "status"}, Description: "Status", DefaultOn: true},
	{Field: "description", Header: tabledata.Header{Title: "DESCRIPTION", Key: "description"}, Description: "Description", DefaultOn: false},
	{Field: "labels", Header: tabledata.Header{Title: "LABELS", Key: "labels"}, Description: "Labels", DefaultOn: false},
	{Field: "createdAt", Header: tabledata.Header{Title: "CREATED", Key: "createdAt", SortField: "createdAt"}, Description: "Created time", DefaultOn: false},
	{Field: "updatedAt", Header: tabledata.Header{Title: "UPDATED", Key: "updatedAt", SortField: "updatedAt"}, Description: "Updated time", DefaultOn: true},
	{Field: "creator", Header: tabledata.Header{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"}, Description: "Creator", DefaultOn: false},
	{Field: "integration", Header: tabledata.Header{Title: "INTEGRATION", Key: "integration", SortField: "integration.name"}, Description: "Integration", DefaultOn: false},
	{Field: "resourcesCount", Header: tabledata.Header{Title: "RESOURCES", Key: "resourcesCount", SortField: "resourcesCount"}, Description: "Resource count", DefaultOn: false},
	{Field: "entityName", Header: tabledata.Header{Title: "ENTITY", Key: "entityName"}, Description: "Entity name", DefaultOn: false},
	{Field: "id", Header: tabledata.Header{Title: "ID", Key: "id"}, Description: "Workspace ID", DefaultOn: false},
	{Field: "age", Header: tabledata.Header{Title: "AGE", Key: "age", SortField: "createdAt"}, Description: "Age", DefaultOn: true},
}

func defaultVisibleWorkspaceColumns() map[string]bool {
	visible := make(map[string]bool, len(workspaceColumnOptions))
	for _, option := range workspaceColumnOptions {
		if option.DefaultOn {
			visible[option.Field] = true
		}
	}
	return visible
}

func (a *App) openWorkspaceOverview(id string, name string) {
	a.stopLiveLogStream()
	a.rememberCurrentDetailState()

	title := fmt.Sprintf("Workspace: %s", name)
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.activeExecutorDetail = nil
	a.activeSourceCodeDetail = nil
	a.activeSourceCodeVersionDetail = nil
	a.activeSecretDetail = nil
	a.activeIntegrationDetail = nil
	a.activeStorageDetail = nil
	a.activeWorkerDetail = nil
	a.activeWorkspaceDetail = &entityDetailSelection{ID: id, Name: name, Kind: "workspaces"}
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading workspace overview...")
	a.ui.SetWorkspaceOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Workspace(ctx, id)
		var primitive tview.Primitive
		var jumpActions map[rune]func()
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load workspace overview.\n\n%v", err))
		} else if full != nil {
			a.activeWorkspaceDetail = &entityDetailSelection{ID: full.ID, Name: full.Name, Kind: "workspaces"}
			primitive = workspaceOverviewView(*full)
			jumpActions = a.workspaceOverviewJumpActions(*full)
		} else {
			a.activeWorkspaceDetail = nil
			primitive = errorView("Workspace not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.clearOverviewJumpState()
			for key, action := range jumpActions {
				a.setOverviewJumpAction(key, action)
			}
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetWorkspaceOverviewHotkeys()
		})
	}()
}

func (a *App) workspaceOverviewJumpActions(workspace client.Workspace) map[rune]func() {
	if workspace.ID == "" {
		return nil
	}
	return map[rune]func(){
		'r': func() {
			a.openWorkspaceResources(workspace)
		},
	}
}

func (a *App) openWorkspaceResources(workspace client.Workspace) {
	title := fmt.Sprintf("Workspace Resources: %s", valueOr(workspace.Name, workspace.ID))
	a.overviewJumpSelector = nil
	a.ui.OpenOverlay(title, "Loading workspace resources...")

	go func(workspace client.Workspace) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Resources(ctx, map[string]any{"workspace_id": workspace.ID}, []string{"updated_at", "DESC"}, []int{0, 200})
		var primitive tview.Primitive
		var selector *overviewJumpSelector
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load workspace resources.\n\n%v", err))
		} else if len(result.Items) == 0 {
			primitive = errorView("No resources found for this workspace")
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
	}(workspace)
}

func (a *App) openWorkspaceColumns() {
	if a.activeKind != model.EntityWorkspaces {
		return
	}
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.entitySelectorTable = nil
	a.settingsTable = nil
	a.overviewTree = nil
	a.resourceColumnsTable = nil
	a.templateColumnsTable = nil
	a.workspaceColumnsTable = nil
	primitive, table := resourceColumnsView(workspaceColumnOptions, a.visibleWorkspaceColumns)
	a.workspaceColumnsTable = table
	a.ui.OpenOverlayPrimitive("Workspace Columns", primitive)
}

func (a *App) toggleSelectedWorkspaceColumn() {
	if a.workspaceColumnsTable == nil {
		return
	}
	selectedRow, _ := a.workspaceColumnsTable.GetSelection()
	if selectedRow <= 0 {
		return
	}
	index := selectedRow - 1
	if index < 0 || index >= len(workspaceColumnOptions) {
		return
	}
	option := workspaceColumnOptions[index]
	if a.visibleWorkspaceColumns[option.Field] {
		visibleCount := 0
		for _, current := range workspaceColumnOptions {
			if a.visibleWorkspaceColumns[current.Field] {
				visibleCount++
			}
		}
		if visibleCount == 1 {
			return
		}
		a.visibleWorkspaceColumns[option.Field] = false
	} else {
		a.visibleWorkspaceColumns[option.Field] = true
	}
	primitive, table := resourceColumnsView(workspaceColumnOptions, a.visibleWorkspaceColumns)
	table.Select(selectedRow, 0)
	a.workspaceColumnsTable = table
	a.ui.OpenOverlayPrimitive("Workspace Columns", primitive)
	a.saveViewPreferences()
	a.renderCurrentModel()
}

func (a *App) projectWorkspaceList(_ []tabledata.Header, rows []tabledata.Row) ([]tabledata.Header, []tabledata.Row) {
	projectedHeaders := make([]tabledata.Header, 0, len(workspaceColumnOptions))
	visibleFields := make([]string, 0, len(workspaceColumnOptions))
	for _, option := range workspaceColumnOptions {
		if !a.visibleWorkspaceColumns[option.Field] {
			continue
		}
		projectedHeaders = append(projectedHeaders, option.Header)
		visibleFields = append(visibleFields, option.Field)
	}
	if len(visibleFields) == 0 {
		return render.WorkspaceListHeaders(), rows
	}
	projectedRows := make([]tabledata.Row, 0, len(rows))
	for _, row := range rows {
		fullRow := row
		if workspaceValue, ok := row.Raw.(client.Workspace); ok {
			fullRow = render.WorkspaceListRow(workspaceValue)
			fullRow.SortKey["id"] = strings.ToLower(workspaceValue.ID)
		}
		fields := make([]string, 0, len(visibleFields))
		for _, field := range visibleFields {
			fields = append(fields, workspaceFieldValue(fullRow, field))
		}
		projectedRow := fullRow
		projectedRow.Fields = fields
		projectedRows = append(projectedRows, projectedRow)
	}
	return projectedHeaders, projectedRows
}

func workspaceFieldValue(row tabledata.Row, field string) string {
	indexByField := map[string]int{
		"name":              0,
		"workspaceProvider": 1,
		"status":            2,
		"description":       3,
		"labels":            4,
		"createdAt":         5,
		"updatedAt":         6,
		"creator":           7,
		"integration":       8,
		"resourcesCount":    9,
		"entityName":        10,
		"age":               11,
	}
	if field == "id" {
		if row.ID == "" {
			return "-"
		}
		return row.ID
	}
	index, ok := indexByField[field]
	if !ok || index < 0 || index >= len(row.Fields) {
		return "-"
	}
	return row.Fields[index]
}

func workspaceOverviewView(workspace client.Workspace) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", workspace.Name},
		{"ID", workspace.ID},
		{"Provider", blankDash(workspace.WorkspaceProvider)},
		{"Status", blankDash(workspace.Status)},
		{"Created", workspace.CreatedAt.Format(time.RFC3339)},
		{"Updated", workspace.UpdatedAt.Format(time.RFC3339)},
	})

	usage := kvTable("Usage", [][2]string{
		{"Resources", fmt.Sprintf("%d", workspace.ResourcesCount)},
		{"Integration", storageIntegrationName(workspace.Integration)},
		{"Creator", storageCreatorName(workspace.Creator)},
		{"Entity", blankDash(workspace.EntityName)},
		{"Labels", strings.Join(orSlice(workspace.Labels, []string{"-"}), ", ")},
	})

	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(workspace.Description, "-"))

	configuration := tview.NewTextView()
	configuration.SetBorder(true)
	configuration.SetTitle("Configuration")
	configuration.SetWrap(true)
	configuration.SetDynamicColors(true)
	configuration.SetText(workspaceConfigurationText(workspace))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(usage, 0, 1, false), 10, 0, true)
	root.AddItem(tview.NewFlex().
		AddItem(description, 0, 2, false).
		AddItem(configuration, 0, 3, false), 0, 1, false)
	root.AddItem(overviewFooter(workspaceOverviewHint(workspace)), 1, 0, false)
	return root
}

func workspaceConfigurationText(workspace client.Workspace) string {
	lines := []string{
		"[::b]Provider[::-] " + blankDash(workspace.WorkspaceProvider),
	}
	if workspace.Integration != nil && workspace.Integration.Name != "" {
		lines = append(lines, "[::b]Integration[::-] "+workspace.Integration.Name)
	}
	for _, line := range mapLines(workspace.Configuration) {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func workspaceOverviewHint(workspace client.Workspace) string {
	hints := []string{"y yaml", "l logs", "a audit", "r resources", "D delete", "E edit", "Esc/q close"}
	if workspace.ID == "" {
		hints = []string{"y yaml", "l logs", "a audit", "E edit", "Esc/q close"}
	}
	return strings.Join(hints, "  ")
}
