package render

import (
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestTemplateRendererIncludesStatusInDefaultProjection(t *testing.T) {
	renderer := NewTemplateRenderer()
	headers := renderer.Headers()
	if len(headers) != 5 {
		t.Fatalf("headers len = %d", len(headers))
	}
	if headers[2].Key != "status" || headers[2].Title != "STATUS" {
		t.Fatalf("status header = %#v", headers[2])
	}

	row := renderer.Row(client.Template{
		ID:                 "t1",
		Name:               "aws_redis",
		CloudResourceTypes: []string{"redis"},
		Status:             "ready",
		CreatedAt:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	if len(row.Fields) != 5 {
		t.Fatalf("fields len = %d", len(row.Fields))
	}
	if row.Fields[2] != "ready" {
		t.Fatalf("status field = %q", row.Fields[2])
	}
	if row.SortKey["status"] != "ready" {
		t.Fatalf("status sort key = %q", row.SortKey["status"])
	}
}
