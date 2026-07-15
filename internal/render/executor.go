package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type ExecutorRenderer struct{}

var executorListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "name"},
	{Title: "CODE REPOSITORY", Key: "sourceCode", SortField: "source_code.source_code_url"},
	{Title: "STATE", Key: "state", SortField: "state"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "CREATED", Key: "createdAt", SortField: "created_at"},
	{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"},
	{Title: "CREATOR", Key: "creator", SortField: "creator.identifier"},
	{Title: "LABELS", Key: "labels"},
	{Title: "DESCRIPTION", Key: "description"},
	{Title: "RUNTIME", Key: "runtime"},
	{Title: "COMMAND ARGS", Key: "commandArgs", SortField: "command_args"},
	{Title: "VERSION", Key: "sourceCodeVersion", SortField: "source_code_version"},
	{Title: "BRANCH", Key: "sourceCodeBranch", SortField: "source_code_branch"},
	{Title: "FOLDER", Key: "sourceCodeFolder", SortField: "source_code_folder"},
	{Title: "STORAGE", Key: "storage", SortField: "storage.name"},
	{Title: "STORAGE PATH", Key: "storagePath", SortField: "storage_path"},
	{Title: "INTEGRATIONS", Key: "integrationIds", SortField: "integration_ids.name"},
	{Title: "SECRETS", Key: "secretIds", SortField: "secret_ids.name"},
	{Title: "REV", Key: "revisionNumber", SortField: "revision_number"},
	{Title: "AGE", Key: "age", SortField: "created_at"},
}

func NewExecutorRenderer() *ExecutorRenderer {
	return &ExecutorRenderer{}
}

func (r *ExecutorRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "CODE REPOSITORY", Key: "sourceCode", SortField: "source_code.source_code_url"},
		{Title: "STATE", Key: "state", SortField: "state"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func (r *ExecutorRenderer) Row(executor client.Executor) tabledata.Row {
	row := ExecutorListRow(executor)
	row.Fields = []string{row.Fields[0], row.Fields[1], row.Fields[2], row.Fields[3], row.Fields[19]}
	row.SortKey = map[string]string{
		"name":       row.SortKey["name"],
		"sourceCode": row.SortKey["sourceCode"],
		"state":      row.SortKey["state"],
		"status":     row.SortKey["status"],
		"age":        row.SortKey["age"],
	}
	return row
}

func ExecutorListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), executorListHeaders...)
}

func ExecutorListRow(executor client.Executor) tabledata.Row {
	sourceCode := "-"
	if executor.SourceCode != nil {
		sourceCode = normalizeCell(executor.SourceCode.DisplayName())
	}
	state := normalizeCell(executor.State)
	status := normalizeCell(executor.Status)
	created := normalizeCell(executor.CreatedAt.Format(time.RFC3339))
	updated := normalizeCell(executor.UpdatedAt.Format(time.RFC3339))
	creator := creatorName(executor.Creator)
	labels := joinStrings(executor.Labels)
	description := normalizeCell(executor.Description)
	runtime := normalizeCell(executor.Runtime)
	commandArgs := normalizeCell(executor.CommandArgs)
	version := normalizeCell(executor.SourceCodeVersion)
	branch := normalizeCell(executor.SourceCodeBranch)
	folder := normalizeCell(executor.SourceCodeFolder)
	storage := "-"
	if executor.Storage != nil {
		storage = normalizeCell(executor.Storage.Name)
	}
	storagePath := normalizeCell(executor.StoragePath)
	integrations := joinIntegrationNames(executor.Integrations)
	secrets := joinSecretNames(executor.Secrets)
	revision := strconv.Itoa(executor.RevisionNumber)
	age := ToAge(executor.CreatedAt, time.Now())

	return tabledata.Row{
		ID: executor.ID,
		Fields: []string{
			executor.Name,
			sourceCode,
			state,
			status,
			created,
			updated,
			creator,
			labels,
			description,
			runtime,
			commandArgs,
			version,
			branch,
			folder,
			storage,
			storagePath,
			integrations,
			secrets,
			revision,
			age,
		},
		SortKey: map[string]string{
			"name":              strings.ToLower(executor.Name),
			"sourceCode":        strings.ToLower(sourceCode),
			"state":             strings.ToLower(state),
			"status":            strings.ToLower(status),
			"createdAt":         executor.CreatedAt.Format(time.RFC3339Nano),
			"updatedAt":         executor.UpdatedAt.Format(time.RFC3339Nano),
			"creator":           strings.ToLower(creator),
			"labels":            strings.ToLower(labels),
			"description":       strings.ToLower(description),
			"runtime":           strings.ToLower(runtime),
			"commandArgs":       strings.ToLower(commandArgs),
			"sourceCodeVersion": strings.ToLower(version),
			"sourceCodeBranch":  strings.ToLower(branch),
			"sourceCodeFolder":  strings.ToLower(folder),
			"storage":           strings.ToLower(storage),
			"storagePath":       strings.ToLower(storagePath),
			"integrationIds":    strings.ToLower(integrations),
			"secretIds":         strings.ToLower(secrets),
			"revisionNumber":    revision,
			"age":               executor.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: executor.UpdatedAt,
		ColorKey:  strings.ToLower(status),
		Raw:       executor,
	}
}

func ExecutorWideHeaders() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "CODE REPOSITORY", Key: "sourceCode", SortField: "source_code.source_code_url"},
		{Title: "STATE", Key: "state", SortField: "state"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "STORAGE", Key: "storage", SortField: "storage.name"},
		{Title: "REV", Key: "revisionNumber", SortField: "revision_number"},
		{Title: "ID", Key: "id"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func ExecutorWideRow(executor client.Executor) tabledata.Row {
	row := ExecutorListRow(executor)
	row.Fields = []string{row.Fields[0], row.Fields[1], row.Fields[2], row.Fields[3], row.Fields[14], row.Fields[18], executor.ID, row.Fields[19]}
	return row
}
