package ui

import (
	"fmt"
	"time"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type App struct {
	app          *tview.Application
	header       *Header
	content      *tview.Pages
	table        *Table
	status       *tview.TextView
	pages        *tview.Pages
	overlay      *tview.Flex
	overlayBox   tview.Primitive
	detailBox    tview.Primitive
	filterMode   bool
	filterText   string
	config       config.Config
	version      string
	root         tview.Primitive
	filterDone   func(string)
	refreshFn    func()
	enterFn      func(tabledata.Row)
	logsFn       func(tabledata.Row)
	auditFn      func(tabledata.Row)
	navFn        func(rune)
	sortFn       func(int, bool)
	loadMoreFn   func()
	overlayKeyFn func(*tcell.EventKey) bool
	detailKeyFn  func(*tcell.EventKey) bool
	statusBase   string
}

func NewApp(cfg config.Config, version string) *App {
	applyColors(!cfg.NoColors)
	app := &App{
		app:     tview.NewApplication(),
		header:  NewHeader(cfg, version),
		table:   NewTable(),
		status:  tview.NewTextView(),
		config:  cfg,
		version: version,
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
		AddItem(app.header.Primitive(), 6, 0, false).
		AddItem(app.content, 0, 1, true).
		AddItem(app.status, 3, 0, false)
	main.SetBackgroundColor(colorBg)
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

	return app
}

func (a *App) Application() *tview.Application {
	return a.app
}

func (a *App) Run() error {
	return a.app.Run()
}

func (a *App) Stop() {
	a.app.Stop()
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

func (a *App) SetFilterDone(fn func(string)) {
	a.filterDone = fn
}

func (a *App) SetRefreshFunc(fn func()) {
	a.refreshFn = fn
}

func (a *App) SetEnterFunc(fn func(tabledata.Row)) {
	a.enterFn = fn
}

func (a *App) SetLogsFunc(fn func(tabledata.Row)) {
	a.logsFn = fn
}

func (a *App) SetAuditFunc(fn func(tabledata.Row)) {
	a.auditFn = fn
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

func (a *App) SetOverlayKeyFunc(fn func(*tcell.EventKey) bool) {
	a.overlayKeyFn = fn
}

func (a *App) SetDetailKeyFunc(fn func(*tcell.EventKey) bool) {
	a.detailKeyFn = fn
}

func (a *App) SetEntityTitle(title string, emptyLabel string) {
	a.table.Widget().SetTitle(title)
	a.table.SetEmptyLabel(emptyLabel)
}

func (a *App) OpenOverlay(title string, text string) {
	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetScrollable(true)
	textView.SetWrap(true)
	textView.SetText(text)
	a.setOverlayContent(title, textView)
	a.pages.ShowPage("overlay")
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
}

func (a *App) OpenOverlayPrimitive(title string, primitive tview.Primitive) {
	a.setOverlayContent(title, primitive)
	a.pages.ShowPage("overlay")
	a.app.SetFocus(primitive)
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
	a.header.ResetHotkeys()
}

func (a *App) SetAuditHeaderHotkeys() {
	a.header.SetAuditHotkeys()
}

func (a *App) SetAuditDetailHeaderHotkeys() {
	a.header.SetAuditDetailHotkeys()
}

func (a *App) CloseDetail() {
	a.content.HidePage("detail")
	a.content.SwitchToPage("list")
	a.header.ResetHotkeys()
	a.app.SetFocus(a.table.Widget())
}

func (a *App) CloseOverlay() {
	a.pages.HidePage("overlay")
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
			if entity, ok := a.table.EntityForDigit(event.Rune()); ok {
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

	if a.overlayVisible() {
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
			if event.Rune() == 'l' && a.logsFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.logsFn(row)
					return nil
				}
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
			if a.table.StartEntityMode() {
				a.status.SetTitle("Entity")
				a.status.SetText("Choose entity: " + a.table.EntityHints() + "  Esc cancel")
			}
			return nil
		case 'r':
			if a.refreshFn != nil {
				a.refreshFn()
			}
			return nil
		case 'l':
			if a.logsFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.logsFn(row)
				}
			}
			return nil
		case 'a':
			if a.auditFn != nil {
				if row, ok := a.table.SelectedRow(); ok {
					a.auditFn(row)
				}
			}
			return nil
		}
	}

	return event
}

func (a *App) renderStatus() {
	if a.filterMode {
		a.status.SetText("Filter: " + a.filterText)
		return
	}
	if a.table.SortMode() {
		a.status.SetText("Choose column: " + a.table.SortHints() + "  Esc cancel")
		if a.table.AwaitingSortDirection() {
			a.status.SetText("Choose direction: " + a.table.SortHints() + "  Esc cancel")
		}
		return
	}
	if a.table.EntityMode() {
		a.status.SetText("Choose entity: " + a.table.EntityHints() + "  Esc cancel")
		return
	}
	a.status.SetText(a.statusBase)
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
}
