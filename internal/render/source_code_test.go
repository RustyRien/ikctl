package render

import (
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestSourceCodeRendererIncludesProviderAndLanguageInDefaultProjection(t *testing.T) {
	renderer := NewSourceCodeRenderer()
	headers := renderer.Headers()
	if len(headers) != 6 {
		t.Fatalf("headers len = %d", len(headers))
	}
	if headers[1].Key != "provider" || headers[2].Key != "language" {
		t.Fatalf("unexpected headers = %#v", headers)
	}

	row := renderer.Row(client.SourceCode{
		ID:                 "sc1",
		Identifier:         "github.com/acme/infrastructure",
		SourceCodeURL:      "https://github.com/acme/infrastructure.git",
		SourceCodeProvider: "github",
		SourceCodeLanguage: "opentofu",
		Status:             "ready",
		CreatedAt:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	if len(row.Fields) != 6 {
		t.Fatalf("fields len = %d", len(row.Fields))
	}
	if row.Fields[0] != "infrastructure" {
		t.Fatalf("name field = %q", row.Fields[0])
	}
	if row.Fields[1] != "github" || row.Fields[2] != "opentofu" {
		t.Fatalf("projection = %#v", row.Fields)
	}
	if row.SortKey["provider"] != "github" || row.SortKey["language"] != "opentofu" {
		t.Fatalf("sort keys = %#v", row.SortKey)
	}
}
