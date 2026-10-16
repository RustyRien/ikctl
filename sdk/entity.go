package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/rustyrien/ikctl/sdk/registry"
)

// DefaultLimit is the page size used when ListOptions.Limit is not set.
const DefaultLimit = 100

// Object is an API object decoded as JSON. Numbers are json.Number.
type Object map[string]any

// ID returns the object's id.
func (o Object) ID() string { s, _ := o["id"].(string); return s }

// String returns the string value at a dot-separated path, or "".
func (o Object) String(path string) string {
	v := Lookup(o, path)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// Lookup returns the value at a dot-separated path inside nested maps, or nil.
func Lookup(v any, path string) any {
	if path == "" {
		return v
	}
	for part := range strings.SplitSeq(path, ".") {
		switch m := v.(type) {
		case Object:
			v = m[part]
		case map[string]any:
			v = m[part]
		default:
			return nil
		}
	}
	return v
}

// List is a page of objects.
type List struct {
	Kind   string   `json:"kind"`
	Items  []Object `json:"items"`
	Total  int      `json:"total"`
	Offset int      `json:"offset"`
	Limit  int      `json:"limit"`
}

// Sort orders a list by one field (camelCase or snake_case; dot paths allowed for relations).
type Sort struct {
	Field string
	Desc  bool
}

// ListOptions controls List.
type ListOptions struct {
	Filter Filter
	Sort   *Sort
	Offset int
	// Limit is the page size; DefaultLimit if <= 0.
	Limit int
	// Fields overrides the GraphQL selection, e.g. "id name variables template { name }".
	Fields string
}

// EntityClient performs operations on one kind of entity.
type EntityClient struct {
	c    *Client
	kind registry.Kind
}

// Kind returns the kind definition.
func (e *EntityClient) Kind() registry.Kind { return e.kind }

func (e *EntityClient) require(v registry.Verb) error {
	if !e.kind.Supports(v) {
		return &UnsupportedError{Kind: e.kind.Name, Verb: string(v)}
	}
	return nil
}

// ListRequest builds the GraphQL request used by List.
func (e *EntityClient) ListRequest(opts ListOptions) (Request, error) {
	if err := e.require(registry.VerbList); err != nil {
		return Request{}, err
	}
	if opts.Offset < 0 {
		return Request{}, fmt.Errorf("offset must not be negative")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	fields := opts.Fields
	if fields == "" {
		fields = e.kind.ListFields
	}
	k := e.kind
	q := fmt.Sprintf("query List%[1]s($filter: JSON, $sort: [String!], $range: [Int!]) {\n"+
		"  items: %[2]s(filter: $filter, sort: $sort, range: $range) { %[3]s }\n"+
		"  total: %[4]s(filter: $filter)\n}", k.Title, k.ListOp(), fields, k.CountOp())
	vars := map[string]any{"range": []int{opts.Offset, opts.Offset + limit}}
	if len(opts.Filter) > 0 {
		vars["filter"] = opts.Filter
	}
	if opts.Sort != nil && opts.Sort.Field != "" {
		dir := "ASC"
		if opts.Sort.Desc {
			dir = "DESC"
		}
		vars["sort"] = []string{opts.Sort.Field, dir}
	}
	return Request{Query: q, Variables: vars}, nil
}

// List returns one page of objects together with the total count.
func (e *EntityClient) List(ctx context.Context, opts ListOptions) (*List, error) {
	req, err := e.ListRequest(opts)
	if err != nil {
		return nil, err
	}
	var data struct {
		Items []Object `json:"items"`
		Total int      `json:"total"`
	}
	if err := e.c.Do(ctx, req, &data); err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if data.Items == nil {
		data.Items = []Object{}
	}
	return &List{Kind: e.kind.ListTitle(), Items: data.Items, Total: data.Total, Offset: opts.Offset, Limit: limit}, nil
}

// ListAll iterates over all objects matching opts, fetching pages of opts.Limit.
func (e *EntityClient) ListAll(ctx context.Context, opts ListOptions) iter.Seq2[Object, error] {
	return func(yield func(Object, error) bool) {
		if opts.Limit <= 0 {
			opts.Limit = DefaultLimit
		}
		for {
			page, err := e.List(ctx, opts)
			if err != nil {
				yield(nil, err)
				return
			}
			for _, item := range page.Items {
				if !yield(item, nil) {
					return
				}
			}
			opts.Offset += len(page.Items)
			if len(page.Items) < opts.Limit || opts.Offset >= page.Total {
				return
			}
		}
	}
}

// Count returns the number of objects matching filter.
func (e *EntityClient) Count(ctx context.Context, filter Filter) (int, error) {
	if err := e.require(registry.VerbList); err != nil {
		return 0, err
	}
	q := fmt.Sprintf("query Count%s($filter: JSON) { total: %s(filter: $filter) }", e.kind.Title, e.kind.CountOp())
	vars := map[string]any{}
	if len(filter) > 0 {
		vars["filter"] = filter
	}
	var data struct {
		Total int `json:"total"`
	}
	if err := e.c.Do(ctx, Request{Query: q, Variables: vars}, &data); err != nil {
		return 0, err
	}
	return data.Total, nil
}

// GetRequest builds the GraphQL request used by Get.
func (e *EntityClient) GetRequest(id string, fields string) (Request, error) {
	if err := e.require(registry.VerbGet); err != nil {
		return Request{}, err
	}
	if fields == "" {
		fields = e.kind.DescribeFields
	}
	q := fmt.Sprintf("query Get%s($id: %s) { item: %s(id: $id) { %s } }", e.kind.Title, e.kind.IDType, e.kind.GetOp(), fields)
	return Request{Query: q, Variables: map[string]any{"id": id}}, nil
}

// Get fetches one object by id. fields optionally overrides the selection
// (default: all describe fields).
func (e *EntityClient) Get(ctx context.Context, id string, fields ...string) (Object, error) {
	if !IsUUID(id) && e.kind.IDType == "UUID!" {
		return nil, notFound(e.kind.Name, id)
	}
	req, err := e.GetRequest(id, strings.Join(fields, " "))
	if err != nil {
		return nil, err
	}
	var data struct {
		Item Object `json:"item"`
	}
	if err := e.c.Do(ctx, req, &data); err != nil {
		return nil, err
	}
	if data.Item == nil {
		return nil, notFound(e.kind.Name, id)
	}
	return data.Item, nil
}

// GetByName fetches one object by its name. It returns a NOT_FOUND *APIError if
// nothing matches and an *AmbiguousError if several objects match.
func (e *EntityClient) GetByName(ctx context.Context, name string, fields ...string) (Object, error) {
	if !e.kind.HasNames() {
		return nil, fmt.Errorf("%s objects can only be referenced by id", e.kind.Name)
	}
	sel := strings.Join(fields, " ")
	if sel == "" {
		sel = e.kind.DescribeFields // same default as Get
	}
	list, err := e.List(ctx, ListOptions{Filter: Eq(e.kind.NameFilter, name), Limit: 10, Fields: sel})
	if err != nil {
		return nil, err
	}
	switch len(list.Items) {
	case 0:
		return nil, notFound(e.kind.Name, name)
	case 1:
		return list.Items[0], nil
	default:
		ids := make([]string, 0, len(list.Items))
		for _, it := range list.Items {
			ids = append(ids, it.ID())
		}
		return nil, &AmbiguousError{Kind: e.kind.Name, Name: name, Matches: ids}
	}
}

// Resolve fetches an object by id (if ref is a UUID) or by name.
func (e *EntityClient) Resolve(ctx context.Context, ref string, fields ...string) (Object, error) {
	if IsUUID(ref) || !e.kind.HasNames() {
		return e.Get(ctx, ref, fields...)
	}
	return e.GetByName(ctx, ref, fields...)
}

// ResolveID returns the id for an id or name reference, avoiding a request when ref is already a UUID.
func (e *EntityClient) ResolveID(ctx context.Context, ref string) (string, error) {
	if IsUUID(ref) {
		return ref, nil
	}
	obj, err := e.Resolve(ctx, ref, "id")
	if err != nil {
		return "", err
	}
	return obj.ID(), nil
}

// Actions returns the actions currently allowed for the object.
func (e *EntityClient) Actions(ctx context.Context, id string) ([]string, error) {
	if !e.kind.HasActionsQuery() {
		return nil, &UnsupportedError{Kind: e.kind.Name, Verb: "actions"}
	}
	q := fmt.Sprintf("query %sActions($id: %s) { actions: %s(id: $id) }", e.kind.Title, e.kind.IDType, e.kind.ActionsOp())
	var data struct {
		Actions []string `json:"actions"`
	}
	if err := e.c.Do(ctx, Request{Query: q, Variables: map[string]any{"id": id}}, &data); err != nil {
		return nil, err
	}
	if data.Actions == nil {
		data.Actions = []string{}
	}
	return data.Actions, nil
}

// ActionRequest builds the GraphQL request used by Action.
func (e *EntityClient) ActionRequest(id, action string) (Request, error) {
	if err := e.require(registry.VerbAction); err != nil {
		return Request{}, err
	}
	k := e.kind
	q := fmt.Sprintf("mutation %sAction($id: %s, $input: %s!) { item: %s(id: $id, input: $input) { %s } }",
		k.Title, k.IDType, k.ActionInputType(), k.ActionOp(), k.ListFields)
	return Request{Query: q, Variables: map[string]any{"id": id, "input": map[string]any{"action": action}}}, nil
}

// Action runs a lifecycle action (e.g. approve, execute, destroy, retry). It first
// checks that the action is currently allowed and returns *ActionNotAllowedError if not.
func (e *EntityClient) Action(ctx context.Context, id, action string) (Object, error) {
	req, err := e.ActionRequest(id, action)
	if err != nil {
		return nil, err
	}
	allowed, err := e.Actions(ctx, id)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(allowed, func(a string) bool { return strings.EqualFold(a, action) }) {
		return nil, &ActionNotAllowedError{Kind: e.kind.Name, ID: id, Action: action, Allowed: allowed}
	}
	return e.doItem(ctx, req)
}

// CreateRequest builds the GraphQL request used by Create.
func (e *EntityClient) CreateRequest(input map[string]any) (Request, error) {
	if err := e.require(registry.VerbCreate); err != nil {
		return Request{}, err
	}
	k := e.kind
	q := fmt.Sprintf("mutation Create%s($input: %s!) { item: %s(input: $input) { %s } }",
		k.Title, k.CreateInputType(), k.CreateOp(), k.ListFields)
	return Request{Query: q, Variables: map[string]any{"input": CamelizeKeys(input)}}, nil
}

// Create creates an object. Top-level snake_case input keys are converted to
// camelCase; nested values (variables, configuration, ...) are sent as-is.
func (e *EntityClient) Create(ctx context.Context, input map[string]any) (Object, error) {
	req, err := e.CreateRequest(input)
	if err != nil {
		return nil, err
	}
	return e.doItem(ctx, req)
}

// UpdateRequest builds the GraphQL request used by Update.
func (e *EntityClient) UpdateRequest(id string, input map[string]any) (Request, error) {
	if err := e.require(registry.VerbUpdate); err != nil {
		return Request{}, err
	}
	k := e.kind
	arg := k.UpdateArgName()
	q := fmt.Sprintf("mutation Update%[1]s($id: %[2]s, $input: %[3]s!) { item: %[4]s(id: $id, %[5]s: $input) { %[6]s } }",
		k.Title, k.IDType, k.UpdateInputType(), k.UpdateOp(), arg, k.ListFields)
	return Request{Query: q, Variables: map[string]any{"id": id, "input": CamelizeKeys(input)}}, nil
}

// Update updates an object. Input keys are handled as in Create.
func (e *EntityClient) Update(ctx context.Context, id string, input map[string]any) (Object, error) {
	req, err := e.UpdateRequest(id, input)
	if err != nil {
		return nil, err
	}
	return e.doItem(ctx, req)
}

// DeleteRequest builds the GraphQL request used by Delete.
func (e *EntityClient) DeleteRequest(id string) (Request, error) {
	if err := e.require(registry.VerbDelete); err != nil {
		return Request{}, err
	}
	k := e.kind
	q := fmt.Sprintf("mutation Delete%s($id: %s) { deleted: %s(id: $id) }", k.Title, k.IDType, k.DeleteOp())
	return Request{Query: q, Variables: map[string]any{"id": id}}, nil
}

// Delete deletes an object.
func (e *EntityClient) Delete(ctx context.Context, id string) error {
	req, err := e.DeleteRequest(id)
	if err != nil {
		return err
	}
	var data struct {
		Deleted bool `json:"deleted"`
	}
	if err := e.c.Do(ctx, req, &data); err != nil {
		return err
	}
	if !data.Deleted {
		return &APIError{Code: CodeInternal, Message: fmt.Sprintf("server did not delete %s %s", e.kind.Name, id)}
	}
	return nil
}

func (e *EntityClient) doItem(ctx context.Context, req Request) (Object, error) {
	var data struct {
		Item Object `json:"item"`
	}
	if err := e.c.Do(ctx, req, &data); err != nil {
		return nil, err
	}
	return data.Item, nil
}

// ResourceClient adds resource-specific operations.
type ResourceClient struct {
	*EntityClient
}

// treeDepth is how many levels of the resource tree are fetched.
const treeDepth = 8

// Tree returns the resource dependency tree (up to 8 levels deep).
// direction is "parents" or "children".
func (r *ResourceClient) Tree(ctx context.Context, id, direction string) (json.RawMessage, error) {
	const node = "id name state status templateName"
	sel := node
	for range treeDepth - 1 {
		sel = node + " children { " + sel + " }"
	}
	q := "query ResourceTree($id: UUID!, $direction: String!) {\n  tree: resourceTree(id: $id, direction: $direction) { " + sel + " }\n}"
	var data struct {
		Tree json.RawMessage `json:"tree"`
	}
	if err := r.c.Do(ctx, Request{Query: q, Variables: map[string]any{"id": id, "direction": direction}}, &data); err != nil {
		return nil, err
	}
	return data.Tree, nil
}

// Metadata returns cloud metadata reported for a resource.
func (r *ResourceClient) Metadata(ctx context.Context, id string) (any, error) {
	const q = `query ResourceMetadata($id: UUID!) { metadata: resourceMetadata(id: $id) }`
	var data struct {
		Metadata any `json:"metadata"`
	}
	if err := r.c.Do(ctx, Request{Query: q, Variables: map[string]any{"id": id}}, &data); err != nil {
		return nil, err
	}
	return data.Metadata, nil
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{12}$`)

// IsUUID reports whether s looks like a UUID.
func IsUUID(s string) bool { return uuidRe.MatchString(s) }

// CamelizeKeys converts top-level snake_case or kebab-case keys to camelCase.
// Nested values are left untouched because they often hold user data
// (template variables, configuration) whose keys must be preserved.
func CamelizeKeys(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[ToCamel(k)] = v
	}
	return out
}

// ToCamel converts snake_case or kebab-case to camelCase. camelCase is returned unchanged.
func ToCamel(s string) string {
	if !strings.ContainsAny(s, "_-") {
		return s
	}
	var b strings.Builder
	upper := false
	for i, r := range s {
		switch {
		case r == '_' || r == '-':
			upper = i > 0
		case upper:
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
