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
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func loginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store an InfraKitchen bearer token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")
			flags.HasNoColorsFlag = cmd.Flags().Changed("no-colors")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}
			persistConfig(cmd, &cfg)

			token := strings.TrimSpace(flags.Token)
			if !isPersistentFlagSet(cmd, "token") {
				value, err := promptToken(cmd)
				if err != nil {
					return err
				}
				token = value
			}
			if token == "" {
				return fmt.Errorf("token cannot be empty")
			}

			cfg.Token = token
			if err := cfg.Save(); err != nil {
				return err
			}

			store, err := auth.OpenStore("")
			if err != nil {
				return err
			}
			if err := store.Delete(cfg.Endpoint); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "stored token for %s\n", cfg.Endpoint)
			return err
		},
	}
	return cmd
}

func promptToken(cmd *cobra.Command) (string, error) {
	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "token: "); err != nil {
		return "", err
	}
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("read token: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.ErrOrStderr()); err != nil {
		return "", err
	}
	return strings.TrimSpace(string(secret)), nil
}

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored InfraKitchen login credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")
			flags.HasNoColorsFlag = cmd.Flags().Changed("no-colors")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}
			persistConfig(cmd, &cfg)
			cfg.Token = ""
			if err := cfg.Save(); err != nil {
				return err
			}
			store, err := auth.OpenStore("")
			if err != nil {
				return err
			}
			cred, ok := store.Get(cfg.Endpoint)
			if ok {
				cli := client.New(cfg)
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				_ = cli.Logout(ctx, cred.Provider, cred.RefreshToken)
				cancel()
			}
			if err := store.Delete(cfg.Endpoint); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "logged out from %s\n", cfg.Endpoint)
			return err
		},
	}
}
