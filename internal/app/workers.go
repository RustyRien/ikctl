package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/rivo/tview"
)

func (a *App) openWorkerOverview(id string, name string) {
	a.stopLiveLogStream()

	title := fmt.Sprintf("Worker: %s", name)
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.activeSourceCodeDetail = nil
	a.activeSourceCodeVersionDetail = nil
	a.activeIntegrationDetail = nil
	a.activeStorageDetail = nil
	a.activeWorkerDetail = &entityDetailSelection{ID: id, Name: name, Kind: "workers"}
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading worker overview...")
	a.ui.SetWorkerOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Worker(ctx, id)
		var primitive tview.Primitive
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load worker overview.\n\n%v", err))
		} else if full != nil {
			a.activeWorkerDetail = &entityDetailSelection{ID: full.ID, Name: full.Name, Kind: "workers"}
			primitive = workerOverviewView(*full)
		} else {
			a.activeWorkerDetail = nil
			primitive = errorView("Worker not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.clearOverviewJumpState()
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetWorkerOverviewHotkeys()
		})
	}()
}

func workerOverviewView(worker client.Worker) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", worker.Name},
		{"ID", worker.ID},
		{"Status", blankDash(worker.Status)},
		{"Host", blankDash(worker.Host)},
		{"Created", worker.CreatedAt.Format(time.RFC3339)},
		{"Updated", worker.UpdatedAt.Format(time.RFC3339)},
	})

	completed := "0"
	if worker.TasksCompleted != nil {
		completed = fmt.Sprintf("%d", *worker.TasksCompleted)
	}
	usage := kvTable("Activity", [][2]string{
		{"Tasks completed", completed},
		{"Current task", workerCurrentTaskDescription(worker.CurrentTask)},
	})

	hostInfo := tview.NewTextView()
	hostInfo.SetBorder(true)
	hostInfo.SetTitle("Host Info")
	hostInfo.SetWrap(true)
	hostInfo.SetDynamicColors(true)
	hostInfo.SetText(workerMetadataText(worker.HostMetadata))

	currentTask := tview.NewTextView()
	currentTask.SetBorder(true)
	currentTask.SetTitle("Current Task")
	currentTask.SetWrap(true)
	currentTask.SetDynamicColors(true)
	currentTask.SetText(workerTaskText(worker.CurrentTask))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(usage, 0, 1, false), 8, 0, true)
	root.AddItem(tview.NewFlex().
		AddItem(hostInfo, 0, 1, false).
		AddItem(currentTask, 0, 1, false), 0, 1, false)
	root.AddItem(overviewFooter(workerOverviewHint()), 1, 0, false)
	return root
}

func workerOverviewHint() string {
	return strings.Join([]string{"y yaml", "l logs", "a audit", "Esc/q close"}, "  ")
}

func workerCurrentTaskDescription(task map[string]any) string {
	if len(task) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(task))
	keys := make([]string, 0, len(task))
	for key := range task {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s: %v", key, task[key]))
	}
	return strings.Join(parts, "\n")
}

func workerMetadataText(metadata map[string]any) string {
	if len(metadata) == 0 {
		return "-"
	}
	flat := render.WorkerFlattenMap(metadata, "")
	keys := make([]string, 0, len(flat))
	for key := range flat {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("[::b]%s[::-] %v", key, flat[key]))
	}
	return strings.Join(lines, "\n")
}

func workerTaskText(task map[string]any) string {
	if len(task) == 0 {
		return "-"
	}
	return workerMetadataText(task)
}
