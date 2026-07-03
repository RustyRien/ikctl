package client

import "context"

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
