package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type ResourceRenderer struct{}

func NewResourceRenderer() *ResourceRenderer {
	return &ResourceRenderer{}
}

func (r *ResourceRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "TEMPLATE", Key: "template"},
		{Title: "STATE", Key: "state", SortField: "state"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "WORKSPACE", Key: "workspace"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func (r *ResourceRenderer) Row(resource client.Resource) tabledata.Row {
	template := "-"
	if resource.Template != nil && resource.Template.Name != "" {
		template = resource.Template.Name
	}

	workspace := "-"
	if resource.Workspace != nil && resource.Workspace.Name != "" {
		workspace = resource.Workspace.Name
	}

	age := ToAge(resource.CreatedAt, time.Now())
	status := normalizeCell(resource.Status)
	state := normalizeCell(resource.State)

	return tabledata.Row{
		ID: resource.ID,
		Fields: []string{
			resource.Name,
			template,
			state,
			status,
			workspace,
			age,
		},
		SortKey: map[string]string{
			"name":      strings.ToLower(resource.Name),
			"template":  strings.ToLower(template),
			"state":     strings.ToLower(state),
			"status":    strings.ToLower(status),
			"workspace": strings.ToLower(workspace),
			"age":       resource.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: resource.UpdatedAt,
		ColorKey:  strings.ToLower(status),
		Raw:       resource,
	}
}

func ResourceWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "TEMPLATE", Key: "template"},
		{Title: "STATE", Key: "state", SortField: "state"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "WORKSPACE", Key: "workspace"},
		{Title: "REV", Key: "revision"},
		{Title: "CREATOR", Key: "creator"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func ResourceWideRow(resource client.Resource) tabledata.Row {
	row := NewResourceRenderer().Row(resource)
	creator := "-"
	if resource.Creator != nil {
		if resource.Creator.DisplayName != "" {
			creator = resource.Creator.DisplayName
		} else if resource.Creator.Identifier != "" {
			creator = resource.Creator.Identifier
		} else {
			creator = resource.Creator.ID
		}
	}
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[3],
		row.Fields[4],
		strconv.Itoa(resource.RevisionNumber),
		creator,
		resource.ID,
		row.Fields[5],
	}
	return row
}
