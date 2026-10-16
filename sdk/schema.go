package sdk

import (
	"context"
	"fmt"
)

// TypeRef is a GraphQL type reference.
type TypeRef struct {
	Kind   string   `json:"kind"`
	Name   string   `json:"name,omitempty"`
	OfType *TypeRef `json:"ofType,omitempty"`
}

// String renders the reference in GraphQL notation, e.g. "[String!]!".
func (t *TypeRef) String() string {
	if t == nil {
		return ""
	}
	switch t.Kind {
	case "NON_NULL":
		return t.OfType.String() + "!"
	case "LIST":
		return "[" + t.OfType.String() + "]"
	default:
		return t.Name
	}
}

// Named returns the innermost named type.
func (t *TypeRef) Named() string {
	for t != nil && t.Name == "" {
		t = t.OfType
	}
	if t == nil {
		return ""
	}
	return t.Name
}

// FieldInfo describes an object field or input field.
type FieldInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Type        *TypeRef `json:"-"`
	TypeName    string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
}

// TypeInfo describes a GraphQL type.
type TypeInfo struct {
	Name        string      `json:"name"`
	Kind        string      `json:"kind"`
	Description string      `json:"description,omitempty"`
	Fields      []FieldInfo `json:"fields,omitempty"`
	InputFields []FieldInfo `json:"inputFields,omitempty"`
	EnumValues  []string    `json:"enumValues,omitempty"`
}

// SchemaService provides GraphQL schema introspection.
type SchemaService struct{ c *Client }

const typeRefFragment = `kind name ofType { kind name ofType { kind name ofType { kind name ofType { kind name } } } }`

// Type returns the description of a named GraphQL type.
func (s *SchemaService) Type(ctx context.Context, name string) (*TypeInfo, error) {
	q := `query Type($name: String!) { __type(name: $name) {
  name kind description
  fields { name description type { ` + typeRefFragment + ` } }
  inputFields { name description type { ` + typeRefFragment + ` } }
  enumValues { name }
} }`
	var data struct {
		Type *struct {
			Name        string `json:"name"`
			Kind        string `json:"kind"`
			Description string `json:"description"`
			Fields      []struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Type        *TypeRef `json:"type"`
			} `json:"fields"`
			InputFields []struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Type        *TypeRef `json:"type"`
			} `json:"inputFields"`
			EnumValues []struct {
				Name string `json:"name"`
			} `json:"enumValues"`
		} `json:"__type"`
	}
	if err := s.c.Do(ctx, Request{Query: q, Variables: map[string]any{"name": name}}, &data); err != nil {
		return nil, err
	}
	if data.Type == nil {
		return nil, &APIError{Code: CodeNotFound, Message: fmt.Sprintf("GraphQL type %q not found", name)}
	}
	t := data.Type
	info := &TypeInfo{Name: t.Name, Kind: t.Kind, Description: t.Description}
	for _, f := range t.Fields {
		info.Fields = append(info.Fields, FieldInfo{Name: f.Name, Description: f.Description, Type: f.Type,
			TypeName: f.Type.String(), Required: f.Type != nil && f.Type.Kind == "NON_NULL"})
	}
	for _, f := range t.InputFields {
		info.InputFields = append(info.InputFields, FieldInfo{Name: f.Name, Description: f.Description, Type: f.Type,
			TypeName: f.Type.String(), Required: f.Type != nil && f.Type.Kind == "NON_NULL"})
	}
	for _, v := range t.EnumValues {
		info.EnumValues = append(info.EnumValues, v.Name)
	}
	return info, nil
}
