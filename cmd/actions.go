package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/auth"
	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/spf13/cobra"
)

func enableCmd() *cobra.Command {
	return entityActionCmd("enable", "Enable an integration or template", func(ctx context.Context, cli *client.Client, entity string, id string) error {
		switch entity {
		case "integrations":
			return cli.EnableIntegration(ctx, id)
		case "templates":
			return cli.EnableTemplate(ctx, id)
		default:
			return fmt.Errorf("enable is currently supported only for integrations and templates")
		}
	})
}

func disableCmd() *cobra.Command {
	return entityActionCmd("disable", "Disable an integration or template", func(ctx context.Context, cli *client.Client, entity string, id string) error {
		switch entity {
		case "integrations":
			return cli.DisableIntegration(ctx, id)
		case "templates":
			return cli.DisableTemplate(ctx, id)
		default:
			return fmt.Errorf("disable is currently supported only for integrations and templates")
		}
	})
}

func deleteCmd() *cobra.Command {
	return entityActionCmd("delete", "Delete an integration or template", func(ctx context.Context, cli *client.Client, entity string, id string) error {
		switch entity {
		case "integrations":
			return cli.DeleteIntegration(ctx, id)
		case "templates":
			return cli.DeleteTemplate(ctx, id)
		default:
			return fmt.Errorf("delete is currently supported only for integrations and templates")
		}
	})
}

func entityActionCmd(use string, short string, action func(context.Context, *client.Client, string, string) error) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <integrations|templates> <name-or-id>",
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEntityAction(cmd, use, args[0], args[1], action)
		},
	}
}

func runEntityAction(cmd *cobra.Command, verb string, entity string, nameOrID string, action func(context.Context, *client.Client, string, string) error) error {
	flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")
	flags.HasNoColorsFlag = cmd.Flags().Changed("no-colors")

	cfg, err := config.Load(flags)
	if err != nil {
		return err
	}
	persistConfig(cmd, &cfg)

	cli, err := auth.NewClient(cfg)
	if err != nil {
		return err
	}
	registry := resource.DefaultRegistry(cli)
	descriptor, ok := registry.Resolve(entity)
	if !ok {
		return fmt.Errorf("unknown entity %q (valid: %s)", entity, strings.Join(registry.Names(), ", "))
	}
	if descriptor.Name != "integrations" && descriptor.Name != "templates" {
		return fmt.Errorf("%s is currently supported only for integrations and templates", verb)
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

	target, err := actionTarget(descriptor.Name, raw)
	if err != nil {
		return err
	}

	if err := action(ctx, cli, descriptor.Name, target.ID); err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s request sent: %s (%s)\n", strings.TrimSuffix(target.Kind, "s"), verb, target.Name, target.ID)
	return err
}

type actionEntityTarget struct {
	Kind string
	ID   string
	Name string
}

func actionTarget(entity string, raw any) (actionEntityTarget, error) {
	switch entity {
	case "integrations":
		integration, ok := raw.(client.Integration)
		if !ok {
			return actionEntityTarget{}, fmt.Errorf("unexpected integration payload type %T", raw)
		}
		return actionEntityTarget{Kind: entity, ID: integration.ID, Name: integration.Name}, nil
	case "templates":
		template, ok := raw.(client.Template)
		if !ok {
			return actionEntityTarget{}, fmt.Errorf("unexpected template payload type %T", raw)
		}
		return actionEntityTarget{Kind: entity, ID: template.ID, Name: template.Name}, nil
	default:
		return actionEntityTarget{}, fmt.Errorf("unsupported entity %q", entity)
	}
}
