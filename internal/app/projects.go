package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/model"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

var projectColumnOptions = []resourceColumnOption{
	{Field: "name", Header: tabledata.Header{Title: "NAME", Key: "name", SortField: "name"}, Description: "Project name", DefaultOn: true},
	{Field: "description", Header: tabledata.Header{Title: "DESCRIPTION", Key: "description"}, Description: "Description", DefaultOn: false},
	{Field: "labels", Header: tabledata.Header{Title: "LABELS", Key: "labels"}, Description: "Labels", DefaultOn: false},
	{Field: "status", Header: tabledata.Header{Title: "STATUS", Key: "status", SortField: "status"}, Description: "Status", DefaultOn: true},
	{Field: "resourcesCount", Header: tabledata.Header{Title: "RESOURCES", Key: "resourcesCount", SortField: "resources_count"}, Description: "Resource count", DefaultOn: true},
	{Field: "createdAt", Header: tabledata.Header{Title: "CREATED", Key: "createdAt", SortField: "created_at"}, Description: "Created time", DefaultOn: false},
	{Field: "updatedAt", Header: tabledata.Header{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"}, Description: "Last updated", DefaultOn: true},
	{Field: "entityName", Header: tabledata.Header{Title: "ENTITY", Key: "entityName"}, Description: "Entity name", DefaultOn: false},
	{Field: "id", Header: tabledata.Header{Title: "ID", Key: "id"}, Description: "Project ID", DefaultOn: false},
	{Field: "age", Header: tabledata.Header{Title: "AGE", Key: "age", SortField: "created_at"}, Description: "Age", DefaultOn: true},
}

func defaultVisibleProjectColumns() map[string]bool {
	visible := make(map[string]bool, len(projectColumnOptions))
	for _, option := range projectColumnOptions {
		if option.DefaultOn {
			visible[option.Field] = true
		}
	}
	return visible
}

func (a *App) openProjectOverview(id string, name string) {
	a.stopLiveLogStream()
	a.rememberCurrentDetailState()
	a.ui.SetDetailActionRow(tabledata.Row{ID: id, Raw: client.Project{ID: id, Name: name}})
	title := fmt.Sprintf("Project: %s", name)
	a.clearOverviewJumpState()
	a.activeProjectDetail = &entityDetailSelection{ID: id, Name: name, Kind: "projects"}
	a.activeTemplateDetail = nil
	a.activeExecutorDetail = nil
	a.activeSourceCodeDetail = nil
	a.activeSourceCodeVersionDetail = nil
	a.activeSecretDetail = nil
	a.activeIntegrationDetail = nil
	a.activeStorageDetail = nil
	a.activeWorkerDetail = nil
	a.activeWorkspaceDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading project overview...")
	a.ui.SetProjectOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()
		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()
		full, err := a.client.Project(ctx, id)
		var primitive tview.Primitive
		var jumpActions map[rune]func()
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load project overview.\n\n%v", err))
		} else if full != nil {
			a.activeProjectDetail = &entityDetailSelection{ID: full.ID, Name: full.Name, Kind: "projects"}
			primitive = projectOverviewView(*full)
			jumpActions = a.projectOverviewJumpActions(*full)
		} else {
			a.activeProjectDetail = nil
			primitive = errorView("Project not found")
		}
		a.ui.Application().QueueUpdateDraw(func() {
			a.clearOverviewJumpState()
			for key, action := range jumpActions {
				a.setOverviewJumpAction(key, action)
			}
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetProjectOverviewHotkeys()
		})
	}()
}

func (a *App) projectOverviewJumpActions(project client.Project) map[rune]func() {
	if project.ID == "" {
		return nil
	}
	return map[rune]func(){
		'r': func() {
			a.openProjectResources(project)
		},
	}
}

func (a *App) openProjectResources(project client.Project) {
	title := fmt.Sprintf("Project Resources: %s", valueOr(project.Name, project.ID))
	a.overviewJumpSelector = nil
	a.ui.OpenOverlay(title, "Loading project resources...")

	go func(project client.Project) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Resources(ctx, map[string]any{"project_id": project.ID}, []string{"updated_at", "DESC"}, []int{0, 200})
		var primitive tview.Primitive
		var selector *overviewJumpSelector
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load project resources.\n\n%v", err))
		} else if len(result.Items) == 0 {
			primitive = errorView("No resources found for this project")
		} else {
			options := templateResourceJumpOptions(result.Items)
			primitive, selector = templateResourceSelectionView(options, len(result.Items), result.Total)
			selector.onSelect = func(option overviewJumpOption) {
				resourceItem, ok := option.Value.(client.Resource)
				if !ok {
					return
				}
				a.ui.SetDetailActionRow(tabledata.Row{ID: resourceItem.ID, Raw: resourceItem})
				a.openResourceOverview(client.Resource{ID: resourceItem.ID, Name: resourceItem.Name})
			}
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.overviewJumpSelector = selector
			a.ui.OpenOverlayPrimitive(title, primitive)
		})
	}(project)
}

func (a *App) openProjectColumns() {
	if a.activeKind != model.EntityProjects {
		return
	}
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.entitySelectorTable = nil
	a.settingsTable = nil
	a.overviewTree = nil
	a.resourceColumnsTable = nil
	primitive, table := resourceColumnsView(projectColumnOptions, a.visibleProjectColumns)
	a.projectColumnsTable = table
	a.ui.OpenOverlayPrimitive("Project Columns", primitive)
}

func (a *App) toggleSelectedProjectColumn() {
	if a.projectColumnsTable == nil {
		return
	}
	selectedRow, _ := a.projectColumnsTable.GetSelection()
	if selectedRow <= 0 {
		return
	}
	index := selectedRow - 1
	if index < 0 || index >= len(projectColumnOptions) {
		return
	}
	option := projectColumnOptions[index]
	if a.visibleProjectColumns[option.Field] {
		visibleCount := 0
		for _, current := range projectColumnOptions {
			if a.visibleProjectColumns[current.Field] {
				visibleCount++
			}
		}
		if visibleCount == 1 {
			return
		}
		a.visibleProjectColumns[option.Field] = false
	} else {
		a.visibleProjectColumns[option.Field] = true
	}
	primitive, table := resourceColumnsView(projectColumnOptions, a.visibleProjectColumns)
	table.Select(selectedRow, 0)
	a.projectColumnsTable = table
	a.ui.OpenOverlayPrimitive("Project Columns", primitive)
	a.saveViewPreferences()
	a.renderCurrentModel()
}

func (a *App) projectProjectList(_ []tabledata.Header, rows []tabledata.Row) ([]tabledata.Header, []tabledata.Row) {
	projectedHeaders := make([]tabledata.Header, 0, len(projectColumnOptions))
	visibleFields := make([]string, 0, len(projectColumnOptions))
	for _, option := range projectColumnOptions {
		if !a.visibleProjectColumns[option.Field] {
			continue
		}
		projectedHeaders = append(projectedHeaders, option.Header)
		visibleFields = append(visibleFields, option.Field)
	}
	if len(visibleFields) == 0 {
		return render.ProjectListHeaders(), rows
	}
	projectedRows := make([]tabledata.Row, 0, len(rows))
	for _, row := range rows {
		fullRow := row
		if projectValue, ok := row.Raw.(client.Project); ok {
			fullRow = render.ProjectListRow(projectValue)
			fullRow.SortKey["id"] = strings.ToLower(projectValue.ID)
		}
		fields := make([]string, 0, len(visibleFields))
		for _, field := range visibleFields {
			fields = append(fields, projectFieldValue(fullRow, field))
		}
		projectedRow := fullRow
		projectedRow.Fields = fields
		projectedRows = append(projectedRows, projectedRow)
	}
	return projectedHeaders, projectedRows
}

func projectFieldValue(row tabledata.Row, field string) string {
	indexByField := map[string]int{"name": 0, "description": 1, "labels": 2, "status": 3, "resourcesCount": 4, "createdAt": 5, "updatedAt": 6, "entityName": 7, "age": 8}
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

func projectOverviewView(project client.Project) tview.Primitive {
	summary := kvTable("Summary", [][2]string{{"Name", project.Name}, {"ID", project.ID}, {"Status", blankDash(project.Status)}, {"Workspace", projectWorkspaceName(project)}, {"Resources", strconv.Itoa(project.ResourcesCount)}, {"Revision", strconv.Itoa(project.RevisionNumber)}, {"Created", project.CreatedAt.Format(time.RFC3339)}, {"Updated", project.UpdatedAt.Format(time.RFC3339)}})
	meta := kvTable("Ownership / Metadata", [][2]string{{"Owners", projectOwners(project.Owners)}, {"Labels", strings.Join(orSlice(project.Labels, []string{"-"}), ", ")}, {"Entity", blankDash(project.EntityName)}, {"Workspace ID", blankDash(project.WorkspaceID)}})
	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(project.Description, "-"))
	configuration := mapDetailsTable("Configuration", project.Configuration)
	dependencyTags := arrayDetailsTable("Dependency Tags", project.DependencyTags)
	dependencyConfig := arrayDetailsTable("Dependency Config", project.DependencyConfig)
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().AddItem(summary, 0, 1, false).AddItem(meta, 0, 1, false), 10, 0, true)
	root.AddItem(description, 6, 0, false)
	root.AddItem(tview.NewFlex().AddItem(configuration, 0, 1, false).AddItem(dependencyTags, 0, 1, false).AddItem(dependencyConfig, 0, 1, false), 0, 1, false)
	root.AddItem(overviewFooter("y yaml  A actions  l logs  a audit  E edit  Esc/q close"), 1, 0, false)
	return root
}

func projectOwners(owners []client.Creator) string {
	if len(owners) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(owners))
	for _, owner := range owners {
		parts = append(parts, displayCreator(owner))
	}
	return strings.Join(parts, ", ")
}

func projectWorkspaceName(project client.Project) string {
	if project.Workspace == nil {
		return "-"
	}
	return valueOr(project.Workspace.Name, project.Workspace.ID)
}

func arrayDetailsTable(title string, values []map[string]any) tview.Primitive {
	view := tview.NewTextView()
	view.SetBorder(true)
	view.SetTitle(title)
	if len(values) == 0 {
		view.SetText("-")
		return view
	}
	lines := make([]string, 0, len(values))
	for _, item := range values {
		lines = append(lines, stringify(item))
	}
	view.SetText(strings.Join(lines, "\n"))
	return view
}
