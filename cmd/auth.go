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
	var provider string
	var scope string
	var refreshToken string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store an InfraKitchen refresh token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")
			flags.HasNoColorsFlag = cmd.Flags().Changed("no-colors")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}
			persistConfig(cmd, &cfg)

			providerName, impliedScope, err := auth.ParseProvider(provider)
			if err != nil {
				return err
			}
			if providerName == "guest" {
				if strings.TrimSpace(scope) == "" {
					scope = impliedScope
				}
				if err := auth.ValidateGuestScope(scope); err != nil {
					return err
				}
			} else if strings.TrimSpace(scope) != "" {
				return fmt.Errorf("--scope is only supported for the guest provider")
			}
			if strings.TrimSpace(refreshToken) == "" {
				value, err := promptSecret(cmd, providerName)
				if err != nil {
					return err
				}
				refreshToken = value
			}
			if strings.TrimSpace(refreshToken) == "" {
				return fmt.Errorf("refresh token cannot be empty")
			}

			store, err := auth.OpenStore("")
			if err != nil {
				return err
			}
			entry := auth.Credentials{Provider: providerName, RefreshToken: refreshToken, Scope: scope}
			if err := store.Put(cfg.Endpoint, entry); err != nil {
				return err
			}

			cli, err := auth.NewClient(cfg)
			if err != nil {
				return err
			}
			refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer refreshCancel()
			refreshed, err := cli.RefreshAuthToken(refreshCtx, providerName, refreshToken)
			if err != nil {
				return err
			}
			entry.Token = refreshed.Token
			entry.TokenExpiry = refreshed.Expiration.Time
			entry.RefreshToken = refreshed.RefreshToken
			if err := store.Put(cfg.Endpoint, entry); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "stored %s refresh token for %s\n", providerName, cfg.Endpoint)
			return err
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Auth provider: microsoft, github, guest, guest_default, guest_super, guest_infra")
	cmd.Flags().StringVar(&scope, "scope", "", "Guest scope: default, super, infra")
	cmd.Flags().StringVar(&refreshToken, "refresh-token", "", "Refresh token or guest token from InfraKitchen")
	_ = cmd.MarkFlagRequired("provider")
	return cmd
}

func promptSecret(cmd *cobra.Command, provider string) (string, error) {
	label := "refresh token"
	if provider == "guest" {
		label = "guest token"
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s: ", label); err != nil {
		return "", err
	}
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", label, err)
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
