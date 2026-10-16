package cli

import (
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
)

func newGraphQLCmd(f *Factory) *cobra.Command {
	var (
		query   string
		file    string
		vars    []string
		varsRaw string
		output  string
	)
	cmd := &cobra.Command{
		Use:   "graphql [QUERY]",
		Short: "Run a raw GraphQL query or mutation",
		Long: `Run a raw GraphQL query or mutation against /api/graphql and print the "data" field.

Use it for anything the other commands do not cover. Discover operations with
'ikctl explain' or by introspection, e.g. '{ __schema { queryType { fields { name } } } }'.
Variables given with --var are parsed as JSON when possible, otherwise as strings.`,
		Example: `  ikctl graphql '{ serverInfo { version } }'
  ikctl graphql -q 'query($id: UUID!) { resource(id: $id) { name outputs } }' --var id=1063a79d-c17e-41c3-b576-52e21dbb7239
  ikctl graphql -f query.graphql --vars '{"filter": {"status": "ERROR"}}' -o yaml`,
		Args: usageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := parseOutput(output, printers.FormatJSON, printers.FormatYAML)
			if err != nil {
				return err
			}
			sources := 0
			if len(args) == 1 {
				query = args[0]
				sources++
			}
			if cmd.Flags().Changed("query") {
				sources++
			}
			if file != "" {
				sources++
				var data []byte
				if file == "-" {
					data, err = io.ReadAll(f.IO.In)
				} else {
					data, err = os.ReadFile(file) //nolint:gosec // reading user-specified files is the purpose
				}
				if err != nil {
					return err
				}
				query = string(data)
			}
			if sources != 1 || query == "" {
				return usageErrorf("provide exactly one query: as an argument, with --query or with --file")
			}

			variables := map[string]any{}
			if varsRaw != "" {
				if err := json.Unmarshal([]byte(varsRaw), &variables); err != nil {
					return usageErrorf("--vars must be a JSON object: %v", err)
				}
			}
			kv, err := parseKeyValues(vars, "var")
			if err != nil {
				return err
			}
			for k, v := range kv {
				variables[k] = v
			}

			c, err := f.Client()
			if err != nil {
				return err
			}
			var data json.RawMessage
			if err := c.GraphQL(cmd.Context(), query, variables, &data); err != nil {
				return err
			}
			var v any
			if err := json.Unmarshal(data, &v); err != nil {
				return err
			}
			return printers.PrintStructured(f.IO.Out, out.Format, v)
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "GraphQL document")
	cmd.Flags().StringVarP(&file, "file", "f", "", "read the GraphQL document from a file (- for stdin)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable as key=value (repeatable)")
	cmd.Flags().StringVar(&varsRaw, "vars", "", "variables as a JSON object")
	addOutputFlag(cmd, &output, printers.FormatJSON, printers.FormatJSON, printers.FormatYAML)
	return cmd
}
