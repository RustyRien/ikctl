package model

import (
	"context"
	"sync"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/render"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type ResourcesModel struct {
	client      *client.Client
	renderer    *render.ResourceRenderer
	headers     []tabledata.Header
	rows        []tabledata.Row
	total       int
	lastUpdated time.Time
	lastError   error
	mx          sync.RWMutex
}

func NewResourcesModel(client *client.Client) *ResourcesModel {
	renderer := render.NewResourceRenderer()
	return &ResourcesModel{
		client:   client,
		renderer: renderer,
		headers:  renderer.Headers(),
	}
}

func (m *ResourcesModel) Refresh(ctx context.Context) error {
	result, err := m.client.Resources(ctx, nil, []string{"created_at", "DESC"}, []int{0, 100})

	m.mx.Lock()
	defer m.mx.Unlock()

	if err != nil {
		m.lastError = err
		return err
	}

	rows := make([]tabledata.Row, 0, len(result.Items))
	for _, resource := range result.Items {
		rows = append(rows, m.renderer.Row(resource))
	}

	m.rows = rows
	m.total = result.Total
	m.lastUpdated = time.Now()
	m.lastError = nil

	return nil
}

func (m *ResourcesModel) Snapshot() ([]tabledata.Header, []tabledata.Row, int, time.Time, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	headers := append([]tabledata.Header(nil), m.headers...)
	rows := append([]tabledata.Row(nil), m.rows...)
	return headers, rows, m.total, m.lastUpdated, m.lastError
}
