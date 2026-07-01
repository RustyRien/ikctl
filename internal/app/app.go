package app

import (
	"context"
	"fmt"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/model"
	uiapp "github.com/electrolux-oss/ik-tui/internal/ui"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type App struct {
	config     config.Config
	build      BuildInfo
	client     *client.Client
	model      *model.ResourcesModel
	ui         *uiapp.App
	manualKick chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
}

func New(cfg config.Config, build BuildInfo) *App {
	ctx, cancel := context.WithCancel(context.Background())
	client := client.New(cfg)
	model := model.NewResourcesModel(client)
	ui := uiapp.NewApp(cfg, build.Version)

	app := &App{
		config:     cfg,
		build:      build,
		client:     client,
		model:      model,
		ui:         ui,
		manualKick: make(chan struct{}, 1),
		ctx:        ctx,
		cancel:     cancel,
	}

	ui.SetRefreshFunc(app.requestRefresh)

	return app
}

func (a *App) Run() error {
	defer a.cancel()
	a.refreshInitial()
	go a.loop()
	return a.ui.Run()
}

func (a *App) loop() {
	ticker := time.NewTicker(a.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.refresh()
		case <-a.manualKick:
			a.refresh()
		}
	}
}

func (a *App) requestRefresh() {
	select {
	case a.manualKick <- struct{}{}:
	default:
	}
}

func (a *App) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_ = a.model.Refresh(ctx)
	headers, rows, total, lastUpdated, lastErr := a.model.Snapshot()
	a.ui.Application().QueueUpdateDraw(func() {
		a.ui.Update(headers, rows, total, lastUpdated, lastErr)
	})
}

func (a *App) refreshInitial() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_ = a.model.Refresh(ctx)
	headers, rows, total, lastUpdated, lastErr := a.model.Snapshot()
	a.ui.Update(headers, rows, total, lastUpdated, lastErr)
}

func (a *App) VersionString() string {
	return fmt.Sprintf("%s (%s %s)", a.build.Version, a.build.Commit, a.build.Date)
}
