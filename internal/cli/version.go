package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/internal/version"
	"github.com/rustyrien/ikctl/sdk"
)

// VersionResult is the structured output of version.
type VersionResult struct {
	Client version.Info    `json:"client"`
	Server *sdk.ServerInfo `json:"server,omitempty"`
}

func newVersionCmd(f *Factory) *cobra.Command {
	var (
		clientOnly bool
		output     string
	)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the client and server versions",
		Args:  usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := parseOutput(output, printers.FormatTable, printers.FormatJSON, printers.FormatYAML)
			if err != nil {
				return err
			}
			res := VersionResult{Client: version.Get()}
			var serverErr error
			if !clientOnly {
				c, err := f.Client()
				if err == nil {
					res.Server, err = c.Server.Info(cmd.Context())
				}
				serverErr = err
			}
			if out.IsStructured() {
				if err := printers.PrintStructured(f.IO.Out, out.Format, res); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(f.IO.Out, "Client Version: %s\n", res.Client.Version)
				if res.Server != nil {
					fmt.Fprintf(f.IO.Out, "Server Version: %s\n", res.Server.Version)
				}
			}
			if serverErr != nil {
				return fmt.Errorf("get server version: %w", serverErr)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&clientOnly, "client", false, "print the client version only")
	addOutputFlag(cmd, &output, printers.FormatTable, printers.FormatTable, printers.FormatJSON, printers.FormatYAML)
	return cmd
}
