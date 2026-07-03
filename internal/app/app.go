package app

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/model"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	uiapp "github.com/electrolux-oss/ik-tui/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type overlayTemplateJump struct {
	ID   string
	Name string
}

type templateDetailSelection struct {
	ID   string
	Name string
}

type auditLogSelection struct {
	ResourceID     string
	ResourceName   string
	AuditLogID     string
	Action         string
	CreatedAt      time.Time
	RevisionNumber int
	Creator        string
}

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

var logLevelPrefixRX = regexp.MustCompile(`(?i)^\[(trace|debug|info|warn|warning|error|fatal)\]\s*`)

type App struct {
	config                    config.Config
	build                     BuildInfo
	client                    *client.Client
	models                    map[model.EntityKind]*model.EntityModel
	registry                  *resource.Registry
	kindByName                map[string]model.EntityKind
	nameByKind                map[model.EntityKind]string
	activeKind                model.EntityKind
	ui                        *uiapp.App
	manualKick                chan struct{}
	settingsChanged           chan struct{}
	ctx                       context.Context
	cancel                    context.CancelFunc
	overlayTemplateJump       *overlayTemplateJump
	auditLogRows              []tabledata.Row
	auditLogTable             *tview.Table
	entitySelectorTable       *tview.Table
	settingsTable             *tview.Table
	templateTree              *tview.TreeView
	templateFilterAllRows     []client.Template
	templateFilterRows        []client.Template
	templateFilterTable       *tview.Table
	templateFilterQuery       string
	templateFilterMode        bool
	integrationFilterAllRows  []client.Integration
	integrationFilterRows     []client.Integration
	integrationFilterTable    *tview.Table
	integrationFilterQuery    string
	integrationFilterMode     bool
	resourceColumnsTable      *tview.Table
	visibleResourceColumns    map[string]bool
	resourceTemplateFilter    *client.Template
	resourceIntegrationFilter *client.Integration
	hideDestroyedResources    bool
	activeTemplateDetail      *templateDetailSelection
}

func New(cfg config.Config, build BuildInfo, activeEntity string) *App {
	client := client.New(cfg)
	return NewWithClient(cfg, build, activeEntity, client)
}

func NewWithClient(cfg config.Config, build BuildInfo, activeEntity string, cli *client.Client) *App {
	ctx, cancel := context.WithCancel(context.Background())
	ui := uiapp.NewApp(cfg, build.Version)
	registry := resource.DefaultRegistry(cli)

	app := &App{
		config:          cfg,
		build:           build,
		client:          cli,
		models:          map[model.EntityKind]*model.EntityModel{},
		registry:        registry,
		kindByName:      map[string]model.EntityKind{},
		nameByKind:      map[model.EntityKind]string{},
		activeKind:      model.EntityResources,
		ui:              ui,
		manualKick:      make(chan struct{}, 1),
		settingsChanged: make(chan struct{}, 1),
		ctx:             ctx,
		cancel:          cancel,
	}
	app.visibleResourceColumns = defaultVisibleResourceColumns()

	ordered := registry.Ordered()
	for index, descriptor := range ordered {
		kind := model.EntityKind(descriptor.Name)
		app.models[kind] = model.NewModelFromDescriptorWithSortOrder(kind, descriptor, cfg.DefaultSortDescending())
		app.kindByName[descriptor.Name] = kind
		app.nameByKind[kind] = descriptor.Name
		if index == 0 {
			app.activeKind = kind
		}
	}
	if kind, ok := app.kindByName[activeEntity]; ok {
		app.activeKind = kind
	}

	ui.SetRefreshFunc(app.requestRefresh)
	ui.SetEnterFunc(app.openOverview)
	ui.SetLogsFunc(app.openLogs)
	ui.SetAuditFunc(app.openAuditLogs)
	ui.SetNavFunc(app.handleNav)
	ui.SetSortFunc(app.handleSort)
	ui.SetLoadMoreFunc(app.requestLoadMore)
	ui.SetTemplateFilterFunc(app.openTemplateFilter)
	ui.SetIntegrationFilterFunc(app.openIntegrationFilter)
	ui.SetResourceColumnsFunc(app.openResourceColumns)
	ui.SetToggleDestroyedFunc(app.toggleHideDestroyedResources)
	ui.SetEntitySelectorFunc(app.openEntitySelector)
	ui.SetSettingsFunc(app.openSettings)
	ui.SetOverlayKeyFunc(app.handleOverlayKey)
	ui.SetDetailKeyFunc(app.handleOverlayKey)
	ui.SetEntityTitle(app.currentEntityTitle(), entityEmptyLabel(app.activeKind))

	return app
}

func (a *App) Run() error {
	defer a.cancel()
	a.loadCurrentUser()
	a.refreshInitial()
	go a.loop()
	return a.ui.Run()
}

func (a *App) loop() {
	refreshInterval := a.config.RefreshInterval
	autoRefresh := a.config.AutoRefresh
	timer := time.NewTimer(refreshInterval)
	defer timer.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-timer.C:
			if autoRefresh {
				a.refresh()
			}
			timer.Reset(refreshInterval)
		case <-a.manualKick:
			a.refresh()
		case <-a.settingsChanged:
			refreshInterval = a.config.RefreshInterval
			autoRefresh = a.config.AutoRefresh
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(refreshInterval)
		}
	}
}

func (a *App) requestRefresh() {
	select {
	case a.manualKick <- struct{}{}:
	default:
	}
}

func (a *App) loadCurrentUser() {
	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		user, err := a.client.CurrentUser(ctx)
		if err != nil || user == nil {
			return
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.ui.SetHeaderUser(user.Identifier, user.DisplayName, user.Email)
		})
	}()
}

func (a *App) requestLoadMore() {
	entityModel := a.currentModel()
	if entityModel.LoadingMore() || !entityModel.HasMore() {
		return
	}

	go func(kind model.EntityKind, m *model.EntityModel) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		_ = m.LoadMore(ctx)
		a.ui.Application().QueueUpdateDraw(func() {
			if a.activeKind != kind {
				return
			}
			a.renderCurrentModel()
		})
	}(a.activeKind, entityModel)
}

func (a *App) refresh() {
	done := a.ui.BeginLoading()
	defer done()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	entityModel := a.currentModel()
	_ = entityModel.Refresh(ctx)
	headers, rows, total, lastUpdated, lastErr := entityModel.Snapshot()
	if a.activeKind == model.EntityResources {
		headers, rows = a.projectResourceList(headers, rows)
	}
	sortColumn, sortAsc := entityModel.SortStateForHeaders(headers)
	a.ui.Application().QueueUpdateDraw(func() {
		a.ui.SetEntityTitle(a.currentEntityTitle(), entityEmptyLabel(a.activeKind))
		a.ui.Update(headers, rows, len(rows), total, lastUpdated, lastErr)
		a.ui.SetSortState(sortColumn, sortAsc)
	})
}

func (a *App) refreshInitial() {
	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		model := a.currentModel()
		_ = model.Refresh(ctx)
		a.ui.Application().QueueUpdateDraw(func() {
			a.renderCurrentModel()
		})
	}()
}

func (a *App) renderCurrentModel() {
	entityModel := a.currentModel()
	headers, rows, total, lastUpdated, lastErr := entityModel.Snapshot()
	if a.activeKind == model.EntityResources {
		headers, rows = a.projectResourceList(headers, rows)
	}
	sortColumn, sortAsc := entityModel.SortStateForHeaders(headers)
	a.ui.SetEntityTitle(a.currentEntityTitle(), entityEmptyLabel(a.activeKind))
	a.ui.Update(headers, rows, len(rows), total, lastUpdated, lastErr)
	a.ui.SetSortState(sortColumn, sortAsc)
}

func (a *App) currentModel() *model.EntityModel {
	return a.models[a.activeKind]
}

func (a *App) handleNav(key rune) {
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
	a.overlayTemplateJump = nil
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
	a.activeTemplateDetail = nil
	a.templateTree = nil
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
	if !a.currentModel().SetSortByColumn(column, asc) {
		return
	}
	a.requestRefresh()
}

func (a *App) toggleHideDestroyedResources() {
	if a.activeKind != model.EntityResources {
		return
	}
	a.hideDestroyedResources = !a.hideDestroyedResources
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

func (a *App) openSettings() {
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.entitySelectorTable = nil
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

	primitive, table := settingsView(a.config.AutoRefresh, a.config.DefaultSortOrder, a.config.RefreshSeconds, a.config.ShowBreadcrumbs)
	a.settingsTable = table
	a.ui.OpenOverlayPrimitive("Settings", primitive)
}

func (a *App) applySelectedSetting() {
	if a.settingsTable == nil {
		return
	}

	selectedRow, _ := a.settingsTable.GetSelection()
	switch selectedRow {
	case 1:
		a.toggleAutoRefresh()
	case 2:
		a.toggleBreadcrumbs()
	case 3:
		a.toggleDefaultSortOrder()
	case 4:
		a.adjustRefreshInterval(1)
	}
}

func (a *App) toggleAutoRefresh() {
	a.config.AutoRefresh = !a.config.AutoRefresh
	_ = a.config.Save()
	a.notifySettingsChanged()
	a.renderSettingsOverlay(1)
}

func (a *App) toggleDefaultSortOrder() {
	if a.config.DefaultSortDescending() {
		a.config.DefaultSortOrder = "asc"
	} else {
		a.config.DefaultSortOrder = "desc"
	}
	_ = a.config.Save()

	for _, entityModel := range a.models {
		entityModel.SetDefaultSortDescending(a.config.DefaultSortDescending())
	}

	a.notifySettingsChanged()
	a.renderSettingsOverlay(3)
	a.requestRefresh()
}

func (a *App) toggleBreadcrumbs() {
	a.config.ShowBreadcrumbs = !a.config.ShowBreadcrumbs
	_ = a.config.Save()
	a.ui.SetBreadcrumbsVisible(a.config.ShowBreadcrumbs)
	a.notifySettingsChanged()
	a.renderSettingsOverlay(2)
}

func (a *App) adjustRefreshInterval(delta float64) {
	next := a.config.RefreshSeconds + delta
	if next < 1 {
		next = 1
	}
	a.config.RefreshSeconds = next
	a.config.RefreshInterval = time.Duration(next * float64(time.Second))
	_ = a.config.Save()
	a.notifySettingsChanged()
	a.renderSettingsOverlay(4)
}

func (a *App) renderSettingsOverlay(selectedRow int) {
	primitive, table := settingsView(a.config.AutoRefresh, a.config.DefaultSortOrder, a.config.RefreshSeconds, a.config.ShowBreadcrumbs)
	if selectedRow >= 1 && selectedRow <= 4 {
		table.Select(selectedRow, 0)
	}
	a.settingsTable = table
	a.ui.OpenOverlayPrimitive("Settings", primitive)
}

func (a *App) notifySettingsChanged() {
	select {
	case a.settingsChanged <- struct{}{}:
	default:
	}
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

func (a *App) openOverview(row tabledata.Row) {
	switch value := row.Raw.(type) {
	case client.Resource:
		a.openResourceOverview(value)
	case client.Template:
		a.openTemplateOverview(value.ID, value.Name)
	case client.Integration:
		a.openIntegrationOverview(value.ID, value.Name)
	}
}

func (a *App) openResourceOverview(resource client.Resource) {
	title := fmt.Sprintf("Resource: %s", resource.Name)
	a.overlayTemplateJump = nil
	a.activeTemplateDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading resource overview...")

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
		})
	}()
}

func (a *App) openTemplateOverview(id string, name string) {
	title := fmt.Sprintf("Template: %s", name)
	a.overlayTemplateJump = nil
	a.activeTemplateDetail = &templateDetailSelection{ID: id, Name: name}
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading template overview...")
	a.ui.SetTemplateOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Template(ctx, id)
		var primitive tview.Primitive
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load template overview.\n\n%v", err))
		} else if full != nil {
			a.activeTemplateDetail = &templateDetailSelection{ID: full.ID, Name: full.Name}
			primitive = templateOverviewView(*full)
		} else {
			a.activeTemplateDetail = nil
			primitive = errorView("Template not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetTemplateOverviewHotkeys()
		})
	}()
}

func (a *App) openIntegrationOverview(id string, name string) {
	title := fmt.Sprintf("Integration: %s", name)
	a.overlayTemplateJump = nil
	a.activeTemplateDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading integration overview...")

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
		})
	}()
}

func (a *App) handleOverlayKey(event *tcell.EventKey) bool {
	if a.settingsTable != nil {
		switch event.Key() {
		case tcell.KeyEnter:
			a.applySelectedSetting()
			return true
		case tcell.KeyEsc:
			a.settingsTable = nil
			return false
		case tcell.KeyCtrlD, tcell.KeyCtrlU:
			return false
		case tcell.KeyRune:
			switch event.Rune() {
			case ' ', 't':
				if row, _ := a.settingsTable.GetSelection(); row == 1 {
					a.toggleAutoRefresh()
					return true
				}
				if row, _ := a.settingsTable.GetSelection(); row == 2 {
					a.toggleBreadcrumbs()
					return true
				}
				return true
			case 'd':
				a.toggleDefaultSortOrder()
				return true
			case '+', '=':
				if row, _ := a.settingsTable.GetSelection(); row == 4 {
					a.adjustRefreshInterval(1)
					return true
				}
			case '-':
				if row, _ := a.settingsTable.GetSelection(); row == 4 {
					a.adjustRefreshInterval(-1)
					return true
				}
			case 'q':
				a.settingsTable = nil
				return false
			}
		}
		return false
	}

	if a.templateTree != nil {
		switch event.Key() {
		case tcell.KeyEnter:
			a.openSelectedTemplateTreeNode()
			return true
		case tcell.KeyEsc:
			a.templateTree = nil
			return false
		case tcell.KeyCtrlD, tcell.KeyCtrlU:
			return false
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				a.templateTree = nil
				return false
			}
		}
		return false
	}

	if a.entitySelectorTable != nil {
		switch event.Key() {
		case tcell.KeyEnter:
			a.applySelectedEntity()
			return true
		case tcell.KeyEsc:
			a.entitySelectorTable = nil
			a.templateTree = nil
			return false
		case tcell.KeyCtrlD, tcell.KeyCtrlU:
			return false
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				a.entitySelectorTable = nil
				return false
			case 'r', 't', 'i':
				a.entitySelectorTable = nil
				a.ui.CloseOverlay()
				a.handleNav(event.Rune())
				return true
			}
		}
		return false
	}

	if a.resourceColumnsTable != nil {
		switch event.Key() {
		case tcell.KeyEnter:
			a.toggleSelectedResourceColumn()
			return true
		case tcell.KeyEsc:
			a.resourceColumnsTable = nil
			return false
		case tcell.KeyCtrlD, tcell.KeyCtrlU:
			return false
		case tcell.KeyRune:
			switch event.Rune() {
			case ' ':
				a.toggleSelectedResourceColumn()
				return true
			case 'q':
				a.resourceColumnsTable = nil
				return false
			}
		}
		return false
	}

	if a.integrationFilterTable != nil {
		if a.integrationFilterMode {
			switch event.Key() {
			case tcell.KeyEsc:
				a.integrationFilterMode = false
				a.renderIntegrationFilterOverlay()
				return true
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				a.integrationFilterQuery = trimLastRune(a.integrationFilterQuery)
				a.renderIntegrationFilterOverlay()
				return true
			case tcell.KeyEnter:
				a.applySelectedIntegrationFilter()
				return true
			case tcell.KeyRune:
				if event.Rune() != '/' {
					a.integrationFilterQuery += string(event.Rune())
					a.renderIntegrationFilterOverlay()
				}
				return true
			}
		}

		switch event.Key() {
		case tcell.KeyEnter:
			a.applySelectedIntegrationFilter()
			return true
		case tcell.KeyEsc:
			a.integrationFilterAllRows = nil
			a.integrationFilterRows = nil
			a.integrationFilterTable = nil
			a.integrationFilterQuery = ""
			a.integrationFilterMode = false
			a.templateTree = nil
			return false
		case tcell.KeyCtrlD, tcell.KeyCtrlU:
			return false
		case tcell.KeyRune:
			switch event.Rune() {
			case '/':
				a.integrationFilterMode = true
				a.renderIntegrationFilterOverlay()
				return true
			case 'c':
				a.clearIntegrationFilter()
				return true
			case 'q':
				a.integrationFilterAllRows = nil
				a.integrationFilterRows = nil
				a.integrationFilterTable = nil
				a.integrationFilterQuery = ""
				a.integrationFilterMode = false
				a.templateTree = nil
				return false
			}
		}
		return false
	}

	if a.templateFilterTable != nil {
		if a.templateFilterMode {
			switch event.Key() {
			case tcell.KeyEsc:
				a.templateFilterMode = false
				a.renderTemplateFilterOverlay()
				return true
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				a.templateFilterQuery = trimLastRune(a.templateFilterQuery)
				a.renderTemplateFilterOverlay()
				return true
			case tcell.KeyEnter:
				a.applySelectedTemplateFilter()
				return true
			case tcell.KeyRune:
				if event.Rune() != '/' {
					a.templateFilterQuery += string(event.Rune())
					a.renderTemplateFilterOverlay()
				}
				return true
			}
		}

		switch event.Key() {
		case tcell.KeyEnter:
			a.applySelectedTemplateFilter()
			return true
		case tcell.KeyEsc:
			a.templateFilterAllRows = nil
			a.templateFilterRows = nil
			a.templateFilterTable = nil
			a.templateFilterQuery = ""
			a.templateFilterMode = false
			a.templateTree = nil
			return false
		case tcell.KeyCtrlD, tcell.KeyCtrlU:
			return false
		case tcell.KeyRune:
			switch event.Rune() {
			case '/':
				a.templateFilterMode = true
				a.renderTemplateFilterOverlay()
				return true
			case 'c':
				a.clearTemplateFilter()
				return true
			case 'q':
				a.templateFilterAllRows = nil
				a.templateFilterRows = nil
				a.templateFilterTable = nil
				a.templateFilterQuery = ""
				a.templateFilterMode = false
				a.templateTree = nil
				return false
			}
		}
		return false
	}

	if a.auditLogRows != nil {
		switch event.Key() {
		case tcell.KeyEnter:
			a.openSelectedAuditLog()
			return true
		case tcell.KeyRune:
			switch event.Rune() {
			case 'l':
				a.openSelectedAuditLog()
				return true
			case 'q':
				a.auditLogRows = nil
				a.auditLogTable = nil
				a.templateTree = nil
				return false
			}
		}
		return false
	}

	if event.Key() != tcell.KeyRune {
		return false
	}
	if event.Rune() == 't' && a.activeTemplateDetail != nil {
		a.openTemplateTree(a.activeTemplateDetail.ID, valueOr(a.activeTemplateDetail.Name, a.activeTemplateDetail.ID))
		return true
	}
	if event.Rune() != 't' || a.overlayTemplateJump == nil {
		return false
	}

	jump := a.overlayTemplateJump
	a.openTemplateOverview(jump.ID, valueOr(jump.Name, jump.ID))
	return true
}

func (a *App) openTemplateTree(id string, name string) {
	title := fmt.Sprintf("Template Tree: %s", name)
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.templateTree = nil
	a.ui.OpenOverlay(title, "Loading template tree...")

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		tree, err := a.client.TemplateTree(ctx, id, "children")
		var primitive tview.Primitive
		var treeView *tview.TreeView
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load template tree.\n\n%v", err))
		} else if tree != nil {
			primitive, treeView = templateTreeView(*tree)
		} else {
			primitive = errorView("Template tree not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.templateTree = treeView
			a.ui.OpenOverlayPrimitive(title, primitive)
		})
	}()
}

func (a *App) openSelectedTemplateTreeNode() {
	if a.templateTree == nil {
		return
	}
	node := a.templateTree.GetCurrentNode()
	if node == nil {
		return
	}
	reference := node.GetReference()
	selection, ok := reference.(client.TemplateTreeNode)
	if !ok || selection.ID == "" {
		return
	}
	a.templateTree = nil
	a.ui.CloseOverlay()
	a.openTemplateOverview(selection.ID, valueOr(selection.Name, selection.ID))
}

func (a *App) openLogs(row tabledata.Row) {
	if selection, ok := row.Raw.(auditLogSelection); ok {
		a.openAuditLogDetail(selection)
		return
	}

	resource, ok := row.Raw.(client.Resource)
	if !ok {
		return
	}

	title := fmt.Sprintf("Logs: %s", resource.Name)
	a.overlayTemplateJump = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading resource logs...")

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		logs, total, err := a.client.LogsForResource(ctx, resource.ID, []int{0, 200})
		text := "No logs"
		if err != nil {
			text = fmt.Sprintf("Failed to load resource logs.\n\n%v", err)
		} else {
			text = formatLogs(logs, total, a.config.NoColors)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.ui.OpenDetail(title, text)
		})
	}()
}

func (a *App) openAuditLogs(row tabledata.Row) {
	resource, ok := row.Raw.(client.Resource)
	if !ok {
		return
	}

	title := fmt.Sprintf("Audit Logs: %s", resource.Name)
	a.overlayTemplateJump = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading audit logs...")
	a.ui.SetAuditHeaderHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		auditLogs, err := a.client.AuditLogsForResource(ctx, resource.ID, []int{0, 200})
		var primitive tview.Primitive
		var rows []tabledata.Row
		var table *tview.Table
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load audit logs.\n\n%v", err))
		} else {
			primitive, rows, table = auditLogsView(resource, auditLogs)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.auditLogRows = rows
			a.auditLogTable = table
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetAuditHeaderHotkeys()
		})
	}()
}

func (a *App) openSelectedAuditLog() {
	if len(a.auditLogRows) == 0 {
		return
	}
	if a.auditLogTable == nil {
		return
	}
	selectedRow, _ := a.auditLogTable.GetSelection()
	if selectedRow <= 0 {
		return
	}
	index := selectedRow - 1
	if index < 0 || index >= len(a.auditLogRows) {
		return
	}
	selection, ok := a.auditLogRows[index].Raw.(auditLogSelection)
	if !ok {
		return
	}
	a.openAuditLogDetail(selection)
}

func (a *App) openAuditLogDetail(selection auditLogSelection) {
	title := fmt.Sprintf("Logs: %s / %s", selection.ResourceName, selection.Action)
	a.overlayTemplateJump = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading audit log details...")
	a.ui.SetAuditDetailHeaderHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		logs, total, err := a.client.LogsForAudit(ctx, selection.ResourceID, selection.AuditLogID, 0, []int{0, 400})
		text := "No logs"
		if err != nil {
			text = fmt.Sprintf("Failed to load audit log details.\n\n%v", err)
		} else {
			text = formatLogs(logs, total, a.config.NoColors)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.ui.OpenDetail(title, text)
			a.ui.SetAuditDetailHeaderHotkeys()
		})
	}()
}

func formatLogs(logs []client.Log, total int, noColors bool) string {
	if len(logs) == 0 {
		return "No logs found for this resource.\n\nEsc or q to close"
	}

	lines := []string{fmt.Sprintf("Showing %d of %d logs", len(logs), total), ""}
	for i := len(logs) - 1; i >= 0; i-- {
		lines = append(lines, formatLogRow(logs[i], noColors))
	}
	lines = append(lines, "", "Esc or q to close")
	return strings.Join(lines, "\n")
}

func formatLogRow(log client.Log, noColors bool) string {
	prefix := log.CreatedAt.Format(time.RFC3339) + "  "
	indent := strings.Repeat(" ", len(prefix))
	bodyLines := strings.Split(log.Data, "\n")
	for i, line := range bodyLines {
		bodyLines[i] = logLevelPrefixRX.ReplaceAllString(line, "")
	}
	body := strings.Join(bodyLines, "\n"+indent)
	if noColors {
		return prefix + body
	}
	colored := logLevelANSI(log.Level) + prefix + body + "\x1b[0m"
	return tview.TranslateANSI(colored)
}

func logLevelANSI(level string) string {
	switch strings.ToLower(level) {
	case "trace", "debug":
		return "\x1b[38;5;110m"
	case "warn", "warning":
		return "\x1b[33m"
	case "error", "fatal":
		return "\x1b[31m"
	default:
		return "\x1b[38;5;153m"
	}
}

func displayCreator(creator client.Creator) string {
	if creator.DisplayName != "" {
		return creator.DisplayName
	}
	if creator.Identifier != "" {
		return creator.Identifier
	}
	return creator.ID
}

func blankDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
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

func templateOverviewView(template client.Template) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", template.Name},
		{"ID", template.ID},
		{"Created", template.CreatedAt.Format(time.RFC3339)},
		{"Updated", template.UpdatedAt.Format(time.RFC3339)},
		{"Cloud Types", strings.Join(orSlice(template.CloudResourceTypes, []string{"-"}), ", ")},
	})

	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(template.Description, "-"))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(summary, 7, 0, true)
	root.AddItem(description, 0, 1, false)
	root.AddItem(overviewFooter("t tree view  Esc/q close"), 1, 0, false)
	return root
}

func templateTreeView(tree client.TemplateTreeNode) (tview.Primitive, *tview.TreeView) {
	view := tview.NewTreeView()
	view.SetBorder(true)
	view.SetTitle("Tree View")
	root := buildTemplateTreeNode(tree)
	root.SetExpanded(true)
	expandTemplateTree(root)
	view.SetRoot(root)
	view.SetCurrentNode(root)

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.AddItem(view, 0, 1, true)
	container.AddItem(overviewFooter("Enter open template  Esc/q close"), 1, 0, false)
	return container, view
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

func auditLogsView(resource client.Resource, auditLogs []client.AuditLog) (tview.Primitive, []tabledata.Row, *tview.Table) {
	headers := []string{"ACTION", "CREATOR", "REV", "CREATED", "AGE", "AUDIT ID"}
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 0)
	table.SetBorder(true)
	table.SetTitle("Audit Runs")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetBorderColor(tcell.ColorCadetBlue)

	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorCadetBlue).
			SetSelectable(false).
			SetExpansion(1))
	}

	rows := make([]tabledata.Row, 0, len(auditLogs))
	now := time.Now()
	for rowIndex, auditLog := range auditLogs {
		selection := auditLogSelection{
			ResourceID:     resource.ID,
			ResourceName:   resource.Name,
			AuditLogID:     auditLog.ID,
			Action:         blankDash(auditLog.Action),
			CreatedAt:      auditLog.CreatedAt,
			RevisionNumber: auditLog.RevisionNumber,
			Creator:        auditLogCreator(auditLog),
		}
		fields := []string{
			selection.Action,
			selection.Creator,
			fmt.Sprintf("%d", selection.RevisionNumber),
			selection.CreatedAt.Format(time.RFC3339),
			render.ToAge(selection.CreatedAt, now),
			selection.AuditLogID,
		}
		rows = append(rows, tabledata.Row{ID: selection.AuditLogID, Fields: fields, Raw: selection})
		for col, field := range fields {
			table.SetCell(rowIndex+1, col, tview.NewTableCell(field).
				SetTextColor(tcell.ColorLightSteelBlue).
				SetExpansion(1))
		}
	}

	if len(rows) == 0 {
		table.SetCell(1, 0, tview.NewTableCell("No audit logs found").SetTextColor(tcell.ColorCadetBlue))
	} else {
		table.Select(1, 0)
	}

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(table, 0, 1, true)
	root.AddItem(overviewFooter("Enter/l open logs  Esc/q close"), 1, 0, false)
	return root, rows, table
}

func overviewFooter(text string) tview.Primitive {
	view := tview.NewTextView()
	view.SetTextColor(tcell.ColorGray)
	view.SetText(text)
	return view
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

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
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

func settingsView(autoRefresh bool, defaultSortOrder string, refreshSeconds float64, showBreadcrumbs bool) (tview.Primitive, *tview.Table) {
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 0)
	table.SetBorder(true)
	table.SetTitle("Settings")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetBorderColor(tcell.ColorCadetBlue)

	headers := []string{"SETTING", "VALUE"}
	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorCadetBlue).
			SetSelectable(false).
			SetExpansion(1))
	}

	value := "off"
	if autoRefresh {
		value = "on"
	}
	breadcrumbsValue := "off"
	if showBreadcrumbs {
		breadcrumbsValue = "on"
	}
	sortOrder := strings.ToLower(strings.TrimSpace(defaultSortOrder))
	if sortOrder == "" {
		sortOrder = "desc"
	}

	table.SetCell(1, 0, tview.NewTableCell("Auto refresh").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(1, 1, tview.NewTableCell(value).SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(2, 0, tview.NewTableCell("Show breadcrumbs").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(2, 1, tview.NewTableCell(breadcrumbsValue).SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(3, 0, tview.NewTableCell("Default sort order").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(3, 1, tview.NewTableCell(sortOrder).SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(4, 0, tview.NewTableCell("Refresh interval (sec)").SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.SetCell(4, 1, tview.NewTableCell(formatRefreshSeconds(refreshSeconds)).SetTextColor(tcell.ColorLightSteelBlue).SetExpansion(1))
	table.Select(1, 0)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(table, 0, 1, true)
	root.AddItem(overviewFooter("Enter toggle/inc  space/t toggle  d sort order  +/- interval  Esc/q close"), 1, 0, false)
	return root, table
}

func resourceColumnsView(options []resourceColumnOption, visible map[string]bool) (tview.Primitive, *tview.Table) {
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 0)
	table.SetBorder(true)
	table.SetTitle("Columns")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetBorderColor(tcell.ColorCadetBlue)

	headers := []string{"COLUMN", "VISIBLE", "DESCRIPTION"}
	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorCadetBlue).
			SetSelectable(false).
			SetExpansion(1))
	}

	for rowIndex, option := range options {
		state := "off"
		if visible[option.Field] {
			state = "on"
		}
		fields := []string{option.Header.Title, state, option.Description}
		for col, field := range fields {
			table.SetCell(rowIndex+1, col, tview.NewTableCell(field).
				SetTextColor(tcell.ColorLightSteelBlue).
				SetExpansion(1))
		}
	}

	table.Select(1, 0)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(table, 0, 1, true)
	root.AddItem(overviewFooter("Enter/space toggle  Esc/q close"), 1, 0, false)
	return root, table
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

func formatRefreshSeconds(seconds float64) string {
	if seconds == float64(int64(seconds)) {
		return fmt.Sprintf("%.0f", seconds)
	}
	return fmt.Sprintf("%.1f", seconds)
}

func auditLogCreator(auditLog client.AuditLog) string {
	if auditLog.Creator == nil {
		return "-"
	}
	return displayCreator(*auditLog.Creator)
}

func errorView(message string) tview.Primitive {
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetScrollable(true)
	view.SetWrap(true)
	view.SetText(message)
	return view
}

func kvTable(title string, pairs [][2]string) tview.Primitive {
	table := tview.NewTable().SetBorders(false)
	table.SetBorder(true)
	table.SetTitle(title)
	for row, pair := range pairs {
		table.SetCell(row, 0, tview.NewTableCell(pair[0]).SetTextColor(tcell.ColorSteelBlue).SetSelectable(false))
		table.SetCell(row, 1, tview.NewTableCell(pair[1]).SetSelectable(false).SetExpansion(1))
	}
	return table
}

func simpleList(title string, lines []string) tview.Primitive {
	view := tview.NewTextView()
	view.SetBorder(true)
	view.SetTitle(title)
	view.SetWrap(true)
	if len(lines) == 0 {
		view.SetText("-")
	} else {
		view.SetText(strings.Join(lines, "\n"))
	}
	return view
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

func buildTemplateTreeNode(node client.TemplateTreeNode) *tview.TreeNode {
	label := node.Name
	if node.Status != "" {
		label = fmt.Sprintf("%s [%s]", node.Name, node.Status)
	}
	treeNode := tview.NewTreeNode(label)
	treeNode.SetReference(node)
	for _, child := range node.Children {
		treeNode.AddChild(buildTemplateTreeNode(child))
	}
	return treeNode
}

func expandTemplateTree(node *tview.TreeNode) {
	if node == nil {
		return
	}
	node.SetExpanded(true)
	for _, child := range node.GetChildren() {
		expandTemplateTree(child)
	}
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

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return "-"
	case string:
		return blankDash(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, stringify(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprint(v)
	}
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

func valueOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func orSlice(values []string, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}
	return values
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

func (a *App) VersionString() string {
	return fmt.Sprintf("%s (%s %s)", a.build.Version, a.build.Commit, a.build.Date)
}
