package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type WorkspaceRenderer struct{}

var workspaceListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "name"},
	{Title: "PROVIDER", Key: "workspaceProvider", SortField: "workspaceProvider"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "DESCRIPTION", Key: "description"},
	{Title: "LABELS", Key: "labels"},
	{Title: "CREATED", Key: "createdAt", SortField: "createdAt"},
	{Title: "UPDATED", Key: "updatedAt", SortField: "updatedAt"},
	{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"},
	{Title: "INTEGRATION", Key: "integration", SortField: "integration.name"},
	{Title: "RESOURCES", Key: "resourcesCount", SortField: "resourcesCount"},
	{Title: "ENTITY", Key: "entityName"},
	{Title: "AGE", Key: "age", SortField: "createdAt"},
}

func NewWorkspaceRenderer() *WorkspaceRenderer {
	return &WorkspaceRenderer{}
}

func (r *WorkspaceRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "PROVIDER", Key: "provider", SortField: "workspaceProvider"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "UPDATED", Key: "updated", SortField: "updatedAt"},
		{Title: "AGE", Key: "age", SortField: "createdAt"},
	}
}

func (r *WorkspaceRenderer) Row(workspace client.Workspace) tabledata.Row {
	row := WorkspaceListRow(workspace)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[6],
		row.Fields[11],
	}
	row.SortKey = map[string]string{
		"name":     row.SortKey["name"],
		"provider": row.SortKey["workspaceProvider"],
		"status":   row.SortKey["status"],
		"updated":  row.SortKey["updatedAt"],
		"age":      row.SortKey["age"],
	}
	return row
}

func WorkspaceListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), workspaceListHeaders...)
}

func WorkspaceListRow(workspace client.Workspace) tabledata.Row {
	provider := normalizeCell(workspace.WorkspaceProvider)
	status := normalizeCell(workspace.Status)
	description := normalizeCell(workspace.Description)
	labels := joinStrings(workspace.Labels)
	created := normalizeCell(workspace.CreatedAt.Format(time.RFC3339))
	updated := normalizeCell(workspace.UpdatedAt.Format(time.RFC3339))
	creator := creatorName(workspace.Creator)
	integration := integrationName(workspace.Integration)
	resourcesCount := strconv.Itoa(workspace.ResourcesCount)
	entityName := normalizeCell(workspace.EntityName)
	age := ToAge(workspace.CreatedAt, time.Now())

	return tabledata.Row{
		ID: workspace.ID,
		Fields: []string{
			workspace.Name,
			provider,
			status,
			description,
			labels,
			created,
			updated,
			creator,
			integration,
			resourcesCount,
			entityName,
			age,
		},
		SortKey: map[string]string{
			"name":              strings.ToLower(workspace.Name),
			"workspaceProvider": strings.ToLower(provider),
			"status":            strings.ToLower(status),
			"description":       strings.ToLower(description),
			"labels":            strings.ToLower(labels),
			"createdAt":         workspace.CreatedAt.Format(time.RFC3339Nano),
			"updatedAt":         workspace.UpdatedAt.Format(time.RFC3339Nano),
			"creator":           strings.ToLower(creator),
			"integration":       strings.ToLower(integration),
			"resourcesCount":    resourcesCount,
			"entityName":        strings.ToLower(entityName),
			"age":               workspace.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: workspace.UpdatedAt,
		ColorKey:  strings.ToLower(status),
		Raw:       workspace,
	}
}

func WorkspaceWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "PROVIDER", Key: "workspaceProvider", SortField: "workspaceProvider"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "RESOURCES", Key: "resourcesCount", SortField: "resourcesCount"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "createdAt"},
	}
}

func WorkspaceWideRow(workspace client.Workspace) tabledata.Row {
	row := WorkspaceListRow(workspace)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[9],
		workspace.ID,
		row.Fields[11],
	}
	return row
}
