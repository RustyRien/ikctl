package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/spf13/cobra"
)

var streamLogLevelPrefixRX = regexp.MustCompile(`(?i)^\[(trace|debug|info|warn|warning|error|fatal)\]\s*`)

const initialLogHistoryLimit = 200

func logCmd() *cobra.Command {
	var since string
	var follow bool

	cmd := &cobra.Command{
		Use:   "log <entity> <name-or-id>",
		Short: "Show logs for a single InfraKitchen entity",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLog(cmd, args[0], args[1], since, follow)
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "Only show logs since a duration like 1h30m or an RFC3339 timestamp")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow live log output after printing history")
	return cmd
}

func runLog(cmd *cobra.Command, entity string, nameOrID string, sinceValue string, follow bool) error {
	flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")
	flags.HasNoColorsFlag = cmd.Flags().Changed("no-colors")

	cfg, err := config.Load(flags)
	if err != nil {
		return err
	}

	cli := client.New(cfg)
	registry := resource.DefaultRegistry(cli)
	descriptor, ok := registry.Resolve(entity)
	if !ok {
		return fmt.Errorf("unknown entity %q (valid: %s)", entity, strings.Join(registry.Names(), ", "))
	}
	if descriptor.Name != "resources" {
		return fmt.Errorf("live log streaming is currently supported only for resources")
	}

	since, err := parseSince(sinceValue, time.Now())
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	resolveCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	_, raw, err := descriptor.GetByID(resolveCtx, nameOrID)
	if err != nil {
		_, raw, err = descriptor.ResolveByName(resolveCtx, nameOrID)
		if err != nil {
			return err
		}
	}

	resourceItem, ok := raw.(client.Resource)
	if !ok {
		return fmt.Errorf("unexpected resource payload type %T", raw)
	}

	if _, err := fmt.Fprintln(os.Stderr, logStartMessage(resourceItem, since, follow)); err != nil {
		return err
	}

	historyCtx, historyCancel := context.WithTimeout(ctx, 20*time.Second)
	historyLogs, _, err := cli.LogsForResource(historyCtx, resourceItem.ID, []int{0, initialLogHistoryLimit})
	historyCancel()
	if err != nil {
		return err
	}
	historyLogs = filterLogsSince(historyLogs, since)

	deduper := newLogDeduper(historyLogs)
	slices.Reverse(historyLogs)
	for _, log := range historyLogs {
		if _, err := fmt.Fprintln(os.Stdout, formatHistoricalLog(log, cfg.NoColors)); err != nil {
			return err
		}
	}
	if !follow {
		return nil
	}

	return cli.StreamLogs(ctx, descriptor.Singular, resourceItem.ID, func(message client.LogStreamMessage) error {
		if deduper.ShouldSuppress(message) {
			return nil
		}
		formatted := formatStreamLog(message, cfg.NoColors)
		_, err := fmt.Fprintln(os.Stdout, formatted)
		return err
	})
}

func formatHistoricalLog(log client.Log, noColors bool) string {
	return formatStreamLog(client.LogStreamMessage{Data: log.Data, Level: log.Level}, noColors)
}

func formatStreamLog(message client.LogStreamMessage, noColors bool) string {
	body := normalizeStreamBody(message.Data)
	if noColors {
		return body
	}
	return logLevelANSI(message.Level) + body + "\x1b[0m"
}

func normalizeStreamBody(data string) string {
	bodyLines := strings.Split(data, "\n")
	for i, line := range bodyLines {
		bodyLines[i] = streamLogLevelPrefixRX.ReplaceAllString(line, "")
	}
	return strings.Join(bodyLines, "\n")
}

func logLevelANSI(level string) string {
	switch strings.ToLower(level) {
	case "trace", "debug":
		return "\x1b[38;5;110m"
	case "warn", "warning":
		return "\x1b[33m"
	case "error", "fatal":
		return "\x1b[31m"
	default:
		return "\x1b[38;5;153m"
	}
}

type logDeduper struct {
	pending map[string]int
	active  bool
}

func newLogDeduper(history []client.Log) *logDeduper {
	if len(history) == 0 {
		return &logDeduper{}
	}

	start := max(0, len(history)-50)
	pending := make(map[string]int, len(history)-start)
	for _, log := range history[start:] {
		pending[logMessageKey(log.Level, log.Data)]++
	}

	return &logDeduper{pending: pending, active: len(pending) > 0}
}

func (d *logDeduper) ShouldSuppress(message client.LogStreamMessage) bool {
	if d == nil || !d.active {
		return false
	}

	key := logMessageKey(message.Level, message.Data)
	if count := d.pending[key]; count > 0 {
		if count == 1 {
			delete(d.pending, key)
		} else {
			d.pending[key] = count - 1
		}
		return true
	}

	d.active = false
	clear(d.pending)
	return false
}

func logMessageKey(level string, data string) string {
	return strings.ToLower(strings.TrimSpace(level)) + "\x00" + normalizeStreamBody(data)
}

func parseSince(value string, now time.Time) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if duration, err := time.ParseDuration(value); err == nil {
		since := now.Add(-duration)
		return &since, nil
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("invalid --since value %q: use a duration like 1h30m or an RFC3339 timestamp", value)
	}
	return &timestamp, nil
}

func filterLogsSince(logs []client.Log, since *time.Time) []client.Log {
	if since == nil {
		return logs
	}

	filtered := make([]client.Log, 0, len(logs))
	for _, log := range logs {
		if !log.CreatedAt.Before(*since) {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func logStartMessage(resourceItem client.Resource, since *time.Time, follow bool) string {
	action := "Showing recent logs"
	if since != nil {
		action = fmt.Sprintf("Showing logs since %s", since.Format(time.RFC3339))
	}
	message := fmt.Sprintf("%s for resource %s (%s).", action, resourceItem.Name, resourceItem.ID)
	if !follow {
		return message
	}
	return message + " Following live output until Ctrl-C."
}
