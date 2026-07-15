package edit

import (
	"fmt"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

type fieldPair struct {
	dst string
	src string
}

func ResourceInputFromYAML(data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"name", "name"},
		fieldPair{"description", "description"},
		fieldPair{"storagePath", "storage_path"},
		fieldPair{"labels", "labels"},
		fieldPair{"variables", "variables"},
		fieldPair{"dependencyTags", "dependency_tags"},
		fieldPair{"dependencyConfig", "dependency_config"},
	)
	copyNilableID(input, decoded, "sourceCodeVersionId", "source_code_version_id", "sourceCodeVersion", "source_code_version")
	copyNilableID(input, decoded, "storageId", "storage_id", "storage", "storage")
	copyNilableID(input, decoded, "workspaceId", "workspace_id", "workspace", "workspace")
	copyIDSlice(input, decoded, "integrationIds", "integration_ids")
	copyIDSlice(input, decoded, "secretIds", "secret_ids")
	return input, nil
}

func TemplateInputFromYAML(data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"name", "name"},
		fieldPair{"description", "description"},
		fieldPair{"documentation", "documentation"},
		fieldPair{"cloudResourceTypes", "cloud_resource_types"},
		fieldPair{"configuration", "configuration"},
		fieldPair{"labels", "labels"},
	)
	copyIDSlice(input, decoded, "parents", "parents")
	copyIDSlice(input, decoded, "children", "children")
	return input, nil
}

func IntegrationInputFromYAML(data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"name", "name"},
		fieldPair{"description", "description"},
		fieldPair{"labels", "labels"},
		fieldPair{"configuration", "configuration"},
	)
	return input, nil
}

func ResourceYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.Resource)
	if !ok {
		return nil, fmt.Errorf("expected resource, got %T", raw)
	}
	return YAMLBytes(ResourceEditableState(value))
}

func ResourceEditableState(value client.Resource) map[string]any {
	return map[string]any{
		"name":                   value.Name,
		"description":            value.Description,
		"source_code_version_id": optionalSourceCodeVersionID(value.SourceCodeVersion),
		"integration_ids":        integrationIDs(value.Integrations),
		"secret_ids":             secretIDs(value.Secrets),
		"storage_id":             optionalStorageID(value.Storage),
		"storage_path":           nilIfEmpty(value.StoragePath),
		"variables":              value.Variables,
		"dependency_tags":        value.DependencyTags,
		"dependency_config":      value.DependencyConfig,
		"labels":                 value.Labels,
		"workspace_id":           optionalWorkspaceID(value.Workspace),
	}
}

func TemplateYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.Template)
	if !ok {
		return nil, fmt.Errorf("expected template, got %T", raw)
	}
	return YAMLBytes(map[string]any{
		"name":                 value.Name,
		"description":          value.Description,
		"documentation":        value.Documentation,
		"parents":              templateReferenceIDs(value.Parents),
		"children":             templateReferenceIDs(value.Children),
		"cloud_resource_types": value.CloudResourceTypes,
		"configuration":        value.Configuration,
		"labels":               value.Labels,
	})
}

func IntegrationYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.Integration)
	if !ok {
		return nil, fmt.Errorf("expected integration, got %T", raw)
	}
	return YAMLBytes(map[string]any{
		"name":          value.Name,
		"description":   value.Description,
		"labels":        value.Labels,
		"configuration": value.Configuration,
	})
}

func optionalWorkspaceID(value *client.Workspace) any {
	if value == nil || value.ID == "" {
		return nil
	}
	return value.ID
}

func optionalStorageID(value *client.Storage) any {
	if value == nil || value.ID == "" {
		return nil
	}
	return value.ID
}

func optionalSourceCodeVersionID(value *client.SourceCodeVersion) any {
	if value == nil || value.ID == "" {
		return nil
	}
	return value.ID
}

func integrationIDs(values []client.Integration) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		if value.ID != "" {
			ids = append(ids, value.ID)
		}
	}
	return ids
}

func secretIDs(values []client.Secret) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		if value.ID != "" {
			ids = append(ids, value.ID)
		}
	}
	return ids
}

func templateReferenceIDs(values []client.TemplateReference) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		if value.ID != "" {
			ids = append(ids, value.ID)
		}
	}
	return ids
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func copyOptional(dst map[string]any, src map[string]any, fields ...fieldPair) {
	for _, field := range fields {
		if value, ok := src[field.src]; ok {
			dst[field.dst] = value
		}
	}
}

func copyNilableID(dst map[string]any, src map[string]any, dstKey string, srcKey string, objectDstKey string, objectSrcKey string) {
	if value, ok := src[srcKey]; ok {
		dst[dstKey] = normalizeNilString(value)
		return
	}
	if value, ok := src[dstKey]; ok {
		dst[dstKey] = normalizeNilString(value)
		return
	}
	value, ok := src[objectSrcKey]
	if !ok {
		value, ok = src[objectDstKey]
	}
	if !ok {
		return
	}
	if value == nil {
		dst[dstKey] = nil
		return
	}
	ref, ok := value.(map[string]any)
	if !ok {
		return
	}
	if id, ok := ref["id"]; ok {
		dst[dstKey] = normalizeNilString(id)
	}
}

func copyIDSlice(dst map[string]any, src map[string]any, dstKey string, srcKey string) {
	value, ok := src[srcKey]
	if !ok {
		value, ok = src[dstKey]
	}
	if !ok {
		return
	}
	items, ok := value.([]any)
	if !ok {
		return
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		switch current := item.(type) {
		case string:
			if current != "" {
				ids = append(ids, current)
			}
		case map[string]any:
			if id, ok := current["id"].(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
	}
	dst[dstKey] = ids
}

func normalizeNilString(value any) any {
	text, ok := value.(string)
	if !ok {
		return value
	}
	if text == "" {
		return nil
	}
	return text
}
