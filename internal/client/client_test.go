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
			fmt.Fprint(w, `{"data":{"resource":{"id":"r1","name":"redis-prod","entityName":"resource","description":"Managed Redis","state":"provisioned","status":"done","actions":["has_temporary_state","approve"],"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","revisionNumber":2,"abstract":false,"storagePath":"ik-catalog/redis/prod.tfstate","labels":["prod"],"variables":[{"name":"size","value":"small"}],"outputs":[{"name":"host","value":"redis.example"}],"dependencyTags":[{"name":"env","value":"prod"}],"dependencyConfig":[{"name":"region","value":"eu-west-1"}],"template":{"id":"t1","name":"aws_redis","cloudResourceTypes":["redis"]},"workspace":{"id":"w1","name":"platform"},"storage":{"id":"st1","name":"terraform-state"},"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"},"integrationIds":[{"id":"i1","name":"aws-prod","integrationProvider":"aws","integrationType":"cloud"}],"secretIds":[{"id":"s1","name":"redis-password"}],"sourceCodeVersion":{"id":"scv1","identifier":"modules/redis:v1.2.3","sourceCodeFolder":"modules/redis","sourceCodeVersion":"v1.2.3","sourceCodeBranch":"","status":"ready"},"parents":[],"children":[]}}}`)
		case strings.Contains(req.Query, "UpdateResource"):
			if req.Variables["id"] != "r1" {
				t.Fatalf("update resource id = %#v", req.Variables["id"])
			}
			fmt.Fprint(w, `{"data":{"updateResource":{"id":"r1","name":"redis-prod","entityName":"resource"}}}`)
		case strings.Contains(req.Query, "ResourceActions"):
			fmt.Fprint(w, `{"data":{"resourceActions":["approve","has_temporary_state"]}}`)
		case strings.Contains(req.Query, "ResourceTempState"):
			fmt.Fprint(w, `{"data":{"resourceTempStateByResource":{"id":"ts1","resourceId":"r1","value":{"name":"redis-prod","description":"Managed Redis updated","integration_ids":["i1"],"secret_ids":["s1"],"storage_id":"st1","storage_path":"ik-catalog/redis/prod.tfstate","variables":[{"name":"size","value":"large"}],"dependency_tags":[{"name":"env","value":"prod"}],"dependency_config":[{"name":"region","value":"eu-west-1"}],"labels":["prod"],"workspace_id":"w1","source_code_version_id":"scv1"},"createdAt":"2026-01-02T00:00:00Z","updatedAt":"2026-01-02T01:00:00Z"}}}`)
		case strings.Contains(req.Query, "ResourceAction"):
			input, _ := req.Variables["input"].(map[string]any)
			action := input["action"]
			if action != "approve" && action != "reject" && action != "execute" && action != "disable" {
				t.Fatalf("resource action = %#v", action)
			}
			fmt.Fprintf(w, `{"data":{"resourceAction":{"id":"r1","entityName":"resource","status":"%s"}}}`, action)
		case strings.Contains(req.Query, "DeleteResource"):
			if req.Variables["id"] != "r1" {
				t.Fatalf("delete resource id = %#v", req.Variables["id"])
			}
			fmt.Fprint(w, `{"data":{"deleteResource":true}}`)
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
	if err := client.UpdateResource(context.Background(), "r1", map[string]any{"name": "redis-prod"}); err != nil {
		t.Fatalf("update resource: %v", err)
	}
	actions, err := client.ResourceActions(context.Background(), "r1")
	if err != nil {
		t.Fatalf("resource actions: %v", err)
	}
	if len(actions) != 2 || actions[0] != "approve" {
		t.Fatalf("actions = %#v", actions)
	}
	tempState, err := client.ResourceTempState(context.Background(), "r1")
	if err != nil {
		t.Fatalf("resource temp state: %v", err)
	}
	if tempState == nil || tempState.Value["description"] != "Managed Redis updated" {
		t.Fatalf("tempState = %#v", tempState)
	}
	if err := client.ApproveResource(context.Background(), "r1"); err != nil {
		t.Fatalf("approve resource: %v", err)
	}
	if err := client.RejectResource(context.Background(), "r1"); err != nil {
		t.Fatalf("reject resource: %v", err)
	}
	if err := client.ResourceAction(context.Background(), "r1", "execute"); err != nil {
		t.Fatalf("resource action execute: %v", err)
	}
	if err := client.DeleteResource(context.Background(), "r1"); err != nil {
		t.Fatalf("delete resource: %v", err)
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
		case strings.Contains(req.Query, "ListResources"):
			filter, _ := req.Variables["filter"].(map[string]any)
			if templateID := filter["template_id"]; templateID != "t1" {
				t.Fatalf("template resource filter = %#v", templateID)
			}
			fmt.Fprint(w, `{"data":{"resources":[{"id":"r1","name":"redis-prod","state":"provisioned","status":"ready","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","labels":["prod"],"template":{"id":"t1","name":"aws_redis","cloudResourceTypes":["redis"]},"workspace":{"id":"w1","name":"platform"},"storage":{"id":"st1","name":"terraform-state"},"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"},"integrationIds":[],"secretIds":[],"revisionNumber":2,"storagePath":"ik/state/redis-prod.tfstate","sourceCodeVersion":{"id":"scv1","identifier":"modules/redis:v1.2.3","sourceCodeFolder":"modules/redis","sourceCodeVersion":"v1.2.3","sourceCodeBranch":"","status":"ready"},"parents":[],"children":[],"variables":[],"outputs":[],"dependencyTags":[],"dependencyConfig":[]}],"resourcesCount":1}}`)
		case strings.Contains(req.Query, "ListAuditLogs"):
			filter, _ := req.Variables["filter"].(map[string]any)
			if entityID := filter["entity_id"]; entityID != "t1" {
				t.Fatalf("audit entity_id filter = %#v", entityID)
			}
			fmt.Fprint(w, `{"data":{"auditLogs":[{"id":"a1","model":"template","userId":"u1","action":"apply","entityId":"t1","createdAt":"2026-01-02T00:00:00Z","revisionNumber":3,"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"}}]}}`)
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

	resources, err := client.Resources(context.Background(), map[string]any{"template_id": "t1"}, []string{"updated_at", "DESC"}, []int{0, 50})
	if err != nil {
		t.Fatalf("template resources query: %v", err)
	}
	if resources.Total != 1 || len(resources.Items) != 1 || resources.Items[0].Name != "redis-prod" {
		t.Fatalf("template resources = %#v", resources)
	}

	auditLogs, err := client.AuditLogsForEntity(context.Background(), "t1", []int{0, 50})
	if err != nil {
		t.Fatalf("template audit logs query: %v", err)
	}
	if len(auditLogs) != 1 || auditLogs[0].Model != "template" || auditLogs[0].EntityID != "t1" {
		t.Fatalf("template auditLogs = %#v", auditLogs)
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
			fmt.Fprint(w, `{"data":{"integration":{"id":"i1","name":"aws-prod","entityName":"integration","description":"AWS","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","integrationProvider":"aws","integrationType":"cloud","labels":["prod"],"configuration":{"role":"admin"},"status":"ready","creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"}}}}`)
		case strings.Contains(req.Query, "ListResources"):
			filter, _ := req.Variables["filter"].(map[string]any)
			got, _ := filter["integration_ids__any"].([]any)
			if len(got) != 1 || got[0] != "i1" {
				t.Fatalf("integration resource filter = %#v", filter["integration_ids__any"])
			}
			fmt.Fprint(w, `{"data":{"resources":[{"id":"r1","name":"redis-prod","state":"provisioned","status":"ready","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","labels":["prod"],"template":{"id":"t1","name":"aws_redis","cloudResourceTypes":["redis"]},"workspace":{"id":"w1","name":"platform"},"storage":{"id":"st1","name":"terraform-state"},"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"},"integrationIds":[{"id":"i1","name":"aws-prod","integrationProvider":"aws","integrationType":"cloud"}],"secretIds":[],"revisionNumber":2,"storagePath":"ik/state/redis-prod.tfstate","sourceCodeVersion":{"id":"scv1","identifier":"modules/redis:v1.2.3","sourceCodeFolder":"modules/redis","sourceCodeVersion":"v1.2.3","sourceCodeBranch":"","status":"ready"},"parents":[],"children":[],"variables":[],"outputs":[],"dependencyTags":[],"dependencyConfig":[]}],"resourcesCount":1}}`)
		case strings.Contains(req.Query, "ListAuditLogs"):
			filter, _ := req.Variables["filter"].(map[string]any)
			if entityID := filter["entity_id"]; entityID != "i1" {
				t.Fatalf("integration audit entity_id filter = %#v", entityID)
			}
			fmt.Fprint(w, `{"data":{"auditLogs":[{"id":"a1","model":"integration","userId":"u1","action":"sync","entityId":"i1","createdAt":"2026-01-02T00:00:00Z","revisionNumber":1,"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"}}]}}`)
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
		case strings.Contains(req.Query, "UpdateIntegration"):
			if req.Variables["id"] != "i1" {
				t.Fatalf("update integration id = %#v", req.Variables["id"])
			}
			fmt.Fprint(w, `{"data":{"updateIntegration":{"id":"i1","name":"aws-prod","entityName":"integration","integrationProvider":"aws"}}}`)
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL, Token: "token-123"})

	resources, err := client.Resources(context.Background(), map[string]any{"integration_ids__any": []string{"i1"}}, []string{"updated_at", "DESC"}, []int{0, 50})
	if err != nil {
		t.Fatalf("integration resources query: %v", err)
	}
	if resources.Total != 1 || len(resources.Items) != 1 || resources.Items[0].Name != "redis-prod" {
		t.Fatalf("integration resources = %#v", resources)
	}

	auditLogs, err := client.AuditLogsForEntity(context.Background(), "i1", []int{0, 50})
	if err != nil {
		t.Fatalf("integration audit logs query: %v", err)
	}
	if len(auditLogs) != 1 || auditLogs[0].Model != "integration" || auditLogs[0].EntityID != "i1" {
		t.Fatalf("integration auditLogs = %#v", auditLogs)
	}

	if err := client.EnableIntegration(context.Background(), "i1"); err != nil {
		t.Fatalf("enable integration: %v", err)
	}
	if err := client.DisableIntegration(context.Background(), "i1"); err != nil {
		t.Fatalf("disable integration: %v", err)
	}
	if err := client.DeleteIntegration(context.Background(), "i1"); err != nil {
		t.Fatalf("delete integration: %v", err)
	}
	if err := client.UpdateIntegration(context.Background(), "i1", map[string]any{"name": "aws-updated"}); err != nil {
		t.Fatalf("update integration: %v", err)
	}
}

func TestTemplateActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(req.Query, "GetTemplate"):
			fmt.Fprint(w, `{"data":{"template":{"id":"t1","name":"aws_redis","description":"Redis","documentation":"https://example.invalid/docs","template":"resource {}","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","cloudResourceTypes":["redis"],"labels":["cache"],"status":"ready","abstract":false,"configuration":{},"revisionNumber":3,"resourcesCount":7,"sourceCodeVersionsCount":2,"entityName":"template","creator":null,"parents":[],"children":[]}}}`)
		case strings.Contains(req.Query, "TemplateAction"):
			input, _ := req.Variables["input"].(map[string]any)
			action := input["action"]
			if req.Variables["id"] != "t1" {
				t.Fatalf("action id = %#v", req.Variables["id"])
			}
			if action != "enable" && action != "disable" {
				t.Fatalf("unexpected action = %#v", action)
			}
			fmt.Fprintf(w, `{"data":{"templateAction":{"id":"t1","entityName":"template","status":"%s"}}}`, action)
		case strings.Contains(req.Query, "DeleteTemplate"):
			if req.Variables["id"] != "t1" {
				t.Fatalf("delete id = %#v", req.Variables["id"])
			}
			fmt.Fprint(w, `{"data":{"deleteTemplate":true}}`)
		case strings.Contains(req.Query, "UpdateTemplate"):
			if req.Variables["id"] != "t1" {
				t.Fatalf("update template id = %#v", req.Variables["id"])
			}
			fmt.Fprint(w, `{"data":{"updateTemplate":{"id":"t1","name":"aws_redis","template":"resource {}","entityName":"template"}}}`)
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL, Token: "token-123"})

	if err := client.EnableTemplate(context.Background(), "t1"); err != nil {
		t.Fatalf("enable template: %v", err)
	}
	if err := client.DisableTemplate(context.Background(), "t1"); err != nil {
		t.Fatalf("disable template: %v", err)
	}
	if err := client.DeleteTemplate(context.Background(), "t1"); err != nil {
		t.Fatalf("delete template: %v", err)
	}
	if err := client.UpdateTemplate(context.Background(), "t1", map[string]any{"name": "aws_redis"}); err != nil {
		t.Fatalf("update template: %v", err)
	}
}

func TestStorages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(req.Query, "ListStorages"):
			fmt.Fprint(w, `{"data":{"storages":[{"id":"st1","name":"terraform-state","storageType":"tofu","storageProvider":"aws","description":"Primary bucket","labels":["prod"],"state":"ready","status":"ready","resourcesCount":4,"executorsCount":1,"revisionNumber":2,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","entityName":"storage","integration":{"id":"i1","name":"aws-prod","entityName":"integration","integrationProvider":"aws"},"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"}}],"storagesCount":1}}`)
		case strings.Contains(req.Query, "GetStorage"):
			fmt.Fprint(w, `{"data":{"storage":{"id":"st1","name":"terraform-state","storageType":"tofu","storageProvider":"aws","configuration":{"aws_bucket_name":"terraform-state"},"description":"Primary bucket","labels":["prod"],"state":"ready","status":"ready","resourcesCount":4,"executorsCount":1,"revisionNumber":2,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","entityName":"storage","integration":{"id":"i1","name":"aws-prod","entityName":"integration","integrationProvider":"aws","integrationType":"cloud"},"creator":{"id":"u1","identifier":"alice","email":"alice@example.com","displayName":"Alice"}}}}`)
		case strings.Contains(req.Query, "UpdateStorage"):
			if req.Variables["id"] != "st1" {
				t.Fatalf("update storage id = %#v", req.Variables["id"])
			}
			fmt.Fprint(w, `{"data":{"updateStorage":{"id":"st1","name":"terraform-state","entityName":"storage"}}}`)
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	defer server.Close()

	client := New(config.Config{Endpoint: server.URL, Token: "token-123"})

	storages, err := client.Storages(context.Background(), map[string]any{"storage_provider": "aws"}, []string{"updated_at", "DESC"}, []int{0, 50})
	if err != nil {
		t.Fatalf("storages query: %v", err)
	}
	if storages.Total != 1 || len(storages.Items) != 1 || storages.Items[0].Name != "terraform-state" {
		t.Fatalf("storages = %#v", storages)
	}

	storage, err := client.Storage(context.Background(), "st1")
	if err != nil {
		t.Fatalf("storage query: %v", err)
	}
	if storage == nil || storage.StorageProvider != "aws" || storage.Integration == nil || storage.Integration.Name != "aws-prod" {
		t.Fatalf("storage = %#v", storage)
	}

	if err := client.UpdateStorage(context.Background(), "st1", map[string]any{"description": "Updated"}); err != nil {
		t.Fatalf("update storage: %v", err)
	}
}
