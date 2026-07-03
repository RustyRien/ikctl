package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

type Descriptor struct {
	Name          string
	Singular      string
	Aliases       []string
	Headers       []tabledata.Header
	WideHeaders   []tabledata.Header
	DefaultSort   []string
	SortFields    map[string]string
	FilterKeys    map[string]string
	List          func(context.Context, map[string]any, []string, []int) ([]tabledata.Row, []any, int, error)
	GetByID       func(context.Context, string) (tabledata.Row, any, error)
	ResolveByName func(context.Context, string) (tabledata.Row, any, error)
	WideRow       func(any) tabledata.Row
}

type Registry struct {
	ordered []*Descriptor
	lookup  map[string]*Descriptor
}

func NewRegistry(descriptors ...*Descriptor) *Registry {
	lookup := make(map[string]*Descriptor, len(descriptors)*3)
	for _, descriptor := range descriptors {
		lookup[descriptor.Name] = descriptor
		lookup[descriptor.Singular] = descriptor
		for _, alias := range descriptor.Aliases {
			lookup[alias] = descriptor
		}
	}
	return &Registry{ordered: descriptors, lookup: lookup}
}

func (r *Registry) Ordered() []*Descriptor {
	return append([]*Descriptor(nil), r.ordered...)
}

func (r *Registry) Resolve(name string) (*Descriptor, bool) {
	descriptor, ok := r.lookup[strings.ToLower(strings.TrimSpace(name))]
	return descriptor, ok
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.ordered))
	for _, descriptor := range r.ordered {
		names = append(names, descriptor.Name)
	}
	sort.Strings(names)
	return names
}

func ResolveSortField(descriptor *Descriptor, name string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return "", false
	}
	field, ok := descriptor.SortFields[key]
	return field, ok
}

func ResolveFilter(descriptor *Descriptor, filters map[string]string) (map[string]any, error) {
	if len(filters) == 0 {
		return nil, nil
	}
	resolved := make(map[string]any, len(filters))
	for key, value := range filters {
		mapped, ok := descriptor.FilterKeys[strings.ToLower(strings.TrimSpace(key))]
		if !ok {
			return nil, fmt.Errorf("unsupported filter %q for %s", key, descriptor.Name)
		}
		resolved[mapped] = value
	}
	return resolved, nil
}

func SortFields(headers []tabledata.Header) map[string]string {
	fields := make(map[string]string, len(headers)*2)
	for _, header := range headers {
		if header.SortField == "" {
			continue
		}
		fields[strings.ToLower(header.Key)] = header.SortField
		fields[strings.ToLower(header.Title)] = header.SortField
	}
	return fields
}

type NamedEntity interface {
	GetName() string
}

func ResolveByName[T NamedEntity](ctx context.Context, name string, rowFn func(T) tabledata.Row, listFn func(context.Context, string) ([]T, error)) (tabledata.Row, any, error) {
	items, err := listFn(ctx, name)
	if err != nil {
		return tabledata.Row{}, nil, err
	}
	needle := strings.TrimSpace(name)
	for _, item := range items {
		if item.GetName() == needle {
			return rowFn(item), item, nil
		}
	}
	return tabledata.Row{}, nil, errors.New("item not found")
}
