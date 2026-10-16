package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/registry"
)

// Manifest is an object definition read from a file.
//
//	kind: resource
//	id: <optional, apply only: update this object>
//	spec:
//	  name: my-resource
//	  template_id: ...
type Manifest struct {
	Kind string         `json:"kind"`
	ID   string         `json:"id,omitempty"`
	Spec map[string]any `json:"spec"`
}

// MutationResult is the structured output of create/apply.
type MutationResult struct {
	Kind      string     `json:"kind"`
	Operation string     `json:"operation"` // created, configured
	Item      sdk.Object `json:"item,omitempty"`
}

// DryRunResult is printed by --dry-run.
type DryRunResult struct {
	Kind      string      `json:"kind"`
	Operation string      `json:"operation"`
	Request   sdk.Request `json:"request"`
}

type applyOptions struct {
	files  []string
	output string
	dryRun bool
}

const manifestHelp = `Files contain one or more YAML or JSON documents (separated by "---"):

  kind: resource          # any kind from 'ikctl api-resources'
  id: <uuid>              # optional; apply updates this object
  spec:                   # the GraphQL create/update input
    name: my-resource
    description: ""
    template_id: <uuid>   # top-level keys may be snake_case or camelCase
    variables: [...]      # nested values are sent unchanged

Run 'ikctl explain KIND --input' to see the input fields.`

func newCreateCmd(f *Factory) *cobra.Command {
	o := &applyOptions{}
	cmd := &cobra.Command{
		Use:   "create -f FILE",
		Short: "Create objects from YAML or JSON files",
		Long:  "Create objects from YAML or JSON files.\n\n" + manifestHelp,
		Example: `  # Preview the GraphQL request without sending it
  ikctl create -f resource.yaml --dry-run

  # Create and print the new object as JSON
  ikctl create -f resource.yaml -o json

  # Read from stdin
  cat resource.json | ikctl create -f -`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error { return runApply(cmd.Context(), f, o, false) },
	}
	addApplyFlags(cmd, o)
	return cmd
}

func newApplyCmd(f *Factory) *cobra.Command {
	o := &applyOptions{}
	cmd := &cobra.Command{
		Use:   "apply -f FILE",
		Short: "Create or update objects from YAML or JSON files",
		Long: `Create or update objects from YAML or JSON files.

An object is updated when the document has an "id", or when an object with the
same spec.name already exists; otherwise it is created.

` + manifestHelp,
		Example: `  ikctl apply -f project.yaml
  ikctl apply -f project.yaml --dry-run -o json`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error { return runApply(cmd.Context(), f, o, true) },
	}
	addApplyFlags(cmd, o)
	return cmd
}

func addApplyFlags(cmd *cobra.Command, o *applyOptions) {
	cmd.Flags().StringArrayVarP(&o.files, "filename", "f", nil, "file to read, or - for stdin (repeatable)")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "print the GraphQL requests instead of sending them")
	addOutputFlag(cmd, &o.output, "", printers.FormatName, printers.FormatJSON, printers.FormatYAML)
	_ = cmd.MarkFlagRequired("filename")
}

func runApply(ctx context.Context, f *Factory, o *applyOptions, upsert bool) error {
	out := printers.Output{}
	if o.output != "" {
		var err error
		if out, err = parseOutput(o.output, printers.FormatName, printers.FormatJSON, printers.FormatYAML); err != nil {
			return err
		}
	}
	var manifests []Manifest
	for _, file := range o.files {
		ms, err := readManifests(f, file)
		if err != nil {
			return err
		}
		manifests = append(manifests, ms...)
	}
	if len(manifests) == 0 {
		return usageErrorf("no objects found in the given files")
	}

	for i, m := range manifests {
		verb := registry.VerbCreate
		if upsert && m.ID != "" {
			verb = registry.VerbUpdate
		}
		if _, err := lookupKind(m.Kind, verb); err != nil {
			return fmt.Errorf("document %d: %w", i+1, err)
		}
		if m.Spec == nil {
			return usageErrorf("document %d: missing spec", i+1)
		}
		if m.ID != "" && !upsert {
			return usageErrorf("document %d: 'id' is only allowed with apply", i+1)
		}
	}

	c, err := f.Client()
	if err != nil {
		return err
	}
	var results []any
	for _, m := range manifests {
		ec, err := c.Entities(m.Kind)
		if err != nil {
			return err
		}
		kind := ec.Kind()
		id := m.ID
		if upsert && id == "" && kind.HasNames() {
			if name, ok := m.Spec["name"].(string); ok && name != "" {
				obj, err := ec.GetByName(ctx, name, "id")
				switch {
				case err == nil:
					id = obj.ID()
				case !sdk.IsNotFound(err):
					return err
				}
			}
		}

		var req sdk.Request
		op := "created"
		if id != "" {
			op = "configured"
			req, err = ec.UpdateRequest(id, m.Spec)
		} else {
			req, err = ec.CreateRequest(m.Spec)
		}
		if err != nil {
			return err
		}
		if o.dryRun {
			dryOp := "create"
			if id != "" {
				dryOp = "update"
			}
			results = append(results, DryRunResult{Kind: kind.Name, Operation: dryOp, Request: req})
			continue
		}

		var item sdk.Object
		if id != "" {
			item, err = ec.Update(ctx, id, m.Spec)
		} else {
			item, err = ec.Create(ctx, m.Spec)
		}
		if err != nil {
			return fmt.Errorf("%s %s: %w", kind.Name, op, err)
		}
		switch {
		case out.IsStructured():
			results = append(results, MutationResult{Kind: kind.Name, Operation: op, Item: item})
		case out.Format == printers.FormatName:
			fmt.Fprintln(f.IO.Out, objectRef(kind, item))
		default:
			fmt.Fprintf(f.IO.Out, "%s %s\n", objectRef(kind, item), op)
		}
	}

	if o.dryRun {
		format := out.Format
		if format == "" || format == printers.FormatName {
			format = printers.FormatYAML
		}
		return printResults(f, format, results)
	}
	if out.IsStructured() {
		return printResults(f, out.Format, results)
	}
	return nil
}

func printResults(f *Factory, format string, results []any) error {
	if len(results) == 1 {
		return printers.PrintStructured(f.IO.Out, format, results[0])
	}
	return printers.PrintStructured(f.IO.Out, format, map[string]any{"kind": "List", "items": results})
}

var docSeparator = regexp.MustCompile(`(?m)^---\s*$`)

func readManifests(f *Factory, file string) ([]Manifest, error) {
	var data []byte
	var err error
	if file == "-" {
		data, err = io.ReadAll(f.IO.In)
	} else {
		data, err = os.ReadFile(file) //nolint:gosec // reading user-specified files is the purpose
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	var out []Manifest
	for i, doc := range docSeparator.Split(string(data), -1) {
		if len(bytes.TrimSpace([]byte(doc))) == 0 {
			continue
		}
		var m Manifest
		if err := yaml.UnmarshalStrict([]byte(doc), &m); err != nil {
			return nil, usageErrorf("%s: document %d: %v", file, i+1, err)
		}
		if m.Kind == "" {
			return nil, usageErrorf("%s: document %d: missing kind", file, i+1)
		}
		out = append(out, m)
	}
	return out, nil
}
