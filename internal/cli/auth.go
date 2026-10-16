package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/config"
)

// LoginResult is the structured output of login and whoami.
type LoginResult struct {
	Server  string    `json:"server"`
	Context string    `json:"context,omitempty"`
	User    *sdk.User `json:"user"`
}

func newLoginCmd(f *Factory) *cobra.Command {
	var (
		tokenStdin bool
		output     string
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with an API token and save it to the config file",
		Long: `Log in to an InfraKitchen server with a personal access token (ik_...) or a
service-account token, validate it, and save it as a context in
~/.config/ikctl/config.yaml (mode 0600). The context becomes the current context.

Create a personal access token in the InfraKitchen UI (user menu > Access tokens).
The token is read, in order of preference, from:
  --token-stdin       standard input (recommended for scripts)
  an interactive prompt (hidden input) when running in a terminal
  --token / IKCTL_TOKEN

The context is named after the server host unless --context is given.`,
		Example: `  # Interactive
  ikctl login --server https://infrakitchen.example.com

  # Non-interactive
  echo "$IK_TOKEN" | ikctl login --server http://localhost:7777 --token-stdin

  # Save as a named context
  ikctl login --server https://ik.prod.example.com --context prod`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := printers.Output{}
			var err error
			if output != "" {
				if out, err = parseOutput(output, printers.FormatJSON, printers.FormatYAML); err != nil {
					return err
				}
			}
			path, err := f.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}

			// Server: flag > env > named/current context > prompt.
			server := firstNonEmpty(f.flags.server, os.Getenv(config.EnvServer))
			if server == "" {
				name := firstNonEmpty(f.flags.context, cfg.CurrentContext)
				if existing, ok := cfg.Get(name); ok {
					server = existing.Server
				}
			}
			if server == "" {
				if !f.IO.InTTY || tokenStdin {
					return usageErrorf("--server is required")
				}
				if server, err = promptLine(f, "Server URL: "); err != nil {
					return err
				}
			}
			u, err := sdk.ParseServerURL(server)
			if err != nil {
				return usageError{err}
			}

			token, err := readToken(f, tokenStdin)
			if err != nil {
				return err
			}
			if err := validateTokenFormat(token); err != nil {
				return usageError{err}
			}

			opts := append([]sdk.Option{sdk.WithServer(u.String()), sdk.WithToken(token),
				sdk.WithInsecureSkipTLSVerify(f.flags.insecure)}, f.ClientOptions()...)
			c, err := sdk.New(opts...)
			if err != nil {
				return err
			}
			user, err := c.Auth.Me(cmd.Context())
			if err != nil {
				return fmt.Errorf("token validation failed: %w", err)
			}

			name := f.flags.context
			if name == "" {
				name = u.Host
			}
			cfg.Upsert(config.Context{Name: name, Server: c.ServerURL(), Token: token,
				User: firstNonEmpty(user.Email, user.Identifier), InsecureSkipTLSVerify: f.flags.insecure})
			cfg.CurrentContext = name
			if err := config.Save(path, cfg); err != nil {
				return err
			}

			if out.IsStructured() {
				return printers.PrintStructured(f.IO.Out, out.Format, LoginResult{Server: c.ServerURL(), Context: name, User: user})
			}
			fmt.Fprintf(f.IO.Out, "Logged in to %s as %s (context %q, saved to %s)\n",
				c.ServerURL(), userLabel(user), name, path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&tokenStdin, "token-stdin", false, "read the token from standard input")
	addOutputFlag(cmd, &output, "", printers.FormatJSON, printers.FormatYAML)
	return cmd
}

func readToken(f *Factory, fromStdin bool) (string, error) {
	switch {
	case fromStdin:
		data, err := io.ReadAll(io.LimitReader(f.IO.In, 64*1024))
		if err != nil {
			return "", fmt.Errorf("read token from stdin: %w", err)
		}
		token := strings.TrimSpace(string(data))
		if token == "" {
			return "", usageErrorf("no token on standard input")
		}
		return token, nil
	case f.flags.token != "":
		warnf(f.IO.ErrOut, "passing --token on the command line may expose it in shell history; prefer --token-stdin")
		return f.flags.token, nil
	case os.Getenv(config.EnvToken) != "":
		return strings.TrimSpace(os.Getenv(config.EnvToken)), nil
	case f.IO.InTTY:
		file, ok := f.IO.In.(*os.File)
		if !ok {
			return promptLine(f, "Token: ")
		}
		fmt.Fprint(f.IO.ErrOut, "Token: ")
		b, err := term.ReadPassword(int(file.Fd())) //nolint:gosec // fd fits in int
		fmt.Fprintln(f.IO.ErrOut)
		if err != nil {
			return "", fmt.Errorf("read token: %w", err)
		}
		token := strings.TrimSpace(string(b))
		if token == "" {
			return "", usageErrorf("empty token")
		}
		return token, nil
	default:
		return "", usageErrorf("no token given: use --token-stdin, IKCTL_TOKEN or run interactively")
	}
}

func validateTokenFormat(token string) error {
	if sdk.IsPersonalAccessToken(token) {
		_, err := sdk.ParseToken(token)
		return err
	}
	if strings.Count(token, ".") == 2 { // service-account JWT
		return nil
	}
	return errors.New("unrecognized token: expected a personal access token (ik_...) or a service-account JWT")
}

func promptLine(f *Factory, prompt string) (string, error) {
	fmt.Fprint(f.IO.ErrOut, prompt)
	line, err := bufio.NewReader(f.IO.In).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", usageErrorf("no value entered")
	}
	return line, nil
}

func userLabel(u *sdk.User) string {
	switch {
	case u.Email != "" && u.Identifier != "" && u.Email != u.Identifier:
		return fmt.Sprintf("%s <%s>", u.Identifier, u.Email)
	case u.Email != "":
		return u.Email
	default:
		return u.Identifier
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func newLogoutCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved token of the current (or --context) context",
		Args:  usageArgs(cobra.NoArgs),
		RunE: func(_ *cobra.Command, _ []string) error {
			path, err := f.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			name := firstNonEmpty(f.flags.context, os.Getenv(config.EnvContext), cfg.CurrentContext)
			ctx, ok := cfg.Get(name)
			if !ok {
				return &sdk.APIError{Code: sdk.CodeNotFound, Message: "no context to log out of"}
			}
			ctx.Token = ""
			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "Logged out of context %q (%s). Revoke the token in the InfraKitchen UI if it is no longer needed.\n", name, ctx.Server)
			return nil
		},
	}
}

func newWhoamiCmd(f *Factory) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the user the current token belongs to",
		Example: `  ikctl whoami
  ikctl whoami -o json`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := parseOutput(output, printers.FormatTable, printers.FormatJSON, printers.FormatYAML)
			if err != nil {
				return err
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			user, err := c.Auth.Me(cmd.Context())
			if err != nil {
				return err
			}
			res := LoginResult{Server: c.ServerURL(), User: user}
			if r, err := config.Resolve(f.Overrides()); err == nil {
				res.Context = r.ContextName
			}
			if out.IsStructured() {
				return printers.PrintStructured(f.IO.Out, out.Format, res)
			}
			fmt.Fprintf(f.IO.Out, "%s (provider: %s, id: %s) on %s\n", userLabel(user), user.Provider, user.ID, res.Server)
			return nil
		},
	}
	addOutputFlag(cmd, &output, printers.FormatTable, printers.FormatTable, printers.FormatJSON, printers.FormatYAML)
	return cmd
}
