package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode"

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
		data, err := yaml.Marshal(normalizeYAMLValue(normalizeRaw(raw)))
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

func normalizeYAMLValue(value any) *yaml.Node {
	return normalizeYAMLReflect(reflect.ValueOf(value))
}

func normalizeYAMLReflect(value reflect.Value) *yaml.Node {
	if !value.IsValid() {
		return scalarYAMLNode(nil)
	}

	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return scalarYAMLNode(nil)
		}
		value = value.Elem()
	}

	typeOf := value.Type()
	if typeOf.PkgPath() == "time" && typeOf.Name() == "Time" {
		return scalarYAMLNode(value.Interface())
	}

	switch value.Kind() {
	case reflect.Struct:
		keys := make([]string, 0, value.NumField())
		values := make(map[string]reflect.Value, value.NumField())
		for i := 0; i < value.NumField(); i++ {
			field := typeOf.Field(i)
			if !field.IsExported() {
				continue
			}
			key, ok := yamlFieldName(field)
			if !ok {
				continue
			}
			keys = append(keys, key)
			values[key] = value.Field(i)
		}
		sort.Strings(keys)
		node := &yaml.Node{Kind: yaml.MappingNode}
		for _, key := range keys {
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
				normalizeYAMLReflect(values[key]),
			)
		}
		return node
	case reflect.Slice, reflect.Array:
		node := &yaml.Node{Kind: yaml.SequenceNode}
		for i := 0; i < value.Len(); i++ {
			node.Content = append(node.Content, normalizeYAMLReflect(value.Index(i)))
		}
		return node
	case reflect.Map:
		if value.IsNil() {
			return &yaml.Node{Kind: yaml.MappingNode}
		}
		keys := make([]string, 0, value.Len())
		values := make(map[string]reflect.Value, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			key := fmt.Sprint(iter.Key().Interface())
			keys = append(keys, key)
			values[key] = iter.Value()
		}
		sort.Strings(keys)
		node := &yaml.Node{Kind: yaml.MappingNode}
		for _, key := range keys {
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
				normalizeYAMLReflect(values[key]),
			)
		}
		return node
	default:
		return scalarYAMLNode(value.Interface())
	}
}

func yamlFieldName(field reflect.StructField) (string, bool) {
	if tag := field.Tag.Get("yaml"); tag != "" {
		name := strings.Split(tag, ",")[0]
		if name == "-" {
			return "", false
		}
		if name != "" {
			return name, true
		}
	}
	if tag := field.Tag.Get("json"); tag != "" {
		name := strings.Split(tag, ",")[0]
		if name == "-" {
			return "", false
		}
		if name != "" {
			return toSnakeCase(name), true
		}
	}
	return toSnakeCase(field.Name), true
}

func toSnakeCase(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	var out strings.Builder
	for i, r := range runes {
		if r == '-' || r == ' ' {
			out.WriteRune('_')
			continue
		}
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				out.WriteRune('_')
			}
			out.WriteRune(unicode.ToLower(r))
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func scalarYAMLNode(value any) *yaml.Node {
	node := &yaml.Node{}
	if err := node.Encode(value); err != nil {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprint(value)}
	}
	return node
}
