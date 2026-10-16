package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// Request is a GraphQL request document with variables.
type Request struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

// GraphQLError is a single error entry of a GraphQL response.
type GraphQLError struct {
	Message    string         `json:"message"`
	Path       []any          `json:"path,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors"`
}

// Do executes a GraphQL request and decodes the "data" field into out
// (a pointer). If the response contains errors, an *APIError is returned.
func (c *Client) Do(ctx context.Context, req Request, out any) error {
	if out == nil {
		return errNilOut
	}
	if c.log.Enabled(ctx, slog.LevelDebug) {
		c.log.Debug("graphql request", "query", compactQuery(req.Query), "variables", req.Variables)
	}
	body, err := c.doHTTP(ctx, http.MethodPost, GraphQLPath, req)
	if err != nil {
		return err
	}
	var resp graphQLResponse
	if err := decodeJSON(body, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return toAPIError(resp.Errors)
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return &APIError{Code: CodeInternal, Message: "empty response data"}
	}
	if raw, ok := out.(*json.RawMessage); ok {
		*raw = resp.Data
		return nil
	}
	return decodeJSON(resp.Data, out)
}

// GraphQL executes a raw GraphQL query or mutation. The "data" field of the
// response is decoded into out (e.g. *map[string]any or *json.RawMessage).
func (c *Client) GraphQL(ctx context.Context, query string, vars map[string]any, out any) error {
	return c.Do(ctx, Request{Query: query, Variables: vars}, out)
}

func toAPIError(errs []GraphQLError) error {
	first := errs[0]
	e := &APIError{Code: CodeInternal, Message: first.Message}
	if code, ok := first.Extensions["code"].(string); ok && code != "" {
		e.Code = code
	}
	for _, p := range first.Path {
		e.Path = append(e.Path, fmt.Sprint(p))
	}
	if strings.HasPrefix(first.Message, "Not authenticated") && e.Code == CodeInternal {
		e.Code = CodeAccessDenied
	}
	if len(errs) > 1 {
		msgs := make([]string, 0, len(errs)-1)
		for _, other := range errs[1:] {
			msgs = append(msgs, other.Message)
		}
		e.Message += "; " + strings.Join(msgs, "; ")
	}
	return e
}

func compactQuery(q string) string {
	return strings.Join(strings.Fields(q), " ")
}
