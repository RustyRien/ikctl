package client

import "time"

type Resource struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	State        string        `json:"state"`
	Status       string        `json:"status"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	Labels       []string      `json:"labels"`
	Template     *Template     `json:"template"`
	Workspace    *Workspace    `json:"workspace"`
	Creator      *Creator      `json:"creator"`
	Integrations []Integration `json:"integrationIds"`
}

type Template struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	CloudResourceTypes []string `json:"cloudResourceTypes"`
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
	ID                  string `json:"id"`
	Name                string `json:"name"`
	IntegrationProvider string `json:"integrationProvider"`
	IntegrationType     string `json:"integrationType"`
}

type resourcesQueryData struct {
	Resources      []Resource `json:"resources"`
	ResourcesCount int        `json:"resourcesCount"`
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
