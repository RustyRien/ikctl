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
	app        *tview.Application
	header     *tview.TextView
	table      *Table
	status     *tview.TextView
	filterMode bool
	filterText string
	config     config.Config
	version    string
	root       tview.Primitive
	filterDone func(string)
	refreshFn  func()
}

func NewApp(cfg config.Config, version string) *App {
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
	app.status.SetBorder(true)
	app.status.SetTitle("Status")

	app.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(app.header, 2, 0, false).
		AddItem(app.table.Widget(), 0, 1, true).
		AddItem(app.status, 3, 0, false)

	app.app.SetRoot(app.root, true)
	app.app.SetInputCapture(app.capture)
	app.table.Widget().SetBorder(true).SetTitle("Resources")

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

func (a *App) Update(headers []tabledata.Header, rows []tabledata.Row, total int, lastUpdated time.Time, lastErr error) {
	a.table.SetData(headers, rows)
	status := fmt.Sprintf("Total: %d", total)
	if !lastUpdated.IsZero() {
		status += fmt.Sprintf("  Last refresh: %s", lastUpdated.Format(time.Kitchen))
	}
	if filter := a.table.Filter(); filter != "" {
		status += fmt.Sprintf("  Filter: %s", filter)
	}
	if lastErr != nil {
		status += fmt.Sprintf("  [red]Error[-]: %v", lastErr)
	}
	a.status.SetText(status)
}

func (a *App) SetFilterDone(fn func(string)) {
	a.filterDone = fn
}

func (a *App) SetRefreshFunc(fn func()) {
	a.refreshFn = fn
}

func (a *App) capture(event *tcell.EventKey) *tcell.EventKey {
	if a.filterMode {
		switch event.Key() {
		case tcell.KeyEsc:
			a.filterMode = false
			a.status.SetTitle("Status")
			return nil
		case tcell.KeyEnter:
			a.filterMode = false
			a.table.SetFilter(a.filterText)
			if a.filterDone != nil {
				a.filterDone(a.filterText)
			}
			a.status.SetTitle("Status")
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

	switch event.Key() {
	case tcell.KeyCtrlC:
		a.Stop()
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
		case '1', '2', '3', '4', '5', '6':
			a.table.ToggleSort(int(event.Rune() - '1'))
			return nil
		case 'r':
			if a.refreshFn != nil {
				a.refreshFn()
			}
			return nil
		}
	}

	return event
}
