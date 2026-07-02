package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/spf13/cobra"
)

func enableCmd() *cobra.Command {
	return integrationActionCmd("enable", "Enable an integration", func(ctx context.Context, cli *client.Client, id string) error {
		return cli.EnableIntegration(ctx, id)
	})
}

func disableCmd() *cobra.Command {
	return integrationActionCmd("disable", "Disable an integration", func(ctx context.Context, cli *client.Client, id string) error {
		return cli.DisableIntegration(ctx, id)
	})
}

func deleteCmd() *cobra.Command {
	return integrationActionCmd("delete", "Delete an integration", func(ctx context.Context, cli *client.Client, id string) error {
		return cli.DeleteIntegration(ctx, id)
	})
}

func integrationActionCmd(use string, short string, action func(context.Context, *client.Client, string) error) *cobra.Command {
	return &cobra.Command{
		Use:   use + " integrations <name-or-id>",
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIntegrationAction(cmd, use, args[0], args[1], action)
		},
	}
}

func runIntegrationAction(cmd *cobra.Command, verb string, entity string, nameOrID string, action func(context.Context, *client.Client, string) error) error {
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
	if descriptor.Name != "integrations" {
		return fmt.Errorf("%s is currently supported only for integrations", verb)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, raw, err := descriptor.GetByID(ctx, nameOrID)
	if err != nil {
		_, raw, err = descriptor.ResolveByName(ctx, nameOrID)
		if err != nil {
			return err
		}
	}

	integration, ok := raw.(client.Integration)
	if !ok {
		return fmt.Errorf("unexpected integration payload type %T", raw)
	}

	if err := action(ctx, cli, integration.ID); err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "integration %s request sent: %s (%s)\n", verb, integration.Name, integration.ID)
	return err
}
