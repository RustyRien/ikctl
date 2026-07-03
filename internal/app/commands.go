package app

import (
	"fmt"
	"sort"
	"strings"
)

var availableCommands = []string{
	"audit",
	"columns",
	"destroyed",
	"entity",
	"integration-filter",
	"integrations",
	"logs",
	"open",
	"q",
	"refresh",
	"resources",
	"settings",
	"template-filter",
	"templates",
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
	case "t", "tpl", "template", "templates":
		a.handleNav('t')
	case "i", "int", "integration", "integrations":
		a.handleNav('i')
	case "refresh", "reload":
		a.requestRefresh()
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
	case "template-filter", "tf":
		a.openTemplateFilter()
	case "integration-filter", "if":
		a.openIntegrationFilter()
	case "destroyed", "hide-destroyed":
		a.toggleHideDestroyedResources()
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
	view := errorView(fmt.Sprintf("%s\n\nCommand: :%s\n\nAvailable commands:\n  :q\n  :resources\n  :templates\n  :integrations\n  :refresh\n  :settings\n  :columns\n  :entity\n  :open\n  :logs\n  :audit\n  :template-filter\n  :integration-filter\n  :destroyed", message, strings.TrimSpace(strings.TrimPrefix(command, ":"))))
	a.ui.OpenOverlayPrimitive("Command", view)
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
