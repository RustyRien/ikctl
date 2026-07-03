package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type ResourceRenderer struct{}

var resourceListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "name"},
	{Title: "TEMPLATE", Key: "template"},
	{Title: "TEMPLATE VERSION", Key: "sourceCodeVersion", SortField: "source_code_version.tag"},
	{Title: "STATE", Key: "state", SortField: "state"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "CREATED", Key: "createdAt", SortField: "created_at"},
	{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"},
	{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"},
	{Title: "STORAGE", Key: "storage"},
	{Title: "WORKSPACE", Key: "workspace"},
	{Title: "INTEGRATIONS", Key: "integrationIds"},
	{Title: "SECRETS", Key: "secretIds", SortField: "secret_ids.name"},
	{Title: "PARENTS", Key: "parents", SortField: "parents.name"},
	{Title: "CHILDREN", Key: "children", SortField: "children.name"},
	{Title: "VARIABLES", Key: "variables"},
	{Title: "OUTPUTS", Key: "outputs"},
	{Title: "LABELS", Key: "labels"},
	{Title: "DEPENDENCY TAGS", Key: "dependencyTags"},
	{Title: "DEPENDENCY CONFIG", Key: "dependencyConfig"},
	{Title: "AGE", Key: "age", SortField: "created_at"},
}

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
	row := ResourceListRow(resource)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[3],
		row.Fields[4],
		row.Fields[9],
		row.Fields[19],
	}
	row.SortKey = map[string]string{
		"name":      row.SortKey["name"],
		"template":  row.SortKey["template"],
		"state":     row.SortKey["state"],
		"status":    row.SortKey["status"],
		"workspace": row.SortKey["workspace"],
		"age":       row.SortKey["age"],
	}
	return row
}

func ResourceListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), resourceListHeaders...)
}

func ResourceListRow(resource client.Resource) tabledata.Row {
	template := "-"
	if resource.Template != nil && resource.Template.Name != "" {
		template = resource.Template.Name
	}

	workspace := "-"
	if resource.Workspace != nil && resource.Workspace.Name != "" {
		workspace = resource.Workspace.Name
	}

	storage := "-"
	if resource.Storage != nil && resource.Storage.Name != "" {
		storage = resource.Storage.Name
	}

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

	sourceCodeVersion := "-"
	if resource.SourceCodeVersion != nil {
		if resource.SourceCodeVersion.SourceCodeVersion != "" {
			sourceCodeVersion = resource.SourceCodeVersion.SourceCodeVersion
		} else if resource.SourceCodeVersion.SourceCodeBranch != "" {
			sourceCodeVersion = resource.SourceCodeVersion.SourceCodeBranch
		}
	}

	age := ToAge(resource.CreatedAt, time.Now())
	created := resource.CreatedAt.Format(time.RFC3339)
	updated := resource.UpdatedAt.Format(time.RFC3339)
	status := normalizeCell(resource.Status)
	state := normalizeCell(resource.State)
	integrations := joinIntegrationNames(resource.Integrations)
	secrets := joinSecretNames(resource.Secrets)
	parents := joinResourceReferenceNames(resource.Parents)
	children := joinResourceReferenceNames(resource.Children)
	variables := joinMapNames(resource.Variables)
	outputs := joinMapNames(resource.Outputs)
	labels := joinStrings(resource.Labels)
	dependencyTags := joinNamedKeyValueMaps(resource.DependencyTags)
	dependencyConfig := joinNamedKeyValueMaps(resource.DependencyConfig)

	return tabledata.Row{
		ID: resource.ID,
		Fields: []string{
			resource.Name,
			template,
			sourceCodeVersion,
			state,
			status,
			created,
			updated,
			creator,
			storage,
			workspace,
			integrations,
			secrets,
			parents,
			children,
			variables,
			outputs,
			labels,
			dependencyTags,
			dependencyConfig,
			age,
		},
		SortKey: map[string]string{
			"name":              strings.ToLower(resource.Name),
			"template":          strings.ToLower(template),
			"sourceCodeVersion": strings.ToLower(sourceCodeVersion),
			"state":             strings.ToLower(state),
			"status":            strings.ToLower(status),
			"createdAt":         created,
			"updatedAt":         updated,
			"creator":           strings.ToLower(creator),
			"storage":           strings.ToLower(storage),
			"workspace":         strings.ToLower(workspace),
			"integrationIds":    strings.ToLower(integrations),
			"secretIds":         strings.ToLower(secrets),
			"parents":           strings.ToLower(parents),
			"children":          strings.ToLower(children),
			"variables":         strings.ToLower(variables),
			"outputs":           strings.ToLower(outputs),
			"labels":            strings.ToLower(labels),
			"dependencyTags":    strings.ToLower(dependencyTags),
			"dependencyConfig":  strings.ToLower(dependencyConfig),
			"age":               resource.CreatedAt.Format(time.RFC3339Nano),
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
	row := ResourceListRow(resource)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[3],
		row.Fields[4],
		row.Fields[9],
		strconv.Itoa(resource.RevisionNumber),
		row.Fields[7],
		resource.ID,
		row.Fields[19],
	}
	return row
}

func joinIntegrationNames(items []client.Integration) string {
	if len(items) == 0 {
		return "-"
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			values = append(values, name)
		}
	}
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func joinSecretNames(items []client.Secret) string {
	if len(items) == 0 {
		return "-"
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			values = append(values, name)
		}
	}
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func joinResourceReferenceNames(items []client.ResourceReference) string {
	if len(items) == 0 {
		return "-"
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			values = append(values, name)
		}
	}
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func joinMapNames(items []map[string]any) string {
	if len(items) == 0 {
		return "-"
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		name, _ := item["name"].(string)
		name = strings.TrimSpace(name)
		if name != "" {
			values = append(values, name)
		}
	}
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func joinNamedKeyValueMaps(items []map[string]any) string {
	if len(items) == 0 {
		return "-"
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		name, _ := item["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		values = append(values, name+":"+stringValue(item["value"]))
	}
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func joinStrings(items []string) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			values = append(values, item)
		}
	}
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(strconv.FormatBool(false))
}
