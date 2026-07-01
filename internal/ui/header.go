package ui

import (
	"fmt"

	"github.com/derailed/tview"
	"github.com/electrolux-oss/ik-tui/internal/config"
)

func NewHeader(cfg config.Config, version string) *tview.TextView {
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetTextColor(colorTitle)
	view.SetText(fmt.Sprintf("ik-tui  %s\nEndpoint: %s", version, cfg.Endpoint))
	return view
}
