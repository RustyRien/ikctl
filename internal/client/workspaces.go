package client

import (
	"context"
	"errors"
)

const workspacesQuery = `query ListWorkspaces($filter: JSON, $sort: [String!], $range: [Int!]) {
  workspaces(filter: $filter, sort: $sort, range: $range) {
    id
    name
    workspaceProvider
    status
    description
    labels
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
  workspacesCount(filter: $filter)
}`

const workspaceQuery = `query GetWorkspace($id: UUID!) {
  workspace(id: $id) {
    id
    name
    workspaceProvider
    configuration
    status
    description
    labels
    resourcesCount
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

const updateWorkspaceMutation = `mutation UpdateWorkspace($id: UUID!, $input: WorkspaceUpdateInput!) {
  updateWorkspace(id: $id, input: $input) {
    id
    name
    entityName
  }
}`

const deleteWorkspaceMutation = `mutation DeleteWorkspace($id: UUID!) {
  deleteWorkspace(id: $id)
}`

func (c *Client) Workspaces(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (WorkspacesResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[workspacesQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     workspacesQuery,
		Variables: variables,
	})
	if err != nil {
		return WorkspacesResult{}, err
	}

	return WorkspacesResult{Items: resp.Workspaces, Total: resp.WorkspacesCount}, nil
}

func (c *Client) Workspace(ctx context.Context, id string) (*Workspace, error) {
	resp, err := query[workspaceQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: workspaceQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Workspace, nil
}

func (c *Client) UpdateWorkspace(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateWorkspaceMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateWorkspaceMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
}

func (c *Client) DeleteWorkspace(ctx context.Context, id string) error {
	resp, err := query[deleteWorkspaceMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: deleteWorkspaceMutation,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return err
	}
	if !resp.DeleteWorkspace {
		return errors.New("workspace delete failed")
	}
	return nil
}
