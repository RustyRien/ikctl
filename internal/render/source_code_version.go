package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type SourceCodeVersionRenderer struct{}

var sourceCodeVersionListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "identifier"},
	{Title: "TEMPLATE", Key: "template", SortField: "template.name"},
	{Title: "CODE REPOSITORY", Key: "sourceCode", SortField: "source_code.source_code_url"},
	{Title: "TAG", Key: "sourceCodeVersion", SortField: "source_code_version"},
	{Title: "BRANCH", Key: "sourceCodeBranch", SortField: "source_code_branch"},
	{Title: "FOLDER", Key: "sourceCodeFolder", SortField: "source_code_folder"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "CREATED", Key: "createdAt", SortField: "created_at"},
	{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"},
	{Title: "RESOURCES", Key: "resourcesCount", SortField: "resources_count"},
	{Title: "LABELS", Key: "labels"},
	{Title: "AGE", Key: "age", SortField: "created_at"},
}

func NewSourceCodeVersionRenderer() *SourceCodeVersionRenderer {
	return &SourceCodeVersionRenderer{}
}

func (r *SourceCodeVersionRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "identifier"},
		{Title: "TEMPLATE", Key: "template", SortField: "template.name"},
		{Title: "REPOSITORY", Key: "repository", SortField: "source_code.source_code_url"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "CREATED", Key: "created", SortField: "created_at"},
		{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"},
	}
}

func (r *SourceCodeVersionRenderer) Row(sourceCodeVersion client.SourceCodeVersion) tabledata.Row {
	row := SourceCodeVersionListRow(sourceCodeVersion)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[6],
		row.Fields[7],
		row.Fields[8],
	}
	row.SortKey = map[string]string{
		"name":       row.SortKey["name"],
		"template":   row.SortKey["template"],
		"repository": row.SortKey["sourceCode"],
		"status":     row.SortKey["status"],
		"created":    row.SortKey["createdAt"],
		"creator":    row.SortKey["creator"],
	}
	return row
}

func SourceCodeVersionListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), sourceCodeVersionListHeaders...)
}

func SourceCodeVersionListRow(sourceCodeVersion client.SourceCodeVersion) tabledata.Row {
	name := normalizeCell(sourceCodeVersionName(sourceCodeVersion))
	template := templateName(sourceCodeVersion.Template)
	repository := sourceCodeDisplayName(sourceCodeVersion.SourceCode)
	tag := normalizeCell(sourceCodeVersion.SourceCodeVersion)
	branch := normalizeCell(sourceCodeVersion.SourceCodeBranch)
	folder := normalizeCell(sourceCodeVersion.SourceCodeFolder)
	status := normalizeCell(sourceCodeVersion.Status)
	created := normalizeCell(sourceCodeVersion.CreatedAt.Format(time.RFC3339))
	creator := creatorName(sourceCodeVersion.Creator)
	resourcesCount := strconv.Itoa(sourceCodeVersion.ResourcesCount)
	labels := joinStrings(sourceCodeVersion.Labels)
	age := ToAge(sourceCodeVersion.CreatedAt, time.Now())

	return tabledata.Row{
		ID: sourceCodeVersion.ID,
		Fields: []string{
			name,
			template,
			repository,
			tag,
			branch,
			folder,
			status,
			created,
			creator,
			resourcesCount,
			labels,
			age,
		},
		SortKey: map[string]string{
			"name":              strings.ToLower(name),
			"template":          strings.ToLower(template),
			"sourceCode":        strings.ToLower(repository),
			"sourceCodeVersion": strings.ToLower(tag),
			"sourceCodeBranch":  strings.ToLower(branch),
			"sourceCodeFolder":  strings.ToLower(folder),
			"status":            strings.ToLower(status),
			"createdAt":         sourceCodeVersion.CreatedAt.Format(time.RFC3339Nano),
			"creator":           strings.ToLower(creator),
			"resourcesCount":    resourcesCount,
			"labels":            strings.ToLower(labels),
			"age":               sourceCodeVersion.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: sourceCodeVersion.UpdatedAt,
		ColorKey:  strings.ToLower(status),
		Raw:       sourceCodeVersion,
	}
}

func SourceCodeVersionWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "identifier"},
		{Title: "TEMPLATE", Key: "template", SortField: "template.name"},
		{Title: "REPOSITORY", Key: "sourceCode", SortField: "source_code.source_code_url"},
		{Title: "TAG", Key: "sourceCodeVersion", SortField: "source_code_version"},
		{Title: "BRANCH", Key: "sourceCodeBranch", SortField: "source_code_branch"},
		{Title: "FOLDER", Key: "sourceCodeFolder", SortField: "source_code_folder"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "RESOURCES", Key: "resourcesCount", SortField: "resources_count"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func SourceCodeVersionWideRow(sourceCodeVersion client.SourceCodeVersion) tabledata.Row {
	row := SourceCodeVersionListRow(sourceCodeVersion)
	row.Fields = []string{
		row.Fields[0],
		row.Fields[1],
		row.Fields[2],
		row.Fields[3],
		row.Fields[4],
		row.Fields[5],
		row.Fields[6],
		row.Fields[9],
		sourceCodeVersion.ID,
		row.Fields[11],
	}
	return row
}

func sourceCodeVersionName(value client.SourceCodeVersion) string {
	if strings.TrimSpace(value.Identifier) != "" {
		return value.Identifier
	}
	if strings.TrimSpace(value.SourceCodeVersion) != "" {
		return value.SourceCodeVersion
	}
	if strings.TrimSpace(value.SourceCodeBranch) != "" {
		return value.SourceCodeBranch
	}
	return value.ID
}

func sourceCodeDisplayName(sourceCode *client.SourceCode) string {
	if sourceCode == nil {
		return "-"
	}
	return normalizeCell(sourceCode.DisplayName())
}

func templateName(template *client.Template) string {
	if template == nil || strings.TrimSpace(template.Name) == "" {
		return "-"
	}
	return template.Name
}
