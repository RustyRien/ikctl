package app

import (
	"fmt"
	"sort"
	"strings"
)

var availableCommands = []string{
	"audit",
	"columns",
	"delete",
	"edit",
	"destroyed",
	"disable",
	"entity",
	"enable",
	"integration-filter",
	"integrations",
	"logs",
	"open",
	"q",
	"refresh",
	"reset-filters",
	"resources",
	"secret-filter",
	"secrets",
	"source-codes",
	"source-code-versions",
	"settings",
	"storage-filter",
	"storages",
	"workers",
	"template-filter",
	"version-filter",
	"templates",
	"yaml",
}

func (a *App) runCommand(input string) {
	fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(input, ":")))
	if len(fields) == 0 {
		return
	}

	command := strings.ToLower(fields[0])
	switch command {
	case "q", "quit", "qa", "qall":
		a.ui.Stop()
	case "r", "res", "resource", "resources":
		a.handleNav('r')
	case "c", "sc", "source-code", "source-codes", "source_code", "source_codes":
		a.handleNav('c')
	case "v", "scv", "source-code-version", "source-code-versions", "source_code_version", "source_code_versions":
		a.handleNav('v')
	case "k", "sec", "secret", "secrets":
		a.handleNav('k')
	case "s", "stg", "storage", "storages":
		a.handleNav('s')
	case "w", "wrk", "worker", "workers":
		a.handleNav('w')
	case "t", "tpl", "template", "templates":
		a.handleNav('t')
	case "i", "int", "integration", "integrations":
		a.handleNav('i')
	case "refresh", "reload":
		a.requestRefresh()
	case "enable":
		a.runEntityActionCommand("enable", input)
	case "disable":
		a.runEntityActionCommand("disable", input)
	case "delete", "del", "rm", "remove":
		a.runEntityActionCommand("delete", input)
	case "edit":
		if row, ok := a.ui.SelectedRow(); ok {
			a.openEntityEditor(row)
			return
		}
		a.commandError(input, "No row selected for edit")
	case "settings", "set", "config":
		a.openSettings()
	case "columns", "cols", "col":
		a.openColumns()
	case "entity", "entities":
		a.openEntitySelector()
	case "logs", "log":
		if row, ok := a.ui.SelectedRow(); ok {
			a.openLogs(row)
			return
		}
		a.commandError(input, "No row selected for logs")
	case "audit":
		if row, ok := a.ui.SelectedRow(); ok {
			a.openAuditLogs(row)
			return
		}
		a.commandError(input, "No row selected for audit logs")
	case "open", "overview":
		if row, ok := a.ui.SelectedRow(); ok {
			a.openOverview(row)
			return
		}
		a.commandError(input, "No row selected for overview")
	case "yaml", "show":
		if row, ok := a.ui.SelectedRow(); ok {
			a.openYAML(row)
			return
		}
		a.commandError(input, "No row selected for yaml")
	case "template-filter", "tf":
		a.openTemplateFilter()
	case "version-filter", "vf", "source-code-version-filter", "scvf":
		a.openSourceCodeVersionFilter()
	case "storage-filter", "sf":
		a.openStorageFilter()
	case "secret-filter", "kf", "secf":
		a.openSecretFilter()
	case "integration-filter", "if":
		a.openIntegrationFilter()
	case "destroyed", "hide-destroyed":
		a.toggleHideDestroyedResources()
	case "reset-filters", "clear-filters", "filters-reset":
		a.resetAllResourceFilters()
	default:
		a.commandError(input, fmt.Sprintf("Unknown command: %s", input))
	}
}

func (a *App) suggestCommand(input string) (string, []string) {
	command := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(input, ":")))
	if command == "" {
		return "", append([]string(nil), availableCommands...)
	}
	if strings.Contains(command, " ") {
		return input, nil
	}

	matches := make([]string, 0, len(availableCommands))
	for _, candidate := range availableCommands {
		if strings.HasPrefix(candidate, command) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 0 {
		return input, nil
	}
	if len(matches) == 1 {
		return matches[0], matches
	}

	sort.Strings(matches)
	prefix := matches[0]
	for _, candidate := range matches[1:] {
		prefix = sharedPrefix(prefix, candidate)
		if prefix == command {
			break
		}
	}
	if len(prefix) < len(command) {
		prefix = command
	}

	return prefix, matches
}

func (a *App) commandError(command string, message string) {
	view := errorView(fmt.Sprintf("%s\n\nCommand: :%s\n\nAvailable commands:\n  :q\n  :resources\n  :source-codes\n  :source-code-versions\n  :secrets\n  :templates\n  :integrations\n  :storages\n  :workers\n  :refresh\n  :enable\n  :disable\n  :delete\n  :edit\n  :settings\n  :columns\n  :entity\n  :open\n  :yaml\n  :logs\n  :audit\n  :storage-filter\n  :secret-filter\n  :template-filter\n  :version-filter\n  :integration-filter\n  :destroyed\n  :reset-filters", message, strings.TrimSpace(strings.TrimPrefix(command, ":"))))
	a.ui.OpenOverlayPrimitive("Command", view)
}

func (a *App) runEntityActionCommand(verb string, input string) {
	if row, ok := a.ui.SelectedRow(); ok {
		a.openEntityActionPrompt(row, verb)
		return
	}
	a.commandError(input, fmt.Sprintf("No row selected for %s", verb))
}

func sharedPrefix(left string, right string) string {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] != right[i] {
			return left[:i]
		}
	}
	return left[:limit]
}
