package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/auth"
	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	editcore "github.com/electrolux-oss/ik-tui/internal/edit"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/spf13/cobra"
)

func editCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <entity> <name-or-id>",
		Short: "Edit a single InfraKitchen entity in your editor",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEdit(cmd, args[0], args[1])
		},
	}
	return cmd
}

func runEdit(cmd *cobra.Command, entity string, nameOrID string) error {
	flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")
	flags.HasNoColorsFlag = cmd.Flags().Changed("no-colors")

	cfg, err := loadConfigForOneShot(cmd)
	if err != nil {
		return err
	}
	cli, err := auth.NewClient(cfg)
	if err != nil {
		return err
	}

	registry := resource.DefaultRegistry(cli)
	descriptor, ok := registry.Resolve(entity)
	if !ok {
		return fmt.Errorf("unknown entity %q (valid: %s)", entity, strings.Join(registry.Names(), ", "))
	}

	session, err := buildEditSession(context.Background(), descriptor, nameOrID)
	if err != nil {
		return err
	}

	loadCtx, loadCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer loadCancel()
	applyCtx, applyCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer applyCancel()
	if err := editcore.RunPhased(loadCtx, applyCtx, session, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "%s %s updated\n", descriptor.Singular, session.Name)
	return nil
}

func buildEditSession(ctx context.Context, descriptor *resource.Descriptor, nameOrID string) (editcore.Session, error) {
	_, raw, err := descriptor.GetByID(ctx, nameOrID)
	if err != nil {
		_, raw, err = descriptor.ResolveByName(ctx, nameOrID)
		if err != nil {
			return editcore.Session{}, err
		}
	}
	id, name, err := editMeta(raw)
	if err != nil {
		return editcore.Session{}, err
	}
	if descriptor.EditLoad == nil || descriptor.ApplyEdit == nil {
		return editcore.Session{}, fmt.Errorf("edit is not supported for %s", descriptor.Name)
	}
	return editcore.Session{
		Kind:  descriptor.Singular,
		ID:    id,
		Name:  name,
		Load:  descriptor.EditLoad,
		Apply: descriptor.ApplyEdit,
	}, nil
}

func editMeta(raw any) (string, string, error) {
	switch value := raw.(type) {
	case client.Resource:
		return value.ID, value.Name, nil
	case client.Template:
		return value.ID, value.Name, nil
	case client.Integration:
		return value.ID, value.Name, nil
	case client.SourceCode:
		return value.ID, value.GetName(), nil
	case client.SourceCodeVersion:
		return value.ID, value.GetName(), nil
	}
	return "", "", fmt.Errorf("unsupported entity type %T", raw)
}

func loadConfigForOneShot(cmd *cobra.Command) (config.Config, error) {
	cfg, err := config.Load(flags)
	if err != nil {
		return config.Config{}, err
	}
	persistConfig(cmd, &cfg)
	return cfg, nil
}
