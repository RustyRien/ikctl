package render

import (
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type SourceCodeRenderer struct{}

var sourceCodeListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "source_code_url"},
	{Title: "IDENTIFIER", Key: "identifier", SortField: "identifier"},
	{Title: "URL", Key: "sourceCodeUrl", SortField: "source_code_url"},
	{Title: "PROVIDER", Key: "sourceCodeProvider", SortField: "source_code_provider"},
	{Title: "LANGUAGE", Key: "sourceCodeLanguage", SortField: "source_code_language"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "INTEGRATION", Key: "integration", SortField: "integration.name"},
	{Title: "LABELS", Key: "labels"},
	{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"},
	{Title: "CREATED", Key: "createdAt", SortField: "created_at"},
	{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"},
	{Title: "AGE", Key: "age", SortField: "created_at"},
}

func NewSourceCodeRenderer() *SourceCodeRenderer {
	return &SourceCodeRenderer{}
}

func (r *SourceCodeRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "source_code_url"},
		{Title: "PROVIDER", Key: "provider", SortField: "source_code_provider"},
		{Title: "LANGUAGE", Key: "language", SortField: "source_code_language"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "UPDATED", Key: "updated", SortField: "updated_at"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func (r *SourceCodeRenderer) Row(sourceCode client.SourceCode) tabledata.Row {
	row := SourceCodeListRow(sourceCode)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[3],
		row.Fields[4],
		row.Fields[5],
		row.Fields[8],
		row.Fields[11],
	}
	row.SortKey = map[string]string{
		"name":     row.SortKey["name"],
		"provider": row.SortKey["sourceCodeProvider"],
		"language": row.SortKey["sourceCodeLanguage"],
		"status":   row.SortKey["status"],
		"updated":  row.SortKey["updatedAt"],
		"age":      row.SortKey["age"],
	}
	return row
}

func SourceCodeListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), sourceCodeListHeaders...)
}

func SourceCodeListRow(sourceCode client.SourceCode) tabledata.Row {
	name := normalizeCell(sourceCode.DisplayName())
	identifier := normalizeCell(sourceCode.Identifier)
	url := normalizeCell(sourceCode.SourceCodeURL)
	provider := normalizeCell(sourceCode.SourceCodeProvider)
	language := normalizeCell(sourceCode.SourceCodeLanguage)
	status := normalizeCell(sourceCode.Status)
	integration := integrationName(sourceCode.Integration)
	labels := joinStrings(sourceCode.Labels)
	updated := normalizeCell(sourceCode.UpdatedAt.Format(time.RFC3339))
	created := normalizeCell(sourceCode.CreatedAt.Format(time.RFC3339))
	creator := creatorName(sourceCode.Creator)
	age := ToAge(sourceCode.CreatedAt, time.Now())

	return tabledata.Row{
		ID: sourceCode.ID,
		Fields: []string{
			name,
			identifier,
			url,
			provider,
			language,
			status,
			integration,
			labels,
			updated,
			created,
			creator,
			age,
		},
		SortKey: map[string]string{
			"name":               strings.ToLower(name),
			"identifier":         strings.ToLower(identifier),
			"sourceCodeUrl":      strings.ToLower(url),
			"sourceCodeProvider": strings.ToLower(provider),
			"sourceCodeLanguage": strings.ToLower(language),
			"status":             strings.ToLower(status),
			"integration":        strings.ToLower(integration),
			"labels":             strings.ToLower(labels),
			"updatedAt":          sourceCode.UpdatedAt.Format(time.RFC3339Nano),
			"createdAt":          sourceCode.CreatedAt.Format(time.RFC3339Nano),
			"creator":            strings.ToLower(creator),
			"age":                sourceCode.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: sourceCode.UpdatedAt,
		ColorKey:  strings.ToLower(status),
		Raw:       sourceCode,
	}
}

func SourceCodeWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name"},
		{Title: "PROVIDER", Key: "provider", SortField: "source_code_provider"},
		{Title: "LANGUAGE", Key: "language", SortField: "source_code_language"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "INTEGRATION", Key: "integration", SortField: "integration.name"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func SourceCodeWideRow(sourceCode client.SourceCode) tabledata.Row {
	row := SourceCodeListRow(sourceCode)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[3],
		row.Fields[4],
		row.Fields[5],
		row.Fields[6],
		sourceCode.ID,
		row.Fields[11],
	}
	return row
}
