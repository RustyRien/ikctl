package ui

import (
	"fmt"
	"strings"

	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type headerHints struct {
	main    [][]menuHint
	filters [][]menuHint
}

var defaultHeaderHints = headerHints{
	main: [][]menuHint{
		{{key: "s", label: "sort"}, {key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
		{{key: "r", label: "refresh"}, {key: "enter", label: "overview"}, {key: "y", label: "yaml"}},
		{{key: "l", label: "logs"}, {key: "a", label: "audit"}},
		{{key: "A", label: "actions"}, {key: "D", label: "delete"}, {key: "E", label: "edit"}},
		{{key: "e", label: "entity"}, {key: "o", label: "settings"}},
		{{key: "q", label: "quit"}, {key: "ctrl-c", label: "stop"}},
	},
	filters: [][]menuHint{
		{{key: "/", label: "search"}, {key: "f", label: "filters"}},
		{{key: "c", label: "reset"}, {key: "v", label: "columns"}},
		{{key: ":", label: "command"}},
	},
}

var filterMenuHeaderHints = headerHints{
	main: defaultHeaderHints.main,
	filters: [][]menuHint{
		{{key: "s", label: "storage"}},
		{{key: "i", label: "integration"}},
		{{key: "t", label: "template"}},
		{{key: "d", label: "hide destroyed"}},
		{{key: "c", label: "reset all"}},
		{{key: "esc", label: "back"}},
	},
}

var auditHeaderHints = headerHints{
	main: [][]menuHint{
		{{key: "up/down", label: "select"}, {key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
		{{key: "enter", label: "open logs"}, {key: "l", label: "open logs"}},
		{{key: "e", label: "entity"}, {key: "esc", label: "back"}, {key: "q", label: "back"}},
	},
}

var auditDetailHeaderHints = headerHints{
	main: [][]menuHint{
		{{key: "up/down", label: "scroll"}, {key: "pgup", label: "up"}, {key: "pgdn", label: "down"}},
		{{key: "esc", label: "back"}, {key: "q", label: "back"}},
	},
}

var detailHeaderHints = headerHints{
	main: [][]menuHint{
		{{key: "up/down", label: "scroll"}, {key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
		{{key: "esc", label: "back"}, {key: "q", label: "back"}},
	},
}

var resourceOverviewHints = headerHints{
	main: [][]menuHint{
		{{key: "up/down", label: "scroll"}, {key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
		{{key: "y", label: "yaml"}, {key: "t", label: "template"}, {key: "s", label: "storage"}},
		{{key: "i", label: "integrations"}},
		{{key: "T", label: "tree"}},
		{{key: "A", label: "actions"}, {key: "D", label: "delete"}, {key: "E", label: "edit"}},
		{{key: "R", label: "review"}},
		{{key: "esc", label: "back"}, {key: "q", label: "back"}},
	},
}

var templateOverviewHints = headerHints{
	main: [][]menuHint{
		{{key: "up/down", label: "scroll"}, {key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
		{{key: "y", label: "yaml"}, {key: "l", label: "logs"}, {key: "a", label: "audit"}},
		{{key: "A", label: "actions"}, {key: "D", label: "delete"}, {key: "E", label: "edit"}},
		{{key: "r", label: "resources"}, {key: "t", label: "tree view"}},
		{{key: "esc", label: "back"}, {key: "q", label: "back"}},
	},
}

var integrationOverviewHints = headerHints{
	main: [][]menuHint{
		{{key: "up/down", label: "scroll"}, {key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
		{{key: "y", label: "yaml"}, {key: "l", label: "logs"}, {key: "a", label: "audit"}},
		{{key: "A", label: "actions"}, {key: "D", label: "delete"}, {key: "E", label: "edit"}},
		{{key: "r", label: "resources"}},
		{{key: "esc", label: "back"}, {key: "q", label: "back"}},
	},
}

var storageOverviewHints = headerHints{
	main: [][]menuHint{
		{{key: "up/down", label: "scroll"}, {key: "ctrl-u", label: "up"}, {key: "ctrl-d", label: "down"}},
		{{key: "y", label: "yaml"}, {key: "l", label: "logs"}, {key: "a", label: "audit"}},
		{{key: "r", label: "resources"}, {key: "E", label: "edit"}},
		{{key: "esc", label: "back"}, {key: "q", label: "back"}},
	},
}

var ikLogo = []string{
	`.___ ___  __.`,
	`|   |   |/ _|`,
	`|   |     <  `,
	`|   |   |  \ `,
	`|___|___|__ \`,
	`           \/`,
}

type Header struct {
	root          *tview.Flex
	body          *tview.Flex
	hotkeyColumns *tview.Flex
	crumbs        *tview.TextView
	info          *tview.Table
	hotkeys       *tview.Table
	filters       *tview.Table
}

func NewHeader(cfg config.Config, version string) *Header {
	crumbs := newBreadcrumbs()
	info := newHeaderInfo(cfg, version)
	hotkeys := newHeaderHotkeys(defaultHeaderHints.main)
	filters := newHeaderHotkeys(defaultHeaderHints.filters)

	logo := tview.NewTextView()
	logo.SetWrap(false)
	logo.SetWordWrap(false)
	logo.SetTextAlign(tview.AlignRight)
	logo.SetBackgroundColor(colorBg)
	logo.SetTextColor(colorLogo)
	logo.SetText(strings.Join(ikLogo, "\n"))

	hotkeyColumns := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(hotkeys, 0, 3, false).
		AddItem(filters, 0, 2, false)
	hotkeyColumns.SetBackgroundColor(colorBg)

	body := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(info, 0, 2, false).
		AddItem(hotkeyColumns, 0, 4, false).
		AddItem(logo, 18, 0, false)
	body.SetBackgroundColor(colorBg)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(body, 6, 0, false).
		AddItem(crumbs, 3, 0, false)
	root.SetBackgroundColor(colorBg)

	header := &Header{root: root, body: body, hotkeyColumns: hotkeyColumns, crumbs: crumbs, info: info, hotkeys: hotkeys, filters: filters}
	header.SetBreadcrumbsVisible(cfg.ShowBreadcrumbs)
	return header
}

func (h *Header) Primitive() tview.Primitive {
	return h.root
}

func (h *Header) ResetHotkeys() {
	h.SetHotkeys(defaultHeaderHints)
}

func (h *Header) SetAuditHotkeys() {
	h.SetHotkeys(auditHeaderHints)
}

func (h *Header) SetFilterMenuHotkeys() {
	h.SetHotkeys(filterMenuHeaderHints)
}

func (h *Header) SetAuditDetailHotkeys() {
	h.SetHotkeys(auditDetailHeaderHints)
}

func (h *Header) SetDetailHotkeys() {
	h.SetHotkeys(detailHeaderHints)
}

func (h *Header) SetResourceOverviewHotkeys() {
	h.SetHotkeys(resourceOverviewHints)
}

func (h *Header) SetTemplateOverviewHotkeys() {
	h.SetHotkeys(templateOverviewHints)
}

func (h *Header) SetIntegrationOverviewHotkeys() {
	h.SetHotkeys(integrationOverviewHints)
}

func (h *Header) SetStorageOverviewHotkeys() {
	h.SetHotkeys(storageOverviewHints)
}

func (h *Header) SetHotkeys(hints headerHints) {
	h.hotkeys.Clear()
	h.filters.Clear()
	for row := 0; row < 6; row++ {
		if row < len(hints.main) {
			fillHeaderHintRow(h.hotkeys, row, hints.main[row])
		} else {
			fillHeaderHintRow(h.hotkeys, row, nil)
		}
		if row < len(hints.filters) {
			fillHeaderHintRow(h.filters, row, hints.filters[row])
		} else {
			fillHeaderHintRow(h.filters, row, nil)
		}
	}
}

func (h *Header) SetBreadcrumbs(items []string) {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts = append(parts, escapeHeaderText(item))
	}
	if len(parts) == 0 {
		h.crumbs.SetText("")
		return
	}

	var line strings.Builder
	for i, part := range parts {
		if i > 0 {
			line.WriteString(" ")
		}
		bg := hexColor(colorCrumbBg)
		if i == len(parts)-1 {
			bg = hexColor(colorCrumbLive)
		}
		line.WriteString("[")
		line.WriteString(hexColor(colorCrumbFg))
		line.WriteString(":")
		line.WriteString(bg)
		line.WriteString(":b] <")
		line.WriteString(part)
		line.WriteString("> [-:")
		line.WriteString(hexColor(colorBg))
		line.WriteString(":-]")
	}
	h.crumbs.SetText(line.String())
}

func (h *Header) SetBreadcrumbsVisible(visible bool) {
	h.root.Clear()
	h.root.AddItem(h.body, 6, 0, false)
	if visible {
		h.root.AddItem(h.crumbs, 3, 0, false)
	}
}

func (h *Header) SetUser(identifier string, displayName string, email string) {
	setHeaderPairRow(h.info, 4, "user", firstNonEmpty(displayName, identifier, "-"), colorTitle, colorInfo)
	setHeaderPairRow(h.info, 5, "email", firstNonEmpty(email, "-"), colorTitle, colorHeader)
}

func newHeaderInfo(cfg config.Config, version string) *tview.Table {
	info := tview.NewTable()
	info.SetBackgroundColor(colorBg)
	info.SetBorders(false)
	info.SetSelectable(false, false)
	info.SetSeparator(' ')

	setHeaderPairRow(info, 0, "app", "ikctl", colorTitleInfo, colorTitle)
	setHeaderPairRow(info, 1, "version", version, colorTitle, colorLogo)
	setHeaderPairRow(info, 2, "endpoint", cfg.Endpoint, colorTitle, colorHeader)
	setHeaderPairRow(info, 3, "api", "POST /api/graphql", colorTitle, colorHeader)
	setHeaderPairRow(info, 4, "user", "-", colorTitle, colorInfo)
	setHeaderPairRow(info, 5, "email", "-", colorTitle, colorHeader)

	return info
}

func newBreadcrumbs() *tview.TextView {
	crumbs := tview.NewTextView()
	crumbs.SetDynamicColors(true)
	crumbs.SetWrap(false)
	crumbs.SetWordWrap(false)
	crumbs.SetScrollable(false)
	crumbs.SetBorder(true)
	crumbs.SetTitle("Breadcrumbs")
	crumbs.SetBorderColor(colorHeader)
	crumbs.SetBackgroundColor(colorBg)
	crumbs.SetTextColor(colorInfo)
	crumbs.SetText("[#000000:#00ffff:b] <Resources> [-:#000000:-]")
	return crumbs
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func escapeHeaderText(value string) string {
	value = strings.ReplaceAll(value, "[", "[[")
	value = strings.ReplaceAll(value, "]", "]]")
	return value
}

func hexColor(color tcell.Color) string {
	return fmt.Sprintf("#%06x", uint32(color.Hex()))
}
