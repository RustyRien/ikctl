// Package printers renders API objects as tables, JSON, YAML or names.
package printers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/registry"
)

// Output formats.
const (
	FormatTable         = "table"
	FormatWide          = "wide"
	FormatJSON          = "json"
	FormatYAML          = "yaml"
	FormatName          = "name"
	FormatCustomColumns = "custom-columns"
)

// Formats lists the formats accepted by -o for list-like commands.
var Formats = []string{FormatTable, FormatWide, FormatJSON, FormatYAML, FormatName, FormatCustomColumns + "=<HEADER>:<path>,..."}

// Output describes a parsed -o flag.
type Output struct {
	Format  string
	Columns []registry.Column // custom-columns only
}

// IsStructured reports whether the output is machine-readable JSON or YAML.
func (o Output) IsStructured() bool { return o.Format == FormatJSON || o.Format == FormatYAML }

// ParseOutput parses an -o value. An empty value means "table".
func ParseOutput(s string) (Output, error) {
	switch {
	case s == "" || s == FormatTable:
		return Output{Format: FormatTable}, nil
	case s == FormatWide, s == FormatJSON, s == FormatYAML, s == FormatName:
		return Output{Format: s}, nil
	case strings.HasPrefix(s, FormatCustomColumns+"="):
		spec := strings.TrimPrefix(s, FormatCustomColumns+"=")
		var cols []registry.Column
		for part := range strings.SplitSeq(spec, ",") {
			header, path, ok := strings.Cut(part, ":")
			if !ok || header == "" || path == "" {
				return Output{}, fmt.Errorf("invalid custom-columns entry %q: expected HEADER:path", part)
			}
			path = strings.TrimPrefix(strings.TrimPrefix(path, "."), "{.")
			path = strings.TrimSuffix(path, "}")
			cols = append(cols, registry.Column{Header: strings.ToUpper(header), Path: path})
		}
		return Output{Format: FormatCustomColumns, Columns: cols}, nil
	default:
		return Output{}, fmt.Errorf("unknown output format %q (allowed: %s)", s, strings.Join(Formats, ", "))
	}
}

// PrintJSON writes v as indented JSON with stable key order.
func PrintJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// PrintYAML writes v as YAML with stable key order.
func PrintYAML(w io.Writer, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	out, err := yaml.JSONToYAML(buf.Bytes())
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// PrintStructured writes v as JSON or YAML according to format.
func PrintStructured(w io.Writer, format string, v any) error {
	if format == FormatYAML {
		return PrintYAML(w, v)
	}
	return PrintJSON(w, v)
}

// TableOptions controls PrintTable.
type TableOptions struct {
	NoHeaders bool
	Wide      bool
	Now       time.Time
}

// PrintTable writes items as an aligned table.
func PrintTable(w io.Writer, cols []registry.Column, items []sdk.Object, opts TableOptions) error {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	visible := slices.DeleteFunc(slices.Clone(cols), func(c registry.Column) bool { return c.Wide && !opts.Wide })
	tw := tabwriter.NewWriter(w, 0, 8, 3, ' ', 0)
	if !opts.NoHeaders {
		headers := make([]string, len(visible))
		for i, c := range visible {
			headers[i] = c.Header
		}
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
	}
	for _, item := range items {
		cells := make([]string, len(visible))
		for i, c := range visible {
			cells[i] = FormatValue(sdk.Lookup(item, c.Path), c.Type, opts.Now)
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	return tw.Flush()
}

// PrintNames writes one "kind/name" line per item (name falls back to id).
func PrintNames(w io.Writer, kind registry.Kind, items []sdk.Object) error {
	for _, item := range items {
		ref := item.ID()
		if kind.NamePath != "" {
			if n := item.String(kind.NamePath); n != "" {
				ref = n
			}
		}
		if _, err := fmt.Fprintf(w, "%s/%s\n", kind.Name, ref); err != nil {
			return err
		}
	}
	return nil
}

// FormatValue renders a value for a table cell.
func FormatValue(v any, typ registry.ColumnType, now time.Time) string {
	if v == nil {
		return "<none>"
	}
	if typ == registry.ColumnAge {
		if s, ok := v.(string); ok {
			if t, ok := parseTime(s); ok {
				return HumanDuration(now.Sub(t))
			}
		}
	}
	switch x := v.(type) {
	case string:
		if x == "" {
			return "<none>"
		}
		return singleLine(x)
	case bool:
		return fmt.Sprint(x)
	case json.Number:
		return x.String()
	case []any:
		if len(x) == 0 {
			return "<none>"
		}
		parts := make([]string, 0, len(x))
		for _, e := range x {
			if m, ok := e.(map[string]any); ok {
				if n, ok := m["name"].(string); ok {
					parts = append(parts, n)
					continue
				}
			}
			parts = append(parts, FormatValue(e, registry.ColumnText, now))
		}
		return strings.Join(parts, ",")
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprint(x)
		}
		return string(b)
	}
}

// parseTime parses API timestamps; values without a zone are UTC.
func parseTime(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func singleLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	const maxLen = 80
	if r := []rune(s); len(r) > maxLen {
		return string(r[:maxLen-3]) + "..."
	}
	return s
}

// HumanDuration renders a duration compactly, e.g. 45s, 12m, 5h, 3d, 2y.
func HumanDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/24/365))
	}
}
