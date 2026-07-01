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
		{Title: "NAME", Key: "name"},
		{Title: "TEMPLATE", Key: "template"},
		{Title: "STATE", Key: "state"},
		{Title: "STATUS", Key: "status"},
		{Title: "WORKSPACE", Key: "workspace"},
		{Title: "AGE", Key: "age"},
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

func ToAge(ts time.Time, now time.Time) string {
	if ts.IsZero() {
		return "-"
	}

	d := now.Sub(ts)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return plural(int(d.Minutes()), "m")
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "h")
	case d < 30*24*time.Hour:
		return plural(int(d.Hours()/24), "d")
	default:
		return plural(int(d.Hours()/(24*30)), "mo")
	}
}

func plural(value int, unit string) string {
	if value <= 0 {
		return "now"
	}
	return strconv.Itoa(value) + unit
}

func normalizeCell(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
