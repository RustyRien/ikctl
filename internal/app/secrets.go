package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/rivo/tview"
)

func (a *App) openSecretOverview(id string, name string) {
	a.stopLiveLogStream()
	a.rememberCurrentDetailState()

	title := fmt.Sprintf("Secret: %s", name)
	a.clearOverviewJumpState()
	a.activeTemplateDetail = nil
	a.activeSourceCodeDetail = nil
	a.activeSourceCodeVersionDetail = nil
	a.activeSecretDetail = &entityDetailSelection{ID: id, Name: name, Kind: "secrets"}
	a.activeIntegrationDetail = nil
	a.activeStorageDetail = nil
	a.activeWorkerDetail = nil
	a.activeWorkspaceDetail = nil
	a.auditLogRows = nil
	a.auditLogTable = nil
	a.ui.OpenDetail(title, "Loading secret overview...")
	a.ui.SetSecretOverviewHotkeys()

	go func() {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		full, err := a.client.Secret(ctx, id)
		var primitive tview.Primitive
		var jumpActions map[rune]func()
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load secret overview.\n\n%v", err))
		} else if full != nil {
			a.activeSecretDetail = &entityDetailSelection{ID: full.ID, Name: full.Name, Kind: "secrets"}
			primitive = secretOverviewView(*full)
			jumpActions = a.secretOverviewJumpActions(*full)
		} else {
			a.activeSecretDetail = nil
			a.activeWorkerDetail = nil
			a.activeWorkspaceDetail = nil
			primitive = errorView("Secret not found")
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.clearOverviewJumpState()
			for key, action := range jumpActions {
				a.setOverviewJumpAction(key, action)
			}
			a.ui.OpenDetailPrimitive(title, primitive)
			a.ui.SetSecretOverviewHotkeys()
		})
	}()
}

func (a *App) secretOverviewJumpActions(secret client.Secret) map[rune]func() {
	if secret.ID == "" {
		return nil
	}
	return map[rune]func(){
		'r': func() {
			a.openSecretResources(secret)
		},
	}
}

func (a *App) openSecretResources(secret client.Secret) {
	title := fmt.Sprintf("Secret Resources: %s", valueOr(secret.Name, secret.ID))
	a.overviewJumpSelector = nil
	a.ui.OpenOverlay(title, "Loading secret resources...")

	go func(secret client.Secret) {
		done := a.ui.BeginLoading()
		defer done()

		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()

		result, err := a.client.Resources(ctx, map[string]any{"secret_ids__any": []string{secret.ID}}, []string{"updated_at", "DESC"}, []int{0, 200})
		var primitive tview.Primitive
		var selector *overviewJumpSelector
		if err != nil {
			primitive = errorView(fmt.Sprintf("Failed to load secret resources.\n\n%v", err))
		} else if len(result.Items) == 0 {
			primitive = errorView("No resources found for this secret")
		} else {
			options := templateResourceJumpOptions(result.Items)
			primitive, selector = templateResourceSelectionView(options, len(result.Items), result.Total)
			selector.onSelect = func(option overviewJumpOption) {
				resourceItem, ok := option.Value.(client.Resource)
				if !ok {
					return
				}
				a.openResourceOverview(client.Resource{ID: resourceItem.ID, Name: resourceItem.Name})
			}
		}

		a.ui.Application().QueueUpdateDraw(func() {
			a.overviewJumpSelector = selector
			a.ui.OpenOverlayPrimitive(title, primitive)
		})
	}(secret)
}

func secretOverviewView(secret client.Secret) tview.Primitive {
	summary := kvTable("Summary", [][2]string{
		{"Name", secret.Name},
		{"ID", secret.ID},
		{"Type", blankDash(secret.SecretType)},
		{"Provider", blankDash(secret.SecretProvider)},
		{"State", blankDash(secret.State)},
		{"Status", blankDash(secret.Status)},
		{"Created", secret.CreatedAt.Format(time.RFC3339)},
		{"Updated", secret.UpdatedAt.Format(time.RFC3339)},
	})

	usage := kvTable("Usage", [][2]string{
		{"Resources", fmt.Sprintf("%d", secret.ResourcesCount)},
		{"Executors", fmt.Sprintf("%d", secret.ExecutorsCount)},
		{"Revision", fmt.Sprintf("%d", secret.RevisionNumber)},
		{"Integration", storageIntegrationName(secret.Integration)},
		{"Creator", storageCreatorName(secret.Creator)},
	})

	description := tview.NewTextView()
	description.SetBorder(true)
	description.SetTitle("Description")
	description.SetWrap(true)
	description.SetText(valueOr(secret.Description, "-"))

	configuration := tview.NewTextView()
	configuration.SetBorder(true)
	configuration.SetTitle("Configuration")
	configuration.SetWrap(true)
	configuration.SetDynamicColors(true)
	configuration.SetText(secretConfigurationText(secret))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(tview.NewFlex().
		AddItem(summary, 0, 1, false).
		AddItem(usage, 0, 1, false), 10, 0, true)
	root.AddItem(tview.NewFlex().
		AddItem(description, 0, 2, false).
		AddItem(configuration, 0, 3, false), 0, 1, false)
	root.AddItem(overviewFooter(secretOverviewHint(secret)), 1, 0, false)
	return root
}

func secretConfigurationText(secret client.Secret) string {
	lines := []string{
		"[::b]Provider[::-] " + blankDash(secret.SecretProvider),
		"[::b]Type[::-] " + blankDash(secret.SecretType),
	}
	if secret.Integration != nil && secret.Integration.Name != "" {
		lines = append(lines, "[::b]Integration[::-] "+secret.Integration.Name)
	}
	if len(secret.Labels) > 0 {
		lines = append(lines, "[::b]Labels[::-] "+strings.Join(secret.Labels, ", "))
	}
	if secret.SecretProvider == "custom" {
		if secrets, ok := secret.Configuration["secrets"].([]any); ok && len(secrets) > 0 {
			lines = append(lines, "[::b]Secret List[::-]")
			lines = append(lines, customSecretLines(secrets)...)
			return strings.Join(lines, "\n")
		}
		lines = append(lines, "[::b]Secret List[::-] -")
		return strings.Join(lines, "\n")
	}
	for _, line := range mapLines(secret.Configuration) {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func customSecretLines(values []any) []string {
	lines := make([]string, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		name, _ := item["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  • %s", name))
	}
	if len(lines) == 0 {
		return []string{"  -"}
	}
	sort.Strings(lines)
	return lines
}

func secretOverviewHint(secret client.Secret) string {
	hints := []string{"y yaml", "l logs", "a audit", "r resources", "E edit", "Esc/q close"}
	if secret.ID == "" {
		hints = []string{"y yaml", "l logs", "a audit", "E edit", "Esc/q close"}
	}
	return strings.Join(hints, "  ")
}
