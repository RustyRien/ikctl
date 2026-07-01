package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/config"
)

func TestResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"resources":[{"id":"1","name":"redis-prod","state":"provisioned","status":"done","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","labels":["prod"],"template":{"id":"t1","name":"aws_redis","cloudResourceTypes":["redis"]},"workspace":{"id":"w1","name":"platform"},"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"},"integrationIds":[{"id":"i1","name":"aws-prod","integrationProvider":"aws","integrationType":"cloud"}]}],"resourcesCount":1}}`)
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL, Token: "token-123"})
	result, err := client.Resources(context.Background(), map[string]any{"state": "provisioned"}, []string{"created_at", "DESC"}, []int{0, 25})
	if err != nil {
		t.Fatalf("resources query: %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("total = %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "redis-prod" {
		t.Fatalf("items = %#v", result.Items)
	}
	if result.Items[0].CreatedAt != time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("createdAt = %v", result.Items[0].CreatedAt)
	}
}

func TestResourcesGraphQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"errors":[{"message":"Not authenticated"}]}`)
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL})
	_, err := client.Resources(context.Background(), nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "Not authenticated") {
		t.Fatalf("err = %v", err)
	}
}
