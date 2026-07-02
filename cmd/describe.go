package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/auth"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/spf13/cobra"
)

func describeCmd() *cobra.Command {
	output := "yaml"
	cmd := &cobra.Command{
		Use:   "describe <entity> <name-or-id>",
		Short: "Describe a single InfraKitchen entity",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDescribe(cmd, args[0], args[1], output)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", output, "Output format: yaml, json, table, wide, name")
	return cmd
}

func runDescribe(cmd *cobra.Command, entity string, nameOrID string, output string) error {
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

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return printSingle(ctx, descriptor, nameOrID, output)
}
