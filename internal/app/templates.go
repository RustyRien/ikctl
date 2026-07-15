package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/model"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

type templateDetailSelection struct {
	ID   string
	Name string
}

var templateColumnOptions = []resourceColumnOption{
	{Field: "name", Header: tabledata.Header{Title: "NAME", Key: "name", SortField: "name"}, Description: "Template name", DefaultOn: true},
	{Field: "cloudResourceTypes", Header: tabledata.Header{Title: "CLOUD TYPES", Key: "cloudResourceTypes"}, Description: "Cloud resource types", DefaultOn: true},
	{Field: "description", Header: tabledata.Header{Title: "DESCRIPTION", Key: "description"}, Description: "Description", DefaultOn: false},
	{Field: "labels", Header: tabledata.Header{Title: "LABELS", Key: "labels"}, Description: "Labels", DefaultOn: false},
	{Field: "status", Header: tabledata.Header{Title: "STATUS", Key: "status", SortField: "status"}, Description: "Status", DefaultOn: false},
	{Field: "abstract", Header: tabledata.Header{Title: "ABSTRACT", Key: "abstract"}, Description: "Abstract flag", DefaultOn: false},
	{Field: "createdAt", Header: tabledata.Header{Title: "CREATED", Key: "createdAt", SortField: "created_at"}, Description: "Created time", DefaultOn: false},
	{Field: "updatedAt", Header: tabledata.Header{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"}, Description: "Last updated", DefaultOn: true},
	{Field: "entityName", Header: tabledata.Header{Title: "ENTITY", Key: "entityName"}, Description: "Entity name", DefaultOn: false},
	{Field: "id", Header: tabledata.Header{Title: "ID", Key: "id"}, Description: "Template ID", DefaultOn: false},
	{Field: "age", Header: tabledata.Header{Title: "AGE", Key: "age", SortField: "created_at"}, Description: "Age", DefaultOn: true},
}

func defaultVisibleTemplateColumns() map[string]bool {
	visible := make(map[string]bool, len(templateColumnOptions))
	for _, option := range templateColumnOptions {
		if option.DefaultOn {
			visible[option.Field] = true
		}
	}
	return visible
}

func (a *App) openTemplateOverview(id string, name string) {
	a.stopLiveLogStream()

	title := fmt.Sprintf("Template: %s", name)
	a.clearOverviewJumpState()
	a.activeTemplateDetail = &templateDetailSelection{ID: id, Name: name}
	a.activeIntegrationDetail = nil
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

func (a *App) openTemplateTree(id string, name string) {
	a.stopLiveLogStream()

	title := fmt.Sprintf("Template Tree: %s", name)
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.overviewTree = nil
	a.ui.OpenOverlay(title, "Loading template tree...")

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		tree, err := a.client.TemplateTree(ctx, id, "children")
		var primitive tview.Primitive
		var treeSelection *overviewTreeSelection
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load template tree.\n\n%v", err))
		} else if tree != nil {
			var treeView *tview.TreeView
			primitive, treeView = overviewTreeView("Tree View", templateTreeNode(*tree), "Enter open template  Esc/q close")
			treeSelection = &overviewTreeSelection{view: treeView, onSelect: func(reference any) {
				selection, ok := reference.(client.TemplateTreeNode)
				if !ok || selection.ID == "" {
					return
				}
				a.openTemplateOverview(selection.ID, valueOr(selection.Name, selection.ID))
			}}
		} else {
			primitive = errorView("Template tree not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.overviewTree = treeSelection
			if treeSelection != nil && treeSelection.view != nil {
				a.ui.OpenOverlayPrimitiveWithFocus(title, primitive, treeSelection.view)
				return
			}
			a.ui.OpenOverlayPrimitive(title, primitive)
		})
	}()
}

func (a *App) openTemplateColumns() {
	if a.activeKind != model.EntityTemplates {
		return
	}
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.entitySelectorTable = nil
	a.settingsTable = nil
	a.overviewTree = nil
	a.resourceColumnsTable = nil
	primitive, table := resourceColumnsView(templateColumnOptions, a.visibleTemplateColumns)
	a.templateColumnsTable = table
	a.ui.OpenOverlayPrimitive("Template Columns", primitive)
}

func (a *App) toggleSelectedTemplateColumn() {
	if a.templateColumnsTable == nil {
		return
	}
	selectedRow, _ := a.templateColumnsTable.GetSelection()
	if selectedRow <= 0 {
		return
	}
	index := selectedRow - 1
	if index < 0 || index >= len(templateColumnOptions) {
		return
	}
	option := templateColumnOptions[index]
	if a.visibleTemplateColumns[option.Field] {
		visibleCount := 0
		for _, current := range templateColumnOptions {
			if a.visibleTemplateColumns[current.Field] {
				visibleCount++
			}
		}
		if visibleCount == 1 {
			return
		}
		a.visibleTemplateColumns[option.Field] = false
	} else {
		a.visibleTemplateColumns[option.Field] = true
	}
	primitive, table := resourceColumnsView(templateColumnOptions, a.visibleTemplateColumns)
	table.Select(selectedRow, 0)
	a.templateColumnsTable = table
	a.ui.OpenOverlayPrimitive("Template Columns", primitive)
	a.saveViewPreferences()
	a.renderCurrentModel()
}

func (a *App) projectTemplateList(_ []tabledata.Header, rows []tabledata.Row) ([]tabledata.Header, []tabledata.Row) {
	projectedHeaders := make([]tabledata.Header, 0, len(templateColumnOptions))
	visibleFields := make([]string, 0, len(templateColumnOptions))
	for _, option := range templateColumnOptions {
		if !a.visibleTemplateColumns[option.Field] {
			continue
		}
		projectedHeaders = append(projectedHeaders, option.Header)
		visibleFields = append(visibleFields, option.Field)
	}
	if len(visibleFields) == 0 {
		return render.TemplateListHeaders(), rows
	}
	projectedRows := make([]tabledata.Row, 0, len(rows))
	for _, row := range rows {
		fullRow := row
		if templateValue, ok := row.Raw.(client.Template); ok {
			fullRow = render.TemplateListRow(templateValue)
			fullRow.SortKey["id"] = strings.ToLower(templateValue.ID)
		}
		fields := make([]string, 0, len(visibleFields))
		for _, field := range visibleFields {
			fields = append(fields, templateFieldValue(fullRow, field))
		}
		projectedRow := fullRow
		projectedRow.Fields = fields
		projectedRows = append(projectedRows, projectedRow)
	}
	return projectedHeaders, projectedRows
}

func templateFieldValue(row tabledata.Row, field string) string {
	indexByField := map[string]int{
		"name":               0,
		"cloudResourceTypes": 1,
		"description":        2,
		"labels":             3,
		"status":             4,
		"abstract":           5,
		"createdAt":          6,
		"updatedAt":          7,
		"entityName":         8,
		"age":                9,
	}
	if field == "id" {
		if row.ID == "" {
			return "-"
		}
		return row.ID
	}
	index, ok := indexByField[field]
	if !ok || index < 0 || index >= len(row.Fields) {
		return "-"
	}
	return row.Fields[index]
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
	root.AddItem(overviewFooter("y yaml  t tree view  Esc/q close"), 1, 0, false)
	return root
}
