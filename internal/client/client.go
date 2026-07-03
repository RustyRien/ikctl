package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/config"
)

const resourcesQuery = `query ListResources($filter: JSON, $sort: [String!], $range: [Int!]) {
  resources(filter: $filter, sort: $sort, range: $range) {
    id
    name
    description
    state
    status
    createdAt
    updatedAt
    labels
    template {
      id
      name
      cloudResourceTypes
    }
    workspace {
      id
      name
    }
    storage {
      id
      name
    }
    creator {
      id
      identifier
      email
      displayName
    }
    integrationIds {
      id
      name
      integrationProvider
      integrationType
    }
    secretIds {
      id
      name
    }
    revisionNumber
    storagePath
    sourceCodeVersion {
      id
      identifier
      sourceCodeFolder
      sourceCodeVersion
      sourceCodeBranch
      status
    }
    parents {
      id
      name
      state
      status
    }
    children {
      id
      name
      state
      status
    }
    variables
    outputs
    labels
    dependencyTags
    dependencyConfig
  }
  resourcesCount(filter: $filter)
}`

const templatesQuery = `query ListTemplates($filter: JSON, $sort: [String!], $range: [Int!]) {
  templates(filter: $filter, sort: $sort, range: $range) {
    id
    name
    description
    labels
    status
    abstract
    createdAt
    updatedAt
    cloudResourceTypes
    entityName
  }
  templatesCount(filter: $filter)
}`

const resourceQuery = `query GetResource($id: UUID!) {
  resource(id: $id) {
    id
    name
    description
    state
    status
    createdAt
    updatedAt
    revisionNumber
    abstract
    storagePath
    labels
    variables
    outputs
    dependencyTags
    dependencyConfig
    template {
      id
      name
      cloudResourceTypes
    }
    workspace {
      id
      name
    }
    storage {
      id
      name
    }
    creator {
      id
      identifier
      email
      displayName
    }
    integrationIds {
      id
      name
      integrationProvider
      integrationType
    }
    secretIds {
      id
      name
    }
    sourceCodeVersion {
      id
      identifier
      sourceCodeFolder
      sourceCodeVersion
      sourceCodeBranch
      status
    }
    parents {
      id
      name
      state
      status
    }
    children {
      id
      name
      state
      status
    }
  }
}`

const templateQuery = `query GetTemplate($id: UUID!) {
  template(id: $id) {
    id
    name
    description
    documentation
    template
    createdAt
    updatedAt
    cloudResourceTypes
    labels
    status
    abstract
    configuration
    revisionNumber
    resourcesCount
    sourceCodeVersionsCount
    entityName
    creator {
      id
      identifier
      email
      displayName
    }
    parents {
      id
      name
      abstract
      cloudResourceTypes
      entityName
    }
    children {
      id
      name
      abstract
      cloudResourceTypes
      entityName
    }
  }
}`

const templateTreeQuery = `query TemplateTree($id: UUID!, $direction: String!) {
  templateTree(id: $id, direction: $direction) {
    id
    nodeId
    name
    status
    children {
      id
      nodeId
      name
      status
      children {
        id
        nodeId
        name
        status
        children {
          id
          nodeId
          name
          status
          children {
            id
            nodeId
            name
            status
          }
        }
      }
    }
  }
}`

const integrationsQuery = `query ListIntegrations($filter: JSON, $sort: [String!], $range: [Int!]) {
  integrations(filter: $filter, sort: $sort, range: $range) {
    id
    name
    description
    createdAt
    updatedAt
    integrationProvider
    integrationType
  }
  integrationsCount(filter: $filter)
}`

const integrationQuery = `query GetIntegration($id: UUID!) {
  integration(id: $id) {
    id
    name
    description
    createdAt
    updatedAt
    integrationProvider
    integrationType
  }
}`

const integrationActionMutation = `mutation IntegrationAction($id: UUID!, $input: IntegrationActionInput!) {
  integrationAction(id: $id, input: $input) {
    id
    entityName
    status
  }
}`

const deleteIntegrationMutation = `mutation DeleteIntegration($id: UUID!) {
  deleteIntegration(id: $id)
}`

const currentUserQuery = `query CurrentUser {
  currentUser {
    id
    identifier
    displayName
    email
    provider
    entityName
  }
}`

const refreshAuthTokenMutation = `mutation RefreshAuthToken {
  refreshAuthToken {
    token
    expiration
    provider
  }
}`

const logoutMutation = `mutation Logout {
  logout {
    success
  }
}`

const logsQuery = `query ListLogs($filter: JSON, $sort: [String!], $range: [Int!]) {
  logs(filter: $filter, sort: $sort, range: $range) {
    id
    entityId
    entity
    revision
    auditLogId
    level
    data
    createdAt
    executionStart
    expireAt
    traceId
  }
}`

const auditLogsQuery = `query ListAuditLogs($filter: JSON, $sort: [String!], $range: [Int!]) {
  auditLogs(filter: $filter, sort: $sort, range: $range) {
    id
    model
    userId
    action
    entityId
    createdAt
    revisionNumber
    creator {
      id
      identifier
      email
      displayName
    }
  }
}`

type Client struct {
	endpoint      string
	tokenProvider TokenProvider
	httpClient    *http.Client
}

type TokenProvider interface {
	Token(context.Context) (string, error)
}

type StaticToken string

func StaticTokenProvider(token string) TokenProvider {
	return StaticToken(token)
}

func (t StaticToken) Token(context.Context) (string, error) {
	return string(t), nil
}

func New(cfg config.Config) *Client {
	return NewWithTokenProvider(cfg, StaticTokenProvider(cfg.Token))
}

func NewWithTokenProvider(cfg config.Config, tokenProvider TokenProvider) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}
	jar, _ := cookiejar.New(nil)

	return &Client{
		endpoint:      cfg.Endpoint + "/api/graphql",
		tokenProvider: tokenProvider,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
			Jar:       jar,
		},
	}
}

func (c *Client) Resources(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (ResourcesResult, error) {
	variables := map[string]any{}
	if filter != nil {
		variables["filter"] = filter
	}
	if len(sort) > 0 {
		variables["sort"] = sort
	}
	if len(pageRange) > 0 {
		variables["range"] = pageRange
	}

	resp, err := query[resourcesQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     resourcesQuery,
		Variables: variables,
	})
	if err != nil {
		return ResourcesResult{}, err
	}

	return ResourcesResult{
		Items: resp.Resources,
		Total: resp.ResourcesCount,
	}, nil
}

func (c *Client) Resource(ctx context.Context, id string) (*Resource, error) {
	resp, err := query[resourceQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: resourceQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Resource, nil
}

func (c *Client) Templates(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (TemplatesResult, error) {
	variables := map[string]any{}
	if filter != nil {
		variables["filter"] = filter
	}
	if len(sort) > 0 {
		variables["sort"] = sort
	}
	if len(pageRange) > 0 {
		variables["range"] = pageRange
	}

	resp, err := query[templatesQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     templatesQuery,
		Variables: variables,
	})
	if err != nil {
		return TemplatesResult{}, err
	}

	return TemplatesResult{Items: resp.Templates, Total: resp.TemplatesCount}, nil
}

func (c *Client) Template(ctx context.Context, id string) (*Template, error) {
	resp, err := query[templateQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: templateQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Template, nil
}

func (c *Client) TemplateTree(ctx context.Context, id string, direction string) (*TemplateTreeNode, error) {
	resp, err := query[templateTreeQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: templateTreeQuery,
		Variables: map[string]any{
			"id":        id,
			"direction": direction,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.TemplateTree, nil
}

func (c *Client) Integrations(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (IntegrationsResult, error) {
	variables := map[string]any{}
	if filter != nil {
		variables["filter"] = filter
	}
	if len(sort) > 0 {
		variables["sort"] = sort
	}
	if len(pageRange) > 0 {
		variables["range"] = pageRange
	}

	resp, err := query[integrationsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     integrationsQuery,
		Variables: variables,
	})
	if err != nil {
		return IntegrationsResult{}, err
	}

	return IntegrationsResult{Items: resp.Integrations, Total: resp.IntegrationsCount}, nil
}

func (c *Client) Integration(ctx context.Context, id string) (*Integration, error) {
	resp, err := query[integrationQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: integrationQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Integration, nil
}

func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	resp, err := query[currentUserQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: currentUserQuery,
	})
	if err != nil {
		return nil, err
	}
	return resp.CurrentUser, nil
}

func (c *Client) RefreshAuthToken(ctx context.Context, provider string, refreshToken string) (*RefreshAuthTokenResult, error) {
	cookieName := refreshCookieName(provider)
	if cookieName == "" {
		return nil, fmt.Errorf("unsupported auth provider %q", provider)
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("refresh token is required")
	}

	endpointURL, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint: %w", err)
	}
	c.httpClient.Jar.SetCookies(endpointURL, []*http.Cookie{{Name: cookieName, Value: refreshToken, Path: "/"}})

	resp, httpResp, err := queryWithHTTP[refreshAuthTokenMutationData](ctx, c.httpClient, c.endpoint, nil, graphqlRequest{Query: refreshAuthTokenMutation})
	if err != nil {
		return nil, err
	}
	if resp.RefreshAuthToken == nil {
		return nil, errors.New("refreshAuthToken returned no token")
	}
	resp.RefreshAuthToken.RefreshToken = extractRefreshToken(httpResp, cookieName)
	if resp.RefreshAuthToken.RefreshToken == "" {
		resp.RefreshAuthToken.RefreshToken = refreshToken
	}
	return resp.RefreshAuthToken, nil
}

func (c *Client) Logout(ctx context.Context, provider string, refreshToken string) error {
	cookieName := refreshCookieName(provider)
	if cookieName != "" && strings.TrimSpace(refreshToken) != "" {
		endpointURL, err := url.Parse(c.endpoint)
		if err != nil {
			return fmt.Errorf("parse endpoint: %w", err)
		}
		c.httpClient.Jar.SetCookies(endpointURL, []*http.Cookie{{Name: cookieName, Value: refreshToken, Path: "/"}})
	}
	resp, err := query[logoutMutationData](ctx, c.httpClient, c.endpoint, nil, graphqlRequest{Query: logoutMutation})
	if err != nil {
		return err
	}
	if resp.Logout == nil || !resp.Logout.Success {
		return errors.New("logout failed")
	}
	return nil
}

func (c *Client) EnableIntegration(ctx context.Context, id string) error {
	return c.integrationAction(ctx, id, "enable")
}

func (c *Client) DisableIntegration(ctx context.Context, id string) error {
	return c.integrationAction(ctx, id, "disable")
}

func (c *Client) DeleteIntegration(ctx context.Context, id string) error {
	resp, err := query[deleteIntegrationMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: deleteIntegrationMutation,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return err
	}
	if !resp.DeleteIntegration {
		return errors.New("integration delete failed")
	}
	return nil
}

func (c *Client) LogsForEntity(ctx context.Context, entityID string, pageRange []int) ([]Log, int, error) {
	executionStart, err := c.latestExecutionStart(ctx, entityID)
	if err != nil {
		return nil, 0, err
	}
	if executionStart == 0 {
		return nil, 0, nil
	}
	return c.LogsForAudit(ctx, entityID, "", executionStart, pageRange)
}

func (c *Client) AuditLogsForEntity(ctx context.Context, entityID string, pageRange []int) ([]AuditLog, error) {
	variables := map[string]any{
		"filter": map[string]any{
			"entity_id": entityID,
		},
		"sort": []string{"created_at", "DESC"},
	}
	if len(pageRange) > 0 {
		variables["range"] = pageRange
	}

	resp, err := query[auditLogsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     auditLogsQuery,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}

	return resp.AuditLogs, nil
}

func (c *Client) LogsForResource(ctx context.Context, resourceID string, pageRange []int) ([]Log, int, error) {
	return c.LogsForEntity(ctx, resourceID, pageRange)
}

func (c *Client) AuditLogsForResource(ctx context.Context, resourceID string, pageRange []int) ([]AuditLog, error) {
	return c.AuditLogsForEntity(ctx, resourceID, pageRange)
}

func (c *Client) LogsForAudit(ctx context.Context, resourceID string, auditLogID string, executionStart int, pageRange []int) ([]Log, int, error) {

	variables := map[string]any{
		"filter": map[string]any{
			"entity_id": resourceID,
		},
		"sort": []string{"created_at", "DESC"},
	}
	if executionStart > 0 {
		variables["filter"].(map[string]any)["execution_start"] = executionStart
	}
	if auditLogID != "" {
		variables["filter"].(map[string]any)["audit_log_id"] = auditLogID
	}
	if len(pageRange) > 0 {
		variables["range"] = pageRange
	}

	resp, err := query[logsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     logsQuery,
		Variables: variables,
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Logs, len(resp.Logs), nil
}

func (c *Client) latestExecutionStart(ctx context.Context, resourceID string) (int, error) {
	resp, err := query[logsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: logsQuery,
		Variables: map[string]any{
			"filter": map[string]any{
				"entity_id": resourceID,
			},
			"sort":  []string{"execution_start", "DESC"},
			"range": []int{0, 1},
		},
	})
	if err != nil {
		return 0, err
	}
	if len(resp.Logs) == 0 {
		return 0, nil
	}

	return resp.Logs[0].ExecutionStart, nil
}

func (c *Client) integrationAction(ctx context.Context, id string, action string) error {
	_, err := query[integrationActionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: integrationActionMutation,
		Variables: map[string]any{
			"id": id,
			"input": map[string]any{
				"action": action,
			},
		},
	})
	return err
}

func query[T any](ctx context.Context, httpClient *http.Client, endpoint string, tokenProvider TokenProvider, reqBody graphqlRequest) (T, error) {
	data, _, err := queryWithHTTP[T](ctx, httpClient, endpoint, tokenProvider, reqBody)
	return data, err
}

func queryWithHTTP[T any](ctx context.Context, httpClient *http.Client, endpoint string, tokenProvider TokenProvider, reqBody graphqlRequest) (T, *http.Response, error) {
	var zero T

	body, err := json.Marshal(reqBody)
	if err != nil {
		return zero, nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, nil, fmt.Errorf("build graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tokenProvider != nil {
		token, err := tokenProvider.Token(ctx)
		if err != nil {
			return zero, nil, fmt.Errorf("resolve bearer token: %w", err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, nil, fmt.Errorf("perform graphql request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, nil, fmt.Errorf("read graphql response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return zero, nil, fmt.Errorf("graphql request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var payload graphqlResponse[T]
	if err := json.Unmarshal(data, &payload); err != nil {
		return zero, nil, fmt.Errorf("decode graphql response: %w", err)
	}

	if len(payload.Errors) > 0 {
		messages := make([]string, 0, len(payload.Errors))
		for _, gqlErr := range payload.Errors {
			messages = append(messages, gqlErr.Message)
		}
		return zero, nil, errors.New(strings.Join(messages, "; "))
	}

	return payload.Data, resp, nil
}

func refreshCookieName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "microsoft":
		return "microsoft-refresh-token"
	case "github":
		return "github-refresh-token"
	case "guest":
		return "guest-token"
	default:
		return ""
	}
}

func extractRefreshToken(resp *http.Response, cookieName string) string {
	if resp == nil {
		return ""
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == cookieName {
			return cookie.Value
		}
	}
	return ""
}
