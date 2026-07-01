package render

import (
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type TemplateRenderer struct{}

func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{}
}

func (r *TemplateRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "CLOUD TYPES", Key: "cloud_types"},
		{Title: "UPDATED", Key: "updated", SortField: "updated_at"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func (r *TemplateRenderer) Row(template client.Template) tabledata.Row {
	cloudTypes := strings.Join(orSlice(template.CloudResourceTypes, []string{"-"}), ", ")
	updated := normalizeCell(template.UpdatedAt.Format(time.RFC3339))

	return tabledata.Row{
		ID: template.ID,
		Fields: []string{
			template.Name,
			cloudTypes,
			updated,
			ToAge(template.CreatedAt, time.Now()),
		},
		SortKey: map[string]string{
			"name":        strings.ToLower(template.Name),
			"cloud_types": strings.ToLower(cloudTypes),
			"updated":     template.UpdatedAt.Format(time.RFC3339Nano),
			"age":         template.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: template.UpdatedAt,
		ColorKey:  "ready",
		Raw:       template,
	}
}

func orSlice(values []string, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}
	return values
}
