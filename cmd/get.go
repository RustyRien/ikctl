package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/auth"
	"github.com/electrolux-oss/ik-tui/internal/config"
	"github.com/electrolux-oss/ik-tui/internal/printer"
	"github.com/electrolux-oss/ik-tui/internal/resource"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
	"github.com/spf13/cobra"
)

type getOptions struct {
	output    string
	sort      string
	sortOrder string
	limit     int
	state     string
	status    string
	label     string
	provider  string
	typeName  string
	name      string
	filters   []string
}

func getCmd() *cobra.Command {
	options := getOptions{output: "table", sortOrder: "desc", limit: 100}
	cmd := &cobra.Command{
		Use:   "get <entity> [name-or-id]",
		Short: "Get InfraKitchen resources",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd, args, options)
		},
	}
	cmd.Flags().StringVarP(&options.output, "output", "o", options.output, "Output format: table, wide, json, yaml, name")
	cmd.Flags().StringVar(&options.sort, "sort", "", "Sort by column name")
	cmd.Flags().StringVar(&options.sortOrder, "sort-order", options.sortOrder, "Sort order: asc or desc")
	cmd.Flags().IntVar(&options.limit, "limit", options.limit, "Maximum rows to fetch")
	cmd.Flags().StringVar(&options.state, "state", "", "Filter resources by state")
	cmd.Flags().StringVar(&options.status, "status", "", "Filter resources by status")
	cmd.Flags().StringVar(&options.label, "label", "", "Filter resources by label")
	cmd.Flags().StringVar(&options.provider, "provider", "", "Filter integrations by provider")
	cmd.Flags().StringVar(&options.typeName, "type", "", "Filter integrations by type")
	cmd.Flags().StringVar(&options.name, "name", "", "Filter entities by name")
	cmd.Flags().StringArrayVar(&options.filters, "filter", nil, "Additional filter as key=value")
	return cmd
}

func runGet(cmd *cobra.Command, args []string, options getOptions) error {
	flags.HasInsecureFlag = cmd.Flags().Changed("insecure-skip-tls-verify")
	flags.HasNoColorsFlag = cmd.Flags().Changed("no-colors")

	cfg, err := config.Load(flags)
	if err != nil {
		return err
	}
	persistConfig(cmd, &cfg)
	cli, err := auth.NewClient(cfg)
	if err != nil {
		return err
	}

	registry := resource.DefaultRegistry(cli)
	descriptor, ok := registry.Resolve(args[0])
	if !ok {
		return fmt.Errorf("unknown entity %q (valid: %s)", args[0], strings.Join(registry.Names(), ", "))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if len(args) == 2 {
		return printSingle(ctx, descriptor, args[1], options.output)
	}

	filter, err := buildFilters(descriptor, options)
	if err != nil {
		return err
	}
	sortBy, err := buildSort(descriptor, options)
	if err != nil {
		return err
	}
	rows, raw, _, err := descriptor.List(ctx, filter, sortBy, []int{0, options.limit})
	if err != nil {
		return err
	}
	return printRows(descriptor, options.output, rows, raw)
}

func printSingle(ctx context.Context, descriptor *resource.Descriptor, nameOrID string, output string) error {
	row, raw, err := descriptor.GetByID(ctx, nameOrID)
	if err != nil {
		row, raw, err = descriptor.ResolveByName(ctx, nameOrID)
		if err != nil {
			return err
		}
	}
	return printRows(descriptor, output, []tabledata.Row{row}, []any{raw})
}

func printRows(descriptor *resource.Descriptor, output string, rows []tabledata.Row, raw []any) error {
	headers := descriptor.Headers
	if strings.EqualFold(output, "wide") {
		wideRows := make([]tabledata.Row, 0, len(raw))
		for _, item := range raw {
			wideRows = append(wideRows, descriptor.WideRow(item))
		}
		rows = wideRows
		headers = descriptor.WideHeaders
	}
	return printer.Print(os.Stdout, output, headers, rows, raw)
}

func buildSort(descriptor *resource.Descriptor, options getOptions) ([]string, error) {
	if options.sort == "" {
		return descriptor.DefaultSort, nil
	}
	field, ok := resource.ResolveSortField(descriptor, options.sort)
	if !ok {
		return nil, fmt.Errorf("unsupported sort %q for %s", options.sort, descriptor.Name)
	}
	direction := strings.ToUpper(options.sortOrder)
	if direction != "ASC" && direction != "DESC" {
		return nil, fmt.Errorf("unsupported sort order %q", options.sortOrder)
	}
	return []string{field, direction}, nil
}

func buildFilters(descriptor *resource.Descriptor, options getOptions) (map[string]any, error) {
	filters := map[string]string{}
	if options.state != "" {
		filters["state"] = options.state
	}
	if options.status != "" {
		filters["status"] = options.status
	}
	if options.label != "" {
		filters["label"] = options.label
	}
	if options.provider != "" {
		filters["provider"] = options.provider
	}
	if options.typeName != "" {
		filters["type"] = options.typeName
	}
	if options.name != "" {
		filters["name"] = options.name
	}
	for _, item := range options.filters {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("invalid filter %q, expected key=value", item)
		}
		filters[key] = value
	}
	return resource.ResolveFilter(descriptor, filters)
}
