package render

import (
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type TemplateRenderer struct{}

var templateListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "name"},
	{Title: "CLOUD TYPES", Key: "cloudResourceTypes"},
	{Title: "DESCRIPTION", Key: "description"},
	{Title: "LABELS", Key: "labels"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "ABSTRACT", Key: "abstract"},
	{Title: "CREATED", Key: "createdAt", SortField: "created_at"},
	{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"},
	{Title: "ENTITY", Key: "entityName"},
	{Title: "AGE", Key: "age", SortField: "created_at"},
}

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
	row := TemplateListRow(template)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[7],
		row.Fields[9],
	}
	row.SortKey = map[string]string{
		"name":        row.SortKey["name"],
		"cloud_types": row.SortKey["cloudResourceTypes"],
		"updated":     row.SortKey["updatedAt"],
		"age":         row.SortKey["age"],
	}
	return row
}

func TemplateListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), templateListHeaders...)
}

func TemplateListRow(template client.Template) tabledata.Row {
	cloudTypes := strings.Join(orSlice(template.CloudResourceTypes, []string{"-"}), ", ")
	description := normalizeCell(template.Description)
	labels := joinStrings(template.Labels)
	status := normalizeCell(template.Status)
	abstract := "no"
	if template.Abstract {
		abstract = "yes"
	}
	created := normalizeCell(template.CreatedAt.Format(time.RFC3339))
	updated := normalizeCell(template.UpdatedAt.Format(time.RFC3339))
	entityName := normalizeCell(template.EntityName)

	return tabledata.Row{
		ID: template.ID,
		Fields: []string{
			template.Name,
			cloudTypes,
			description,
			labels,
			status,
			abstract,
			created,
			updated,
			entityName,
			ToAge(template.CreatedAt, time.Now()),
		},
		SortKey: map[string]string{
			"name":               strings.ToLower(template.Name),
			"cloudResourceTypes": strings.ToLower(cloudTypes),
			"description":        strings.ToLower(description),
			"labels":             strings.ToLower(labels),
			"status":             strings.ToLower(status),
			"abstract":           abstract,
			"createdAt":          template.CreatedAt.Format(time.RFC3339Nano),
			"updatedAt":          template.UpdatedAt.Format(time.RFC3339Nano),
			"entityName":         strings.ToLower(entityName),
			"age":                template.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: template.UpdatedAt,
		ColorKey:  strings.ToLower(status),
		Raw:       template,
	}
}

func TemplateWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "CLOUD TYPES", Key: "cloud_types"},
		{Title: "UPDATED", Key: "updated", SortField: "updated_at"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func TemplateWideRow(template client.Template) tabledata.Row {
	row := NewTemplateRenderer().Row(template)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		template.ID,
		row.Fields[3],
	}
	return row
}

func orSlice(values []string, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}
	return values
}
