package app

import (
	"context"
	"fmt"
	"regexp"
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

type auditLogSelection struct {
	EntityID       string
	EntityName     string
	EntityLabel    string
	AuditLogID     string
	Action         string
	CreatedAt      time.Time
	RevisionNumber int
	Creator        string
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
	templateColumnsTable      *tview.Table
	visibleTemplateColumns    map[string]bool
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
	app.visibleTemplateColumns = defaultVisibleTemplateColumns()
	app.applySavedViewPreferences()

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
	if resourcesModel := app.models[model.EntityResources]; resourcesModel != nil {
		resourcesModel.SetFilter(app.resourceFilters())
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
	ui.SetResourceColumnsFunc(app.openColumns)
	ui.SetToggleDestroyedFunc(app.toggleHideDestroyedResources)
	ui.SetResetFiltersFunc(app.resetAllResourceFilters)
	ui.SetCommandFunc(app.runCommand)
	ui.SetCommandSuggestFunc(app.suggestCommand)
	ui.SetEntitySelectorFunc(app.openEntitySelector)
	ui.SetSettingsFunc(app.openSettings)
	ui.SetOverlayKeyFunc(app.handleOverlayKey)
	ui.SetDetailKeyFunc(app.handleOverlayKey)
	ui.SetEntityTitle(app.currentEntityTitle(), entityEmptyLabel(app.activeKind))

	return app
}

func (a *App) Run() error {
	defer a.cancel()
	a.renderCurrentModel()
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
	if a.activeKind == model.EntityTemplates {
		headers, rows = a.projectTemplateList(headers, rows)
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
	if a.activeKind == model.EntityTemplates {
		headers, rows = a.projectTemplateList(headers, rows)
	}
	sortColumn, sortAsc := entityModel.SortStateForHeaders(headers)
	a.ui.SetEntityTitle(a.currentEntityTitle(), entityEmptyLabel(a.activeKind))
	a.ui.Update(headers, rows, len(rows), total, lastUpdated, lastErr)
	a.ui.SetSortState(sortColumn, sortAsc)
}

func (a *App) currentModel() *model.EntityModel {
	return a.models[a.activeKind]
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

	if a.templateColumnsTable != nil {
		switch event.Key() {
		case tcell.KeyEnter:
			a.toggleSelectedTemplateColumn()
			return true
		case tcell.KeyEsc:
			a.templateColumnsTable = nil
			return false
		case tcell.KeyCtrlD, tcell.KeyCtrlU:
			return false
		case tcell.KeyRune:
			switch event.Rune() {
			case ' ':
				a.toggleSelectedTemplateColumn()
				return true
			case 'q':
				a.templateColumnsTable = nil
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

func (a *App) openLogs(row tabledata.Row) {
	if selection, ok := row.Raw.(auditLogSelection); ok {
		a.openAuditLogDetail(selection)
		return
	}

	entityID, entityName, entityLabel, ok := auditEntityRowMeta(row)
	if !ok {
		return
	}

	title := fmt.Sprintf("Logs: %s", entityName)
	a.overlayTemplateJump = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, fmt.Sprintf("Loading %s logs...", strings.ToLower(entityLabel)))
	a.ui.SetDetailHotkeys()

	go func(entityID string, entityLabel string) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		logs, total, err := a.client.LogsForEntity(ctx, entityID, []int{0, 200})
		text := "No logs"
		if err != nil {
			text = fmt.Sprintf("Failed to load %s logs.\n\n%v", strings.ToLower(entityLabel), err)
		} else {
			text = formatLogs(logs, total, a.config.NoColors, entityLabel)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.ui.OpenDetail(title, text)
			a.ui.SetDetailHotkeys()
		})
	}(entityID, entityLabel)
}

func (a *App) openAuditLogs(row tabledata.Row) {
	entityID, entityName, entityLabel, ok := auditEntityRowMeta(row)
	if !ok {
		return
	}

	title := fmt.Sprintf("Audit Logs: %s", entityName)
	a.overlayTemplateJump = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading audit logs...")
	a.ui.SetAuditHeaderHotkeys()

	go func(entityID string, entityName string, entityLabel string) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		auditLogs, err := a.client.AuditLogsForEntity(ctx, entityID, []int{0, 200})
		var primitive tview.Primitive
		var rows []tabledata.Row
		var table *tview.Table
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load audit logs.\n\n%v", err))
		} else {
			primitive, rows, table = auditLogsView(entityID, entityName, entityLabel, auditLogs)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.auditLogRows = rows
			a.auditLogTable = table
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetAuditHeaderHotkeys()
		})
	}(entityID, entityName, entityLabel)
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
	title := fmt.Sprintf("Logs: %s / %s", selection.EntityName, selection.Action)
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

		logs, total, err := a.client.LogsForAudit(ctx, selection.EntityID, selection.AuditLogID, 0, []int{0, 400})
		text := "No logs"
		if err != nil {
			text = fmt.Sprintf("Failed to load audit log details.\n\n%v", err)
		} else {
			text = formatLogs(logs, total, a.config.NoColors, selection.EntityLabel)
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.ui.OpenDetail(title, text)
			a.ui.SetAuditDetailHeaderHotkeys()
		})
	}()
}

func formatLogs(logs []client.Log, total int, noColors bool, entityLabel string) string {
	if len(logs) == 0 {
		return fmt.Sprintf("No logs found for this %s.\n\nEsc or q to close", strings.ToLower(blankDash(entityLabel)))
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

func auditLogsView(entityID string, entityName string, entityLabel string, auditLogs []client.AuditLog) (tview.Primitive, []tabledata.Row, *tview.Table) {
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
			EntityID:       entityID,
			EntityName:     entityName,
			EntityLabel:    entityLabel,
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

func auditEntityRowMeta(row tabledata.Row) (entityID string, entityName string, entityLabel string, ok bool) {
	switch value := row.Raw.(type) {
	case client.Resource:
		return value.ID, value.Name, "resource", true
	case client.Template:
		return value.ID, value.Name, "template", true
	case client.Integration:
		return value.ID, value.Name, "integration", true
	default:
		return "", "", "", false
	}
}

func overviewFooter(text string) tview.Primitive {
	view := tview.NewTextView()
	view.SetTextColor(tcell.ColorGray)
	view.SetText(text)
	return view
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
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

func (a *App) VersionString() string {
	return fmt.Sprintf("%s (%s %s)", a.build.Version, a.build.Commit, a.build.Date)
}
