package client

import "context"

const workersQuery = `query ListWorkers($filter: JSON, $sort: [String!], $range: [Int!]) {
  workers(filter: $filter, sort: $sort, range: $range) {
    id
    name
    host
    hostMetadata
    status
    currentTask
    tasksCompleted
    createdAt
    updatedAt
  }
  workersCount(filter: $filter)
}`

const workerQuery = `query GetWorker($id: UUID!) {
  worker(id: $id) {
    id
    name
    host
    hostMetadata
    status
    currentTask
    tasksCompleted
    createdAt
    updatedAt
  }
}`

func (c *Client) Workers(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (WorkersResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[workersQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     workersQuery,
		Variables: variables,
	})
	if err != nil {
		return WorkersResult{}, err
	}

	return WorkersResult{Items: resp.Workers, Total: resp.WorkersCount}, nil
}

func (c *Client) Worker(ctx context.Context, id string) (*Worker, error) {
	resp, err := query[workerQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: workerQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Worker, nil
}
