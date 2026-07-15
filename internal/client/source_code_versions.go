package client

import (
	"context"
	"errors"
)

const sourceCodeVersionsQuery = `query ListSourceCodeVersions($filter: JSON, $sort: [String!], $range: [Int!]) {
  sourceCodeVersions(filter: $filter, sort: $sort, range: $range) {
    id
    identifier
    sourceCodeVersion
    sourceCodeBranch
    sourceCodeFolder
    description
    labels
    status
    revisionNumber
    resourcesCount
    createdAt
    updatedAt
    entityName
    template {
      id
      name
      abstract
      cloudResourceTypes
      entityName
    }
    sourceCode {
      id
      identifier
      sourceCodeUrl
      sourceCodeProvider
      sourceCodeLanguage
      status
      entityName
    }
    creator {
      id
      identifier
      email
      displayName
    }
  }
  sourceCodeVersionsCount(filter: $filter)
}`

const sourceCodeVersionQuery = `query GetSourceCodeVersion($id: UUID!) {
  sourceCodeVersion(id: $id) {
    id
    identifier
    sourceCodeVersion
    sourceCodeBranch
    sourceCodeFolder
    variables
    outputs
    codeSnapshot
    description
    labels
    status
    revisionNumber
    resourcesCount
    createdAt
    updatedAt
    entityName
    template {
      id
      name
      abstract
      cloudResourceTypes
      entityName
    }
    sourceCode {
      id
      identifier
      sourceCodeUrl
      sourceCodeProvider
      sourceCodeLanguage
      status
      entityName
    }
    creator {
      id
      identifier
      email
      displayName
    }
  }
}`

const sourceCodeVersionActionMutation = `mutation SourceCodeVersionAction($id: UUID!, $input: SourceCodeVersionActionInput!) {
  sourceCodeVersionAction(id: $id, input: $input) {
    id
    entityName
    status
  }
}`

const deleteSourceCodeVersionMutation = `mutation DeleteSourceCodeVersion($id: UUID!) {
  deleteSourceCodeVersion(id: $id)
}`

const updateSourceCodeVersionMutation = `mutation UpdateSourceCodeVersion($id: UUID!, $input: SourceCodeVersionUpdateInput!) {
  updateSourceCodeVersion(id: $id, input: $input) {
    id
    identifier
    entityName
  }
}`

func (c *Client) SourceCodeVersions(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (SourceCodeVersionsResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[sourceCodeVersionsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     sourceCodeVersionsQuery,
		Variables: variables,
	})
	if err != nil {
		return SourceCodeVersionsResult{}, err
	}

	return SourceCodeVersionsResult{Items: resp.SourceCodeVersions, Total: resp.SourceCodeVersionsCount}, nil
}

func (c *Client) SourceCodeVersion(ctx context.Context, id string) (*SourceCodeVersion, error) {
	resp, err := query[sourceCodeVersionQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: sourceCodeVersionQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.SourceCodeVersion, nil
}

func (c *Client) EnableSourceCodeVersion(ctx context.Context, id string) error {
	return c.sourceCodeVersionAction(ctx, id, "enable")
}

func (c *Client) DisableSourceCodeVersion(ctx context.Context, id string) error {
	return c.sourceCodeVersionAction(ctx, id, "disable")
}

func (c *Client) SyncSourceCodeVersion(ctx context.Context, id string) error {
	return c.sourceCodeVersionAction(ctx, id, "sync")
}

func (c *Client) DeleteSourceCodeVersion(ctx context.Context, id string) error {
	resp, err := query[deleteSourceCodeVersionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: deleteSourceCodeVersionMutation,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return err
	}
	if !resp.DeleteSourceCodeVersion {
		return errors.New("source code version delete failed")
	}
	return nil
}

func (c *Client) UpdateSourceCodeVersion(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateSourceCodeVersionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateSourceCodeVersionMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
}

func (c *Client) sourceCodeVersionAction(ctx context.Context, id string, action string) error {
	_, err := query[sourceCodeVersionActionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: sourceCodeVersionActionMutation,
		Variables: map[string]any{
			"id": id,
			"input": map[string]any{
				"action": action,
			},
		},
	})
	return err
}
