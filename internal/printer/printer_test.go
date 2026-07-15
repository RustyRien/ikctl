package printer

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

type sampleNested struct {
	InnerValue string `json:"innerValue"`
}

type sampleYAML struct {
	IDValue    string           `json:"idValue"`
	CreatedAt  time.Time        `json:"createdAt"`
	RawName    string           `yaml:"raw_name"`
	HTTPServer string           `json:"HTTPServer"`
	Nested     sampleNested     `json:"nestedObject"`
	Meta       map[string]any   `json:"metaData"`
	Items      []map[string]any `json:"items"`
	Ignored    string           `yaml:"-"`
	Implicit   string
}

func TestPrintYAMLPreservesSnakeCaseStyle(t *testing.T) {
	raw := sampleYAML{
		IDValue:    "abc",
		CreatedAt:  time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC),
		RawName:    "raw",
		HTTPServer: "nginx",
		Nested:     sampleNested{InnerValue: "x"},
		Meta: map[string]any{
			"customKey": "y",
		},
		Items:    []map[string]any{{"itemName": "one"}},
		Ignored:  "skip",
		Implicit: "z",
	}

	var buf bytes.Buffer
	if err := Print(&buf, "yaml", nil, nil, []any{raw}); err != nil {
		t.Fatalf("Print yaml: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"id_value: abc",
		"created_at: 2026-07-15T10:00:00Z",
		"raw_name: raw",
		"http_server: nginx",
		"implicit: z",
		"nested_object:",
		"    inner_value: x",
		"meta_data:",
		"    customKey: \"y\"",
		"items:",
		"    - itemName: one",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("yaml output missing %q in:\n%s", want, out)
		}
	}

	for _, unwanted := range []string{"idValue:", "createdAt:", "HTTPServer:", "Ignored:"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("yaml output unexpectedly contains %q in:\n%s", unwanted, out)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := map[string]string{
		"createdAt":  "created_at",
		"HTTPServer": "http_server",
		"IDValue":    "id_value",
		"Name":       "name",
	}
	for input, want := range tests {
		if got := toSnakeCase(input); got != want {
			t.Fatalf("toSnakeCase(%q) = %q, want %q", input, got, want)
		}
	}
}
