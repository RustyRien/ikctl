package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type SecretRenderer struct{}

var secretListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "name"},
	{Title: "TYPE", Key: "secretType", SortField: "secret_type"},
	{Title: "PROVIDER", Key: "secretProvider", SortField: "secret_provider"},
	{Title: "STATE", Key: "state", SortField: "state"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "CREATED", Key: "createdAt", SortField: "created_at"},
	{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"},
	{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"},
	{Title: "INTEGRATION", Key: "integration", SortField: "integration.name"},
	{Title: "RESOURCES", Key: "resourcesCount", SortField: "resources_count"},
	{Title: "EXECUTORS", Key: "executorsCount", SortField: "executors_count"},
	{Title: "LABELS", Key: "labels"},
	{Title: "AGE", Key: "age", SortField: "created_at"},
}

func NewSecretRenderer() *SecretRenderer {
	return &SecretRenderer{}
}

func (r *SecretRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "TYPE", Key: "type", SortField: "secret_type"},
		{Title: "PROVIDER", Key: "provider", SortField: "secret_provider"},
		{Title: "STATE", Key: "state", SortField: "state"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func (r *SecretRenderer) Row(secret client.Secret) tabledata.Row {
	row := SecretListRow(secret)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[3],
		row.Fields[12],
	}
	row.SortKey = map[string]string{
		"name":     row.SortKey["name"],
		"type":     row.SortKey["secretType"],
		"provider": row.SortKey["secretProvider"],
		"state":    row.SortKey["state"],
		"age":      row.SortKey["age"],
	}
	return row
}

func SecretListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), secretListHeaders...)
}

func SecretListRow(secret client.Secret) tabledata.Row {
	secretType := normalizeCell(secret.SecretType)
	provider := normalizeCell(secret.SecretProvider)
	state := normalizeCell(secret.State)
	status := normalizeCell(secret.Status)
	created := normalizeCell(secret.CreatedAt.Format(time.RFC3339))
	updated := normalizeCell(secret.UpdatedAt.Format(time.RFC3339))
	creator := creatorName(secret.Creator)
	integration := integrationName(secret.Integration)
	resourcesCount := strconv.Itoa(secret.ResourcesCount)
	executorsCount := strconv.Itoa(secret.ExecutorsCount)
	labels := joinStrings(secret.Labels)
	age := ToAge(secret.CreatedAt, time.Now())

	return tabledata.Row{
		ID: secret.ID,
		Fields: []string{
			secret.Name,
			secretType,
			provider,
			state,
			status,
			created,
			updated,
			creator,
			integration,
			resourcesCount,
			executorsCount,
			labels,
			age,
		},
		SortKey: map[string]string{
			"name":           strings.ToLower(secret.Name),
			"secretType":     strings.ToLower(secretType),
			"secretProvider": strings.ToLower(provider),
			"state":          strings.ToLower(state),
			"status":         strings.ToLower(status),
			"createdAt":      secret.CreatedAt.Format(time.RFC3339Nano),
			"updatedAt":      secret.UpdatedAt.Format(time.RFC3339Nano),
			"creator":        strings.ToLower(creator),
			"integration":    strings.ToLower(integration),
			"resourcesCount": resourcesCount,
			"executorsCount": executorsCount,
			"labels":         strings.ToLower(labels),
			"age":            secret.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: secret.UpdatedAt,
		ColorKey:  strings.ToLower(state),
		Raw:       secret,
	}
}

func SecretWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "TYPE", Key: "secretType", SortField: "secret_type"},
		{Title: "PROVIDER", Key: "secretProvider", SortField: "secret_provider"},
		{Title: "STATE", Key: "state", SortField: "state"},
		{Title: "RESOURCES", Key: "resourcesCount", SortField: "resources_count"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func SecretWideRow(secret client.Secret) tabledata.Row {
	row := SecretListRow(secret)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[3],
		row.Fields[9],
		secret.ID,
		row.Fields[12],
	}
	return row
}
