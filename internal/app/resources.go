package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/model"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
)

type resourceColumnOption struct {
	Field       string
	Header      tabledata.Header
	Description string
	DefaultOn   bool
}

var resourceColumnOptions = []resourceColumnOption{
	{Field: "name", Header: tabledata.Header{Title: "NAME", Key: "name", SortField: "name"}, Description: "Resource name", DefaultOn: true},
	{Field: "template", Header: tabledata.Header{Title: "TEMPLATE", Key: "template"}, Description: "Template", DefaultOn: true},
	{Field: "sourceCodeVersion", Header: tabledata.Header{Title: "TEMPLATE VERSION", Key: "sourceCodeVersion", SortField: "source_code_version.tag"}, Description: "Template version", DefaultOn: false},
	{Field: "state", Header: tabledata.Header{Title: "STATE", Key: "state", SortField: "state"}, Description: "Lifecycle state", DefaultOn: true},
	{Field: "status", Header: tabledata.Header{Title: "STATUS", Key: "status", SortField: "status"}, Description: "Execution status", DefaultOn: true},
	{Field: "createdAt", Header: tabledata.Header{Title: "CREATED", Key: "createdAt", SortField: "created_at"}, Description: "Created time", DefaultOn: false},
	{Field: "updatedAt", Header: tabledata.Header{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"}, Description: "Last updated", DefaultOn: false},
	{Field: "creator", Header: tabledata.Header{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"}, Description: "Creator", DefaultOn: false},
	{Field: "storage", Header: tabledata.Header{Title: "STORAGE", Key: "storage"}, Description: "Storage", DefaultOn: false},
	{Field: "workspace", Header: tabledata.Header{Title: "WORKSPACE", Key: "workspace"}, Description: "Workspace", DefaultOn: false},
	{Field: "integrationIds", Header: tabledata.Header{Title: "INTEGRATIONS", Key: "integrationIds"}, Description: "Integrations", DefaultOn: false},
	{Field: "secretIds", Header: tabledata.Header{Title: "SECRETS", Key: "secretIds", SortField: "secret_ids.name"}, Description: "Secrets", DefaultOn: false},
	{Field: "parents", Header: tabledata.Header{Title: "PARENTS", Key: "parents", SortField: "parents.name"}, Description: "Parent resources", DefaultOn: false},
	{Field: "children", Header: tabledata.Header{Title: "CHILDREN", Key: "children", SortField: "children.name"}, Description: "Child resources", DefaultOn: false},
	{Field: "variables", Header: tabledata.Header{Title: "VARIABLES", Key: "variables"}, Description: "Variables", DefaultOn: false},
	{Field: "outputs", Header: tabledata.Header{Title: "OUTPUTS", Key: "outputs"}, Description: "Outputs", DefaultOn: false},
	{Field: "labels", Header: tabledata.Header{Title: "LABELS", Key: "labels"}, Description: "Labels", DefaultOn: false},
	{Field: "dependencyTags", Header: tabledata.Header{Title: "DEPENDENCY TAGS", Key: "dependencyTags"}, Description: "Dependency tags", DefaultOn: false},
	{Field: "dependencyConfig", Header: tabledata.Header{Title: "DEPENDENCY CONFIG", Key: "dependencyConfig"}, Description: "Dependency config", DefaultOn: false},
	{Field: "age", Header: tabledata.Header{Title: "AGE", Key: "age", SortField: "created_at"}, Description: "Age", DefaultOn: true},
}

func defaultVisibleResourceColumns() map[string]bool {
	visible := make(map[string]bool, len(resourceColumnOptions))
	for _, option := range resourceColumnOptions {
		if option.DefaultOn {
			visible[option.Field] = true
		}
	}
	return visible
}

func (a *App) toggleHideDestroyedResources() {
	if a.activeKind != model.EntityResources {
		return
	}
	a.hideDestroyedResources = !a.hideDestroyedResources
	a.applyResourceFilters()
}

func (a *App) resetAllResourceFilters() {
	if a.activeKind != model.EntityResources {
		return
	}
	a.resourceTemplateFilter = nil
	a.resourceIntegrationFilter = nil
	a.hideDestroyedResources = false
	a.applyResourceFilters()
}

func (a *App) openTemplateFilter() {
	if a.activeKind != model.EntityResources {
		return
	}

	selectedTemplateID := ""
	if a.resourceTemplateFilter != nil {
		selectedTemplateID = a.resourceTemplateFilter.ID
	}
	a.resetResourceFilterOverlays()

	a.ui.OpenOverlay("Resource Template Filter", "Loading templates...")

	go func(selectedID string) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Templates(ctx, nil, []string{"name", "ASC"}, []int{0, 200})
		var primitive tview.Primitive
		var templates []client.Template
		var table *tview.Table
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load templates.\n\n%v", err))
		} else {
			templates = result.Items
			primitive, table = templateFilterView(result.Items, len(result.Items), result.Total, selectedID, "", false)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.templateFilterAllRows = templates
			a.templateFilterRows = templates
			a.templateFilterTable = table
			a.ui.OpenOverlayPrimitive("Resource Template Filter", primitive)
		})
	}(selectedTemplateID)
}

func (a *App) openIntegrationFilter() {
	if a.activeKind != model.EntityResources {
		return
	}

	selectedIntegrationID := ""
	if a.resourceIntegrationFilter != nil {
		selectedIntegrationID = a.resourceIntegrationFilter.ID
	}
	a.resetResourceFilterOverlays()

	a.ui.OpenOverlay("Resource Integration Filter", "Loading integrations...")

	go func(selectedID string) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Integrations(ctx, map[string]any{"integration_type": "cloud"}, []string{"name", "ASC"}, []int{0, 200})
		var primitive tview.Primitive
		var integrations []client.Integration
		var table *tview.Table
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load integrations.\n\n%v", err))
		} else {
			integrations = result.Items
			primitive, table = integrationFilterView(result.Items, len(result.Items), result.Total, selectedID, "", false)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.integrationFilterAllRows = integrations
			a.integrationFilterRows = integrations
			a.integrationFilterTable = table
			a.ui.OpenOverlayPrimitive("Resource Integration Filter", primitive)
		})
	}(selectedIntegrationID)
}

func (a *App) renderTemplateFilterOverlay() {
	selectedID := ""
	if a.resourceTemplateFilter != nil {
		selectedID = a.resourceTemplateFilter.ID
	}

	filtered := filterTemplates(a.templateFilterAllRows, a.templateFilterQuery)
	primitive, table := templateFilterView(filtered, len(filtered), len(a.templateFilterAllRows), selectedID, a.templateFilterQuery, a.templateFilterMode)
	a.templateFilterRows = filtered
	a.templateFilterTable = table
	a.ui.OpenOverlayPrimitive("Resource Template Filter", primitive)
}

func (a *App) renderIntegrationFilterOverlay() {
	selectedID := ""
	if a.resourceIntegrationFilter != nil {
		selectedID = a.resourceIntegrationFilter.ID
	}

	filtered := filterIntegrations(a.integrationFilterAllRows, a.integrationFilterQuery)
	primitive, table := integrationFilterView(filtered, len(filtered), len(a.integrationFilterAllRows), selectedID, a.integrationFilterQuery, a.integrationFilterMode)
	a.integrationFilterRows = filtered
	a.integrationFilterTable = table
	a.ui.OpenOverlayPrimitive("Resource Integration Filter", primitive)
}

func (a *App) applySelectedTemplateFilter() {
	if a.templateFilterTable == nil {
		return
	}

	selectedRow, _ := a.templateFilterTable.GetSelection()
	if selectedRow <= 0 {
		return
	}

	if selectedRow == 1 {
		a.clearTemplateFilter()
		return
	}

	index := selectedRow - 2
	if index < 0 || index >= len(a.templateFilterRows) {
		return
	}

	template := a.templateFilterRows[index]
	a.resourceTemplateFilter = &template
	a.applyResourceFilters()
}

func (a *App) clearTemplateFilter() {
	a.resourceTemplateFilter = nil
	a.applyResourceFilters()
}

func (a *App) applySelectedIntegrationFilter() {
	if a.integrationFilterTable == nil {
		return
	}

	selectedRow, _ := a.integrationFilterTable.GetSelection()
	if selectedRow <= 0 {
		return
	}

	if selectedRow == 1 {
		a.clearIntegrationFilter()
		return
	}

	index := selectedRow - 2
	if index < 0 || index >= len(a.integrationFilterRows) {
		return
	}

	integration := a.integrationFilterRows[index]
	a.resourceIntegrationFilter = &integration
	a.applyResourceFilters()
}

func (a *App) clearIntegrationFilter() {
	a.resourceIntegrationFilter = nil
	a.applyResourceFilters()
}

func (a *App) applyResourceFilters() {
	a.entitySelectorTable = nil
	a.settingsTable = nil
	a.resetResourceFilterOverlays()
	a.ui.CloseOverlay()

	resourcesModel := a.models[model.EntityResources]
	if resourcesModel == nil {
		return
	}
	filter := a.resourceFilters()
	a.saveViewPreferences()
	if !resourcesModel.SetFilter(filter) {
		a.renderCurrentModel()
		return
	}

	a.renderCurrentModel()
	a.requestRefresh()
}

func (a *App) resetResourceFilterOverlays() {
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
}

func (a *App) resourceFilters() map[string]any {
	filter := map[string]any{}
	if a.resourceTemplateFilter != nil {
		filter["template_id"] = a.resourceTemplateFilter.ID
	}
	if a.resourceIntegrationFilter != nil {
		filter["integration_ids__any"] = []string{a.resourceIntegrationFilter.ID}
	}
	if a.hideDestroyedResources {
		filter["state__in"] = []string{"provision", "provisioned", "update"}
	}
	if len(filter) == 0 {
		return nil
	}
	return filter
}

func (a *App) openResourceOverview(resource client.Resource) {
	title := fmt.Sprintf("Resource: %s", resource.Name)
	a.overlayTemplateJump = nil
	a.activeTemplateDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading resource overview...")
	a.ui.SetResourceOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Resource(ctx, resource.ID)
		var primitive tview.Primitive
		var jump *overlayTemplateJump
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load resource overview.\n\n%v", err))
		} else if full != nil {
			primitive = resourceOverviewView(*full)
			if full.Template != nil && full.Template.ID != "" {
				jump = &overlayTemplateJump{ID: full.Template.ID, Name: full.Template.Name}
			}
		} else {
			primitive = errorView("Resource not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.overlayTemplateJump = jump
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetResourceOverviewHotkeys()
		})
	}()
}

func (a *App) openResourceColumns() {
	if a.activeKind != model.EntityResources {
		return
	}
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.entitySelectorTable = nil
	a.settingsTable = nil
	a.resetResourceFilterOverlays()
	primitive, table := resourceColumnsView(resourceColumnOptions, a.visibleResourceColumns)
	a.resourceColumnsTable = table
	a.ui.OpenOverlayPrimitive("Resource Columns", primitive)
}

func (a *App) toggleSelectedResourceColumn() {
	if a.resourceColumnsTable == nil {
		return
	}
	selectedRow, _ := a.resourceColumnsTable.GetSelection()
	if selectedRow <= 0 {
		return
	}
	index := selectedRow - 1
	if index < 0 || index >= len(resourceColumnOptions) {
		return
	}
	option := resourceColumnOptions[index]
	if a.visibleResourceColumns[option.Field] {
		visibleCount := 0
		for _, current := range resourceColumnOptions {
			if a.visibleResourceColumns[current.Field] {
				visibleCount++
			}
		}
		if visibleCount == 1 {
			return
		}
		a.visibleResourceColumns[option.Field] = false
	} else {
		a.visibleResourceColumns[option.Field] = true
	}
	primitive, table := resourceColumnsView(resourceColumnOptions, a.visibleResourceColumns)
	table.Select(selectedRow, 0)
	a.resourceColumnsTable = table
	a.ui.OpenOverlayPrimitive("Resource Columns", primitive)
	a.saveViewPreferences()
	a.renderCurrentModel()
}

func (a *App) projectResourceList(_ []tabledata.Header, rows []tabledata.Row) ([]tabledata.Header, []tabledata.Row) {
	headers := render.ResourceListHeaders()
	visibleIndexes := make([]int, 0, len(resourceColumnOptions))
	projectedHeaders := make([]tabledata.Header, 0, len(resourceColumnOptions))
	for index, option := range resourceColumnOptions {
		if !a.visibleResourceColumns[option.Field] {
			continue
		}
		if index >= len(headers) {
			continue
		}
		visibleIndexes = append(visibleIndexes, index)
		projectedHeaders = append(projectedHeaders, headers[index])
	}
	if len(visibleIndexes) == 0 {
		return headers, rows
	}
	projectedRows := make([]tabledata.Row, 0, len(rows))
	for _, row := range rows {
		fullRow := row
		if resourceValue, ok := row.Raw.(client.Resource); ok {
			fullRow = render.ResourceListRow(resourceValue)
		}
		fields := make([]string, 0, len(visibleIndexes))
		for _, index := range visibleIndexes {
			if index < len(fullRow.Fields) {
				fields = append(fields, fullRow.Fields[index])
			}
		}
		projectedRow := fullRow
		projectedRow.Fields = fields
		projectedRows = append(projectedRows, projectedRow)
	}
	return projectedHeaders, projectedRows
}

func resourceOverviewView(resource client.Resource) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", resource.Name},
		{"ID", resource.ID},
		{"State", resource.State},
		{"Status", resource.Status},
		{"Created", resource.CreatedAt.Format(time.RFC3339)},
		{"Updated", resource.UpdatedAt.Format(time.RFC3339)},
		{"Revision", fmt.Sprintf("%d", resource.RevisionNumber)},
		{"Abstract", fmt.Sprintf("%t", resource.Abstract)},
		{"Workspace", resourceWorkspace(resource)},
		{"Creator", resourceCreator(resource)},
	})

	meta := kvTable("Template / Source", [][2]string{
		{"Template", resourceTemplate(resource)},
		{"Cloud Types", strings.Join(orSlice(resourceTemplateTypes(resource), []string{"-"}), ", ")},
		{"Storage Path", blankDash(resource.StoragePath)},
		{"Labels", strings.Join(orSlice(resource.Labels, []string{"-"}), ", ")},
		{"SCV Identifier", sourceCodeIdentifier(resource)},
		{"SCV Folder", sourceCodeFolder(resource)},
		{"SCV Version", sourceCodeVersion(resource)},
		{"SCV Branch", sourceCodeBranch(resource)},
		{"SCV Status", sourceCodeStatus(resource)},
	})

	description := tview.NewTextView()
	description.SetDynamicColors(true)
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(resource.Description, "-"))

	relations := simpleList("Relations", append(
		prefixedRefs("Parent", resource.Parents),
		prefixedRefs("Child", resource.Children)...,
	))

	integrations := simpleList("Integrations", integrationLines(resource.Integrations))
	variables := mapListTable("Variables", resource.Variables)
	outputs := mapListTable("Outputs", resource.Outputs)
	tags := mapListTable("Dependency Tags", resource.DependencyTags)
	config := mapListTable("Dependency Config", resource.DependencyConfig)

	top := tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(meta, 0, 1, false)

	middle := tview.NewFlex().
		AddItem(description, 0, 2, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(relations, 0, 1, false).
			AddItem(integrations, 0, 1, false), 0, 1, false)

	bottom := tview.NewFlex().
		AddItem(variables, 0, 1, false).
		AddItem(outputs, 0, 1, false).
		AddItem(tags, 0, 1, false).
		AddItem(config, 0, 1, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(top, 12, 0, true)
	root.AddItem(middle, 10, 0, false)
	root.AddItem(bottom, 0, 1, false)
	root.AddItem(overviewFooter(resourceTemplateHint(resource)), 1, 0, false)

	return root
}

func templateFilterView(templates []client.Template, shown int, total int, selectedID string, query string, filterMode bool) (tview.Primitive, *tview.Table) {
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 0)
	table.SetBorder(true)
	table.SetTitle("Templates")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetBorderColor(tcell.ColorCadetBlue)

	headers := []string{"TEMPLATE", "CLOUD TYPES", "UPDATED"}
	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorCadetBlue).
			SetSelectable(false).
			SetExpansion(1))
	}

	selectedRow := 1
	allLabel := "All templates"
	if selectedID == "" {
		allLabel = allLabel + "  [active]"
	}
	table.SetCell(1, 0, tview.NewTableCell(allLabel).SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(1, 1, tview.NewTableCell("-").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(1, 2, tview.NewTableCell("-").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))

	for rowIndex, template := range templates {
		name := template.Name
		if template.ID == selectedID {
			name = name + "  [active]"
			selectedRow = rowIndex + 2
		}
		cloudTypes := strings.Join(orSlice(template.CloudResourceTypes, []string{"-"}), ", ")
		fields := []string{name, cloudTypes, template.UpdatedAt.Format(time.RFC3339)}
		for col, field := range fields {
			table.SetCell(rowIndex+2, col, tview.NewTableCell(field).
				SetTextColor(tcell.ColorLightSteelBlue).
				SetExpansion(1))
		}
	}

	table.Select(selectedRow, 0)

	footerText := fmt.Sprintf("Showing %d of %d templates  / filter  Enter apply  c clear  Esc/q close", shown, total)
	if query != "" {
		footerText += fmt.Sprintf("  Filter: %s", query)
	}
	if filterMode {
		footerText += "  typing..."
	}
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(table, 0, 1, true)
	root.AddItem(overviewFooter(footerText), 1, 0, false)
	return root, table
}

func integrationFilterView(integrations []client.Integration, shown int, total int, selectedID string, query string, filterMode bool) (tview.Primitive, *tview.Table) {
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 0)
	table.SetBorder(true)
	table.SetTitle("Integrations")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetBorderColor(tcell.ColorCadetBlue)

	headers := []string{"INTEGRATION", "PROVIDER", "TYPE", "UPDATED"}
	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorCadetBlue).
			SetSelectable(false).
			SetExpansion(1))
	}

	selectedRow := 1
	allLabel := "All integrations"
	if selectedID == "" {
		allLabel = allLabel + "  [active]"
	}
	table.SetCell(1, 0, tview.NewTableCell(allLabel).SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(1, 1, tview.NewTableCell("-").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(1, 2, tview.NewTableCell("-").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(1, 3, tview.NewTableCell("-").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))

	for rowIndex, integration := range integrations {
		name := integration.Name
		if integration.ID == selectedID {
			name = name + "  [active]"
			selectedRow = rowIndex + 2
		}
		fields := []string{
			name,
			blankDash(integration.IntegrationProvider),
			blankDash(integration.IntegrationType),
			integration.UpdatedAt.Format(time.RFC3339),
		}
		for col, field := range fields {
			table.SetCell(rowIndex+2, col, tview.NewTableCell(field).
				SetTextColor(tcell.ColorLightSteelBlue).
				SetExpansion(1))
		}
	}

	table.Select(selectedRow, 0)

	footerText := fmt.Sprintf("Showing %d of %d integrations  / filter  Enter apply  c clear  Esc/q close", shown, total)
	if query != "" {
		footerText += fmt.Sprintf("  Filter: %s", query)
	}
	if filterMode {
		footerText += "  typing..."
	}
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(table, 0, 1, true)
	root.AddItem(overviewFooter(footerText), 1, 0, false)
	return root, table
}

func filterTemplates(templates []client.Template, query string) []client.Template {
	if query == "" {
		return append([]client.Template(nil), templates...)
	}

	targets := make([]string, 0, len(templates))
	for _, template := range templates {
		targets = append(targets, strings.ToLower(template.Name+" "+strings.Join(template.CloudResourceTypes, " ")))
	}

	matches := fuzzy.Find(strings.ToLower(query), targets)
	filtered := make([]client.Template, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, templates[match.Index])
	}
	return filtered
}

func filterIntegrations(integrations []client.Integration, query string) []client.Integration {
	if query == "" {
		return append([]client.Integration(nil), integrations...)
	}

	targets := make([]string, 0, len(integrations))
	for _, integration := range integrations {
		targets = append(targets, strings.ToLower(integration.Name+" "+integration.IntegrationProvider+" "+integration.IntegrationType))
	}

	matches := fuzzy.Find(strings.ToLower(query), targets)
	filtered := make([]client.Integration, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, integrations[match.Index])
	}
	return filtered
}

func mapListTable(title string, items []map[string]any) tview.Primitive {
	table := tview.NewTable().SetBorders(false)
	table.SetBorder(true)
	table.SetTitle(title)
	if len(items) == 0 {
		table.SetCell(0, 0, tview.NewTableCell("-").SetSelectable(false))
		return table
	}

	headers := sortedKeys(items)
	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(strings.ToUpper(header)).SetTextColor(tcell.ColorSteelBlue).SetSelectable(false))
	}
	for row, item := range items {
		for col, header := range headers {
			table.SetCell(row+1, col, tview.NewTableCell(stringify(item[header])).SetSelectable(false).SetExpansion(1))
		}
	}
	return table
}

func sortedKeys(items []map[string]any) []string {
	keySet := map[string]struct{}{}
	for _, item := range items {
		for key := range item {
			keySet[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func prefixedRefs(prefix string, refs []client.ResourceReference) []string {
	lines := make([]string, 0, len(refs))
	for _, ref := range refs {
		lines = append(lines, fmt.Sprintf("%s: %s [%s/%s]", prefix, ref.Name, ref.State, ref.Status))
	}
	return lines
}

func integrationLines(integrations []client.Integration) []string {
	if len(integrations) == 0 {
		return nil
	}
	lines := make([]string, 0, len(integrations))
	for _, integration := range integrations {
		lines = append(lines, fmt.Sprintf("%s (%s/%s)", integration.Name, integration.IntegrationProvider, integration.IntegrationType))
	}
	return lines
}

func resourceWorkspace(resource client.Resource) string {
	if resource.Workspace == nil {
		return "-"
	}
	return valueOr(resource.Workspace.Name, "-")
}

func resourceCreator(resource client.Resource) string {
	if resource.Creator == nil {
		return "-"
	}
	if resource.Creator.Email != "" {
		return fmt.Sprintf("%s <%s>", displayCreator(*resource.Creator), resource.Creator.Email)
	}
	return displayCreator(*resource.Creator)
}

func resourceTemplate(resource client.Resource) string {
	if resource.Template == nil {
		return "-"
	}
	return valueOr(resource.Template.Name, "-")
}

func resourceTemplateTypes(resource client.Resource) []string {
	if resource.Template == nil {
		return nil
	}
	return resource.Template.CloudResourceTypes
}

func resourceTemplateHint(resource client.Resource) string {
	if resource.Template != nil && resource.Template.ID != "" {
		return "t open template  Esc/q close"
	}
	return "Esc/q close"
}

func sourceCodeIdentifier(resource client.Resource) string {
	if resource.SourceCodeVersion == nil {
		return "-"
	}
	return blankDash(resource.SourceCodeVersion.Identifier)
}

func sourceCodeFolder(resource client.Resource) string {
	if resource.SourceCodeVersion == nil {
		return "-"
	}
	return blankDash(resource.SourceCodeVersion.SourceCodeFolder)
}

func sourceCodeVersion(resource client.Resource) string {
	if resource.SourceCodeVersion == nil {
		return "-"
	}
	return blankDash(resource.SourceCodeVersion.SourceCodeVersion)
}

func sourceCodeBranch(resource client.Resource) string {
	if resource.SourceCodeVersion == nil {
		return "-"
	}
	return blankDash(resource.SourceCodeVersion.SourceCodeBranch)
}

func sourceCodeStatus(resource client.Resource) string {
	if resource.SourceCodeVersion == nil {
		return "-"
	}
	return blankDash(resource.SourceCodeVersion.Status)
}
