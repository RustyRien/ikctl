package client

import (
	"fmt"
	"strings"
	"time"
)

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

func (r Resource) GetName() string {
	return r.Name
}

type Template struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	CloudResourceTypes []string  `json:"cloudResourceTypes"`
}

func (t Template) GetName() string {
	return t.Name
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

type User struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Provider    string `json:"provider"`
	EntityName  string `json:"entityName"`
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

func (i Integration) GetName() string {
	return i.Name
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

type currentUserQueryData struct {
	CurrentUser *User `json:"currentUser"`
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

type integrationActionMutationData struct {
	IntegrationAction *entityActionResult `json:"integrationAction"`
}

type refreshAuthTokenMutationData struct {
	RefreshAuthToken *RefreshAuthTokenResult `json:"refreshAuthToken"`
}

type logoutMutationData struct {
	Logout *logoutResult `json:"logout"`
}

type deleteIntegrationMutationData struct {
	DeleteIntegration bool `json:"deleteIntegration"`
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

type entityActionResult struct {
	ID         string `json:"id"`
	EntityName string `json:"entityName"`
	Status     string `json:"status"`
}

type LogStreamMessage struct {
	Data  string `json:"data"`
	Level string `json:"level"`
}

type RefreshAuthTokenResult struct {
	Token        string       `json:"token"`
	Expiration   FlexibleTime `json:"expiration"`
	Provider     string       `json:"provider"`
	RefreshToken string       `json:"-"`
}

type logoutResult struct {
	Success bool `json:"success"`
}

type FlexibleTime struct {
	time.Time
}

func (t *FlexibleTime) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if raw == "" || raw == "null" {
		t.Time = time.Time{}
		return nil
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999-07:00", "2006-01-02 15:04:05-07:00"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			t.Time = parsed
			return nil
		}
	}

	return fmt.Errorf("parse time %q", raw)
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
