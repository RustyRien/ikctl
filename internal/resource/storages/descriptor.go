package storages

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
	renderer := render.NewStorageRenderer()
	headers := renderer.Headers()
	return &core.Descriptor{
		Name:        "storages",
		Singular:    "storage",
		Aliases:     []string{"store", "stg"},
		Headers:     headers,
		WideHeaders: render.StorageWideHeaders(),
		DefaultSort: []string{"updated_at", "DESC"},
		SortFields:  core.SortFields(headers),
		FilterKeys: map[string]string{
			"name":     "name",
			"label":    "labels__contains_all",
			"labels":   "labels__contains_all",
			"provider": "storage_provider",
			"type":     "storage_type",
			"state":    "state",
			"status":   "status",
		},
		List: func(ctx context.Context, filter map[string]any, sortBy []string, pageRange []int) ([]tabledata.Row, []any, int, error) {
			result, err := c.Storages(ctx, filter, sortBy, pageRange)
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
			item, err := c.Storage(ctx, id)
			if err != nil {
				return tabledata.Row{}, nil, err
			}
			if item == nil {
				return tabledata.Row{}, nil, errors.New("storage not found")
			}
			return renderer.Row(*item), *item, nil
		},
		ResolveByName: func(ctx context.Context, name string) (tabledata.Row, any, error) {
			return core.ResolveByName(ctx, name, renderer.Row, func(ctx context.Context, name string) ([]client.Storage, error) {
				result, err := c.Storages(ctx, map[string]any{"name__like": name}, []string{"name", "ASC"}, []int{0, 100})
				if err != nil {
					return nil, err
				}
				return result.Items, nil
			})
		},
		WideRow: func(value any) tabledata.Row {
			return render.StorageWideRow(value.(client.Storage))
		},
		EditLoad: func(ctx context.Context, id string) ([]byte, error) {
			item, err := c.Storage(ctx, id)
			if err != nil {
				return nil, err
			}
			if item == nil {
				return nil, errors.New("storage not found")
			}
			return edit.StorageYAML(*item)
		},
		ApplyEdit: func(ctx context.Context, id string, data []byte) error {
			input, err := edit.StorageInputFromYAML(data)
			if err != nil {
				return err
			}
			return c.UpdateStorage(ctx, id, input)
		},
	}
}
