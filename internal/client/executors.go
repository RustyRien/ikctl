package client

import (
	"context"
	"errors"
)

const executorsQuery = `query ListExecutors($filter: JSON, $sort: [String!], $range: [Int!]) {
  executors(filter: $filter, sort: $sort, range: $range) {
    id
    name
    entityName
    description
    runtime
    commandArgs
    sourceCodeVersion
    sourceCodeBranch
    sourceCodeFolder
    storagePath
    labels
    state
    status
    revisionNumber
    createdAt
    updatedAt
    isFavorite
    sourceCode {
      id
      identifier
      sourceCodeUrl
      entityName
    }
    integrationIds {
      id
      name
      entityName
      integrationProvider
      integrationType
    }
    secretIds {
      id
      name
      entityName
    }
    storage {
      id
      name
      entityName
      storageProvider
      storageType
    }
    creator {
      id
      identifier
      email
      displayName
    }
  }
  executorsCount(filter: $filter)
}`

const executorQuery = `query GetExecutor($id: UUID!) {
  executor(id: $id) {
    id
    name
    entityName
    description
    runtime
    commandArgs
    sourceCodeVersion
    sourceCodeBranch
    sourceCodeFolder
    storagePath
    labels
    state
    status
    revisionNumber
    createdAt
    updatedAt
    isFavorite
    sourceCode {
      id
      identifier
      sourceCodeUrl
      sourceCodeProvider
      sourceCodeLanguage
      entityName
    }
    integrationIds {
      id
      name
      entityName
      integrationProvider
      integrationType
    }
    secretIds {
      id
      name
      entityName
      secretProvider
      secretType
    }
    storage {
      id
      name
      entityName
      storageProvider
      storageType
    }
    creator {
      id
      identifier
      email
      displayName
    }
  }
}`

const executorActionsQuery = `query ExecutorActions($id: UUID!) {
  executorActions(id: $id)
}`

const updateExecutorMutation = `mutation UpdateExecutor($id: UUID!, $input: ExecutorUpdateInput!) {
  updateExecutor(id: $id, input: $input) {
    id
    name
    entityName
  }
}`

const executorActionMutation = `mutation ExecutorAction($id: UUID!, $input: ExecutorActionInput!) {
  executorAction(id: $id, input: $input) {
    id
    entityName
    status
  }
}`

const deleteExecutorMutation = `mutation DeleteExecutor($id: UUID!) {
  deleteExecutor(id: $id)
}`

func (c *Client) Executors(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (ExecutorsResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[executorsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     executorsQuery,
		Variables: variables,
	})
	if err != nil {
		return ExecutorsResult{}, err
	}

	return ExecutorsResult{Items: resp.Executors, Total: resp.ExecutorsCount}, nil
}

func (c *Client) Executor(ctx context.Context, id string) (*Executor, error) {
	resp, err := query[executorQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: executorQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Executor, nil
}

func (c *Client) ExecutorActions(ctx context.Context, id string) ([]string, error) {
	resp, err := query[executorActionsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: executorActionsQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.ExecutorActions, nil
}

func (c *Client) UpdateExecutor(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateExecutorMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateExecutorMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
}

func (c *Client) EnableExecutor(ctx context.Context, id string) error {
	return c.executorAction(ctx, id, "enable")
}

func (c *Client) DryRunExecutor(ctx context.Context, id string) error {
	return c.executorAction(ctx, id, "dryrun")
}

func (c *Client) DisableExecutor(ctx context.Context, id string) error {
	return c.executorAction(ctx, id, "disable")
}

func (c *Client) ExecutorAction(ctx context.Context, id string, action string) error {
	return c.executorAction(ctx, id, action)
}

func (c *Client) DeleteExecutor(ctx context.Context, id string) error {
	resp, err := query[deleteExecutorMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: deleteExecutorMutation,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return err
	}
	if !resp.DeleteExecutor {
		return errors.New("executor delete failed")
	}
	return nil
}

func (c *Client) executorAction(ctx context.Context, id string, action string) error {
	_, err := query[executorActionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: executorActionMutation,
		Variables: map[string]any{
			"id": id,
			"input": map[string]any{
				"action": action,
			},
		},
	})
	return err
}
