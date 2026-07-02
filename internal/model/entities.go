package model

import (
	"context"
	"sync"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type EntityKind string

const (
	EntityResources    EntityKind = "resources"
	EntityTemplates    EntityKind = "templates"
	EntityIntegrations EntityKind = "integrations"
)

type EntityModel struct {
	kind        EntityKind
	headers     []tabledata.Header
	rows        []tabledata.Row
	total       int
	lastUpdated time.Time
	lastError   error
	sortField   string
	sortDesc    bool
	pageSize    int
	loadingMore bool
	refreshFn   func(context.Context, []string, []int) ([]tabledata.Row, int, error)
	mx          sync.RWMutex
}

func NewResourcesModel(client *client.Client) *EntityModel {
	renderer := render.NewResourceRenderer()
	return &EntityModel{
		kind:      EntityResources,
		headers:   renderer.Headers(),
		sortField: "created_at",
		sortDesc:  true,
		pageSize:  100,
		refreshFn: func(ctx context.Context, sort []string, pageRange []int) ([]tabledata.Row, int, error) {
			result, err := client.Resources(ctx, nil, sort, pageRange)
			if err != nil {
				return nil, 0, err
			}
			rows := make([]tabledata.Row, 0, len(result.Items))
			for _, item := range result.Items {
				rows = append(rows, renderer.Row(item))
			}
			return rows, result.Total, nil
		},
	}
}

func NewTemplatesModel(client *client.Client) *EntityModel {
	renderer := render.NewTemplateRenderer()
	return &EntityModel{
		kind:      EntityTemplates,
		headers:   renderer.Headers(),
		sortField: "updated_at",
		sortDesc:  true,
		pageSize:  100,
		refreshFn: func(ctx context.Context, sort []string, pageRange []int) ([]tabledata.Row, int, error) {
			result, err := client.Templates(ctx, nil, sort, pageRange)
			if err != nil {
				return nil, 0, err
			}
			rows := make([]tabledata.Row, 0, len(result.Items))
			for _, item := range result.Items {
				rows = append(rows, renderer.Row(item))
			}
			return rows, result.Total, nil
		},
	}
}

func NewIntegrationsModel(client *client.Client) *EntityModel {
	renderer := render.NewIntegrationRenderer()
	return &EntityModel{
		kind:      EntityIntegrations,
		headers:   renderer.Headers(),
		sortField: "updated_at",
		sortDesc:  true,
		pageSize:  100,
		refreshFn: func(ctx context.Context, sort []string, pageRange []int) ([]tabledata.Row, int, error) {
			result, err := client.Integrations(ctx, nil, sort, pageRange)
			if err != nil {
				return nil, 0, err
			}
			rows := make([]tabledata.Row, 0, len(result.Items))
			for _, item := range result.Items {
				rows = append(rows, renderer.Row(item))
			}
			return rows, result.Total, nil
		},
	}
}

func NewModelFromDescriptor(kind EntityKind, descriptor *resource.Descriptor) *EntityModel {
	sortField := ""
	sortDesc := false
	if len(descriptor.DefaultSort) == 2 {
		sortField = descriptor.DefaultSort[0]
		sortDesc = descriptor.DefaultSort[1] == "DESC"
	}
	return &EntityModel{
		kind:      kind,
		headers:   descriptor.Headers,
		sortField: sortField,
		sortDesc:  sortDesc,
		pageSize:  100,
		refreshFn: func(ctx context.Context, sort []string, pageRange []int) ([]tabledata.Row, int, error) {
			rows, _, total, err := descriptor.List(ctx, nil, sort, pageRange)
			return rows, total, err
		},
	}
}

func (m *EntityModel) Kind() EntityKind {
	return m.kind
}

func (m *EntityModel) Refresh(ctx context.Context) error {
	m.mx.RLock()
	sort := currentSort(m.sortField, m.sortDesc)
	loaded := len(m.rows)
	pageSize := m.pageSize
	m.mx.RUnlock()
	if loaded < pageSize {
		loaded = pageSize
	}

	rows, total, err := m.refreshFn(ctx, sort, pageRange(0, loaded))

	m.mx.Lock()
	defer m.mx.Unlock()

	if err != nil {
		m.lastError = err
		return err
	}

	m.rows = rows
	m.total = total
	m.lastUpdated = time.Now()
	m.lastError = nil
	m.loadingMore = false

	return nil
}

func (m *EntityModel) LoadMore(ctx context.Context) error {
	m.mx.Lock()
	if m.loadingMore || (m.total > 0 && len(m.rows) >= m.total) {
		m.mx.Unlock()
		return nil
	}
	m.loadingMore = true
	sort := currentSort(m.sortField, m.sortDesc)
	offset := len(m.rows)
	limit := m.pageSize
	m.mx.Unlock()

	rows, total, err := m.refreshFn(ctx, sort, pageRange(offset, offset+limit))

	m.mx.Lock()
	defer m.mx.Unlock()
	m.loadingMore = false
	if err != nil {
		m.lastError = err
		return err
	}

	m.rows = append(m.rows, rows...)
	m.total = total
	m.lastUpdated = time.Now()
	m.lastError = nil
	return nil
}

func (m *EntityModel) SortState() (int, bool) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	for index, header := range m.headers {
		if header.SortField == m.sortField && header.SortField != "" {
			return index, !m.sortDesc
		}
	}
	return -1, true
}

func (m *EntityModel) SetSortByColumn(column int, asc bool) bool {
	m.mx.Lock()
	defer m.mx.Unlock()

	if column < 0 || column >= len(m.headers) {
		return false
	}
	header := m.headers[column]
	if header.SortField == "" {
		return false
	}
	m.sortField = header.SortField
	m.sortDesc = !asc
	return true
}

func (m *EntityModel) HasMore() bool {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.total == 0 || len(m.rows) < m.total
}

func (m *EntityModel) LoadingMore() bool {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.loadingMore
}

func currentSort(field string, desc bool) []string {
	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	return []string{field, direction}
}

func pageRange(start int, end int) []int {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	return []int{start, end}
}

func (m *EntityModel) Snapshot() ([]tabledata.Header, []tabledata.Row, int, time.Time, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	headers := append([]tabledata.Header(nil), m.headers...)
	rows := append([]tabledata.Row(nil), m.rows...)
	return headers, rows, m.total, m.lastUpdated, m.lastError
}
