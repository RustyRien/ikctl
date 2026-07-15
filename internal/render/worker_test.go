package render

import (
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestWorkerListRowIncludesFrontendInspiredFields(t *testing.T) {
	completed := 7
	worker := client.Worker{
		ID:             "w1",
		Name:           "worker-a",
		Host:           "host-a",
		HostMetadata:   map[string]any{"platform": "linux", "machine": "arm64"},
		Status:         "ready",
		CurrentTask:    map[string]any{"entity": "resource", "action": "apply"},
		TasksCompleted: &completed,
		CreatedAt:      time.Now().Add(-time.Hour),
		UpdatedAt:      time.Now(),
	}

	row := WorkerListRow(worker)
	if row.Fields[0] != "worker-a" {
		t.Fatalf("name field = %q", row.Fields[0])
	}
	if row.Fields[3] != "resource / apply" {
		t.Fatalf("task field = %q", row.Fields[3])
	}
	if row.Fields[4] != "7" {
		t.Fatalf("completed field = %q", row.Fields[4])
	}
	if row.Fields[7] != "linux (arm64)" {
		t.Fatalf("host info field = %q", row.Fields[7])
	}
}

func TestWorkerFlattenMapFlattensNestedMaps(t *testing.T) {
	flat := WorkerFlattenMap(map[string]any{"cpu": map[string]any{"count": 8}, "platform": "linux"}, "")
	if flat["cpu_count"] != 8 {
		t.Fatalf("flat map cpu_count = %#v", flat["cpu_count"])
	}
	if flat["platform"] != "linux" {
		t.Fatalf("flat map platform = %#v", flat["platform"])
	}
}
