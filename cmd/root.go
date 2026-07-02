package cmd

import (
	"fmt"
	"os"

	appcore "github.com/electrolux-oss/ik-tui/internal/app"
	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "dev"
	date    = "unknown"

	flags config.FlagOverrides

	rootCmd = &cobra.Command{
		Use:   "ikctl",
		Short: "InfraKitchen CLI and TUI",
		Long:  "ikctl is a kubectl-style CLI and k9s-inspired TUI for browsing InfraKitchen resources.",
		RunE:  run,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(describeCmd())
	rootCmd.AddCommand(logCmd())
	rootCmd.AddCommand(enableCmd())
	rootCmd.AddCommand(disableCmd())
	rootCmd.AddCommand(deleteCmd())

	rootCmd.PersistentFlags().StringVar(&flags.ConfigPath, "config", "", "Path to config file")
	rootCmd.PersistentFlags().StringVar(&flags.Endpoint, "endpoint", "", "InfraKitchen base URL")
	rootCmd.PersistentFlags().StringVar(&flags.Token, "token", "", "InfraKitchen bearer token")
	rootCmd.PersistentFlags().Float64VarP(&flags.RefreshSeconds, "refresh", "r", 0, "Refresh interval in seconds")
	rootCmd.PersistentFlags().BoolVar(&flags.InsecureSkipVerify, "insecure-skip-tls-verify", false, "Skip TLS certificate verification")
	rootCmd.PersistentFlags().BoolVar(&flags.NoColors, "no-colors", false, "Disable colored ANSI output")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, _ []string) error {
	flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")
	flags.HasNoColorsFlag = cmd.Flags().Changed("no-colors")

	cfg, err := config.Load(flags)
	if err != nil {
		return err
	}

	activeEntity := "resources"
	if len(cmd.Flags().Args()) > 0 {
		registry := resource.DefaultRegistry(client.New(cfg))
		if descriptor, ok := registry.Resolve(cmd.Flags().Arg(0)); ok {
			activeEntity = descriptor.Name
		}
	}

	app := appcore.New(cfg, appcore.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}, activeEntity)

	if err := app.Run(); err != nil {
		return fmt.Errorf("run app: %w", err)
	}

	return nil
}
