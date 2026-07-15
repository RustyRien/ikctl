package client

import (
	"context"
	"errors"
)

const integrationsQuery = `query ListIntegrations($filter: JSON, $sort: [String!], $range: [Int!]) {
  integrations(filter: $filter, sort: $sort, range: $range) {
    id
    name
    entityName
    description
    createdAt
    updatedAt
    integrationProvider
    integrationType
    labels
    configuration
    status
    creator {
      id
      identifier
      email
      displayName
    }
  }
  integrationsCount(filter: $filter)
}`

const integrationQuery = `query GetIntegration($id: UUID!) {
  integration(id: $id) {
    id
    name
    entityName
    description
    createdAt
    updatedAt
    integrationProvider
    integrationType
    labels
    configuration
    status
    creator {
      id
      identifier
      email
      displayName
    }
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

const updateIntegrationMutation = `mutation UpdateIntegration($id: UUID!, $input: IntegrationUpdateInput!) {
  updateIntegration(id: $id, input: $input) {
    id
    name
    entityName
    integrationProvider
  }
}`

func (c *Client) Integrations(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (IntegrationsResult, error) {
	variables := listVariables(filter, sort, pageRange)

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

func (c *Client) UpdateIntegration(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateIntegrationMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateIntegrationMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
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
