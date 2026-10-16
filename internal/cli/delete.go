package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/registry"
)

// DeleteResult is the structured output of delete.
type DeleteResult struct {
	Kind    string   `json:"kind"`
	Deleted []string `json:"deleted"`
	DryRun  bool     `json:"dryRun,omitempty"`
}

func newDeleteCmd(f *Factory) *cobra.Command {
	var (
		yes    bool
		dryRun bool
		output string
	)
	cmd := &cobra.Command{
		Use:   "delete KIND NAME|ID ...",
		Short: "Delete objects by name or id",
		Long: `Delete objects by name or id.

Deleting asks for confirmation on a terminal. In non-interactive use (scripts,
agents) --yes is required. Resources usually must be destroyed before they can
be deleted (see 'ikctl action resource NAME destroy').`,
		Example: `  ikctl delete resource my-resource
  ikctl delete template 3f1c0b9e-1d2a-4c55-9a51-0c7d7d0e2f11 --yes
  ikctl delete resource my-resource --dry-run`,
		Args:              usageArgs(cobra.MinimumNArgs(2)),
		ValidArgsFunction: completeKinds(registry.VerbDelete),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, err := lookupKind(args[0], registry.VerbDelete)
			if err != nil {
				return err
			}
			out := printers.Output{}
			if output != "" {
				if out, err = parseOutput(output, printers.FormatName, printers.FormatJSON, printers.FormatYAML); err != nil {
					return err
				}
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			ec, err := c.Entities(kind.Name)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			objs := make([]sdk.Object, 0, len(args)-1)
			for _, ref := range args[1:] {
				obj, err := ec.Resolve(ctx, ref, refFields(kind))
				if err != nil {
					return err
				}
				objs = append(objs, obj)
			}
			result := DeleteResult{Kind: kind.Name, Deleted: []string{}, DryRun: dryRun}
			if dryRun {
				for _, obj := range objs {
					result.Deleted = append(result.Deleted, obj.ID())
					if !out.IsStructured() {
						fmt.Fprintf(f.IO.Out, "%s deleted (dry run)\n", objectRef(kind, obj))
					}
				}
				if out.IsStructured() {
					return printers.PrintStructured(f.IO.Out, out.Format, result)
				}
				return nil
			}

			for _, obj := range objs {
				ref := objectRef(kind, obj)
				if err := confirm(f, yes, fmt.Sprintf("delete %s (%s)", ref, obj.ID())); err != nil {
					return err
				}
				if err := ec.Delete(ctx, obj.ID()); err != nil {
					return fmt.Errorf("delete %s: %w", ref, err)
				}
				result.Deleted = append(result.Deleted, obj.ID())
				switch {
				case out.IsStructured():
				case out.Format == printers.FormatName:
					fmt.Fprintln(f.IO.Out, ref)
				default:
					fmt.Fprintf(f.IO.Out, "%s deleted\n", ref)
				}
			}
			if out.IsStructured() {
				return printers.PrintStructured(f.IO.Out, out.Format, result)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "only resolve and print the objects that would be deleted")
	addOutputFlag(cmd, &output, "", printers.FormatName, printers.FormatJSON, printers.FormatYAML)
	return cmd
}
