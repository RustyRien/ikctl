package client

import (
	"context"
	"encoding/json"
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
		fmt.Fprint(w, `{"data":{"resources":[{"id":"1","name":"redis-prod","state":"provisioned","status":"done","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","labels":["prod"],"template":{"id":"t1","name":"aws_redis","cloudResourceTypes":["redis"]},"workspace":{"id":"w1","name":"platform"},"storage":{"id":"st1","name":"terraform-state"},"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"},"integrationIds":[{"id":"i1","name":"aws-prod","integrationProvider":"aws","integrationType":"cloud"}],"secretIds":[{"id":"s1","name":"redis-password"}],"revisionNumber":2,"storagePath":"ik/state/redis-prod.tfstate","sourceCodeVersion":{"id":"scv1","identifier":"modules/redis:v1.2.3","sourceCodeFolder":"modules/redis","sourceCodeVersion":"v1.2.3","sourceCodeBranch":"","status":"ready"},"parents":[],"children":[],"variables":[{"name":"size","value":"small"}],"outputs":[{"name":"host","value":"redis.example"}],"dependencyTags":[{"name":"env","value":"prod"}],"dependencyConfig":[{"name":"region","value":"eu-west-1"}]}],"resourcesCount":1}}`)
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

func TestCurrentUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"currentUser":{"id":"u1","identifier":"alice","displayName":"Alice Doe","email":"alice@example.com","provider":"github","entityName":"user"}}}`)
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL, Token: "token-123"})
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		t.Fatalf("current user query: %v", err)
	}
	if user == nil || user.DisplayName != "Alice Doe" || user.Identifier != "alice" || user.Email != "alice@example.com" {
		t.Fatalf("user = %#v", user)
	}
	if user.Provider != "github" || user.EntityName != "user" {
		t.Fatalf("user = %#v", user)
	}
}

func TestRefreshAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("authorization header = %q", got)
		}
		cookie, err := r.Cookie("github-refresh-token")
		if err != nil {
			t.Fatalf("missing refresh cookie: %v", err)
		}
		if cookie.Value != "refresh-1" {
			t.Fatalf("refresh cookie = %q", cookie.Value)
		}
		w.Header().Set("Set-Cookie", "github-refresh-token=refresh-2; Path=/; HttpOnly")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"refreshAuthToken":{"token":"jwt-1","expiration":"2026-07-02 23:39:39.593976+00:00","provider":"github"}}}`)
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL})
	result, err := client.RefreshAuthToken(context.Background(), "github", "refresh-1")
	if err != nil {
		t.Fatalf("refresh auth token: %v", err)
	}
	if result.Token != "jwt-1" || result.Provider != "github" || result.RefreshToken != "refresh-2" {
		t.Fatalf("result = %#v", result)
	}
	if result.Expiration.Time.IsZero() {
		t.Fatalf("expiration was not parsed: %#v", result)
	}
}

func TestLogout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("guest-token")
		if err != nil {
			t.Fatalf("missing logout cookie: %v", err)
		}
		if cookie.Value != "guest-refresh" {
			t.Fatalf("logout cookie = %q", cookie.Value)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"logout":{"success":true}}}`)
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL})
	if err := client.Logout(context.Background(), "guest", "guest-refresh"); err != nil {
		t.Fatalf("logout: %v", err)
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

func TestResourceAndLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(req.Query, "GetResource"):
			fmt.Fprint(w, `{"data":{"resource":{"id":"r1","name":"redis-prod","description":"Managed Redis","state":"provisioned","status":"done","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","revisionNumber":2,"abstract":false,"storagePath":"ik-catalog/redis/prod.tfstate","labels":["prod"],"variables":[{"name":"size","value":"small"}],"outputs":[{"name":"host","value":"redis.example"}],"dependencyTags":[{"name":"env","value":"prod"}],"dependencyConfig":[{"name":"region","value":"eu-west-1"}],"template":{"id":"t1","name":"aws_redis","cloudResourceTypes":["redis"]},"workspace":{"id":"w1","name":"platform"},"storage":{"id":"st1","name":"terraform-state"},"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"},"integrationIds":[{"id":"i1","name":"aws-prod","integrationProvider":"aws","integrationType":"cloud"}],"secretIds":[{"id":"s1","name":"redis-password"}],"sourceCodeVersion":{"id":"scv1","identifier":"modules/redis:v1.2.3","sourceCodeFolder":"modules/redis","sourceCodeVersion":"v1.2.3","sourceCodeBranch":"","status":"ready"},"parents":[],"children":[]}}}`)
		case strings.Contains(req.Query, "ListLogs"):
			filter, _ := req.Variables["filter"].(map[string]any)
			if auditLogID, ok := filter["audit_log_id"]; ok {
				if auditLogID != "a1" {
					t.Fatalf("audit_log_id filter = %#v", auditLogID)
				}
				fmt.Fprint(w, `{"data":{"logs":[{"id":"l1","entityId":"r1","entity":"resource","revision":2,"auditLogId":"a1","level":"info","data":"apply started","createdAt":"2026-01-02T00:00:00Z","executionStart":123,"expireAt":"2026-01-03T00:00:00Z","traceId":"trace-1"}]}}`)
				return
			}
			if executionStart, ok := filter["execution_start"]; ok {
				if executionStart != float64(123) {
					t.Fatalf("execution_start filter = %#v", executionStart)
				}
				fmt.Fprint(w, `{"data":{"logs":[{"id":"l1","entityId":"r1","entity":"resource","revision":2,"auditLogId":"a1","level":"info","data":"apply started","createdAt":"2026-01-02T00:00:00Z","executionStart":123,"expireAt":"2026-01-03T00:00:00Z","traceId":"trace-1"}]}}`)
				return
			}

			fmt.Fprint(w, `{"data":{"logs":[{"id":"l1","entityId":"r1","entity":"resource","revision":2,"auditLogId":"a1","level":"info","data":"apply started","createdAt":"2026-01-02T00:00:00Z","executionStart":123,"expireAt":"2026-01-03T00:00:00Z","traceId":"trace-1"}]}}`)
		case strings.Contains(req.Query, "ListAuditLogs"):
			filter, _ := req.Variables["filter"].(map[string]any)
			if entityID := filter["entity_id"]; entityID != "r1" {
				t.Fatalf("audit entity_id filter = %#v", entityID)
			}
			fmt.Fprint(w, `{"data":{"auditLogs":[{"id":"a1","model":"resource","userId":"u1","action":"apply","entityId":"r1","createdAt":"2026-01-02T00:00:00Z","revisionNumber":2,"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"}}]}}`)
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL, Token: "token-123"})

	resource, err := client.Resource(context.Background(), "r1")
	if err != nil {
		t.Fatalf("resource query: %v", err)
	}
	if resource == nil || resource.SourceCodeVersion == nil || resource.SourceCodeVersion.Identifier != "modules/redis:v1.2.3" {
		t.Fatalf("resource = %#v", resource)
	}

	logs, total, err := client.LogsForEntity(context.Background(), "r1", []int{0, 50})
	if err != nil {
		t.Fatalf("logs query: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].Data != "apply started" {
		t.Fatalf("logs = %#v total=%d", logs, total)
	}
	if logs[0].AuditLogID != "a1" || logs[0].ExecutionStart != 123 {
		t.Fatalf("log = %#v", logs[0])
	}

	auditLogs, err := client.AuditLogsForEntity(context.Background(), "r1", []int{0, 50})
	if err != nil {
		t.Fatalf("audit logs query: %v", err)
	}
	if len(auditLogs) != 1 || auditLogs[0].Action != "apply" {
		t.Fatalf("auditLogs = %#v", auditLogs)
	}
	if auditLogs[0].Creator == nil || auditLogs[0].Creator.DisplayName != "Alice" {
		t.Fatalf("auditLog creator = %#v", auditLogs[0].Creator)
	}

	auditScopedLogs, total, err := client.LogsForAudit(context.Background(), "r1", "a1", 0, []int{0, 50})
	if err != nil {
		t.Fatalf("audit scoped logs query: %v", err)
	}
	if total != 1 || len(auditScopedLogs) != 1 || auditScopedLogs[0].AuditLogID != "a1" {
		t.Fatalf("audit scoped logs = %#v total=%d", auditScopedLogs, total)
	}
}

func TestTemplateAndTree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(req.Query, "GetTemplate"):
			fmt.Fprint(w, `{"data":{"template":{"id":"t1","name":"aws_redis","description":"Redis template","documentation":"https://example.invalid/docs","template":"resource \"aws_elasticache_cluster\" \"this\" {}","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","cloudResourceTypes":["redis"],"labels":["cache","prod"],"status":"ready","abstract":false,"configuration":{"tier":"backend"},"revisionNumber":3,"resourcesCount":7,"sourceCodeVersionsCount":2,"entityName":"template","creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"},"parents":[{"id":"tp1","name":"base_template","abstract":true,"cloudResourceTypes":["network"],"entityName":"template"}],"children":[{"id":"tc1","name":"redis_replica","abstract":false,"cloudResourceTypes":["redis"],"entityName":"template"}]}}}`)
		case strings.Contains(req.Query, "TemplateTree"):
			if req.Variables["id"] != "t1" {
				t.Fatalf("template tree id = %#v", req.Variables["id"])
			}
			if req.Variables["direction"] != "children" {
				t.Fatalf("template tree direction = %#v", req.Variables["direction"])
			}
			fmt.Fprint(w, `{"data":{"templateTree":{"id":"t1","nodeId":"t1","name":"aws_redis","status":"ready","children":[{"id":"t2","nodeId":"t2","name":"aws_redis_cache","status":"ready","children":[]}]}}}`)
		case strings.Contains(req.Query, "ResourceTree"):
			if req.Variables["id"] != "r1" {
				t.Fatalf("resource tree id = %#v", req.Variables["id"])
			}
			if req.Variables["direction"] != "children" {
				t.Fatalf("resource tree direction = %#v", req.Variables["direction"])
			}
			fmt.Fprint(w, `{"data":{"resourceTree":{"id":"r1","nodeId":"r1","name":"redis-prod","state":"provisioned","status":"ready","templateName":"aws_redis","children":[{"id":"r2","nodeId":"r2","name":"redis-replica","state":"provisioned","status":"ready","templateName":"aws_redis_replica","children":[]}]}}}`)
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL, Token: "token-123"})

	template, err := client.Template(context.Background(), "t1")
	if err != nil {
		t.Fatalf("template query: %v", err)
	}
	if template == nil || template.Name != "aws_redis" {
		t.Fatalf("template = %#v", template)
	}
	if template.Creator == nil || template.Creator.DisplayName != "Alice" || template.ResourcesCount != 7 || len(template.Parents) != 1 {
		t.Fatalf("template details = %#v", template)
	}

	tree, err := client.TemplateTree(context.Background(), "t1", "children")
	if err != nil {
		t.Fatalf("template tree query: %v", err)
	}
	if tree == nil || tree.Name != "aws_redis" || len(tree.Children) != 1 || tree.Children[0].Name != "aws_redis_cache" {
		t.Fatalf("tree = %#v", tree)
	}

	resourceTree, err := client.ResourceTree(context.Background(), "r1", "children")
	if err != nil {
		t.Fatalf("resource tree query: %v", err)
	}
	if resourceTree == nil || resourceTree.Name != "redis-prod" || len(resourceTree.Children) != 1 || resourceTree.Children[0].Name != "redis-replica" {
		t.Fatalf("resourceTree = %#v", resourceTree)
	}
}

func TestIntegrationActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(req.Query, "GetIntegration"):
			fmt.Fprint(w, `{"data":{"integration":{"id":"i1","name":"aws-prod","description":"AWS","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","integrationProvider":"aws","integrationType":"cloud"}}}`)
		case strings.Contains(req.Query, "IntegrationAction"):
			input, _ := req.Variables["input"].(map[string]any)
			action := input["action"]
			if req.Variables["id"] != "i1" {
				t.Fatalf("action id = %#v", req.Variables["id"])
			}
			if action != "enable" && action != "disable" {
				t.Fatalf("unexpected action = %#v", action)
			}
			fmt.Fprintf(w, `{"data":{"integrationAction":{"id":"i1","entityName":"integration","status":"%s"}}}`, action)
		case strings.Contains(req.Query, "DeleteIntegration"):
			if req.Variables["id"] != "i1" {
				t.Fatalf("delete id = %#v", req.Variables["id"])
			}
			fmt.Fprint(w, `{"data":{"deleteIntegration":true}}`)
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL, Token: "token-123"})

	if err := client.EnableIntegration(context.Background(), "i1"); err != nil {
		t.Fatalf("enable integration: %v", err)
	}
	if err := client.DisableIntegration(context.Background(), "i1"); err != nil {
		t.Fatalf("disable integration: %v", err)
	}
	if err := client.DeleteIntegration(context.Background(), "i1"); err != nil {
		t.Fatalf("delete integration: %v", err)
	}
}
