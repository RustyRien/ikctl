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

func ExecutorInputFromYAML(data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"description", "description"},
		fieldPair{"commandArgs", "command_args"},
		fieldPair{"runtime", "runtime"},
		fieldPair{"sourceCodeVersion", "source_code_version"},
		fieldPair{"sourceCodeBranch", "source_code_branch"},
		fieldPair{"sourceCodeFolder", "source_code_folder"},
		fieldPair{"storagePath", "storage_path"},
		fieldPair{"labels", "labels"},
	)
	copyNilableID(input, decoded, "sourceCodeId", "source_code_id", "sourceCode", "source_code")
	copyNilableID(input, decoded, "storageId", "storage_id", "storage", "storage")
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

func WorkspaceInputFromYAML(data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"name", "name"},
		fieldPair{"description", "description"},
		fieldPair{"labels", "labels"},
	)
	return input, nil
}

func SecretInputFromYAML(current client.Secret, data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"description", "description"},
		fieldPair{"labels", "labels"},
		fieldPair{"configuration", "configuration"},
	)
	if value, ok := decoded["secret_provider"]; ok {
		input["secretProvider"] = value
	} else if value, ok := decoded["secretProvider"]; ok {
		input["secretProvider"] = value
	} else if _, ok := input["configuration"]; ok && current.SecretProvider != "" {
		input["secretProvider"] = current.SecretProvider
	}
	return input, nil
}

func StorageInputFromYAML(data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"description", "description"},
		fieldPair{"labels", "labels"},
	)
	return input, nil
}

func SourceCodeInputFromYAML(data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"description", "description"},
		fieldPair{"labels", "labels"},
	)
	copyNilableID(input, decoded, "integrationId", "integration_id", "integration", "integration")
	return input, nil
}

func SourceCodeVersionInputFromYAML(data []byte) (map[string]any, error) {
	decoded, err := ParseYAMLMap(data)
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	copyOptional(input, decoded,
		fieldPair{"description", "description"},
		fieldPair{"labels", "labels"},
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

func ExecutorYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.Executor)
	if !ok {
		return nil, fmt.Errorf("expected executor, got %T", raw)
	}
	return YAMLBytes(map[string]any{
		"name":                value.Name,
		"description":         value.Description,
		"runtime":             value.Runtime,
		"command_args":        nilIfEmpty(value.CommandArgs),
		"source_code_id":      optionalSourceCodeID(value.SourceCode),
		"source_code_version": nilIfEmpty(value.SourceCodeVersion),
		"source_code_branch":  nilIfEmpty(value.SourceCodeBranch),
		"source_code_folder":  nilIfEmpty(value.SourceCodeFolder),
		"integration_ids":     integrationIDs(value.Integrations),
		"secret_ids":          secretIDs(value.Secrets),
		"storage_id":          optionalStorageID(value.Storage),
		"storage_path":        nilIfEmpty(value.StoragePath),
		"labels":              value.Labels,
	})
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

func WorkspaceYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.Workspace)
	if !ok {
		return nil, fmt.Errorf("expected workspace, got %T", raw)
	}
	return YAMLBytes(map[string]any{
		"name":        value.Name,
		"description": value.Description,
		"labels":      value.Labels,
	})
}

func SecretYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.Secret)
	if !ok {
		return nil, fmt.Errorf("expected secret, got %T", raw)
	}
	return YAMLBytes(map[string]any{
		"description":     value.Description,
		"labels":          value.Labels,
		"secret_provider": value.SecretProvider,
		"configuration":   value.Configuration,
	})
}

func StorageYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.Storage)
	if !ok {
		return nil, fmt.Errorf("expected storage, got %T", raw)
	}
	return YAMLBytes(map[string]any{
		"description": value.Description,
		"labels":      value.Labels,
	})
}

func SourceCodeYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.SourceCode)
	if !ok {
		return nil, fmt.Errorf("expected source code, got %T", raw)
	}
	return YAMLBytes(map[string]any{
		"description":    value.Description,
		"integration_id": normalizeNilString(value.IntegrationID),
		"labels":         value.Labels,
	})
}

func SourceCodeVersionYAML(raw any) ([]byte, error) {
	value, ok := raw.(client.SourceCodeVersion)
	if !ok {
		return nil, fmt.Errorf("expected source code version, got %T", raw)
	}
	return YAMLBytes(map[string]any{
		"description": value.Description,
		"labels":      value.Labels,
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

func optionalSourceCodeID(value *client.SourceCode) any {
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
