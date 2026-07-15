package source_codes

import (
	"context"
	"errors"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/edit"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/resource/core"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

func Descriptor(c *client.Client) *core.Descriptor {
	renderer := render.NewSourceCodeRenderer()
	headers := renderer.Headers()
	return &core.Descriptor{
		Name:        "source_codes",
		Singular:    "source_code",
		Aliases:     []string{"sc", "sourcecode", "repo", "repos"},
		Headers:     headers,
		WideHeaders: render.SourceCodeWideHeaders(),
		DefaultSort: []string{"updated_at", "DESC"},
		SortFields:  core.SortFields(headers),
		FilterKeys: map[string]string{
			"name":       "source_code_url__like",
			"label":      "labels__contains_all",
			"labels":     "labels__contains_all",
			"provider":   "source_code_provider",
			"language":   "source_code_language",
			"status":     "status",
			"identifier": "identifier__like",
		},
		List: func(ctx context.Context, filter map[string]any, sortBy []string, pageRange []int) ([]tabledata.Row, []any, int, error) {
			result, err := c.SourceCodes(ctx, filter, sortBy, pageRange)
			if err != nil {
				return nil, nil, 0, err
			}
			rows := make([]tabledata.Row, 0, len(result.Items))
			raw := make([]any, 0, len(result.Items))
			for _, item := range result.Items {
				rows = append(rows, renderer.Row(item))
				raw = append(raw, item)
			}
			return rows, raw, result.Total, nil
		},
		GetByID: func(ctx context.Context, id string) (tabledata.Row, any, error) {
			item, err := c.SourceCode(ctx, id)
			if err != nil {
				return tabledata.Row{}, nil, err
			}
			if item == nil {
				return tabledata.Row{}, nil, errors.New("source code not found")
			}
			return renderer.Row(*item), *item, nil
		},
		ResolveByName: func(ctx context.Context, name string) (tabledata.Row, any, error) {
			return core.ResolveByName(ctx, name, renderer.Row, func(ctx context.Context, name string) ([]client.SourceCode, error) {
				result, err := c.SourceCodes(ctx, map[string]any{"source_code_url__like": name}, []string{"identifier", "ASC"}, []int{0, 100})
				if err != nil {
					return nil, err
				}
				return result.Items, nil
			})
		},
		WideRow: func(value any) tabledata.Row {
			return render.SourceCodeWideRow(value.(client.SourceCode))
		},
		EditLoad: func(ctx context.Context, id string) ([]byte, error) {
			item, err := c.SourceCode(ctx, id)
			if err != nil {
				return nil, err
			}
			if item == nil {
				return nil, errors.New("source code not found")
			}
			return edit.SourceCodeYAML(*item)
		},
		ApplyEdit: func(ctx context.Context, id string, data []byte) error {
			input, err := edit.SourceCodeInputFromYAML(data)
			if err != nil {
				return err
			}
			return c.UpdateSourceCode(ctx, id, input)
		},
	}
}
