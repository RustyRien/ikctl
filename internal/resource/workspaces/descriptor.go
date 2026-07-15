package workspaces

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
	renderer := render.NewWorkspaceRenderer()
	headers := renderer.Headers()
	return &core.Descriptor{
		Name:        "workspaces",
		Singular:    "workspace",
		Aliases:     []string{"ws", "wsp"},
		Headers:     headers,
		WideHeaders: render.WorkspaceWideHeaders(),
		DefaultSort: []string{"updatedAt", "DESC"},
		SortFields:  core.SortFields(headers),
		FilterKeys: map[string]string{
			"name":        "name__like",
			"label":       "labels__contains_all",
			"labels":      "labels__contains_all",
			"provider":    "workspaceProvider",
			"status":      "status",
			"integration": "integration.name",
		},
		List: func(ctx context.Context, filter map[string]any, sortBy []string, pageRange []int) ([]tabledata.Row, []any, int, error) {
			result, err := c.Workspaces(ctx, filter, sortBy, pageRange)
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
			item, err := c.Workspace(ctx, id)
			if err != nil {
				return tabledata.Row{}, nil, err
			}
			if item == nil {
				return tabledata.Row{}, nil, errors.New("workspace not found")
			}
			return renderer.Row(*item), *item, nil
		},
		ResolveByName: func(ctx context.Context, name string) (tabledata.Row, any, error) {
			return core.ResolveByName(ctx, name, renderer.Row, func(ctx context.Context, name string) ([]client.Workspace, error) {
				result, err := c.Workspaces(ctx, map[string]any{"name__like": name}, []string{"name", "ASC"}, []int{0, 100})
				if err != nil {
					return nil, err
				}
				return result.Items, nil
			})
		},
		WideRow: func(value any) tabledata.Row {
			return render.WorkspaceWideRow(value.(client.Workspace))
		},
		EditLoad: func(ctx context.Context, id string) ([]byte, error) {
			item, err := c.Workspace(ctx, id)
			if err != nil {
				return nil, err
			}
			if item == nil {
				return nil, errors.New("workspace not found")
			}
			return edit.WorkspaceYAML(*item)
		},
		ApplyEdit: func(ctx context.Context, id string, data []byte) error {
			input, err := edit.WorkspaceInputFromYAML(data)
			if err != nil {
				return err
			}
			return c.UpdateWorkspace(ctx, id, input)
		},
	}
}
