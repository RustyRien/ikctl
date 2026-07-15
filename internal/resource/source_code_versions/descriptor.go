package source_code_versions

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
	renderer := render.NewSourceCodeVersionRenderer()
	headers := renderer.Headers()
	return &core.Descriptor{
		Name:        "source_code_versions",
		Singular:    "source_code_version",
		Aliases:     []string{"scv", "sourcecodeversion", "sourcecodeversions"},
		Headers:     headers,
		WideHeaders: render.SourceCodeVersionWideHeaders(),
		DefaultSort: []string{"created_at", "DESC"},
		SortFields:  core.SortFields(headers),
		FilterKeys: map[string]string{
			"name":       "identifier__like",
			"tag":        "source_code_version__like",
			"version":    "source_code_version__like",
			"branch":     "source_code_branch",
			"folder":     "source_code_folder__like",
			"template":   "template__name__in",
			"label":      "labels__contains_all",
			"labels":     "labels__contains_all",
			"status":     "status",
			"identifier": "identifier__like",
		},
		List: func(ctx context.Context, filter map[string]any, sortBy []string, pageRange []int) ([]tabledata.Row, []any, int, error) {
			result, err := c.SourceCodeVersions(ctx, filter, sortBy, pageRange)
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
			item, err := c.SourceCodeVersion(ctx, id)
			if err != nil {
				return tabledata.Row{}, nil, err
			}
			if item == nil {
				return tabledata.Row{}, nil, errors.New("source code version not found")
			}
			return renderer.Row(*item), *item, nil
		},
		ResolveByName: func(ctx context.Context, name string) (tabledata.Row, any, error) {
			return core.ResolveByName(ctx, name, renderer.Row, func(ctx context.Context, name string) ([]client.SourceCodeVersion, error) {
				result, err := c.SourceCodeVersions(ctx, map[string]any{"identifier__like": name}, []string{"identifier", "ASC"}, []int{0, 100})
				if err != nil {
					return nil, err
				}
				return result.Items, nil
			})
		},
		WideRow: func(value any) tabledata.Row {
			return render.SourceCodeVersionWideRow(value.(client.SourceCodeVersion))
		},
		EditLoad: func(ctx context.Context, id string) ([]byte, error) {
			item, err := c.SourceCodeVersion(ctx, id)
			if err != nil {
				return nil, err
			}
			if item == nil {
				return nil, errors.New("source code version not found")
			}
			return edit.SourceCodeVersionYAML(*item)
		},
		ApplyEdit: func(ctx context.Context, id string, data []byte) error {
			input, err := edit.SourceCodeVersionInputFromYAML(data)
			if err != nil {
				return err
			}
			return c.UpdateSourceCodeVersion(ctx, id, input)
		},
	}
}
