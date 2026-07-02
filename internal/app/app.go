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
	ResourceID     string
	ResourceName   string
	AuditLogID     string
	Action         string
	CreatedAt      time.Time
	RevisionNumber int
	Creator        string
}

var logLevelPrefixRX = regexp.MustCompile(`(?i)^\[(trace|debug|info|warn|warning|error|fatal)\]\s*`)

type App struct {
	config              config.Config
	build               BuildInfo
	client              *client.Client
	models              map[model.EntityKind]*model.EntityModel
	registry            *resource.Registry
	kindByName          map[string]model.EntityKind
	nameByKind          map[model.EntityKind]string
	activeKind          model.EntityKind
	ui                  *uiapp.App
	manualKick          chan struct{}
	ctx                 context.Context
	cancel              context.CancelFunc
	overlayTemplateJump *overlayTemplateJump
	auditLogRows        []tabledata.Row
	auditLogTable       *tview.Table
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
		config:     cfg,
		build:      build,
		client:     cli,
		models:     map[model.EntityKind]*model.EntityModel{},
		registry:   registry,
		kindByName: map[string]model.EntityKind{},
		nameByKind: map[model.EntityKind]string{},
		activeKind: model.EntityResources,
		ui:         ui,
		manualKick: make(chan struct{}, 1),
		ctx:        ctx,
		cancel:     cancel,
	}

	ordered := registry.Ordered()
	for index, descriptor := range ordered {
		kind := model.EntityKind(descriptor.Name)
		app.models[kind] = model.NewModelFromDescriptor(kind, descriptor)
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
	ui.SetOverlayKeyFunc(app.handleOverlayKey)
	ui.SetDetailKeyFunc(app.handleOverlayKey)
	ui.SetEntityTitle(entityTitle(app.activeKind), entityEmptyLabel(app.activeKind))

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
	ticker := time.NewTicker(a.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.refresh()
		case <-a.manualKick:
			a.refresh()
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	user, err := a.client.CurrentUser(ctx)
	if err != nil || user == nil {
		return
	}

	a.ui.SetHeaderUser(user.Identifier, user.DisplayName, user.Email)
}

func (a *App) requestLoadMore() {
	entityModel := a.currentModel()
	if entityModel.LoadingMore() || !entityModel.HasMore() {
		return
	}

	go func(kind model.EntityKind, m *model.EntityModel) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	model := a.currentModel()
	_ = model.Refresh(ctx)
	headers, rows, total, lastUpdated, lastErr := model.Snapshot()
	sortColumn, sortAsc := model.SortState()
	a.ui.Application().QueueUpdateDraw(func() {
		a.ui.SetEntityTitle(entityTitle(a.activeKind), entityEmptyLabel(a.activeKind))
		a.ui.Update(headers, rows, len(rows), total, lastUpdated, lastErr)
		a.ui.SetSortState(sortColumn, sortAsc)
	})
}

func (a *App) refreshInitial() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	model := a.currentModel()
	_ = model.Refresh(ctx)
	a.renderCurrentModel()
}

func (a *App) renderCurrentModel() {
	model := a.currentModel()
	headers, rows, total, lastUpdated, lastErr := model.Snapshot()
	sortColumn, sortAsc := model.SortState()
	a.ui.SetEntityTitle(entityTitle(a.activeKind), entityEmptyLabel(a.activeKind))
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
	a.ui.CloseOverlay()
	a.renderCurrentModel()
	a.requestRefresh()
}

func (a *App) handleSort(column int, asc bool) {
	if !a.currentModel().SetSortByColumn(column, asc) {
		return
	}
	a.requestRefresh()
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
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading resource overview...")

	go func() {
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
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading template overview...")

	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Template(ctx, id)
		var primitive tview.Primitive
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load template overview.\n\n%v", err))
		} else if full != nil {
			primitive = templateOverviewView(*full)
		} else {
			primitive = errorView("Template not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.ui.OpenDetailPrimitive(title, primitive)
		})
	}()
}

func (a *App) openIntegrationOverview(id string, name string) {
	title := fmt.Sprintf("Integration: %s", name)
	a.overlayTemplateJump = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading integration overview...")

	go func() {
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
				return false
			}
		}
		return false
	}

	if event.Key() != tcell.KeyRune {
		return false
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
	root.AddItem(overviewFooter("Esc/q close"), 1, 0, false)
	return root
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
