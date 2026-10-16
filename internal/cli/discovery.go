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

// APIResource is one row of api-resources.
type APIResource struct {
	Name       string          `json:"name"`
	Plural     string          `json:"plural"`
	ShortNames []string        `json:"shortNames"`
	Kind       string          `json:"kind"`
	GraphQL    string          `json:"graphqlType"`
	ByName     bool            `json:"lookupByName"`
	NameField  string          `json:"nameFilter,omitempty"`
	Verbs      []registry.Verb `json:"verbs"`
}

func newAPIResourcesCmd(f *Factory) *cobra.Command {
	var output string
	var noHeaders bool
	cmd := &cobra.Command{
		Use:   "api-resources",
		Short: "List the kinds of objects and the verbs they support",
		Example: `  ikctl api-resources
  ikctl api-resources -o json`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(_ *cobra.Command, _ []string) error {
			out, err := parseOutput(output, printers.FormatTable, printers.FormatWide, printers.FormatJSON, printers.FormatYAML, printers.FormatName)
			if err != nil {
				return err
			}
			var rows []APIResource
			for _, k := range registry.All() {
				short := k.Aliases
				if short == nil {
					short = []string{}
				}
				rows = append(rows, APIResource{Name: k.Name, Plural: k.Plural, ShortNames: short, Kind: k.Title,
					GraphQL: k.TypeName, ByName: k.HasNames(), NameField: k.NameFilter, Verbs: k.Verbs})
			}
			switch out.Format {
			case printers.FormatJSON, printers.FormatYAML:
				return printers.PrintStructured(f.IO.Out, out.Format, map[string]any{"kind": "APIResourceList", "items": rows})
			case printers.FormatName:
				for _, r := range rows {
					fmt.Fprintln(f.IO.Out, r.Name)
				}
				return nil
			}
			tw := tabwriter.NewWriter(f.IO.Out, 0, 8, 3, ' ', 0)
			if !noHeaders {
				header := "NAME\tSHORTNAMES\tKIND\tBY NAME\tVERBS"
				if out.Format == printers.FormatWide {
					header += "\tGRAPHQL TYPE\tNAME FILTER"
				}
				fmt.Fprintln(tw, header)
			}
			for _, r := range rows {
				verbs := make([]string, len(r.Verbs))
				for i, v := range r.Verbs {
					verbs[i] = string(v)
				}
				line := fmt.Sprintf("%s\t%s\t%s\t%t\t%s", r.Plural, strings.Join(r.ShortNames, ","), r.Kind, r.ByName, strings.Join(verbs, ","))
				if out.Format == printers.FormatWide {
					line += fmt.Sprintf("\t%s\t%s", r.GraphQL, r.NameField)
				}
				fmt.Fprintln(tw, line)
			}
			return tw.Flush()
		},
	}
	addOutputFlag(cmd, &output, printers.FormatTable, printers.FormatTable, printers.FormatWide, printers.FormatJSON, printers.FormatYAML, printers.FormatName)
	cmd.Flags().BoolVar(&noHeaders, "no-headers", false, "do not print table headers")
	return cmd
}

// Explanation is the structured output of explain.
type Explanation struct {
	Kind        string        `json:"kind"`
	Path        string        `json:"path,omitempty"`
	Type        *sdk.TypeInfo `json:"type"`
	CreateInput *sdk.TypeInfo `json:"createInput,omitempty"`
	UpdateInput *sdk.TypeInfo `json:"updateInput,omitempty"`
}

func newExplainCmd(f *Factory) *cobra.Command {
	var output string
	var input bool
	cmd := &cobra.Command{
		Use:   "explain KIND[.FIELD...]",
		Short: "Describe the fields of a kind, using the server's GraphQL schema",
		Long: `Describe the fields of a kind (or of a nested field) using GraphQL introspection.

Use the field names with 'ikctl get KIND --select "..."' and in filters
(converted to snake_case). With --input, the create and update input types are
shown as well; use them to write files for 'ikctl create' and 'ikctl apply'.`,
		Example: `  ikctl explain resource
  ikctl explain resource.template
  ikctl explain resource --input -o json`,
		Args:              usageArgs(cobra.ExactArgs(1)),
		ValidArgsFunction: completeKinds(""),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := parseOutput(output, printers.FormatTable, printers.FormatJSON, printers.FormatYAML)
			if err != nil {
				return err
			}
			parts := strings.Split(args[0], ".")
			kind, err := lookupKind(parts[0], "")
			if err != nil {
				return err
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			t, err := c.Schema.Type(ctx, kind.TypeName)
			if err != nil {
				return err
			}
			for _, name := range parts[1:] {
				var next string
				for _, fld := range t.Fields {
					if fld.Name == name {
						next = fld.Type.Named()
					}
				}
				if next == "" {
					return &sdk.APIError{Code: sdk.CodeNotFound, Message: fmt.Sprintf("field %q not found in %s", name, t.Name)}
				}
				if t, err = c.Schema.Type(ctx, next); err != nil {
					return err
				}
			}
			ex := Explanation{Kind: kind.Title, Path: strings.Join(parts[1:], "."), Type: t}
			if input && len(parts) == 1 {
				if kind.Supports(registry.VerbCreate) {
					if ex.CreateInput, err = c.Schema.Type(ctx, kind.CreateInputType()); err != nil {
						return err
					}
				}
				if kind.Supports(registry.VerbUpdate) {
					if ex.UpdateInput, err = c.Schema.Type(ctx, kind.UpdateInputType()); err != nil {
						return err
					}
				}
			}
			if out.IsStructured() {
				return printers.PrintStructured(f.IO.Out, out.Format, ex)
			}
			return printExplanation(f, ex)
		},
	}
	addOutputFlag(cmd, &output, printers.FormatTable, printers.FormatTable, printers.FormatJSON, printers.FormatYAML)
	cmd.Flags().BoolVar(&input, "input", false, "also show the create/update input fields")
	return cmd
}

func printExplanation(f *Factory, ex Explanation) error {
	w := f.IO.Out
	fmt.Fprintf(w, "KIND:  %s\n", ex.Kind)
	if ex.Path != "" {
		fmt.Fprintf(w, "FIELD: %s\n", ex.Path)
	}
	fmt.Fprintf(w, "TYPE:  %s (%s)\n", ex.Type.Name, strings.ToLower(ex.Type.Kind))
	if ex.Type.Description != "" {
		fmt.Fprintf(w, "\n%s\n", ex.Type.Description)
	}
	if len(ex.Type.EnumValues) > 0 {
		fmt.Fprintf(w, "\nVALUES: %s\n", strings.Join(ex.Type.EnumValues, ", "))
	}
	printFields := func(title string, fields []sdk.FieldInfo) error {
		if len(fields) == 0 {
			return nil
		}
		fmt.Fprintf(w, "\n%s:\n", title)
		tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
		for _, fld := range fields {
			line := fmt.Sprintf("  %s\t%s", fld.Name, fld.TypeName)
			if fld.Description != "" {
				line += "\t" + fld.Description
			}
			fmt.Fprintln(tw, line)
		}
		return tw.Flush()
	}
	if err := printFields("FIELDS", ex.Type.Fields); err != nil {
		return err
	}
	if err := printFields("INPUT FIELDS", ex.Type.InputFields); err != nil {
		return err
	}
	if ex.CreateInput != nil {
		if err := printFields("CREATE INPUT ("+ex.CreateInput.Name+", '!' = required)", ex.CreateInput.InputFields); err != nil {
			return err
		}
	}
	if ex.UpdateInput != nil {
		if err := printFields("UPDATE INPUT ("+ex.UpdateInput.Name+")", ex.UpdateInput.InputFields); err != nil {
			return err
		}
	}
	return nil
}
