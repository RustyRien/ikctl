package printers

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/registry"
)

func TestParseOutput(t *testing.T) {
	for _, s := range []string{"", "table", "wide", "json", "yaml", "name"} {
		if _, err := ParseOutput(s); err != nil {
			t.Errorf("ParseOutput(%q): %v", s, err)
		}
	}
	out, err := ParseOutput("custom-columns=NAME:name,TPL:.template.name,ID:{.id}")
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{out.Columns[0].Path, out.Columns[1].Path, out.Columns[2].Path}
	if strings.Join(paths, ",") != "name,template.name,id" {
		t.Errorf("paths = %v", paths)
	}
	for _, bad := range []string{"xml", "custom-columns=", "custom-columns=NAME"} {
		if _, err := ParseOutput(bad); err == nil {
			t.Errorf("ParseOutput(%q) should fail", bad)
		}
	}
}

func TestPrintTable(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	cols := []registry.Column{
		{Header: "NAME", Path: "name"},
		{Header: "TEMPLATE", Path: "template.name"},
		{Header: "AGE", Path: "createdAt", Type: registry.ColumnAge},
		{Header: "ID", Path: "id", Wide: true},
	}
	items := []sdk.Object{{
		"id": "1", "name": "a", "createdAt": "2026-01-07T00:00:00+00:00",
		"template": map[string]any{"name": "Redis"},
	}, {"id": "2", "name": "b", "template": nil, "createdAt": "2026-01-09T23:00:00.000000"}}

	var buf bytes.Buffer
	if err := PrintTable(&buf, cols, items, TableOptions{Now: now}); err != nil {
		t.Fatal(err)
	}
	want := "NAME   TEMPLATE   AGE\na      Redis      3d\nb      <none>     1h\n"
	if buf.String() != want {
		t.Errorf("table:\n%s\nwant:\n%s", buf.String(), want)
	}

	buf.Reset()
	_ = PrintTable(&buf, cols, items, TableOptions{Now: now, Wide: true, NoHeaders: true})
	if buf.String() != "a   Redis    3d   1\nb   <none>   1h   2\n" {
		t.Errorf("wide table:\n%q", buf.String())
	}
}

func TestFormatValue(t *testing.T) {
	now := time.Now()
	tests := []struct {
		in   any
		want string
	}{
		{nil, "<none>"},
		{"", "<none>"},
		{true, "true"},
		{json.Number("12"), "12"},
		{[]any{"a", "b"}, "a,b"},
		{[]any{map[string]any{"name": "x"}, map[string]any{"name": "y"}}, "x,y"},
		{[]any{}, "<none>"},
		{map[string]any{"k": 1}, `{"k":1}`},
		{"line1\nline2", "line1 line2"},
	}
	for _, tt := range tests {
		if got := FormatValue(tt.in, registry.ColumnText, now); got != tt.want {
			t.Errorf("FormatValue(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPrintYAMLAndNames(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintYAML(&buf, map[string]any{"b": 1, "a": "<x>"}); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "a: <x>\nb: 1\n" {
		t.Errorf("yaml = %q", buf.String())
	}
	buf.Reset()
	k, _ := registry.Lookup("resource")
	_ = PrintNames(&buf, k, []sdk.Object{{"id": "1", "name": "a"}, {"id": "2"}})
	if buf.String() != "resource/a\nresource/2\n" {
		t.Errorf("names = %q", buf.String())
	}
}

func TestHumanDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{
		5 * time.Second: "5s", 3 * time.Minute: "3m", 30 * time.Hour: "30h",
		72 * time.Hour: "3d", 800 * 24 * time.Hour: "2y", -time.Second: "0s",
	} {
		if got := HumanDuration(d); got != want {
			t.Errorf("HumanDuration(%v) = %s, want %s", d, got, want)
		}
	}
}
