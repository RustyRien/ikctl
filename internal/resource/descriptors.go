package resource

import (
	"context"
	"errors"
	"strings"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

func DefaultRegistry(c *client.Client) *Registry {
	return NewRegistry(
		resourcesDescriptor(c),
		templatesDescriptor(c),
		integrationsDescriptor(c),
	)
}

func resourcesDescriptor(c *client.Client) *Descriptor {
	renderer := render.NewResourceRenderer()
	headers := renderer.Headers()
	return &Descriptor{
		Name:        "resources",
		Singular:    "resource",
		Aliases:     []string{"res"},
		Headers:     headers,
		WideHeaders: render.ResourceWideHeaders(),
		DefaultSort: []string{"created_at", "DESC"},
		SortFields:  sortFields(headers),
		FilterKeys: map[string]string{
			"state":  "state",
			"status": "status",
			"label":  "labels",
			"name":   "name",
		},
		List: func(ctx context.Context, filter map[string]any, sortBy []string, pageRange []int) ([]tabledata.Row, []any, int, error) {
			result, err := c.Resources(ctx, filter, sortBy, pageRange)
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
			item, err := c.Resource(ctx, id)
			if err != nil {
				return tabledata.Row{}, nil, err
			}
			if item == nil {
				return tabledata.Row{}, nil, errors.New("resource not found")
			}
			return renderer.Row(*item), *item, nil
		},
		ResolveByName: func(ctx context.Context, name string) (tabledata.Row, any, error) {
			return resolveByName(ctx, name, renderer.Row, func(ctx context.Context) ([]client.Resource, error) {
				result, err := c.Resources(ctx, nil, []string{"name", "ASC"}, []int{0, 1000})
				if err != nil {
					return nil, err
				}
				return result.Items, nil
			})
		},
		WideRow: func(value any) tabledata.Row {
			return render.ResourceWideRow(value.(client.Resource))
		},
	}
}

func templatesDescriptor(c *client.Client) *Descriptor {
	renderer := render.NewTemplateRenderer()
	headers := renderer.Headers()
	return &Descriptor{
		Name:        "templates",
		Singular:    "template",
		Aliases:     []string{"tmpl", "tpl"},
		Headers:     headers,
		WideHeaders: render.TemplateWideHeaders(),
		DefaultSort: []string{"updated_at", "DESC"},
		SortFields:  sortFields(headers),
		FilterKeys: map[string]string{
			"name": "name",
		},
		List: func(ctx context.Context, filter map[string]any, sortBy []string, pageRange []int) ([]tabledata.Row, []any, int, error) {
			result, err := c.Templates(ctx, filter, sortBy, pageRange)
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
			item, err := c.Template(ctx, id)
			if err != nil {
				return tabledata.Row{}, nil, err
			}
			if item == nil {
				return tabledata.Row{}, nil, errors.New("template not found")
			}
			return renderer.Row(*item), *item, nil
		},
		ResolveByName: func(ctx context.Context, name string) (tabledata.Row, any, error) {
			return resolveByName(ctx, name, renderer.Row, func(ctx context.Context) ([]client.Template, error) {
				result, err := c.Templates(ctx, nil, []string{"name", "ASC"}, []int{0, 1000})
				if err != nil {
					return nil, err
				}
				return result.Items, nil
			})
		},
		WideRow: func(value any) tabledata.Row {
			return render.TemplateWideRow(value.(client.Template))
		},
	}
}

func integrationsDescriptor(c *client.Client) *Descriptor {
	renderer := render.NewIntegrationRenderer()
	headers := renderer.Headers()
	return &Descriptor{
		Name:        "integrations",
		Singular:    "integration",
		Aliases:     []string{"int", "ints"},
		Headers:     headers,
		WideHeaders: render.IntegrationWideHeaders(),
		DefaultSort: []string{"updated_at", "DESC"},
		SortFields:  sortFields(headers),
		FilterKeys: map[string]string{
			"name":     "name",
			"provider": "integration_provider",
			"type":     "integration_type",
		},
		List: func(ctx context.Context, filter map[string]any, sortBy []string, pageRange []int) ([]tabledata.Row, []any, int, error) {
			result, err := c.Integrations(ctx, filter, sortBy, pageRange)
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
			item, err := c.Integration(ctx, id)
			if err != nil {
				return tabledata.Row{}, nil, err
			}
			if item == nil {
				return tabledata.Row{}, nil, errors.New("integration not found")
			}
			return renderer.Row(*item), *item, nil
		},
		ResolveByName: func(ctx context.Context, name string) (tabledata.Row, any, error) {
			return resolveByName(ctx, name, renderer.Row, func(ctx context.Context) ([]client.Integration, error) {
				result, err := c.Integrations(ctx, nil, []string{"name", "ASC"}, []int{0, 1000})
				if err != nil {
					return nil, err
				}
				return result.Items, nil
			})
		},
		WideRow: func(value any) tabledata.Row {
			return render.IntegrationWideRow(value.(client.Integration))
		},
	}
}

func sortFields(headers []tabledata.Header) map[string]string {
	fields := make(map[string]string, len(headers)*2)
	for _, header := range headers {
		if header.SortField == "" {
			continue
		}
		fields[strings.ToLower(header.Key)] = header.SortField
		fields[strings.ToLower(header.Title)] = header.SortField
	}
	return fields
}

type namedEntity interface {
	GetName() string
}

func resolveByName[T namedEntity](ctx context.Context, name string, rowFn func(T) tabledata.Row, listFn func(context.Context) ([]T, error)) (tabledata.Row, any, error) {
	items, err := listFn(ctx)
	if err != nil {
		return tabledata.Row{}, nil, err
	}
	needle := strings.TrimSpace(name)
	for _, item := range items {
		if item.GetName() == needle {
			return rowFn(item), item, nil
		}
	}
	return tabledata.Row{}, nil, errors.New("item not found")
}
