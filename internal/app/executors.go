package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/rivo/tview"
)

func (a *App) openExecutorOverview(id string, name string) {
	session := a.nextLiveLogSession()
	a.rememberCurrentDetailState()
	a.ui.SetDetailActionRow(tabledata.Row{ID: id, Raw: client.Executor{ID: id, Name: name, EntityName: "executor"}})

	title := fmt.Sprintf("Executor: %s", valueOr(name, id))
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.activeExecutorDetail = &entityDetailSelection{ID: id, Name: name, Kind: "executors"}
	a.activeSourceCodeDetail = nil
	a.activeSourceCodeVersionDetail = nil
	a.activeSecretDetail = nil
	a.activeIntegrationDetail = nil
	a.activeStorageDetail = nil
	a.activeWorkerDetail = nil
	a.activeWorkspaceDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.pendingEntityAction = nil
	a.resourceReview = nil
	a.ui.OpenDetail(title, "Loading executor overview...")
	a.ui.SetExecutorOverviewHotkeys()

	go func(session int) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Executor(ctx, id)
		var primitive tview.Primitive
		var lastLogView *tview.TextView
		var jumpActions map[rune]func()
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load executor overview.\n\n%v", err))
		} else if full != nil {
			a.activeExecutorDetail = &entityDetailSelection{ID: full.ID, Name: full.Name, Kind: "executors"}
			primitive, lastLogView = executorOverviewView(*full)
			jumpActions = a.executorOverviewJumpActions(*full)
		} else {
			a.activeExecutorDetail = nil
			primitive = errorView("Executor not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			if !a.isLiveLogSessionCurrent(session) {
				return
			}
			a.clearOverviewJumpState()
			for key, action := range jumpActions {
				a.setOverviewJumpAction(key, action)
			}
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetExecutorOverviewHotkeys()
			if lastLogView != nil && full != nil {
				a.loadEntityLogsIntoView(session, full.ID, lastLogView, 20, formatRecentLogs, "Failed to load recent logs.")
			}
		})
	}(session)
}

func (a *App) executorOverviewJumpActions(executor client.Executor) map[rune]func() {
	jumpActions := make(map[rune]func())
	if executor.SourceCode != nil && executor.SourceCode.ID != "" {
		sourceCodeID := executor.SourceCode.ID
		sourceCodeName := executor.SourceCode.DisplayName()
		jumpActions['c'] = func() {
			a.openSourceCodeOverview(sourceCodeID, valueOr(sourceCodeName, sourceCodeID))
		}
	}
	if executor.Storage != nil && executor.Storage.ID != "" {
		storageID := executor.Storage.ID
		storageName := executor.Storage.Name
		jumpActions['s'] = func() {
			a.openStorageOverview(storageID, valueOr(storageName, storageID))
		}
	}
	if len(executor.Integrations) > 0 {
		jumpActions['i'] = func() {
			a.openOverviewJumpSelection(
				"Executor Integrations",
				executorIntegrationJumpOptions(executor.Integrations),
				"No integrations available",
				func(option overviewJumpOption) {
					integration, ok := option.Value.(client.Integration)
					if !ok {
						return
					}
					a.openIntegrationOverview(integration.ID, valueOr(integration.Name, integration.ID))
				},
			)
		}
	}
	if len(executor.Secrets) > 0 {
		jumpActions['k'] = func() {
			a.openOverviewJumpSelection(
				"Executor Secrets",
				resourceSecretJumpOptions(executor.Secrets),
				"No secrets available",
				func(option overviewJumpOption) {
					secret, ok := option.Value.(client.Secret)
					if !ok {
						return
					}
					a.openSecretOverview(secret.ID, valueOr(secret.Name, secret.ID))
				},
			)
		}
	}
	if len(jumpActions) == 0 {
		return nil
	}
	return jumpActions
}

func executorOverviewView(executor client.Executor) (tview.Primitive, *tview.TextView) {
	summary := kvTable("Summary", [][2]string{
		{"Name", executor.Name},
		{"ID", executor.ID},
		{"State", blankDash(executor.State)},
		{"Status", blankDash(executor.Status)},
		{"Runtime", blankDash(executor.Runtime)},
		{"Revision", fmt.Sprintf("%d", executor.RevisionNumber)},
		{"Created", executor.CreatedAt.Format(time.RFC3339)},
		{"Updated", executor.UpdatedAt.Format(time.RFC3339)},
	})

	references := kvTable("References", [][2]string{
		{"Source Code", executorSourceCodeName(executor.SourceCode)},
		{"Storage", executorStorageName(executor.Storage)},
		{"Creator", storageCreatorName(executor.Creator)},
		{"Labels", strings.Join(orSlice(executor.Labels, []string{"-"}), ", ")},
	})

	configuration := tview.NewTextView()
	configuration.SetBorder(true)
	configuration.SetTitle("Configuration")
	configuration.SetWrap(true)
	configuration.SetDynamicColors(true)
	configuration.SetText(executorConfigurationText(executor))

	relationLines := executorIntegrationLines(executor.Integrations)
	relationLines = append(relationLines, executorSecretLines(executor.Secrets)...)
	if len(relationLines) == 0 {
		relationLines = []string{"-"}
	}
	relations := simpleList("Relations", relationLines)
	lastLogView := newStreamingLogTextView()
	lastLogView.SetBorder(true)
	lastLogView.SetTitle("Last Log")
	lastLogView.SetText("Waiting for logs...")

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(references, 0, 1, false), 10, 0, true)
	root.AddItem(tview.NewFlex().
		AddItem(configuration, 0, 2, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(relations, 0, 1, false).
			AddItem(lastLogView, 0, 1, false), 0, 1, false), 0, 1, false)
	root.AddItem(overviewFooter(executorOverviewHint(executor)), 1, 0, false)
	return root, lastLogView
}

func executorOverviewHint(executor client.Executor) string {
	hints := []string{"y yaml", "A actions", "l logs", "a audit", "E edit", "Esc/q close"}
	if executor.SourceCode != nil && executor.SourceCode.ID != "" {
		hints = append(hints, "c source code")
	}
	if executor.Storage != nil && executor.Storage.ID != "" {
		hints = append(hints, "s storage")
	}
	if len(executor.Integrations) > 0 {
		hints = append(hints, "i integrations")
	}
	if len(executor.Secrets) > 0 {
		hints = append(hints, "k secrets")
	}
	return strings.Join(hints, "  ")
}

func executorConfigurationText(executor client.Executor) string {
	lines := []string{
		"[::b]Description[::-] " + blankDash(executor.Description),
		"[::b]Runtime[::-] " + blankDash(executor.Runtime),
		"[::b]Command Args[::-] " + blankDash(executor.CommandArgs),
		"[::b]Version[::-] " + blankDash(executor.SourceCodeVersion),
		"[::b]Branch[::-] " + blankDash(executor.SourceCodeBranch),
		"[::b]Folder[::-] " + blankDash(executor.SourceCodeFolder),
		"[::b]Storage Path[::-] " + blankDash(executor.StoragePath),
	}
	return strings.Join(lines, "\n")
}

func executorSourceCodeName(sourceCode *client.SourceCode) string {
	if sourceCode == nil {
		return "-"
	}
	return valueOr(sourceCode.DisplayName(), sourceCode.ID)
}

func executorStorageName(storage *client.Storage) string {
	if storage == nil {
		return "-"
	}
	return valueOr(storage.Name, storage.ID)
}

func executorIntegrationJumpOptions(integrations []client.Integration) []overviewJumpOption {
	return resourceIntegrationJumpOptions(integrations)
}

func executorIntegrationLines(integrations []client.Integration) []string {
	if len(integrations) == 0 {
		return nil
	}
	lines := make([]string, 0, len(integrations))
	for _, integration := range integrations {
		lines = append(lines, fmt.Sprintf("Integration: %s (%s/%s)", valueOr(integration.Name, integration.ID), blankDash(integration.IntegrationProvider), blankDash(integration.IntegrationType)))
	}
	return lines
}

func executorSecretLines(secrets []client.Secret) []string {
	if len(secrets) == 0 {
		return nil
	}
	lines := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		lines = append(lines, fmt.Sprintf("Secret: %s (%s/%s)", valueOr(secret.Name, secret.ID), blankDash(secret.SecretProvider), blankDash(secret.SecretType)))
	}
	return lines
}
