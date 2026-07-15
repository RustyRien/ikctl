package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
)

type Table struct {
	widget      *tview.Table
	headers     []tabledata.Header
	rows        []tabledata.Row
	filtered    []tabledata.Row
	filter      string
	emptyLabel  string
	sortMode    bool
	pendingSort int
	sortColumn  int
	sortAsc     bool
	entityMode  bool
	selectFn    func(int, int)
}

func NewTable() *Table {
	widget := tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0)

	return &Table{
		widget:      widget,
		emptyLabel:  "No rows",
		pendingSort: -1,
		sortColumn:  -1,
		sortAsc:     true,
	}
}

func (t *Table) Widget() *tview.Table {
	return t.widget
}

func (t *Table) SetSelectionChangedFunc(fn func(int, int)) {
	t.selectFn = fn
	t.widget.SetSelectionChangedFunc(fn)
}

func (t *Table) SetData(headers []tabledata.Header, rows []tabledata.Row) {
	t.headers = append([]tabledata.Header(nil), headers...)
	t.rows = append([]tabledata.Row(nil), rows...)
	t.apply()
}

func (t *Table) SetSortState(column int, asc bool) {
	t.sortColumn = column
	t.sortAsc = asc
	t.render()
}

func (t *Table) SetFilter(filter string) {
	t.filter = filter
	t.apply()
}

func (t *Table) Filter() string {
	return t.filter
}

func (t *Table) SetEmptyLabel(label string) {
	if label == "" {
		label = "No rows"
	}
	t.emptyLabel = label
	t.render()
}

func (t *Table) SelectedRow() (tabledata.Row, bool) {
	rowIndex, _ := t.widget.GetSelection()
	if rowIndex <= 0 {
		return tabledata.Row{}, false
	}
	index := rowIndex - 1
	if index < 0 || index >= len(t.filtered) {
		return tabledata.Row{}, false
	}
	return t.filtered[index], true
}

func (t *Table) MoveHalfPage(direction int) {
	if len(t.filtered) == 0 || direction == 0 {
		return
	}

	row, column := t.widget.GetSelection()
	if row <= 0 {
		row = 1
	}

	step := t.halfPageSize()
	next := row + (direction * step)
	if next < 1 {
		next = 1
	}
	if next > len(t.filtered) {
		next = len(t.filtered)
	}

	t.widget.Select(next, column)
	if t.selectFn != nil {
		t.selectFn(next, column)
	}
}

func (t *Table) VisibleCount() int {
	return len(t.filtered)
}

func (t *Table) CurrentSelectionIndex() int {
	row, _ := t.widget.GetSelection()
	if row <= 0 {
		return 0
	}
	return row - 1
}

func (t *Table) StartSortMode() bool {
	if len(t.sortableColumns()) == 0 {
		return false
	}
	t.sortMode = true
	t.pendingSort = -1
	t.render()
	return true
}

func (t *Table) CancelSortMode() bool {
	if !t.sortMode {
		return false
	}
	t.sortMode = false
	t.pendingSort = -1
	t.render()
	return true
}

func (t *Table) ApplySortDigit(digit rune) bool {
	if !t.sortMode {
		return false
	}
	column, ok := t.SortColumnForDigit(digit)
	if !ok {
		return false
	}
	t.pendingSort = column
	t.render()
	return true
}

func (t *Table) ApplySortDirection(direction rune) bool {
	if !t.sortMode || t.pendingSort < 0 {
		return false
	}
	if direction != 'a' && direction != 'd' {
		return false
	}
	return true
}

func (t *Table) FinishSortMode() {
	t.sortMode = false
	t.pendingSort = -1
	t.render()
}

func (t *Table) SortMode() bool {
	return t.sortMode
}

func (t *Table) StartEntityMode() bool {
	t.entityMode = true
	t.render()
	return true
}

func (t *Table) CancelEntityMode() bool {
	if !t.entityMode {
		return false
	}
	t.entityMode = false
	t.render()
	return true
}

func (t *Table) FinishEntityMode() {
	t.entityMode = false
	t.render()
}

func (t *Table) EntityMode() bool {
	return t.entityMode
}

func (t *Table) EntityHints() string {
	return "r:Resources  c:SourceCodes  v:SourceCodeVersions  s:Storages  t:Templates  i:Integrations"
}

func (t *Table) EntityForKey(key rune) (rune, bool) {
	if !t.entityMode {
		return 0, false
	}
	switch unicode.ToLower(key) {
	case 'r', 'c', 'v', 's', 't', 'i':
		return unicode.ToLower(key), true
	default:
		return 0, false
	}
}

func (t *Table) AwaitingSortDirection() bool {
	return t.pendingSort >= 0
}

func (t *Table) SortHints() string {
	if t.pendingSort >= 0 {
		return "a:ASC  d:DESC"
	}
	if len(t.headers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(t.headers))
	for i, column := range t.sortableColumns() {
		header := t.headers[column]
		parts = append(parts, strconv.Itoa(i+1)+":"+header.Title)
	}
	return strings.Join(parts, "  ")
}

func (t *Table) PendingSortColumn() (int, bool) {
	if t.pendingSort < 0 {
		return 0, false
	}
	return t.pendingSort, true
}

func (t *Table) SortColumnForDigit(digit rune) (int, bool) {
	if !t.sortMode {
		return 0, false
	}
	sortable := t.sortableColumns()
	index := int(digit - '1')
	if index < 0 || index >= len(sortable) {
		return 0, false
	}
	return sortable[index], true
}

func (t *Table) apply() {
	t.filtered = t.filterRows()
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

func (t *Table) render() {
	currentRow, currentColumn := t.widget.GetSelection()
	rowOffset, columnOffset := t.widget.GetOffset()
	t.widget.SetSelectionChangedFunc(nil)
	t.widget.Clear()

	sortNumbers := map[int]int{}
	for index, column := range t.sortableColumns() {
		sortNumbers[column] = index + 1
	}

	for col, header := range t.headers {
		text := header.Title
		color := colorHeader
		if t.sortMode {
			if number, ok := sortNumbers[col]; ok {
				text = fmt.Sprintf("[%d] %s", number, header.Title)
				color = colorRunning
			}
			if col == t.pendingSort {
				text = "> " + header.Title
				color = colorSuccess
			}
		}
		if col == t.sortColumn && t.sortColumn >= 0 {
			indicator := " ^"
			if !t.sortAsc {
				indicator = " v"
			}
			text += indicator
			if !t.sortMode {
				color = colorHeader
			}
		}
		if t.entityMode {
			color = colorHeader
		}
		cell := tview.NewTableCell(text).
			SetTextColor(color).
			SetSelectable(false).
			SetExpansion(1)
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
		t.widget.SetOffset(0, 0)
		t.widget.SetCell(1, 0, tview.NewTableCell(t.emptyLabel).SetTextColor(colorInfo))
	} else {
		if currentRow <= 0 || currentRow > len(t.filtered) {
			currentRow = 1
			currentColumn = 0
			rowOffset = 0
			columnOffset = 0
		}
		t.widget.SetOffset(rowOffset, columnOffset)
		t.widget.Select(currentRow, currentColumn)
	}

	t.widget.SetSelectedFunc(nil)
	t.widget.SetSelectionChangedFunc(t.selectFn)
}

func (t *Table) sortableColumns() []int {
	columns := make([]int, 0, len(t.headers))
	for index, header := range t.headers {
		if header.SortField != "" {
			columns = append(columns, index)
		}
	}
	return columns
}

func (t *Table) halfPageSize() int {
	_, _, _, height := t.widget.GetRect()
	step := (height - 4) / 2
	if step < 1 {
		return 1
	}
	return step
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
