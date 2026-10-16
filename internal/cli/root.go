// Package cli implements the ikctl command line on top of the sdk package.
package cli

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rustyrien/ikctl/internal/version"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/config"
)

// IOStreams are the standard streams of a command.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
	// InTTY reports whether In is an interactive terminal (prompts allowed).
	InTTY bool
}

// SystemIO returns streams bound to the process stdin/stdout/stderr.
func SystemIO() IOStreams {
	return IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr, InTTY: term.IsTerminal(int(os.Stdin.Fd()))} //nolint:gosec // fd fits in int
}

type globalFlags struct {
	configPath string
	context    string
	server     string
	token      string
	timeout    time.Duration
	verbose    bool
	insecure   bool
}

// Factory carries shared state and builds API clients for commands.
type Factory struct {
	IO    IOStreams
	flags globalFlags
	// newClient may be replaced in tests.
	newClient func(o config.Overrides, opts ...sdk.Option) (*sdk.Client, error)
}

// NewFactory creates a Factory.
func NewFactory(io IOStreams) *Factory {
	return &Factory{IO: io, newClient: sdk.NewFromConfig}
}

// Overrides returns the connection overrides from global flags.
func (f *Factory) Overrides() config.Overrides {
	return config.Overrides{
		ConfigPath:            f.flags.configPath,
		Context:               f.flags.context,
		Server:                f.flags.server,
		Token:                 f.flags.token,
		InsecureSkipTLSVerify: f.flags.insecure,
	}
}

// ConfigPath returns the effective config file path.
func (f *Factory) ConfigPath() (string, error) {
	if f.flags.configPath != "" {
		return f.flags.configPath, nil
	}
	return config.DefaultPath()
}

// ClientOptions returns the common SDK options (timeout, user agent, logging).
func (f *Factory) ClientOptions() []sdk.Option {
	opts := []sdk.Option{sdk.WithTimeout(f.flags.timeout), sdk.WithUserAgent(version.UserAgent())}
	if f.flags.verbose {
		h := slog.NewTextHandler(f.IO.ErrOut, &slog.HandlerOptions{Level: slog.LevelDebug})
		opts = append(opts, sdk.WithLogger(slog.New(h)))
	}
	return opts
}

// Client builds an API client from flags, environment and the config file.
func (f *Factory) Client() (*sdk.Client, error) {
	if path, err := f.ConfigPath(); err == nil {
		if err := config.CheckPermissions(path); err != nil {
			warnf(f.IO.ErrOut, "%v", err)
		}
	}
	return f.newClient(f.Overrides(), f.ClientOptions()...)
}

// NewRootCmd builds the ikctl command tree.
func NewRootCmd(f *Factory) *cobra.Command {
	root := &cobra.Command{
		Use:   "ikctl",
		Short: "ikctl controls InfraKitchen from the command line",
		Long: `ikctl controls InfraKitchen from the command line.

Commands follow a "verb kind [name]" pattern and every command that returns data
supports -o json and -o yaml for scripting and automation.

Get started:
  ikctl login --server https://infrakitchen.example.com
  ikctl get resources
  ikctl describe resource my-resource
  ikctl api-resources

Environment:
  IKCTL_SERVER, IKCTL_TOKEN, IKCTL_CONTEXT, IKCTL_CONFIG override the config file
  (` + "`~/.config/ikctl/config.yaml`" + `).

Exit codes:
  0 success, 1 error, 2 usage error, 3 authentication/authorization error,
  4 not found, 5 conflict (wrong state, validation, ambiguous name, action not allowed).`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Get().Version,
	}
	root.SetIn(f.IO.In)
	root.SetOut(f.IO.Out)
	root.SetErr(f.IO.ErrOut)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	root.SetVersionTemplate("ikctl {{.Version}}\n")

	pf := root.PersistentFlags()
	pf.StringVar(&f.flags.configPath, "config", "", "path to the config file (env IKCTL_CONFIG, default ~/.config/ikctl/config.yaml)")
	pf.StringVar(&f.flags.context, "context", "", "name of the config context to use (env IKCTL_CONTEXT)")
	pf.StringVarP(&f.flags.server, "server", "s", "", "InfraKitchen server URL (env IKCTL_SERVER)")
	pf.StringVar(&f.flags.token, "token", "", "API token; prefer 'ikctl login' or IKCTL_TOKEN to keep it out of shell history")
	pf.DurationVar(&f.flags.timeout, "timeout", 30*time.Second, "request timeout")
	pf.BoolVarP(&f.flags.verbose, "verbose", "v", false, "log HTTP and GraphQL requests to stderr (tokens are never logged)")
	pf.BoolVar(&f.flags.insecure, "insecure-skip-tls-verify", false, "skip TLS certificate verification (insecure)")

	root.AddGroup(
		&cobra.Group{ID: "basic", Title: "Basic Commands:"},
		&cobra.Group{ID: "manage", Title: "Management Commands:"},
		&cobra.Group{ID: "discovery", Title: "Discovery Commands:"},
		&cobra.Group{ID: "auth", Title: "Authentication & Settings:"},
	)
	add := func(group string, cmds ...*cobra.Command) {
		for _, c := range cmds {
			c.GroupID = group
			root.AddCommand(c)
		}
	}
	add("basic", newGetCmd(f), newDescribeCmd(f))
	add("manage", newCreateCmd(f), newApplyCmd(f), newDeleteCmd(f), newActionCmd(f), newGraphQLCmd(f))
	add("discovery", newAPIResourcesCmd(f), newExplainCmd(f))
	add("auth", newLoginCmd(f), newLogoutCmd(f), newWhoamiCmd(f), newConfigCmd(f))
	root.AddCommand(newVersionCmd(f), newSkillCmd(f))
	return root
}

// Execute runs ikctl with the given arguments and returns the process exit code.
func Execute(ctx context.Context, f *Factory, args []string) int {
	root := NewRootCmd(f)
	root.SetArgs(args)
	cmd, err := root.ExecuteContextC(ctx)
	if err == nil {
		return ExitOK
	}
	reportError(f.IO.ErrOut, cmd, err)
	return exitCode(err)
}
