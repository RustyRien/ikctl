package cmd

import (
	"fmt"
	"os"

	appcore "github.com/electrolux-oss/ik-tui/internal/app"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "dev"
	date    = "unknown"

	flags config.FlagOverrides

	rootCmd = &cobra.Command{
		Use:   "ik-tui",
		Short: "A TUI for InfraKitchen resources",
		Long:  "ik-tui is a k9s-inspired terminal UI for browsing InfraKitchen resources.",
		RunE:  run,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd())

	rootCmd.Flags().StringVar(&flags.ConfigPath, "config", "", "Path to config file")
	rootCmd.Flags().StringVar(&flags.Endpoint, "endpoint", "", "InfraKitchen base URL")
	rootCmd.Flags().StringVar(&flags.Token, "token", "", "InfraKitchen bearer token")
	rootCmd.Flags().Float64VarP(&flags.RefreshSeconds, "refresh", "r", 0, "Refresh interval in seconds")
	rootCmd.Flags().BoolVar(&flags.InsecureSkipVerify, "insecure-skip-tls-verify", false, "Skip TLS certificate verification")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, _ []string) error {
	flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")

	cfg, err := config.Load(flags)
	if err != nil {
		return err
	}

	app := appcore.New(cfg, appcore.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})

	if err := app.Run(); err != nil {
		return fmt.Errorf("run app: %w", err)
	}

	return nil
}
