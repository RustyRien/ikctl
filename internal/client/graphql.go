package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func query[T any](ctx context.Context, httpClient *http.Client, endpoint string, tokenProvider TokenProvider, reqBody graphqlRequest) (T, error) {
	data, _, err := queryWithHTTP[T](ctx, httpClient, endpoint, tokenProvider, reqBody)
	return data, err
}

func queryWithHTTP[T any](ctx context.Context, httpClient *http.Client, endpoint string, tokenProvider TokenProvider, reqBody graphqlRequest) (T, *http.Response, error) {
	var zero T

	body, err := json.Marshal(reqBody)
	if err != nil {
		return zero, nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, nil, fmt.Errorf("build graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tokenProvider != nil {
		token, err := tokenProvider.Token(ctx)
		if err != nil {
			return zero, nil, fmt.Errorf("resolve bearer token: %w", err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, nil, fmt.Errorf("perform graphql request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, nil, fmt.Errorf("read graphql response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return zero, nil, fmt.Errorf("graphql request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var payload graphqlResponse[T]
	if err := json.Unmarshal(data, &payload); err != nil {
		return zero, nil, fmt.Errorf("decode graphql response: %w", err)
	}

	if len(payload.Errors) > 0 {
		messages := make([]string, 0, len(payload.Errors))
		for _, gqlErr := range payload.Errors {
			messages = append(messages, gqlErr.Message)
		}
		return zero, nil, errors.New(strings.Join(messages, "; "))
	}

	return payload.Data, resp, nil
}
