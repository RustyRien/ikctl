package render

import (
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestStorageRendererIncludesProviderInDefaultProjection(t *testing.T) {
	renderer := NewStorageRenderer()
	headers := renderer.Headers()
	if len(headers) != 5 {
		t.Fatalf("headers len = %d", len(headers))
	}
	if headers[2].Key != "provider" || headers[2].Title != "PROVIDER" {
		t.Fatalf("provider header = %#v", headers[2])
	}

	row := renderer.Row(client.Storage{
		ID:              "st1",
		Name:            "terraform-state",
		StorageType:     "tofu",
		StorageProvider: "aws",
		State:           "ready",
		CreatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	if len(row.Fields) != 5 {
		t.Fatalf("fields len = %d", len(row.Fields))
	}
	if row.Fields[2] != "aws" {
		t.Fatalf("provider field = %q", row.Fields[2])
	}
	if row.SortKey["provider"] != "aws" {
		t.Fatalf("provider sort key = %q", row.SortKey["provider"])
	}
}
