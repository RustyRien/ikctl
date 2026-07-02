package render

import (
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type IntegrationRenderer struct{}

func NewIntegrationRenderer() *IntegrationRenderer {
	return &IntegrationRenderer{}
}

func (r *IntegrationRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "PROVIDER", Key: "provider", SortField: "integration_provider"},
		{Title: "TYPE", Key: "type", SortField: "integration_type"},
		{Title: "UPDATED", Key: "updated", SortField: "updated_at"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func (r *IntegrationRenderer) Row(integration client.Integration) tabledata.Row {
	provider := normalizeCell(integration.IntegrationProvider)
	integrationType := normalizeCell(integration.IntegrationType)
	updated := normalizeCell(integration.UpdatedAt.Format(time.RFC3339))

	return tabledata.Row{
		ID: integration.ID,
		Fields: []string{
			integration.Name,
			provider,
			integrationType,
			updated,
			ToAge(integration.CreatedAt, time.Now()),
		},
		SortKey: map[string]string{
			"name":     strings.ToLower(integration.Name),
			"provider": strings.ToLower(provider),
			"type":     strings.ToLower(integrationType),
			"updated":  integration.UpdatedAt.Format(time.RFC3339Nano),
			"age":      integration.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: integration.UpdatedAt,
		ColorKey:  "ready",
		Raw:       integration,
	}
}

func IntegrationWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "PROVIDER", Key: "provider", SortField: "integration_provider"},
		{Title: "TYPE", Key: "type", SortField: "integration_type"},
		{Title: "UPDATED", Key: "updated", SortField: "updated_at"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func IntegrationWideRow(integration client.Integration) tabledata.Row {
	row := NewIntegrationRenderer().Row(integration)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[3],
		integration.ID,
		row.Fields[4],
	}
	return row
}
