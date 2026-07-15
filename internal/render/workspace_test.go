package render

import (
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestWorkspaceRendererIncludesProviderAndStatusInDefaultProjection(t *testing.T) {
	renderer := NewWorkspaceRenderer()
	headers := renderer.Headers()
	if len(headers) != 5 {
		t.Fatalf("headers len = %d", len(headers))
	}
	if headers[1].Key != "provider" || headers[1].Title != "PROVIDER" {
		t.Fatalf("provider header = %#v", headers[1])
	}
	if headers[2].Key != "status" || headers[2].Title != "STATUS" {
		t.Fatalf("status header = %#v", headers[2])
	}

	row := renderer.Row(client.Workspace{
		ID:                "w1",
		Name:              "platform",
		WorkspaceProvider: "github",
		Status:            "ready",
		CreatedAt:         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	if len(row.Fields) != 5 {
		t.Fatalf("fields len = %d", len(row.Fields))
	}
	if row.Fields[1] != "github" {
		t.Fatalf("provider field = %q", row.Fields[1])
	}
	if row.Fields[2] != "ready" {
		t.Fatalf("status field = %q", row.Fields[2])
	}
	if row.SortKey["provider"] != "github" {
		t.Fatalf("provider sort key = %q", row.SortKey["provider"])
	}
	if row.SortKey["status"] != "ready" {
		t.Fatalf("status sort key = %q", row.SortKey["status"])
	}
}
