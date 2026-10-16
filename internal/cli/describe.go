package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/registry"
)

// Description is the structured output of describe.
type Description struct {
	Kind           string     `json:"kind"`
	Item           sdk.Object `json:"item"`
	AllowedActions []string   `json:"allowedActions"`
}

func newDescribeCmd(f *Factory) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "describe KIND NAME|ID",
		Short: "Show details of an object, including the actions currently allowed",
		Long: `Show all details of an object and the lifecycle actions currently allowed for it.

With -o json or -o yaml the output is {"kind": ..., "item": {...}, "allowedActions": [...]}.`,
		Example: `  ikctl describe resource my-resource
  ikctl describe template 3f1c0b9e-1d2a-4c55-9a51-0c7d7d0e2f11 -o json`,
		Args:              usageArgs(cobra.ExactArgs(2)),
		ValidArgsFunction: completeKinds(registry.VerbGet),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, err := lookupKind(args[0], registry.VerbGet)
			if err != nil {
				return err
			}
			out, err := parseOutput(output, printers.FormatTable, printers.FormatJSON, printers.FormatYAML)
			if err != nil {
				return err
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			ec, err := c.Entities(kind.Name)
			if err != nil {
				return err
			}
			obj, err := ec.Resolve(cmd.Context(), args[1])
			if err != nil {
				return err
			}
			d := Description{Kind: kind.Title, Item: obj, AllowedActions: []string{}}
			if kind.HasActionsQuery() {
				if d.AllowedActions, err = ec.Actions(cmd.Context(), obj.ID()); err != nil {
					return err
				}
			}
			if out.IsStructured() {
				return printers.PrintStructured(f.IO.Out, out.Format, d)
			}
			return printDescription(f, kind, d)
		},
	}
	addOutputFlag(cmd, &output, printers.FormatTable, printers.FormatTable, printers.FormatJSON, printers.FormatYAML)
	return cmd
}

func printDescription(f *Factory, kind registry.Kind, d Description) error {
	tw := tabwriter.NewWriter(f.IO.Out, 0, 8, 1, ' ', 0)
	fmt.Fprintf(tw, "Kind:\t%s\n", kind.Title)
	if kind.NamePath != "" {
		fmt.Fprintf(tw, "Name:\t%s\n", d.Item.String(kind.NamePath))
	}
	fmt.Fprintf(tw, "ID:\t%s\n", d.Item.ID())
	if kind.HasActionsQuery() {
		actions := "<none>"
		if len(d.AllowedActions) > 0 {
			actions = strings.Join(d.AllowedActions, ", ")
		}
		fmt.Fprintf(tw, "Allowed Actions:\t%s\n", actions)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Fprintln(f.IO.Out, "Details:")
	var b strings.Builder
	if err := printers.PrintYAML(&b, d.Item); err != nil {
		return err
	}
	for line := range strings.Lines(b.String()) {
		fmt.Fprint(f.IO.Out, "  "+line)
	}
	return nil
}
