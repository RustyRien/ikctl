package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/spf13/cobra"
)

var streamLogLevelPrefixRX = regexp.MustCompile(`(?i)^\[(trace|debug|info|warn|warning|error|fatal)\]\s*`)

func logCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log <entity> <name-or-id>",
		Short: "Stream live logs for a single InfraKitchen entity",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLog(cmd, args[0], args[1])
		},
	}
	return cmd
}

func runLog(cmd *cobra.Command, entity string, nameOrID string) error {
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

	if _, err := fmt.Fprintf(os.Stderr, "Streaming logs for resource %s (%s). Press Ctrl-C to stop.\n", resourceItem.Name, resourceItem.ID); err != nil {
		return err
	}

	return cli.StreamLogs(ctx, descriptor.Singular, resourceItem.ID, func(message client.LogStreamMessage) error {
		formatted := formatStreamLog(message, cfg.NoColors)
		_, err := fmt.Fprintln(os.Stdout, formatted)
		return err
	})
}

func formatStreamLog(message client.LogStreamMessage, noColors bool) string {
	bodyLines := strings.Split(message.Data, "\n")
	for i, line := range bodyLines {
		bodyLines[i] = streamLogLevelPrefixRX.ReplaceAllString(line, "")
	}
	body := strings.Join(bodyLines, "\n")
	if noColors {
		return body
	}
	return logLevelANSI(message.Level) + body + "\x1b[0m"
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
