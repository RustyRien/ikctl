package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/registry"
)

type getOptions struct {
	output         string
	noHeaders      bool
	labels         []string
	fieldSelectors []string
	filter         string
	nameLike       string
	sortBy         string
	limit          int
	offset         int
	all            bool
	selectFields   string
}

// listOperators are filter operators whose value is a list (comma-separated on the command line).
var listOperators = []string{"__in", "__any", "__contains_all"}

func newGetCmd(f *Factory) *cobra.Command {
	o := &getOptions{}
	cmd := &cobra.Command{
		Use:   "get KIND [NAME|ID ...]",
		Short: "List objects or get specific objects by name or id",
		Long: `List objects of a kind, or get specific objects by name or id.

Without names, objects are listed page by page (default --limit 100). Use
--all to fetch every page. Filters are combined with AND:

  -l, --selector       objects having all given labels (comma-separated)
  --name-like          case-insensitive substring match on the name
  --field-selector     field conditions (repeatable), e.g.
                         status=DONE                  equality
                         state!=DESTROYED             not equal
                         name__like=redis             substring
                         status__in=ERROR,CANCELLED   any of the values
                         template__name__like=eks     relation field
  --filter             raw JSON filter, e.g. '{"or":[{"status":"ERROR"},{"state":"DESTROY"}]}'

Field names in filters are snake_case. With -o json or -o yaml a list is
printed as {"kind": "<Kind>List", "items": [...], "total": N, ...} and a single
object is printed as-is.`,
		Example: `  # List resources
  ikctl get resources

  # Resources whose name contains "redis", as JSON
  ikctl get rs --name-like redis -o json

  # Failed resources created from an EKS template, newest first
  ikctl get resources --field-selector status=ERROR --field-selector template__name__like=eks --sort-by createdAt:desc

  # One resource by name or id, as YAML
  ikctl get resource my-resource -o yaml

  # Only selected GraphQL fields
  ikctl get resources --select "id name outputs" -o json`,
		Args:              usageArgs(cobra.MinimumNArgs(1)),
		ValidArgsFunction: completeKinds(registry.VerbGet),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd, f, o, args)
		},
	}
	fl := cmd.Flags()
	addOutputFlag(cmd, &o.output, printers.FormatTable, printers.Formats...)
	fl.BoolVar(&o.noHeaders, "no-headers", false, "do not print table headers")
	fl.StringSliceVarP(&o.labels, "selector", "l", nil, "filter by labels (objects must have all of them)")
	fl.StringArrayVar(&o.fieldSelectors, "field-selector", nil, "filter by field: key=value, key!=value, key__<op>=value (repeatable)")
	fl.StringVar(&o.filter, "filter", "", "raw JSON filter object")
	fl.StringVar(&o.nameLike, "name-like", "", "case-insensitive substring match on the name")
	fl.StringVar(&o.sortBy, "sort-by", "", "sort by field, optionally with :asc or :desc (e.g. createdAt:desc)")
	fl.IntVar(&o.limit, "limit", sdk.DefaultLimit, "maximum number of objects per page")
	fl.IntVar(&o.offset, "offset", 0, "number of objects to skip")
	fl.BoolVar(&o.all, "all", false, "fetch all pages")
	fl.StringVar(&o.selectFields, "select", "", "GraphQL field selection to fetch instead of the defaults (see 'ikctl explain KIND')")
	return cmd
}

func runGet(cmd *cobra.Command, f *Factory, o *getOptions, args []string) error {
	kind, err := lookupKind(args[0], registry.VerbGet)
	if err != nil {
		return err
	}
	out, err := parseOutput(o.output)
	if err != nil {
		return err
	}
	if o.limit <= 0 || o.offset < 0 {
		return usageErrorf("--limit must be positive and --offset must not be negative")
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

	fields := o.selectFields
	if fields == "" && out.Format != printers.FormatTable && out.Format != printers.FormatName {
		fields = kind.DescribeFields
	}

	// Specific objects.
	if refs := args[1:]; len(refs) > 0 {
		items := make([]sdk.Object, 0, len(refs))
		for _, ref := range refs {
			obj, err := ec.Resolve(ctx, ref, fields)
			if err != nil {
				return err
			}
			items = append(items, obj)
		}
		if out.IsStructured() && len(items) == 1 {
			return printers.PrintStructured(f.IO.Out, out.Format, items[0])
		}
		return printList(f, out, o.noHeaders, kind, &sdk.List{Kind: kind.ListTitle(), Items: items, Total: len(items), Limit: len(items)})
	}

	if !kind.Supports(registry.VerbList) {
		return usageErrorf("%s objects cannot be listed; pass an id", kind.Name)
	}
	opts, err := o.listOptions(kind)
	if err != nil {
		return err
	}
	opts.Fields = fields

	if o.all {
		list := &sdk.List{Kind: kind.ListTitle(), Items: []sdk.Object{}, Offset: o.offset}
		for item, err := range ec.ListAll(ctx, opts) {
			if err != nil {
				return err
			}
			list.Items = append(list.Items, item)
		}
		list.Total = o.offset + len(list.Items)
		list.Limit = len(list.Items)
		return printList(f, out, o.noHeaders, kind, list)
	}

	list, err := ec.List(ctx, opts)
	if err != nil {
		return err
	}
	if err := printList(f, out, o.noHeaders, kind, list); err != nil {
		return err
	}
	if !out.IsStructured() && out.Format != printers.FormatName {
		if len(list.Items) == 0 {
			fmt.Fprintf(f.IO.ErrOut, "No %s found.\n", kind.Plural)
		} else if shown := list.Offset + len(list.Items); shown < list.Total {
			fmt.Fprintf(f.IO.ErrOut, "Showing %d-%d of %d %s (use --offset/--limit or --all for more).\n",
				list.Offset+1, shown, list.Total, kind.Plural)
		}
	}
	return nil
}

func (o *getOptions) listOptions(kind registry.Kind) (sdk.ListOptions, error) {
	filter := sdk.Filter{}
	if o.filter != "" {
		v, ok := parseValue(o.filter).(map[string]any)
		if !ok {
			return sdk.ListOptions{}, usageErrorf("--filter must be a JSON object")
		}
		filter = filter.And(v)
	}
	if len(o.labels) > 0 {
		filter = filter.And(sdk.Labels(o.labels...))
	}
	if o.nameLike != "" {
		if !kind.HasNames() {
			return sdk.ListOptions{}, usageErrorf("%s objects have no name; use --field-selector", kind.Name)
		}
		filter = filter.And(sdk.Like(kind.NameFilter, o.nameLike))
	}
	for _, sel := range o.fieldSelectors {
		fs, err := parseFieldSelector(sel)
		if err != nil {
			return sdk.ListOptions{}, err
		}
		filter = filter.And(fs)
	}

	opts := sdk.ListOptions{Filter: filter, Offset: o.offset, Limit: o.limit}
	if o.sortBy != "" {
		field, dir, _ := strings.Cut(o.sortBy, ":")
		switch strings.ToLower(dir) {
		case "", "asc":
			opts.Sort = &sdk.Sort{Field: field}
		case "desc":
			opts.Sort = &sdk.Sort{Field: field, Desc: true}
		default:
			return sdk.ListOptions{}, usageErrorf("invalid --sort-by %q: direction must be asc or desc", o.sortBy)
		}
	}
	return opts, nil
}

// parseFieldSelector parses "key=value", "key!=value" or "key__op=value".
func parseFieldSelector(s string) (sdk.Filter, error) {
	if k, v, ok := strings.Cut(s, "!="); ok && k != "" {
		return sdk.NotEq(strings.TrimSpace(k), parseValue(v)), nil
	}
	k, v, ok := strings.Cut(s, "=")
	k = strings.TrimSpace(k)
	if !ok || k == "" {
		return nil, usageErrorf("invalid --field-selector %q: expected key=value", s)
	}
	for _, op := range listOperators {
		if strings.HasSuffix(k, op) {
			if parsed, isList := parseValue(v).([]any); isList {
				return sdk.Filter{k: parsed}, nil
			}
			parts := strings.Split(v, ",")
			vals := make([]any, len(parts))
			for i, p := range parts {
				vals[i] = strings.TrimSpace(p)
			}
			return sdk.Filter{k: vals}, nil
		}
	}
	if strings.HasSuffix(k, "__like") || strings.HasSuffix(k, "__not_like") {
		return sdk.Filter{k: v}, nil
	}
	return sdk.Filter{k: parseValue(v)}, nil
}

func printList(f *Factory, out printers.Output, noHeaders bool, kind registry.Kind, list *sdk.List) error {
	switch out.Format {
	case printers.FormatJSON, printers.FormatYAML:
		return printers.PrintStructured(f.IO.Out, out.Format, list)
	case printers.FormatName:
		return printers.PrintNames(f.IO.Out, kind, list.Items)
	case printers.FormatCustomColumns:
		return printers.PrintTable(f.IO.Out, out.Columns, list.Items, printers.TableOptions{NoHeaders: noHeaders})
	default:
		if len(list.Items) == 0 {
			return nil
		}
		return printers.PrintTable(f.IO.Out, kind.Columns, list.Items,
			printers.TableOptions{NoHeaders: noHeaders, Wide: out.Format == printers.FormatWide})
	}
}
