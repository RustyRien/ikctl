package client

import "context"

const secretsQuery = `query ListSecrets($filter: JSON, $sort: [String!], $range: [Int!]) {
  secrets(filter: $filter, sort: $sort, range: $range) {
    id
    name
    entityName
    description
    secretType
    secretProvider
    labels
    configuration
    state
    status
    resourcesCount
    executorsCount
    revisionNumber
    createdAt
    updatedAt
    integration {
      id
      name
      entityName
      integrationProvider
      integrationType
    }
    creator {
      id
      identifier
      email
      displayName
    }
  }
  secretsCount(filter: $filter)
}`

const secretQuery = `query GetSecret($id: UUID!) {
  secret(id: $id) {
    id
    name
    entityName
    description
    secretType
    secretProvider
    labels
    configuration
    state
    status
    resourcesCount
    executorsCount
    revisionNumber
    createdAt
    updatedAt
    integration {
      id
      name
      entityName
      integrationProvider
      integrationType
    }
    creator {
      id
      identifier
      email
      displayName
    }
  }
}`

const updateSecretMutation = `mutation UpdateSecret($id: UUID!, $input: SecretUpdateInput!) {
  updateSecret(id: $id, input: $input) {
    id
    name
    entityName
    secretProvider
  }
}`

func (c *Client) Secrets(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (SecretsResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[secretsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     secretsQuery,
		Variables: variables,
	})
	if err != nil {
		return SecretsResult{}, err
	}

	return SecretsResult{Items: resp.Secrets, Total: resp.SecretsCount}, nil
}

func (c *Client) Secret(ctx context.Context, id string) (*Secret, error) {
	resp, err := query[secretQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: secretQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Secret, nil
}

func (c *Client) UpdateSecret(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateSecretMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateSecretMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
}
