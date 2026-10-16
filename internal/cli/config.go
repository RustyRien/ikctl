package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/config"
)

func newConfigCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage contexts in the config file",
		Long: `Manage the contexts saved in the config file
($IKCTL_CONFIG, $XDG_CONFIG_HOME/ikctl/config.yaml or ~/.config/ikctl/config.yaml).

Use 'ikctl login' to add a context.`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		newConfigViewCmd(f),
		newGetContextsCmd(f),
		newCurrentContextCmd(f),
		newUseContextCmd(f),
		newDeleteContextCmd(f),
	)
	return cmd
}

func loadConfig(f *Factory) (string, *config.Config, error) {
	path, err := f.ConfigPath()
	if err != nil {
		return "", nil, err
	}
	cfg, err := config.Load(path)
	return path, cfg, err
}

func redacted(cfg *config.Config) *config.Config {
	out := &config.Config{CurrentContext: cfg.CurrentContext, Contexts: make([]config.Context, len(cfg.Contexts))}
	for i, c := range cfg.Contexts {
		c.Token = sdk.RedactToken(c.Token)
		out.Contexts[i] = c
	}
	return out
}

func newConfigViewCmd(f *Factory) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Show the config file with tokens redacted",
		Args:  usageArgs(cobra.NoArgs),
		RunE: func(_ *cobra.Command, _ []string) error {
			out, err := parseOutput(output, printers.FormatJSON, printers.FormatYAML)
			if err != nil {
				return err
			}
			_, cfg, err := loadConfig(f)
			if err != nil {
				return err
			}
			return printers.PrintStructured(f.IO.Out, out.Format, redacted(cfg))
		},
	}
	addOutputFlag(cmd, &output, printers.FormatYAML, printers.FormatJSON, printers.FormatYAML)
	return cmd
}

func newGetContextsCmd(f *Factory) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "get-contexts",
		Short: "List the saved contexts",
		Args:  usageArgs(cobra.NoArgs),
		RunE: func(_ *cobra.Command, _ []string) error {
			out, err := parseOutput(output, printers.FormatTable, printers.FormatJSON, printers.FormatYAML, printers.FormatName)
			if err != nil {
				return err
			}
			_, cfg, err := loadConfig(f)
			if err != nil {
				return err
			}
			switch out.Format {
			case printers.FormatJSON, printers.FormatYAML:
				return printers.PrintStructured(f.IO.Out, out.Format, redacted(cfg))
			case printers.FormatName:
				for _, c := range cfg.Contexts {
					fmt.Fprintln(f.IO.Out, c.Name)
				}
				return nil
			}
			tw := tabwriter.NewWriter(f.IO.Out, 0, 8, 3, ' ', 0)
			fmt.Fprintln(tw, "CURRENT\tNAME\tSERVER\tUSER\tLOGGED IN")
			for _, c := range cfg.Contexts {
				cur := ""
				if c.Name == cfg.CurrentContext {
					cur = "*"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%t\n", cur, c.Name, c.Server, c.User, c.Token != "")
			}
			return tw.Flush()
		},
	}
	addOutputFlag(cmd, &output, printers.FormatTable, printers.FormatTable, printers.FormatJSON, printers.FormatYAML, printers.FormatName)
	return cmd
}

func newCurrentContextCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "current-context",
		Short: "Print the current context",
		Args:  usageArgs(cobra.NoArgs),
		RunE: func(_ *cobra.Command, _ []string) error {
			_, cfg, err := loadConfig(f)
			if err != nil {
				return err
			}
			if cfg.CurrentContext == "" {
				return &sdk.APIError{Code: sdk.CodeNotFound, Message: "current context is not set"}
			}
			fmt.Fprintln(f.IO.Out, cfg.CurrentContext)
			return nil
		},
	}
}

func completeContexts(f *Factory) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		_, cfg, err := loadConfig(f)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []cobra.Completion
		for _, c := range cfg.Contexts {
			names = append(names, c.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func newUseContextCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:               "use-context NAME",
		Short:             "Set the current context",
		Args:              usageArgs(cobra.ExactArgs(1)),
		ValidArgsFunction: completeContexts(f),
		RunE: func(_ *cobra.Command, args []string) error {
			path, cfg, err := loadConfig(f)
			if err != nil {
				return err
			}
			if _, ok := cfg.Get(args[0]); !ok {
				return &sdk.APIError{Code: sdk.CodeNotFound, Message: fmt.Sprintf("context %q not found", args[0])}
			}
			cfg.CurrentContext = args[0]
			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "Switched to context %q.\n", args[0])
			return nil
		},
	}
}

func newDeleteContextCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:               "delete-context NAME",
		Short:             "Delete a context and its token from the config file",
		Args:              usageArgs(cobra.ExactArgs(1)),
		ValidArgsFunction: completeContexts(f),
		RunE: func(_ *cobra.Command, args []string) error {
			path, cfg, err := loadConfig(f)
			if err != nil {
				return err
			}
			if !cfg.Delete(args[0]) {
				return &sdk.APIError{Code: sdk.CodeNotFound, Message: fmt.Sprintf("context %q not found", args[0])}
			}
			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "Deleted context %q.\n", args[0])
			return nil
		},
	}
}
