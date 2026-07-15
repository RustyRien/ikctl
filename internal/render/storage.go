package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type StorageRenderer struct{}

var storageListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "name"},
	{Title: "TYPE", Key: "storageType", SortField: "storage_type"},
	{Title: "PROVIDER", Key: "storageProvider", SortField: "storage_provider"},
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

func NewStorageRenderer() *StorageRenderer {
	return &StorageRenderer{}
}

func (r *StorageRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "TYPE", Key: "type", SortField: "storage_type"},
		{Title: "PROVIDER", Key: "provider", SortField: "storage_provider"},
		{Title: "STATE", Key: "state", SortField: "state"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func (r *StorageRenderer) Row(storage client.Storage) tabledata.Row {
	row := StorageListRow(storage)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[3],
		row.Fields[12],
	}
	row.SortKey = map[string]string{
		"name":     row.SortKey["name"],
		"type":     row.SortKey["storageType"],
		"provider": row.SortKey["storageProvider"],
		"state":    row.SortKey["state"],
		"age":      row.SortKey["age"],
	}
	return row
}

func StorageListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), storageListHeaders...)
}

func StorageListRow(storage client.Storage) tabledata.Row {
	storageType := normalizeCell(storage.StorageType)
	provider := normalizeCell(storage.StorageProvider)
	state := normalizeCell(storage.State)
	status := normalizeCell(storage.Status)
	created := normalizeCell(storage.CreatedAt.Format(time.RFC3339))
	updated := normalizeCell(storage.UpdatedAt.Format(time.RFC3339))
	creator := creatorName(storage.Creator)
	integration := integrationName(storage.Integration)
	resourcesCount := strconv.Itoa(storage.ResourcesCount)
	executorsCount := strconv.Itoa(storage.ExecutorsCount)
	labels := joinStrings(storage.Labels)
	age := ToAge(storage.CreatedAt, time.Now())

	return tabledata.Row{
		ID: storage.ID,
		Fields: []string{
			storage.Name,
			storageType,
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
			"name":            strings.ToLower(storage.Name),
			"storageType":     strings.ToLower(storageType),
			"storageProvider": strings.ToLower(provider),
			"state":           strings.ToLower(state),
			"status":          strings.ToLower(status),
			"createdAt":       storage.CreatedAt.Format(time.RFC3339Nano),
			"updatedAt":       storage.UpdatedAt.Format(time.RFC3339Nano),
			"creator":         strings.ToLower(creator),
			"integration":     strings.ToLower(integration),
			"resourcesCount":  resourcesCount,
			"executorsCount":  executorsCount,
			"labels":          strings.ToLower(labels),
			"age":             storage.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: storage.UpdatedAt,
		ColorKey:  strings.ToLower(state),
		Raw:       storage,
	}
}

func StorageWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "TYPE", Key: "storageType", SortField: "storage_type"},
		{Title: "PROVIDER", Key: "storageProvider", SortField: "storage_provider"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "RESOURCES", Key: "resourcesCount", SortField: "resources_count"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func StorageWideRow(storage client.Storage) tabledata.Row {
	row := StorageListRow(storage)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[4],
		row.Fields[9],
		storage.ID,
		row.Fields[12],
	}
	return row
}

func creatorName(creator *client.Creator) string {
	if creator == nil {
		return "-"
	}
	if creator.DisplayName != "" {
		return creator.DisplayName
	}
	if creator.Identifier != "" {
		return creator.Identifier
	}
	if creator.ID != "" {
		return creator.ID
	}
	return "-"
}

func integrationName(integration *client.Integration) string {
	if integration == nil || strings.TrimSpace(integration.Name) == "" {
		return "-"
	}
	return integration.Name
}
