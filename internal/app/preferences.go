package app

import (
	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
)

func (a *App) applySavedViewPreferences() {
	if visible := visibleColumnsFromFields(a.config.View.Resources.Columns, resourceColumnOptions); len(visible) > 0 {
		a.visibleResourceColumns = visible
	}
	if visible := visibleColumnsFromFields(a.config.View.Templates.Columns, templateColumnOptions); len(visible) > 0 {
		a.visibleTemplateColumns = visible
	}
	if ref := a.config.View.Resources.TemplateFilter; ref.ID != "" {
		a.resourceTemplateFilter = &client.Template{ID: ref.ID, Name: ref.Name}
	}
	if ref := a.config.View.Resources.IntegrationFilter; ref.ID != "" {
		a.resourceIntegrationFilter = &client.Integration{ID: ref.ID, Name: ref.Name}
	}
	a.hideDestroyedResources = a.config.View.Resources.HideDestroyed
}

func (a *App) saveViewPreferences() {
	a.config.View.Resources.Columns = selectedFields(a.visibleResourceColumns, resourceColumnOptions)
	a.config.View.Templates.Columns = selectedFields(a.visibleTemplateColumns, templateColumnOptions)
	a.config.View.Resources.TemplateFilter = templateFilterRef(a.resourceTemplateFilter)
	a.config.View.Resources.IntegrationFilter = integrationFilterRef(a.resourceIntegrationFilter)
	a.config.View.Resources.HideDestroyed = a.hideDestroyedResources
	_ = a.config.Save()
}

func visibleColumnsFromFields(fields []string, options []resourceColumnOption) map[string]bool {
	if len(fields) == 0 {
		return nil
	}
	known := make(map[string]struct{}, len(options))
	for _, option := range options {
		known[option.Field] = struct{}{}
	}
	visible := make(map[string]bool, len(fields))
	for _, field := range fields {
		if _, ok := known[field]; ok {
			visible[field] = true
		}
	}
	if len(visible) == 0 {
		return nil
	}
	return visible
}

func selectedFields(visible map[string]bool, options []resourceColumnOption) []string {
	fields := make([]string, 0, len(options))
	for _, option := range options {
		if visible[option.Field] {
			fields = append(fields, option.Field)
		}
	}
	return fields
}

func templateFilterRef(value *client.Template) config.FilterRef {
	if value == nil {
		return config.FilterRef{}
	}
	return config.FilterRef{ID: value.ID, Name: value.Name}
}

func integrationFilterRef(value *client.Integration) config.FilterRef {
	if value == nil {
		return config.FilterRef{}
	}
	return config.FilterRef{ID: value.ID, Name: value.Name}
}
