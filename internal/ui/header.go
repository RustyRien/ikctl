package ui

import (
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/electrolux-oss/ik-tui/internal/config"
)

var defaultHeaderHotkeys = [][]menuHint{
	{{key: "s", label: "sort"}, {key: "/", label: "filter"}},
	{{key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
	{{key: "r", label: "refresh"}, {key: "enter", label: "overview"}},
	{{key: "l", label: "logs"}, {key: "a", label: "audit"}, {key: "esc", label: "close"}},
	{{key: "e", label: "entity"}},
	{{key: "q", label: "quit"}, {key: "ctrl-c", label: "stop"}},
}

var auditHeaderHotkeys = [][]menuHint{
	{{key: "up/down", label: "select"}, {key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
	{{key: "enter", label: "open logs"}, {key: "l", label: "open logs"}},
	{{key: "e", label: "entity"}, {key: "esc", label: "back"}, {key: "q", label: "back"}},
}

var auditDetailHeaderHotkeys = [][]menuHint{
	{{key: "up/down", label: "scroll"}, {key: "pgup", label: "up"}, {key: "pgdn", label: "down"}},
	{{key: "esc", label: "back"}, {key: "q", label: "back"}},
}

var ikLogo = []string{
	` ___ _  __`,
	`|_ _| |/ /`,
	` | || ' / `,
	` | || . \ `,
	`|___|_|\_\`,
	`          `,
}

type Header struct {
	root    *tview.Flex
	hotkeys *tview.Table
}

func NewHeader(cfg config.Config, version string) *Header {
	info := newHeaderInfo(cfg, version)
	hotkeys := newHeaderHotkeys(defaultHeaderHotkeys)

	logo := tview.NewTextView()
	logo.SetWrap(false)
	logo.SetWordWrap(false)
	logo.SetTextAlign(tview.AlignRight)
	logo.SetBackgroundColor(colorBg)
	logo.SetTextColor(colorLogo)
	logo.SetText(strings.Join(ikLogo, "\n"))

	root := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(info, 0, 2, false).
		AddItem(hotkeys, 0, 3, false).
		AddItem(logo, 12, 0, false)
	root.SetBackgroundColor(colorBg)

	return &Header{root: root, hotkeys: hotkeys}
}

func (h *Header) Primitive() tview.Primitive {
	return h.root
}

func (h *Header) ResetHotkeys() {
	h.SetHotkeys(defaultHeaderHotkeys)
}

func (h *Header) SetAuditHotkeys() {
	h.SetHotkeys(auditHeaderHotkeys)
}

func (h *Header) SetAuditDetailHotkeys() {
	h.SetHotkeys(auditDetailHeaderHotkeys)
}

func (h *Header) SetHotkeys(rows [][]menuHint) {
	h.hotkeys.Clear()
	for row := 0; row < 6; row++ {
		if row < len(rows) {
			fillHeaderHintRow(h.hotkeys, row, rows[row])
			continue
		}
		fillHeaderHintRow(h.hotkeys, row, nil)
	}
}

func newHeaderInfo(cfg config.Config, version string) tview.Primitive {
	info := tview.NewTable()
	info.SetBackgroundColor(colorBg)
	info.SetBorders(false)
	info.SetSelectable(false, false)
	info.SetSeparator(' ')

	setHeaderPairRow(info, 0, "app", "ik-tui", colorTitleInfo, colorTitle)
	setHeaderPairRow(info, 1, "version", version, colorTitle, colorLogo)
	setHeaderPairRow(info, 2, "endpoint", cfg.Endpoint, colorTitle, colorHeader)
	setHeaderPairRow(info, 3, "api", "POST /api/graphql", colorTitle, colorHeader)
	setHeaderSpacerRow(info, 4)
	setHeaderSpacerRow(info, 5)

	return info
}

func newHeaderHotkeys(rows [][]menuHint) *tview.Table {
	hotkeys := tview.NewTable()
	hotkeys.SetBackgroundColor(colorBg)
	hotkeys.SetBorders(false)
	hotkeys.SetSelectable(false, false)
	hotkeys.SetSeparator(' ')

	for row, hints := range rows {
		fillHeaderHintRow(hotkeys, row, hints)
	}

	return hotkeys
}

type menuHint struct {
	key     string
	label   string
	numeric bool
}

func fillHeaderHintRow(table *tview.Table, row int, hints []menuHint) {
	for col, hint := range hints {
		keyColor := colorKey
		if hint.numeric {
			keyColor = colorNumKey
		}

		keyCell := tview.NewTableCell("<" + hint.key + ">")
		keyCell.SetTextColor(keyColor)
		keyCell.SetBackgroundColor(colorBg)
		keyCell.SetExpansion(1)
		keyCell.SetSelectable(false)
		table.SetCell(row, col*2, keyCell)

		labelCell := tview.NewTableCell(hint.label)
		labelCell.SetTextColor(colorTitle)
		labelCell.SetBackgroundColor(colorBg)
		labelCell.SetExpansion(2)
		labelCell.SetSelectable(false)
		table.SetCell(row, col*2+1, labelCell)
	}
}

func setHeaderPairRow(table *tview.Table, row int, label string, value string, labelColor, valueColor tcell.Color) {
	labelCell := tview.NewTableCell(label)
	labelCell.SetTextColor(labelColor)
	labelCell.SetBackgroundColor(colorBg)
	labelCell.SetSelectable(false)
	table.SetCell(row, 0, labelCell)

	valueCell := tview.NewTableCell(value)
	valueCell.SetTextColor(valueColor)
	valueCell.SetBackgroundColor(colorBg)
	valueCell.SetExpansion(1)
	valueCell.SetSelectable(false)
	table.SetCell(row, 1, valueCell)
}

func setHeaderSpacerRow(table *tview.Table, row int) {
	left := tview.NewTableCell("")
	left.SetBackgroundColor(colorBg)
	left.SetSelectable(false)
	table.SetCell(row, 0, left)

	right := tview.NewTableCell("")
	right.SetBackgroundColor(colorBg)
	right.SetSelectable(false)
	table.SetCell(row, 1, right)
}
