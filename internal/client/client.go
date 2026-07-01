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
  }
  resourcesCount(filter: $filter)
}`

type Client struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

func New(cfg config.Config) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}

	return &Client{
		endpoint: cfg.Endpoint + "/api/graphql",
		token:    cfg.Token,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
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

	resp, err := query[resourcesQueryData](ctx, c.httpClient, c.endpoint, c.token, graphqlRequest{
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

func query[T any](ctx context.Context, httpClient *http.Client, endpoint string, token string, reqBody graphqlRequest) (T, error) {
	var zero T

	body, err := json.Marshal(reqBody)
	if err != nil {
		return zero, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("build graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, fmt.Errorf("perform graphql request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("read graphql response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return zero, fmt.Errorf("graphql request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var payload graphqlResponse[T]
	if err := json.Unmarshal(data, &payload); err != nil {
		return zero, fmt.Errorf("decode graphql response: %w", err)
	}

	if len(payload.Errors) > 0 {
		messages := make([]string, 0, len(payload.Errors))
		for _, gqlErr := range payload.Errors {
			messages = append(messages, gqlErr.Message)
		}
		return zero, errors.New(strings.Join(messages, "; "))
	}

	return payload.Data, nil
}
