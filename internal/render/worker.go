package render

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type WorkerRenderer struct{}

var workerListHeaders = []tabledata.Header{
	{Title: "NAME", Key: "name", SortField: "name"},
	{Title: "STATUS", Key: "status", SortField: "status"},
	{Title: "HOST", Key: "host", SortField: "host"},
	{Title: "LAST TASK", Key: "currentTask"},
	{Title: "COMPLETED", Key: "tasksCompleted", SortField: "tasks_completed"},
	{Title: "CREATED", Key: "createdAt", SortField: "created_at"},
	{Title: "UPDATED", Key: "updatedAt", SortField: "updated_at"},
	{Title: "HOST INFO", Key: "hostMetadata"},
	{Title: "AGE", Key: "age", SortField: "created_at"},
}

func NewWorkerRenderer() *WorkerRenderer {
	return &WorkerRenderer{}
}

func (r *WorkerRenderer) Headers() []tabledata.Header {
	return []tabledata.Header{
		{Title: "NAME", Key: "name", SortField: "name"},
		{Title: "STATUS", Key: "status", SortField: "status"},
		{Title: "HOST", Key: "host", SortField: "host"},
		{Title: "COMPLETED", Key: "completed", SortField: "tasks_completed"},
		{Title: "AGE", Key: "age", SortField: "created_at"},
	}
}

func (r *WorkerRenderer) Row(worker client.Worker) tabledata.Row {
	row := WorkerListRow(worker)
	row.Fields = []string{row.Fields[0], row.Fields[1], row.Fields[2], row.Fields[4], row.Fields[8]}
	row.SortKey = map[string]string{
		"name":      row.SortKey["name"],
		"status":    row.SortKey["status"],
		"host":      row.SortKey["host"],
		"completed": row.SortKey["tasksCompleted"],
		"age":       row.SortKey["age"],
	}
	return row
}

func WorkerListHeaders() []tabledata.Header {
	return append([]tabledata.Header(nil), workerListHeaders...)
}

func WorkerListRow(worker client.Worker) tabledata.Row {
	status := normalizeCell(worker.Status)
	host := normalizeCell(worker.Host)
	lastTask := workerCurrentTaskSummary(worker.CurrentTask)
	completed := workerTasksCompleted(worker.TasksCompleted)
	created := normalizeCell(worker.CreatedAt.Format(time.RFC3339))
	updated := normalizeCell(worker.UpdatedAt.Format(time.RFC3339))
	hostInfo := workerHostInfoSummary(worker.HostMetadata)
	age := ToAge(worker.CreatedAt, time.Now())

	return tabledata.Row{
		ID: worker.ID,
		Fields: []string{
			worker.Name,
			status,
			host,
			lastTask,
			completed,
			created,
			updated,
			hostInfo,
			age,
		},
		SortKey: map[string]string{
			"name":           strings.ToLower(worker.Name),
			"status":         strings.ToLower(status),
			"host":           strings.ToLower(host),
			"currentTask":    strings.ToLower(lastTask),
			"tasksCompleted": completed,
			"createdAt":      worker.CreatedAt.Format(time.RFC3339Nano),
			"updatedAt":      worker.UpdatedAt.Format(time.RFC3339Nano),
			"hostMetadata":   strings.ToLower(hostInfo),
			"age":            worker.CreatedAt.Format(time.RFC3339Nano),
		},
		UpdatedAt: worker.UpdatedAt,
		ColorKey:  strings.ToLower(status),
		Raw:       worker,
	}
}

func WorkerWideHeaders() []tabledata.Header {
	return WorkerListHeaders()
}

func WorkerWideRow(worker client.Worker) tabledata.Row {
	return WorkerListRow(worker)
}

func workerTasksCompleted(value *int) string {
	if value == nil {
		return "0"
	}
	return fmt.Sprintf("%d", *value)
}

func workerCurrentTaskSummary(task map[string]any) string {
	if len(task) == 0 {
		return "-"
	}
	entity := strings.TrimSpace(fmt.Sprint(task["entity"]))
	action := strings.TrimSpace(fmt.Sprint(task["action"]))
	if entity == "" && action == "" {
		return "-"
	}
	if entity == "" {
		return action
	}
	if action == "" {
		return entity
	}
	return entity + " / " + action
}

func workerHostInfoSummary(metadata map[string]any) string {
	if len(metadata) == 0 {
		return "-"
	}
	platform := strings.TrimSpace(fmt.Sprint(metadata["platform"]))
	arch := strings.TrimSpace(fmt.Sprint(metadata["machine"]))
	if platform == "" && arch == "" {
		return workerFlatMapSummary(metadata)
	}
	if platform == "" {
		return arch
	}
	if arch == "" {
		return platform
	}
	return fmt.Sprintf("%s (%s)", platform, arch)
}

func workerFlatMapSummary(metadata map[string]any) string {
	flattened := WorkerFlattenMap(metadata, "")
	if len(flattened) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(flattened))
	for key := range flattened {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, flattened[key]))
	}
	return strings.Join(parts, ", ")
}

func WorkerFlattenMap(input map[string]any, prefix string) map[string]any {
	result := map[string]any{}
	for key, value := range input {
		composed := key
		if prefix != "" {
			composed = prefix + "_" + key
		}
		nested, ok := value.(map[string]any)
		if ok {
			for nestedKey, nestedValue := range WorkerFlattenMap(nested, composed) {
				result[nestedKey] = nestedValue
			}
			continue
		}
		result[composed] = value
	}
	return result
}
