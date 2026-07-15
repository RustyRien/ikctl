package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	app                 *tview.Application
	header              *Header
	content             *tview.Pages
	table               *Table
	status              *tview.TextView
	main                *tview.Flex
	pages               *tview.Pages
	overlay             *tview.Flex
	overlayBox          tview.Primitive
	detailBox           tview.Primitive
	detailHistory       []detailPage
	filterMode          bool
	filterMenuMode      bool
	commandMode         bool
	filterText          string
	commandText         string
	commandMatches      []string
	config              config.Config
	version             string
	root                tview.Primitive
	filterDone          func(string)
	refreshFn           func()
	enterFn             func(tabledata.Row)
	yamlFn              func()
	logsFn              func(tabledata.Row)
	auditFn             func(tabledata.Row)
	enableFn            func(tabledata.Row)
	disableFn           func(tabledata.Row)
	deleteFn            func(tabledata.Row)
	editFn              func(tabledata.Row)
	navFn               func(rune)
	sortFn              func(int, bool)
	loadMoreFn          func()
	templateFilterFn    func()
	integrationFilterFn func()
	resourceColumnsFn   func()
	entitySelectorFn    func()
	settingsFn          func()
	toggleDestroyedFn   func()
	resetFiltersFn      func()
	commandFn           func(string)
	commandSuggestFn    func(string) (string, []string)
	overlayKeyFn        func(*tcell.EventKey) bool
	detailKeyFn         func(*tcell.EventKey) bool
	detailClosedFn      func()
	statusBase          string
	loadingMx           sync.Mutex
	loadingCount        int
	loadingFrame        int
	loadingStop         chan struct{}
	loadingStopOnce     sync.Once
	listTitle           string
}

type detailPage struct {
	title     string
	primitive tview.Primitive
	hotkeys   detailHotkeys
}

type detailHotkeys int

const (
	detailHotkeysDefault detailHotkeys = iota
	detailHotkeysDetail
	detailHotkeysAudit
	detailHotkeysAuditDetail
	detailHotkeysResourceOverview
	detailHotkeysTemplateOverview
	detailHotkeysIntegrationOverview
)

var loadingFrames = []string{"|", "/", "-", "\\"}

func NewApp(cfg config.Config, version string) *App {
	applyColors(!cfg.NoColors)
	app := &App{
		app:         tview.NewApplication(),
		header:      NewHeader(cfg, version),
		table:       NewTable(),
		status:      tview.NewTextView(),
		config:      cfg,
		version:     version,
		loadingStop: make(chan struct{}),
	}

	app.status.SetDynamicColors(true)
	app.status.SetTextColor(colorInfo)
	app.status.SetBackgroundColor(colorBg)
	app.status.SetBorder(true)
	app.status.SetTitle("Status")
	app.status.SetBorderColor(colorHeader)
	app.overlay = tview.NewFlex().SetDirection(tview.FlexRow)
	app.overlay.SetBorder(true)
	app.overlay.SetTitle("Details")
	app.overlay.SetBackgroundColor(colorBg)
	app.overlay.SetBorderColor(colorHeader)
	app.content = tview.NewPages().AddPage("list", app.table.Widget(), true, true)

	main := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(app.header.Primitive(), headerHeight(cfg.ShowBreadcrumbs), 0, false).
		AddItem(app.content, 0, 1, true).
		AddItem(app.status, 3, 0, false)
	main.SetBackgroundColor(colorBg)
	app.main = main
	app.pages = tview.NewPages().
		AddPage("main", main, true, true).
		AddPage("overlay", centered(app.overlay, 140, 36), true, false)

	app.root = app.pages
	app.app.SetRoot(app.root, true)
	app.app.SetInputCapture(app.capture)
	app.table.Widget().SetBorder(true).SetTitle("Resources")
	app.table.Widget().SetBackgroundColor(colorBg)
	app.table.Widget().SetBorderColor(colorHeader)
	app.table.SetEmptyLabel("No resources")
	app.table.SetSelectionChangedFunc(app.handleSelectionChanged)
	app.listTitle = "Resources"
	app.updateBreadcrumbs()
	go app.runLoadingSpinner()

	return app
}

func (a *App) Application() *tview.Application {
	return a.app
}

func (a *App) Run() error {
	return a.app.Run()
}

func (a *App) Stop() {
	a.loadingStopOnce.Do(func() {
		close(a.loadingStop)
	})
	a.app.Stop()
}

func (a *App) BeginLoading() func() {
	a.loadingMx.Lock()
	a.loadingCount++
	if a.loadingCount == 1 {
		a.loadingFrame = 0
	}
	a.loadingMx.Unlock()
	a.queueRenderStatus()

	var once sync.Once
	return func() {
		once.Do(func() {
			a.loadingMx.Lock()
			if a.loadingCount > 0 {
				a.loadingCount--
			}
			if a.loadingCount == 0 {
				a.loadingFrame = 0
			}
			a.loadingMx.Unlock()
			a.queueRenderStatus()
		})
	}
}

func (a *App) Update(headers []tabledata.Header, rows []tabledata.Row, shown int, total int, lastUpdated time.Time, lastErr error) {
	a.table.SetData(headers, rows)
	status := fmt.Sprintf("Shown: %d / %d", shown, total)
	if !lastUpdated.IsZero() {
		status += fmt.Sprintf("  Last refresh: %s", lastUpdated.Format(time.Kitchen))
	}
	if filter := a.table.Filter(); filter != "" {
		status += fmt.Sprintf("  Filter: %s", filter)
	}
	if lastErr != nil {
		if a.config.NoColors {
			status += fmt.Sprintf("  Error: %v", lastErr)
		} else {
			status += fmt.Sprintf("  [red]Error[-]: %v", lastErr)
		}
	}
	a.statusBase = status
	a.renderStatus()
}

func (a *App) SetSortState(column int, asc bool) {
	a.table.SetSortState(column, asc)
	a.renderStatus()
}

func (a *App) SetHeaderUser(identifier string, displayName string, email string) {
	a.header.SetUser(identifier, displayName, email)
}

func (a *App) SetBreadcrumbsVisible(visible bool) {
	a.header.SetBreadcrumbsVisible(visible)
	if a.main != nil {
		a.main.ResizeItem(a.header.Primitive(), headerHeight(visible), 0)
	}
	a.updateBreadcrumbs()
}

func (a *App) SetFilterDone(fn func(string)) {
	a.filterDone = fn
}

func (a *App) SetRefreshFunc(fn func()) {
	a.refreshFn = fn
}

func (a *App) SetEnterFunc(fn func(tabledata.Row)) {
	a.enterFn = fn
}

func (a *App) SetYAMLFunc(fn func()) {
	a.yamlFn = fn
}

func (a *App) SetLogsFunc(fn func(tabledata.Row)) {
	a.logsFn = fn
}

func (a *App) SetAuditFunc(fn func(tabledata.Row)) {
	a.auditFn = fn
}

func (a *App) SetEnableFunc(fn func(tabledata.Row)) {
	a.enableFn = fn
}

func (a *App) SetDisableFunc(fn func(tabledata.Row)) {
	a.disableFn = fn
}

func (a *App) SetDeleteFunc(fn func(tabledata.Row)) {
	a.deleteFn = fn
}

func (a *App) SetEditFunc(fn func(tabledata.Row)) {
	a.editFn = fn
}

func (a *App) SetNavFunc(fn func(rune)) {
	a.navFn = fn
}

func (a *App) SetSortFunc(fn func(int, bool)) {
	a.sortFn = fn
}

func (a *App) SetLoadMoreFunc(fn func()) {
	a.loadMoreFn = fn
}

func (a *App) SetTemplateFilterFunc(fn func()) {
	a.templateFilterFn = fn
}

func (a *App) SetIntegrationFilterFunc(fn func()) {
	a.integrationFilterFn = fn
}

func (a *App) SetResourceColumnsFunc(fn func()) {
	a.resourceColumnsFn = fn
}

func (a *App) SetEntitySelectorFunc(fn func()) {
	a.entitySelectorFn = fn
}

func (a *App) SetSettingsFunc(fn func()) {
	a.settingsFn = fn
}

func (a *App) SetToggleDestroyedFunc(fn func()) {
	a.toggleDestroyedFn = fn
}

func (a *App) SetResetFiltersFunc(fn func()) {
	a.resetFiltersFn = fn
}

func (a *App) SetCommandFunc(fn func(string)) {
	a.commandFn = fn
}

func (a *App) SetCommandSuggestFunc(fn func(string) (string, []string)) {
	a.commandSuggestFn = fn
}

func (a *App) SetOverlayKeyFunc(fn func(*tcell.EventKey) bool) {
	a.overlayKeyFn = fn
}

func (a *App) SetDetailKeyFunc(fn func(*tcell.EventKey) bool) {
	a.detailKeyFn = fn
}

func (a *App) SetDetailClosedFunc(fn func()) {
	a.detailClosedFn = fn
}

func (a *App) SetEntityTitle(title string, emptyLabel string) {
	a.listTitle = title
	a.table.Widget().SetTitle(title)
	a.table.SetEmptyLabel(emptyLabel)
	a.updateBreadcrumbs()
}

func (a *App) SelectedRow() (tabledata.Row, bool) {
	return a.table.SelectedRow()
}

func (a *App) DetailVisible() bool {
	return a.detailVisible()
}

func (a *App) OverlayVisible() bool {
	return a.overlayVisible()
}

func (a *App) OpenOverlay(title string, text string) {
	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetScrollable(true)
	textView.SetWrap(true)
	textView.SetText(text)
	a.setOverlayContent(title, textView)
	a.pages.ShowPage("overlay")
	a.updateBreadcrumbs()
	a.app.SetFocus(textView)
}

func (a *App) UpdateOverlay(title string, text string) {
	textView, ok := a.overlayBox.(*tview.TextView)
	if !ok {
		textView = tview.NewTextView()
		textView.SetDynamicColors(true)
		textView.SetScrollable(true)
		textView.SetWrap(true)
		a.setOverlayContent(title, textView)
	}
	a.overlay.SetTitle(title)
	textView.SetText(text)
	textView.ScrollToBeginning()
	if !a.pages.HasPage("overlay") {
		return
	}
	a.pages.ShowPage("overlay")
	a.updateBreadcrumbs()
}

func (a *App) OpenOverlayPrimitive(title string, primitive tview.Primitive) {
	a.OpenOverlayPrimitiveWithFocus(title, primitive, primitive)
}

func (a *App) OpenOverlayPrimitiveWithFocus(title string, primitive tview.Primitive, focus tview.Primitive) {
	if focus == nil {
		focus = primitive
	}
	a.setOverlayContent(title, primitive)
	a.pages.ShowPage("overlay")
	a.updateBreadcrumbs()
	a.app.SetFocus(focus)
}

func (a *App) OpenDetail(title string, text string) {
	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetScrollable(true)
	textView.SetWrap(true)
	textView.SetText(text)
	a.setDetailContent(title, textView)
	a.app.SetFocus(textView)
}

func (a *App) OpenDetailPrimitive(title string, primitive tview.Primitive) {
	a.setDetailContent(title, primitive)
	a.app.SetFocus(primitive)
}

func (a *App) ResetHeaderHotkeys() {
	a.filterMenuMode = false
	a.header.ResetHotkeys()
}

func (a *App) SetAuditHeaderHotkeys() {
	a.header.SetAuditHotkeys()
	a.updateDetailHotkeys(detailHotkeysAudit)
}

func (a *App) SetAuditDetailHeaderHotkeys() {
	a.header.SetAuditDetailHotkeys()
	a.updateDetailHotkeys(detailHotkeysAuditDetail)
}

func (a *App) SetDetailHotkeys() {
	a.filterMenuMode = false
	a.header.SetDetailHotkeys()
	a.updateDetailHotkeys(detailHotkeysDetail)
}

func (a *App) SetResourceOverviewHotkeys() {
	a.filterMenuMode = false
	a.header.SetResourceOverviewHotkeys()
	a.updateDetailHotkeys(detailHotkeysResourceOverview)
}

func (a *App) SetTemplateOverviewHotkeys() {
	a.filterMenuMode = false
	a.header.SetTemplateOverviewHotkeys()
	a.updateDetailHotkeys(detailHotkeysTemplateOverview)
}

func (a *App) SetIntegrationOverviewHotkeys() {
	a.filterMenuMode = false
	a.header.SetIntegrationOverviewHotkeys()
	a.updateDetailHotkeys(detailHotkeysIntegrationOverview)
}

func (a *App) CloseDetail() {
	if a.detailClosedFn != nil {
		a.detailClosedFn()
	}
	if len(a.detailHistory) > 1 {
		a.detailHistory = a.detailHistory[:len(a.detailHistory)-1]
		page := a.detailHistory[len(a.detailHistory)-1]
		a.restoreDetailPage(page)
		return
	}
	a.detailHistory = nil
	a.content.HidePage("detail")
	a.content.SwitchToPage("list")
	a.ResetHeaderHotkeys()
	a.updateBreadcrumbs()
	a.app.SetFocus(a.table.Widget())
}

func (a *App) CloseOverlay() {
	a.pages.HidePage("overlay")
	a.updateBreadcrumbs()
	if a.detailVisible() && a.detailBox != nil {
		a.app.SetFocus(a.detailBox)
		return
	}
	a.app.SetFocus(a.table.Widget())
}

func (a *App) overlayVisible() bool {
	name, _ := a.pages.GetFrontPage()
	return name == "overlay"
}

func (a *App) detailVisible() bool {
	name, _ := a.content.GetFrontPage()
	return name == "detail"
}

func (a *App) capture(event *tcell.EventKey) *tcell.EventKey {
	if a.commandMode {
		switch event.Key() {
		case tcell.KeyEsc:
			a.commandMode = false
			a.commandText = ""
			a.commandMatches = nil
			a.status.SetTitle("Status")
			a.renderStatus()
			return nil
		case tcell.KeyEnter:
			command := strings.TrimSpace(a.commandText)
			a.commandMode = false
			a.commandText = ""
			a.commandMatches = nil
			a.status.SetTitle("Status")
			a.renderStatus()
			if command != "" && a.commandFn != nil {
				a.commandFn(command)
			}
			return nil
		case tcell.KeyTAB:
			if a.commandSuggestFn != nil {
				a.commandText, a.commandMatches = a.commandSuggestFn(a.commandText)
			}
			a.renderStatus()
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(a.commandText) > 0 {
				a.commandText = a.commandText[:len(a.commandText)-1]
			}
			a.commandMatches = nil
			a.status.SetText(a.withLoadingSuffix(":" + a.commandText))
			return nil
		case tcell.KeyRune:
			a.commandText += string(event.Rune())
			a.commandMatches = nil
			a.status.SetText(a.withLoadingSuffix(":" + a.commandText))
			return nil
		default:
			return nil
		}
	}

	if a.filterMode {
		switch event.Key() {
		case tcell.KeyEsc:
			a.filterMode = false
			a.status.SetTitle("Status")
			a.renderStatus()
			return nil
		case tcell.KeyEnter:
			a.filterMode = false
			a.table.SetFilter(a.filterText)
			if a.filterDone != nil {
				a.filterDone(a.filterText)
			}
			a.status.SetTitle("Status")
			a.renderStatus()
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(a.filterText) > 0 {
				a.filterText = a.filterText[:len(a.filterText)-1]
			}
			a.status.SetText("Filter: " + a.filterText)
			return nil
		case tcell.KeyRune:
			a.filterText += string(event.Rune())
			a.status.SetText("Filter: " + a.filterText)
			return nil
		default:
			return nil
		}
	}

	if a.filterMenuMode {
		switch event.Key() {
		case tcell.KeyEsc:
			a.exitFilterMenuMode()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'i':
				a.exitFilterMenuMode()
				if a.integrationFilterFn != nil {
					a.integrationFilterFn()
				}
				return nil
			case 't':
				a.exitFilterMenuMode()
				if a.templateFilterFn != nil {
					a.templateFilterFn()
				}
				return nil
			case 'd':
				a.exitFilterMenuMode()
				if a.toggleDestroyedFn != nil {
					a.toggleDestroyedFn()
				}
				return nil
			case 'c':
				a.exitFilterMenuMode()
				if a.resetFiltersFn != nil {
					a.resetFiltersFn()
				}
				return nil
			case 'q':
				a.exitFilterMenuMode()
				return nil
			}
			return nil
		default:
			return nil
		}
	}

	if a.table.SortMode() {
		switch event.Key() {
		case tcell.KeyEsc:
			a.table.CancelSortMode()
			a.status.SetTitle("Status")
			a.renderStatus()
			return nil
		case tcell.KeyRune:
			if !a.table.AwaitingSortDirection() {
				if a.table.ApplySortDigit(event.Rune()) {
					a.status.SetTitle("Sort")
					a.renderStatus()
					return nil
				}
				return nil
			}
			if a.table.ApplySortDirection(event.Rune()) {
				if a.sortFn != nil {
					if column, ok := a.table.PendingSortColumn(); ok {
						a.sortFn(column, event.Rune() == 'a')
					}
				}
				a.table.FinishSortMode()
				a.status.SetTitle("Status")
				a.renderStatus()
				return nil
			}
			return nil
		default:
			return nil
		}
	}

	if a.table.EntityMode() {
		switch event.Key() {
		case tcell.KeyEsc:
			a.table.CancelEntityMode()
			a.status.SetTitle("Status")
			a.renderStatus()
			return nil
		case tcell.KeyRune:
			if entity, ok := a.table.EntityForKey(event.Rune()); ok {
				if a.navFn != nil {
					a.navFn(entity)
				}
				a.table.FinishEntityMode()
				a.status.SetTitle("Status")
				a.renderStatus()
				return nil
			}
			return nil
		default:
			return nil
		}
	}

	if event.Key() == tcell.KeyRune && event.Rune() == ':' {
		a.enterCommandMode()
		return nil
	}

	if a.overlayVisible() {
		if remapped := remapHalfPageScroll(event); remapped != nil {
			return remapped
		}
		if a.overlayKeyFn != nil && a.overlayKeyFn(event) {
			return nil
		}
		switch event.Key() {
		case tcell.KeyEsc:
			a.CloseOverlay()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				a.CloseOverlay()
				return nil
			}
		}
		return event
	}

	if a.detailVisible() {
		if remapped := remapHalfPageScroll(event); remapped != nil {
			return remapped
		}
		if a.detailKeyFn != nil && a.detailKeyFn(event) {
			return nil
		}
		switch event.Key() {
		case tcell.KeyEsc:
			a.CloseDetail()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				a.CloseDetail()
				return nil
			}
			if event.Rune() == 'x' && a.disableFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.disableFn(row)
					return nil
				}
			}
			if event.Rune() == 'X' && a.enableFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.enableFn(row)
					return nil
				}
			}
			if event.Rune() == 'D' && a.deleteFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.deleteFn(row)
					return nil
				}
			}
			if event.Rune() == 'E' && a.editFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.editFn(row)
					return nil
				}
			}
			if event.Rune() == 'l' && a.logsFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.logsFn(row)
					return nil
				}
			}
			if event.Rune() == 'a' && a.auditFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.auditFn(row)
					return nil
				}
			}
			if event.Rune() == 'y' && a.yamlFn != nil {
				a.yamlFn()
				return nil
			}
		}
		if event.Key() == tcell.KeyEnter && a.enterFn != nil {
			if row, ok := a.table.SelectedRow(); ok {
				a.enterFn(row)
				return nil
			}
		}
		return event
	}

	switch event.Key() {
	case tcell.KeyCtrlC:
		a.Stop()
		return nil
	case tcell.KeyCtrlD:
		a.table.MoveHalfPage(1)
		return nil
	case tcell.KeyCtrlU:
		a.table.MoveHalfPage(-1)
		return nil
	case tcell.KeyEnter:
		if a.enterFn != nil {
			if row, ok := a.table.SelectedRow(); ok {
				a.enterFn(row)
			}
		}
		return nil
	case tcell.KeyRune:
		if event.Rune() == 0x04 {
			a.table.MoveHalfPage(1)
			return nil
		}
		if event.Rune() == 0x15 {
			a.table.MoveHalfPage(-1)
			return nil
		}
		switch event.Rune() {
		case 'q':
			a.Stop()
			return nil
		case '/':
			a.filterMode = true
			a.filterText = a.table.Filter()
			a.status.SetTitle("Filter")
			a.status.SetText("Filter: " + a.filterText)
			return nil
		case 's':
			if a.table.StartSortMode() {
				a.status.SetTitle("Sort")
				a.status.SetText("Choose column: " + a.table.SortHints() + "  Esc cancel")
			}
			return nil
		case 'e':
			if a.entitySelectorFn != nil {
				a.entitySelectorFn()
				return nil
			}
			return nil
		case 'o':
			if a.settingsFn != nil {
				a.settingsFn()
				return nil
			}
			return nil
		case 'f':
			a.enterFilterMenuMode()
			return nil
		case 'c':
			if a.resetFiltersFn != nil {
				a.resetFiltersFn()
				return nil
			}
		case 'r':
			if a.refreshFn != nil {
				a.refreshFn()
			}
			return nil
		case 'v':
			if a.resourceColumnsFn != nil {
				a.resourceColumnsFn()
				return nil
			}
		case 'l':
			if a.logsFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.logsFn(row)
				}
			}
			return nil
		case 'y':
			if a.yamlFn != nil {
				a.yamlFn()
			}
			return nil
		case 'a':
			if a.auditFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.auditFn(row)
				}
			}
			return nil
		case 'x':
			if a.disableFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.disableFn(row)
				}
			}
			return nil
		case 'X':
			if a.enableFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.enableFn(row)
				}
			}
			return nil
		case 'D':
			if a.deleteFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.deleteFn(row)
				}
			}
			return nil
		case 'E':
			if a.editFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.editFn(row)
				}
			}
			return nil
		}
	}

	return event
}

func (a *App) renderStatus() {
	if a.commandMode {
		status := ":" + a.commandText
		if len(a.commandMatches) > 0 {
			status += "  Matches: " + strings.Join(a.commandMatches, ", ")
		}
		a.status.SetText(a.withLoadingSuffix(status))
		return
	}
	if a.filterMode {
		a.status.SetText(a.withLoadingSuffix("Filter: " + a.filterText))
		return
	}
	if a.filterMenuMode {
		a.status.SetText(a.withLoadingSuffix("Choose filter: i integration, t template, d hide destroyed, c reset all  Esc back"))
		return
	}
	if a.table.SortMode() {
		a.status.SetText(a.withLoadingSuffix("Choose column: " + a.table.SortHints() + "  Esc cancel"))
		if a.table.AwaitingSortDirection() {
			a.status.SetText(a.withLoadingSuffix("Choose direction: " + a.table.SortHints() + "  Esc cancel"))
		}
		return
	}
	if a.table.EntityMode() {
		a.status.SetText(a.withLoadingSuffix("Choose entity: " + a.table.EntityHints() + "  Esc cancel"))
		return
	}
	a.status.SetText(a.withLoadingSuffix(a.statusBase))
}

func (a *App) handleSelectionChanged(row int, _ int) {
	if a.loadMoreFn == nil || row <= 0 {
		return
	}
	visible := a.table.VisibleCount()
	if visible == 0 {
		return
	}
	if visible-row <= 5 {
		a.loadMoreFn()
	}
}

func centered(p tview.Primitive, width int, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

func (a *App) setOverlayContent(title string, primitive tview.Primitive) {
	a.overlay.Clear()
	a.overlay.SetTitle(title)
	a.overlay.AddItem(primitive, 0, 1, true)
	a.overlayBox = primitive
}

func (a *App) setDetailContent(title string, primitive tview.Primitive) {
	a.pushDetailPage(title, primitive)
	a.showDetailPage(title, primitive)
}

func (a *App) pushDetailPage(title string, primitive tview.Primitive) {
	if len(a.detailHistory) > 0 {
		current := &a.detailHistory[len(a.detailHistory)-1]
		if current.title == title {
			current.primitive = primitive
			current.hotkeys = a.currentDetailHotkeys()
			return
		}
	}
	a.detailHistory = append(a.detailHistory, detailPage{title: title, primitive: primitive, hotkeys: a.currentDetailHotkeys()})
}

func (a *App) showDetailPage(title string, primitive tview.Primitive) {
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBorder(true)
	container.SetTitle(title)
	container.SetBackgroundColor(colorBg)
	container.SetBorderColor(colorHeader)
	container.AddItem(primitive, 0, 1, true)

	if a.content.HasPage("detail") {
		a.content.RemovePage("detail")
	}
	a.content.AddPage("detail", container, true, true)
	a.content.SwitchToPage("detail")
	a.detailBox = primitive
	a.updateBreadcrumbs()
}

func (a *App) restoreDetailPage(page detailPage) {
	a.showDetailPage(page.title, page.primitive)
	a.applyDetailHotkeys(page.hotkeys)
	a.app.SetFocus(page.primitive)
}

func (a *App) updateBreadcrumbs() {
	items := make([]string, 0, 1+len(a.detailHistory)+1)
	items = append(items, breadcrumbTitle(a.listTitle))
	for _, page := range a.detailHistory {
		items = append(items, breadcrumbTitle(page.title))
	}
	if a.overlayVisible() {
		items = append(items, breadcrumbTitle(a.overlay.GetTitle()))
	}
	a.header.SetBreadcrumbs(items)
}

func (a *App) currentDetailHotkeys() detailHotkeys {
	if len(a.detailHistory) == 0 {
		return detailHotkeysDefault
	}
	return a.detailHistory[len(a.detailHistory)-1].hotkeys
}

func (a *App) updateDetailHotkeys(hotkeys detailHotkeys) {
	if len(a.detailHistory) == 0 {
		return
	}
	a.detailHistory[len(a.detailHistory)-1].hotkeys = hotkeys
}

func (a *App) applyDetailHotkeys(hotkeys detailHotkeys) {
	a.filterMenuMode = false
	switch hotkeys {
	case detailHotkeysDetail:
		a.header.SetDetailHotkeys()
	case detailHotkeysAudit:
		a.header.SetAuditHotkeys()
	case detailHotkeysAuditDetail:
		a.header.SetAuditDetailHotkeys()
	case detailHotkeysResourceOverview:
		a.header.SetResourceOverviewHotkeys()
	case detailHotkeysTemplateOverview:
		a.header.SetTemplateOverviewHotkeys()
	case detailHotkeysIntegrationOverview:
		a.header.SetIntegrationOverviewHotkeys()
	default:
		a.header.ResetHotkeys()
	}
}

func (a *App) enterFilterMenuMode() {
	a.filterMenuMode = true
	a.commandMode = false
	a.header.SetFilterMenuHotkeys()
	a.status.SetTitle("Filters")
	a.renderStatus()
}

func (a *App) exitFilterMenuMode() {
	a.filterMenuMode = false
	a.header.ResetHotkeys()
	a.status.SetTitle("Status")
	a.renderStatus()
}

func (a *App) enterCommandMode() {
	a.commandMode = true
	a.filterMenuMode = false
	a.commandText = ""
	a.commandMatches = nil
	a.status.SetTitle("Command")
	a.renderStatus()
}

func remapHalfPageScroll(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlD:
		return tcell.NewEventKey(tcell.KeyPgDn, 0, event.Modifiers())
	case tcell.KeyCtrlU:
		return tcell.NewEventKey(tcell.KeyPgUp, 0, event.Modifiers())
	case tcell.KeyRune:
		switch event.Rune() {
		case 0x04:
			return tcell.NewEventKey(tcell.KeyPgDn, 0, event.Modifiers())
		case 0x15:
			return tcell.NewEventKey(tcell.KeyPgUp, 0, event.Modifiers())
		}
	}

	return nil
}

func breadcrumbTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.TrimSuffix(title, "...")
	if title == "" {
		return "-"
	}
	return title
}

func headerHeight(showBreadcrumbs bool) int {
	if showBreadcrumbs {
		return 9
	}
	return 6
}

func (a *App) withLoadingSuffix(status string) string {
	a.loadingMx.Lock()
	defer a.loadingMx.Unlock()

	if a.loadingCount == 0 {
		return status
	}

	loading := "Loading " + loadingFrames[a.loadingFrame]
	if status == "" {
		return loading
	}
	return status + "  " + loading
}

func (a *App) queueRenderStatus() {
	a.app.QueueUpdateDraw(func() {
		a.renderStatus()
	})
}

func (a *App) runLoadingSpinner() {
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.loadingStop:
			return
		case <-ticker.C:
			a.loadingMx.Lock()
			if a.loadingCount == 0 {
				a.loadingMx.Unlock()
				continue
			}
			a.loadingFrame = (a.loadingFrame + 1) % len(loadingFrames)
			a.loadingMx.Unlock()
			a.queueRenderStatus()
		}
	}
}
