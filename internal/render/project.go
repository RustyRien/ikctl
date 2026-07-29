package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type ProjectRenderer struct{}

var projectListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "name"},
	{Title: "DESCRIPTION", Key: "description"},
	{Title: "LABELS", Key: "labels"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "RESOURCES", Key: "resourcesCount", SortField: "resources_count"},
	{Title: "CREATED", Key: "createdAt", SortField: "created_at"},
	{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"},
	{Title: "ENTITY", Key: "entityName"},
	{Title: "AGE", Key: "age", SortField: "created_at"},
}

func NewProjectRenderer() *ProjectRenderer { return &ProjectRenderer{} }

func (r *ProjectRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{{Title: "NAME", Key: "name", SortField: "name"}, {Title: "STATUS", Key: "status", SortField: "status"}, {Title: "RESOURCES", Key: "resources", SortField: "resources_count"}, {Title: "UPDATED", Key: "updated", SortField: "updated_at"}, {Title: "AGE", Key: "age", SortField: "created_at"}}
}

func (r *ProjectRenderer) Row(project client.Project) tabledata.Row {
	row := ProjectListRow(project)
	row.Fields = []string{row.Fields[0], row.Fields[3], row.Fields[4], row.Fields[6], row.Fields[8]}
	row.SortKey = map[string]string{"name": row.SortKey["name"], "status": row.SortKey["status"], "resources": row.SortKey["resourcesCount"], "updated": row.SortKey["updatedAt"], "age": row.SortKey["age"]}
	return row
}

func ProjectListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), projectListHeaders...)
}

func ProjectListRow(project client.Project) tabledata.Row {
	description := normalizeCell(project.Description)
	labels := joinStrings(project.Labels)
	status := normalizeCell(project.Status)
	resourcesCount := normalizeCell(fmt.Sprintf("%d", project.ResourcesCount))
	created := normalizeCell(project.CreatedAt.Format(time.RFC3339))
	updated := normalizeCell(project.UpdatedAt.Format(time.RFC3339))
	entityName := normalizeCell(project.EntityName)
	return tabledata.Row{ID: project.ID, Fields: []string{project.Name, description, labels, status, resourcesCount, created, updated, entityName, ToAge(project.CreatedAt, time.Now())}, SortKey: map[string]string{"name": strings.ToLower(project.Name), "description": strings.ToLower(description), "labels": strings.ToLower(labels), "status": strings.ToLower(status), "resourcesCount": fmt.Sprintf("%09d", project.ResourcesCount), "createdAt": project.CreatedAt.Format(time.RFC3339Nano), "updatedAt": project.UpdatedAt.Format(time.RFC3339Nano), "entityName": strings.ToLower(entityName), "age": project.CreatedAt.Format(time.RFC3339Nano)}, UpdatedAt: project.UpdatedAt, ColorKey: strings.ToLower(status), Raw: project}
}

func ProjectWideHeaders() []tabledata.Header {
	return []tabledata.Header{{Title: "NAME", Key: "name", SortField: "name"}, {Title: "STATUS", Key: "status", SortField: "status"}, {Title: "RESOURCES", Key: "resources", SortField: "resources_count"}, {Title: "UPDATED", Key: "updated", SortField: "updated_at"}, {Title: "ID", Key: "id"}, {Title: "AGE", Key: "age", SortField: "created_at"}}
}

func ProjectWideRow(project client.Project) tabledata.Row {
	row := NewProjectRenderer().Row(project)
	row.Fields = []string{row.Fields[0], row.Fields[1], row.Fields[2], row.Fields[3], project.ID, row.Fields[4]}
	return row
}
