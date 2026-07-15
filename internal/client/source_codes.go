package client

import (
	"context"
	"errors"
)

const sourceCodesQuery = `query ListSourceCodes($filter: JSON, $sort: [String!], $range: [Int!]) {
  sourceCodes(filter: $filter, sort: $sort, range: $range) {
    id
    identifier
    description
    sourceCodeUrl
    sourceCodeProvider
    sourceCodeLanguage
    labels
    status
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
  sourceCodesCount(filter: $filter)
}`

const sourceCodeQuery = `query GetSourceCode($id: UUID!) {
  sourceCode(id: $id) {
    id
    identifier
    description
    sourceCodeUrl
    sourceCodeProvider
    sourceCodeLanguage
    integrationId
    gitTags
    gitTagMessages
    gitBranches
    gitBranchMessages
    gitFoldersMap
    labels
    status
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

const sourceCodeActionMutation = `mutation SourceCodeAction($id: UUID!, $input: SourceCodeActionInput!) {
  sourceCodeAction(id: $id, input: $input) {
    id
    entityName
    status
  }
}`

const deleteSourceCodeMutation = `mutation DeleteSourceCode($id: UUID!) {
  deleteSourceCode(id: $id)
}`

const updateSourceCodeMutation = `mutation UpdateSourceCode($id: UUID!, $input: SourceCodeUpdateInput!) {
  updateSourceCode(id: $id, input: $input) {
    id
    identifier
    entityName
  }
}`

func (c *Client) SourceCodes(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (SourceCodesResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[sourceCodesQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     sourceCodesQuery,
		Variables: variables,
	})
	if err != nil {
		return SourceCodesResult{}, err
	}

	return SourceCodesResult{Items: resp.SourceCodes, Total: resp.SourceCodesCount}, nil
}

func (c *Client) SourceCode(ctx context.Context, id string) (*SourceCode, error) {
	resp, err := query[sourceCodeQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: sourceCodeQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.SourceCode, nil
}

func (c *Client) EnableSourceCode(ctx context.Context, id string) error {
	return c.sourceCodeAction(ctx, id, "enable")
}

func (c *Client) DisableSourceCode(ctx context.Context, id string) error {
	return c.sourceCodeAction(ctx, id, "disable")
}

func (c *Client) SyncSourceCode(ctx context.Context, id string) error {
	return c.sourceCodeAction(ctx, id, "sync")
}

func (c *Client) DeleteSourceCode(ctx context.Context, id string) error {
	resp, err := query[deleteSourceCodeMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: deleteSourceCodeMutation,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return err
	}
	if !resp.DeleteSourceCode {
		return errors.New("source code delete failed")
	}
	return nil
}

func (c *Client) UpdateSourceCode(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateSourceCodeMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateSourceCodeMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
}

func (c *Client) sourceCodeAction(ctx context.Context, id string, action string) error {
	_, err := query[sourceCodeActionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: sourceCodeActionMutation,
		Variables: map[string]any{
			"id": id,
			"input": map[string]any{
				"action": action,
			},
		},
	})
	return err
}
