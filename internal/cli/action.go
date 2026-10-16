package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/registry"
)

// ActionResult is the structured output of action.
type ActionResult struct {
	Kind   string     `json:"kind"`
	ID     string     `json:"id"`
	Action string     `json:"action"`
	DryRun bool       `json:"dryRun,omitempty"`
	Item   sdk.Object `json:"item,omitempty"`
}

// ActionsList is the structured output of action --list.
type ActionsList struct {
	Kind           string   `json:"kind"`
	ID             string   `json:"id"`
	AllowedActions []string `json:"allowedActions"`
}

func newActionCmd(f *Factory) *cobra.Command {
	var (
		list   bool
		yes    bool
		dryRun bool
		output string
	)
	cmd := &cobra.Command{
		Use:   "action KIND NAME|ID [ACTION]",
		Short: "Run a lifecycle action (approve, execute, destroy, retry, ...) on an object",
		Long: `Run a lifecycle action on an object, or list the actions currently allowed.

Allowed actions depend on the object's state and status, for example:
  approve, reject     a pending resource awaiting approval
  dryrun              plan changes without applying them
  execute             apply (provision) the resource
  destroy             destroy provisioned infrastructure
  retry, recreate     recover from errors
  sync, enable, disable

Without ACTION (or with --list) the allowed actions are printed. The action is
validated against the allowed actions before it is sent. Actions ask for
confirmation on a terminal; non-interactive use requires --yes.`,
		Example: `  # Show allowed actions
  ikctl action resource my-resource

  # Approve and then apply a resource
  ikctl action resource my-resource approve --yes
  ikctl action resource my-resource execute --yes -o json`,
		Args:              usageArgs(cobra.RangeArgs(2, 3)),
		ValidArgsFunction: completeKinds(registry.VerbAction),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, err := lookupKind(args[0], "")
			if err != nil {
				return err
			}
			if !kind.HasActionsQuery() {
				return usageError{&sdk.UnsupportedError{Kind: kind.Name, Verb: "action"}}
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
			obj, err := ec.Resolve(ctx, args[1], refFields(kind))
			if err != nil {
				return err
			}
			ref := objectRef(kind, obj)

			if list || len(args) == 2 {
				actions, err := ec.Actions(ctx, obj.ID())
				if err != nil {
					return err
				}
				if out.IsStructured() {
					return printers.PrintStructured(f.IO.Out, out.Format, ActionsList{Kind: kind.Name, ID: obj.ID(), AllowedActions: actions})
				}
				if len(actions) == 0 {
					fmt.Fprintf(f.IO.ErrOut, "No actions are currently allowed for %s.\n", ref)
				}
				for _, a := range actions {
					fmt.Fprintln(f.IO.Out, a)
				}
				return nil
			}

			action := strings.ToLower(args[2])
			if !kind.Supports(registry.VerbAction) {
				return usageErrorf("%s objects do not support custom actions; use update or delete", kind.Name)
			}
			if dryRun {
				req, err := ec.ActionRequest(obj.ID(), action)
				if err != nil {
					return err
				}
				format := out.Format
				if !out.IsStructured() {
					format = printers.FormatYAML
				}
				return printers.PrintStructured(f.IO.Out, format, DryRunResult{Kind: kind.Name, Operation: action, Request: req})
			}
			if err := confirm(f, yes, fmt.Sprintf("run %q on %s", action, ref)); err != nil {
				return err
			}
			item, err := ec.Action(ctx, obj.ID(), action)
			if err != nil {
				return err
			}
			switch {
			case out.IsStructured():
				return printers.PrintStructured(f.IO.Out, out.Format, ActionResult{Kind: kind.Name, ID: obj.ID(), Action: action, Item: item})
			case out.Format == printers.FormatName:
				fmt.Fprintln(f.IO.Out, ref)
			default:
				var details []string
				for _, field := range []string{"state", "status"} {
					if v := item.String(field); v != "" {
						details = append(details, field+"="+v)
					}
				}
				suffix := ""
				if len(details) > 0 {
					suffix = " (" + strings.Join(details, ", ") + ")"
				}
				fmt.Fprintf(f.IO.Out, "%s: action %q submitted%s\n", ref, action, suffix)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "list the allowed actions")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the GraphQL request instead of sending it")
	addOutputFlag(cmd, &output, "", printers.FormatName, printers.FormatJSON, printers.FormatYAML)
	return cmd
}
