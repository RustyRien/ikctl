package client

import "time"

type Resource struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	State             string              `json:"state"`
	Status            string              `json:"status"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
	Labels            []string            `json:"labels"`
	Template          *Template           `json:"template"`
	Workspace         *Workspace          `json:"workspace"`
	Creator           *Creator            `json:"creator"`
	Integrations      []Integration       `json:"integrationIds"`
	RevisionNumber    int                 `json:"revisionNumber"`
	Abstract          bool                `json:"abstract"`
	StoragePath       string              `json:"storagePath"`
	Variables         []map[string]any    `json:"variables"`
	Outputs           []map[string]any    `json:"outputs"`
	DependencyTags    []map[string]any    `json:"dependencyTags"`
	DependencyConfig  []map[string]any    `json:"dependencyConfig"`
	SourceCodeVersion *SourceCodeVersion  `json:"sourceCodeVersion"`
	Parents           []ResourceReference `json:"parents"`
	Children          []ResourceReference `json:"children"`
}

type Template struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	CloudResourceTypes []string  `json:"cloudResourceTypes"`
}

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Creator struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type Integration struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
	IntegrationProvider string    `json:"integrationProvider"`
	IntegrationType     string    `json:"integrationType"`
}

type SourceCodeVersion struct {
	ID                string `json:"id"`
	Identifier        string `json:"identifier"`
	SourceCodeFolder  string `json:"sourceCodeFolder"`
	SourceCodeVersion string `json:"sourceCodeVersion"`
	SourceCodeBranch  string `json:"sourceCodeBranch"`
	Status            string `json:"status"`
}

type ResourceReference struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Status string `json:"status"`
}

type Log struct {
	ID             string    `json:"id"`
	EntityID       string    `json:"entityId"`
	Entity         string    `json:"entity"`
	Revision       int       `json:"revision"`
	AuditLogID     string    `json:"auditLogId"`
	Level          string    `json:"level"`
	Data           string    `json:"data"`
	CreatedAt      time.Time `json:"createdAt"`
	ExecutionStart int       `json:"executionStart"`
	ExpireAt       time.Time `json:"expireAt"`
	TraceID        string    `json:"traceId"`
}

type AuditLog struct {
	ID             string    `json:"id"`
	Model          string    `json:"model"`
	UserID         string    `json:"userId"`
	Action         string    `json:"action"`
	EntityID       string    `json:"entityId"`
	CreatedAt      time.Time `json:"createdAt"`
	RevisionNumber int       `json:"revisionNumber"`
	Creator        *Creator  `json:"creator"`
}

type resourcesQueryData struct {
	Resources      []Resource `json:"resources"`
	ResourcesCount int        `json:"resourcesCount"`
}

type resourceQueryData struct {
	Resource *Resource `json:"resource"`
}

type templatesQueryData struct {
	Templates      []Template `json:"templates"`
	TemplatesCount int        `json:"templatesCount"`
}

type templateQueryData struct {
	Template *Template `json:"template"`
}

type integrationsQueryData struct {
	Integrations      []Integration `json:"integrations"`
	IntegrationsCount int           `json:"integrationsCount"`
}

type integrationQueryData struct {
	Integration *Integration `json:"integration"`
}

type logsQueryData struct {
	Logs      []Log `json:"logs"`
	LogsCount int   `json:"logsCount"`
}

type auditLogsQueryData struct {
	AuditLogs []AuditLog `json:"auditLogs"`
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse[T any] struct {
	Data   T              `json:"data"`
	Errors []GraphQLError `json:"errors"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

type ResourcesResult struct {
	Items []Resource
	Total int
}

type TemplatesResult struct {
	Items []Template
	Total int
}

type IntegrationsResult struct {
	Items []Integration
	Total int
}
