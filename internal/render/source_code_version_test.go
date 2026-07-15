package render

import (
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestSourceCodeVersionRendererIncludesTemplateAndRepositoryInDefaultProjection(t *testing.T) {
	renderer := NewSourceCodeVersionRenderer()
	headers := renderer.Headers()
	if len(headers) != 6 {
		t.Fatalf("headers len = %d", len(headers))
	}
	if headers[1].Key != "template" || headers[2].Key != "repository" {
		t.Fatalf("unexpected headers = %#v", headers)
	}

	row := renderer.Row(client.SourceCodeVersion{
		ID:                "scv1",
		Identifier:        "modules/redis:v1.2.3",
		SourceCodeVersion: "v1.2.3",
		SourceCodeFolder:  "modules/redis",
		Status:            "done",
		CreatedAt:         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Template:          &client.Template{Name: "aws_redis"},
		SourceCode:        &client.SourceCode{SourceCodeURL: "https://github.com/acme/infrastructure.git"},
		Creator:           &client.Creator{Identifier: "alice"},
	})
	if len(row.Fields) != 6 {
		t.Fatalf("fields len = %d", len(row.Fields))
	}
	if row.Fields[1] != "aws_redis" || row.Fields[2] != "infrastructure" {
		t.Fatalf("projection = %#v", row.Fields)
	}
	if row.SortKey["template"] != "aws_redis" || row.SortKey["repository"] != "infrastructure" {
		t.Fatalf("sort keys = %#v", row.SortKey)
	}
}
