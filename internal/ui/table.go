package ui

import (
	"sort"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/sahilm/fuzzy"
)

type Table struct {
	widget     *tview.Table
	headers    []tabledata.Header
	rows       []tabledata.Row
	filtered   []tabledata.Row
	filter     string
	sortColumn int
	sortAsc    bool
}

func NewTable() *Table {
	widget := tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0)

	return &Table{
		widget:     widget,
		sortColumn: 0,
		sortAsc:    true,
	}
}

func (t *Table) Widget() *tview.Table {
	return t.widget
}

func (t *Table) SetData(headers []tabledata.Header, rows []tabledata.Row) {
	t.headers = append([]tabledata.Header(nil), headers...)
	t.rows = append([]tabledata.Row(nil), rows...)
	t.apply()
}

func (t *Table) SetFilter(filter string) {
	t.filter = filter
	t.apply()
}

func (t *Table) Filter() string {
	return t.filter
}

func (t *Table) ToggleSort(column int) {
	if column < 0 || column >= len(t.headers) {
		return
	}
	if t.sortColumn == column {
		t.sortAsc = !t.sortAsc
	} else {
		t.sortColumn = column
		t.sortAsc = true
	}
	t.apply()
}

func (t *Table) apply() {
	t.filtered = t.filterRows()
	t.sortRows()
	t.render()
}

func (t *Table) filterRows() []tabledata.Row {
	if t.filter == "" {
		return append([]tabledata.Row(nil), t.rows...)
	}

	targets := make([]string, 0, len(t.rows))
	for _, row := range t.rows {
		targets = append(targets, strings.ToLower(strings.Join(row.Fields, " ")))
	}

	matches := fuzzy.Find(strings.ToLower(t.filter), targets)
	rows := make([]tabledata.Row, 0, len(matches))
	for _, match := range matches {
		rows = append(rows, t.rows[match.Index])
	}
	return rows
}

func (t *Table) sortRows() {
	if len(t.headers) == 0 || t.sortColumn >= len(t.headers) {
		return
	}

	key := t.headers[t.sortColumn].Key
	sort.SliceStable(t.filtered, func(i, j int) bool {
		left := t.filtered[i].SortKey[key]
		right := t.filtered[j].SortKey[key]
		if t.sortAsc {
			return left < right
		}
		return left > right
	})
}

func (t *Table) render() {
	t.widget.Clear()

	for col, header := range t.headers {
		cell := tview.NewTableCell(header.Title).
			SetTextColor(colorHeader).
			SetSelectable(false).
			SetExpansion(1)
		if col == t.sortColumn {
			indicator := " ^"
			if !t.sortAsc {
				indicator = " v"
			}
			cell.SetText(header.Title + indicator)
		}
		t.widget.SetCell(0, col, cell)
	}

	for rowIdx, row := range t.filtered {
		for colIdx, field := range row.Fields {
			cell := tview.NewTableCell(field).
				SetExpansion(1).
				SetTextColor(colorForRow(row))
			t.widget.SetCell(rowIdx+1, colIdx, cell)
		}
	}

	if len(t.filtered) == 0 {
		t.widget.SetCell(1, 0, tview.NewTableCell("No resources").SetTextColor(colorInfo))
	}

	t.widget.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			t.ToggleSort(column)
		}
	})
}

func colorForRow(row tabledata.Row) tcell.Color {
	switch row.ColorKey {
	case "done", "ready", "provisioned":
		return colorSuccess
	case "error", "rejected":
		return colorError
	case "in_progress", "queued", "pending", "approval_pending":
		return colorRunning
	default:
		return colorTableText
	}
}
