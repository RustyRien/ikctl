package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"gopkg.in/yaml.v3"
)

func Print(w io.Writer, format string, headers []tabledata.Header, rows []tabledata.Row, raw []any) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "table", "wide":
		return printTable(w, headers, rows)
	case "name":
		for _, row := range rows {
			if _, err := fmt.Fprintln(w, row.ID); err != nil {
				return err
			}
		}
		return nil
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(normalizeRaw(raw))
	case "yaml":
		data, err := yaml.Marshal(normalizeRaw(raw))
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func printTable(w io.Writer, headers []tabledata.Header, rows []tabledata.Row) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for index, header := range headers {
		if index > 0 {
			if _, err := fmt.Fprint(tw, "\t"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(tw, header.Title); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(tw); err != nil {
		return err
	}
	for _, row := range rows {
		for index, field := range row.Fields {
			if index > 0 {
				if _, err := fmt.Fprint(tw, "\t"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(tw, field); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(tw); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func normalizeRaw(raw []any) any {
	if len(raw) == 1 {
		return raw[0]
	}
	return raw
}
