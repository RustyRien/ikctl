package client

import "context"

const storagesQuery = `query ListStorages($filter: JSON, $sort: [String!], $range: [Int!]) {
  storages(filter: $filter, sort: $sort, range: $range) {
    id
    name
    storageType
    storageProvider
    description
    labels
    state
    status
    resourcesCount
    executorsCount
    revisionNumber
    createdAt
    updatedAt
    entityName
    integration {
      id
      name
      entityName
      integrationProvider
    }
    creator {
      id
      identifier
      email
      displayName
    }
  }
  storagesCount(filter: $filter)
}`

const storageQuery = `query GetStorage($id: UUID!) {
  storage(id: $id) {
    id
    name
    storageType
    storageProvider
    configuration
    description
    labels
    state
    status
    resourcesCount
    executorsCount
    revisionNumber
    createdAt
    updatedAt
    entityName
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

const updateStorageMutation = `mutation UpdateStorage($id: UUID!, $input: StorageUpdateInput!) {
  updateStorage(id: $id, input: $input) {
    id
    name
    entityName
  }
}`

func (c *Client) Storages(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (StoragesResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[storagesQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     storagesQuery,
		Variables: variables,
	})
	if err != nil {
		return StoragesResult{}, err
	}

	return StoragesResult{Items: resp.Storages, Total: resp.StoragesCount}, nil
}

func (c *Client) Storage(ctx context.Context, id string) (*Storage, error) {
	resp, err := query[storageQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: storageQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Storage, nil
}

func (c *Client) UpdateStorage(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateStorageMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateStorageMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
}
