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
	Actions           []string            `json:"actions"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
	Labels            []string            `json:"labels"`
	Template          *Template           `json:"template"`
	Workspace         *Workspace          `json:"workspace"`
	Storage           *Storage            `json:"storage"`
	Creator           *Creator            `json:"creator"`
	Integrations      []Integration       `json:"integrationIds"`
	Secrets           []Secret            `json:"secretIds"`
	EntityName        string              `json:"entityName"`
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
	ID                      string              `json:"id"`
	Name                    string              `json:"name"`
	Description             string              `json:"description"`
	Documentation           string              `json:"documentation"`
	Template                string              `json:"template"`
	CreatedAt               time.Time           `json:"createdAt"`
	UpdatedAt               time.Time           `json:"updatedAt"`
	CloudResourceTypes      []string            `json:"cloudResourceTypes"`
	Labels                  []string            `json:"labels"`
	Status                  string              `json:"status"`
	Abstract                bool                `json:"abstract"`
	Configuration           map[string]any      `json:"configuration"`
	RevisionNumber          int                 `json:"revisionNumber"`
	ResourcesCount          int                 `json:"resourcesCount"`
	SourceCodeVersionsCount int                 `json:"sourceCodeVersionsCount"`
	EntityName              string              `json:"entityName"`
	Creator                 *Creator            `json:"creator"`
	Parents                 []TemplateReference `json:"parents"`
	Children                []TemplateReference `json:"children"`
}

func (t Template) GetName() string {
	return t.Name
}

type TemplateReference struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Abstract           bool     `json:"abstract"`
	CloudResourceTypes []string `json:"cloudResourceTypes"`
	EntityName         string   `json:"entityName"`
}

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Storage struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	StorageType     string         `json:"storageType"`
	StorageProvider string         `json:"storageProvider"`
	Configuration   map[string]any `json:"configuration"`
	Description     string         `json:"description"`
	Labels          []string       `json:"labels"`
	State           string         `json:"state"`
	Status          string         `json:"status"`
	ResourcesCount  int            `json:"resourcesCount"`
	ExecutorsCount  int            `json:"executorsCount"`
	RevisionNumber  int            `json:"revisionNumber"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	EntityName      string         `json:"entityName"`
	Integration     *Integration   `json:"integration"`
	Creator         *Creator       `json:"creator"`
}

func (s Storage) GetName() string {
	return s.Name
}

type RefFolders struct {
	Ref     string   `json:"ref"`
	Folders []string `json:"folders"`
}

type SourceCode struct {
	ID                 string            `json:"id"`
	Identifier         string            `json:"identifier"`
	Description        string            `json:"description"`
	SourceCodeURL      string            `json:"sourceCodeUrl"`
	SourceCodeProvider string            `json:"sourceCodeProvider"`
	SourceCodeLanguage string            `json:"sourceCodeLanguage"`
	IntegrationID      string            `json:"integrationId"`
	GitTags            []string          `json:"gitTags"`
	GitTagMessages     map[string]string `json:"gitTagMessages"`
	GitBranches        []string          `json:"gitBranches"`
	GitBranchMessages  map[string]string `json:"gitBranchMessages"`
	GitFoldersMap      []RefFolders      `json:"gitFoldersMap"`
	Labels             []string          `json:"labels"`
	Status             string            `json:"status"`
	RevisionNumber     int               `json:"revisionNumber"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
	EntityName         string            `json:"entityName"`
	Integration        *Integration      `json:"integration"`
	Creator            *Creator          `json:"creator"`
}

func (s SourceCode) DisplayName() string {
	if name := repoNameFromURL(s.SourceCodeURL); name != "" {
		return name
	}
	if strings.TrimSpace(s.Identifier) != "" {
		return s.Identifier
	}
	return s.SourceCodeURL
}

func (s SourceCode) GetName() string {
	if strings.TrimSpace(s.Identifier) != "" {
		return s.Identifier
	}
	return s.DisplayName()
}

type Secret struct {
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
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	IntegrationProvider string         `json:"integrationProvider"`
	IntegrationType     string         `json:"integrationType"`
	EntityName          string         `json:"entityName"`
	Labels              []string       `json:"labels"`
	Configuration       map[string]any `json:"configuration"`
	Status              string         `json:"status"`
	Creator             *Creator       `json:"creator"`
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

type resourceActionsQueryData struct {
	ResourceActions []string `json:"resourceActions"`
}

type resourceTempStateQueryData struct {
	ResourceTempStateByResource *ResourceTempState `json:"resourceTempStateByResource"`
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

type TemplateTreeNode struct {
	ID       string             `json:"id"`
	NodeID   string             `json:"nodeId"`
	Name     string             `json:"name"`
	Status   string             `json:"status"`
	Children []TemplateTreeNode `json:"children"`
}

type templateTreeQueryData struct {
	TemplateTree *TemplateTreeNode `json:"templateTree"`
}

type ResourceTreeNode struct {
	ID           string             `json:"id"`
	NodeID       string             `json:"nodeId"`
	Name         string             `json:"name"`
	State        string             `json:"state"`
	Status       string             `json:"status"`
	TemplateName string             `json:"templateName"`
	Children     []ResourceTreeNode `json:"children"`
}

type resourceTreeQueryData struct {
	ResourceTree *ResourceTreeNode `json:"resourceTree"`
}

type integrationsQueryData struct {
	Integrations      []Integration `json:"integrations"`
	IntegrationsCount int           `json:"integrationsCount"`
}

type integrationQueryData struct {
	Integration *Integration `json:"integration"`
}

type storagesQueryData struct {
	Storages      []Storage `json:"storages"`
	StoragesCount int       `json:"storagesCount"`
}

type storageQueryData struct {
	Storage *Storage `json:"storage"`
}

type sourceCodesQueryData struct {
	SourceCodes      []SourceCode `json:"sourceCodes"`
	SourceCodesCount int          `json:"sourceCodesCount"`
}

type sourceCodeQueryData struct {
	SourceCode *SourceCode `json:"sourceCode"`
}

type integrationActionMutationData struct {
	IntegrationAction *entityActionResult `json:"integrationAction"`
}

type templateActionMutationData struct {
	TemplateAction *entityActionResult `json:"templateAction"`
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

type deleteSourceCodeMutationData struct {
	DeleteSourceCode bool `json:"deleteSourceCode"`
}

type deleteTemplateMutationData struct {
	DeleteTemplate bool `json:"deleteTemplate"`
}

type deleteResourceMutationData struct {
	DeleteResource bool `json:"deleteResource"`
}

type updateResourceMutationData struct {
	UpdateResource *entityActionResult `json:"updateResource"`
}

type resourceActionMutationData struct {
	ResourceAction *entityActionResult `json:"resourceAction"`
}

type updateTemplateMutationData struct {
	UpdateTemplate *templateUpdateResult `json:"updateTemplate"`
}

type updateIntegrationMutationData struct {
	UpdateIntegration *integrationUpdateResult `json:"updateIntegration"`
}

type updateStorageMutationData struct {
	UpdateStorage *entityActionResult `json:"updateStorage"`
}

type updateSourceCodeMutationData struct {
	UpdateSourceCode *entityActionResult `json:"updateSourceCode"`
}

type sourceCodeActionMutationData struct {
	SourceCodeAction *entityActionResult `json:"sourceCodeAction"`
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

type templateUpdateResult struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Template   string `json:"template"`
	EntityName string `json:"entityName"`
}

type integrationUpdateResult struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	EntityName          string `json:"entityName"`
	IntegrationProvider string `json:"integrationProvider"`
}

type ResourceTempState struct {
	ID         string         `json:"id"`
	ResourceID string         `json:"resourceId"`
	Value      map[string]any `json:"value"`
	CreatedAt  FlexibleTime   `json:"createdAt"`
	UpdatedAt  FlexibleTime   `json:"updatedAt"`
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

type StoragesResult struct {
	Items []Storage
	Total int
}

type SourceCodesResult struct {
	Items []SourceCode
	Total int
}

func repoNameFromURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, sep := range []string{"#", "?"} {
		if index := strings.Index(value, sep); index >= 0 {
			value = value[:index]
		}
	}
	value = strings.TrimSuffix(strings.TrimRight(value, "/"), ".git")
	if index := strings.LastIndex(value, "/"); index >= 0 && index+1 < len(value) {
		return value[index+1:]
	}
	if index := strings.LastIndex(value, ":"); index >= 0 && index+1 < len(value) {
		return value[index+1:]
	}
	return value
}
