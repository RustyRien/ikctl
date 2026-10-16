package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/config"
)

const testToken = "ik_0123456789abcdef0123456789abcdef_secret"

// fakeAPI is a minimal InfraKitchen server for command tests.
func fakeAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"Invalid authentication credentials"}`))
			return
		}
		if r.URL.Path == sdk.MePath {
			_, _ = w.Write([]byte(`{"id":"u1","email":"alice@example.com","identifier":"alice","provider":"guest"}`))
			return
		}
		var req sdk.Request
		_ = json.NewDecoder(r.Body).Decode(&req)
		q := req.Query
		switch {
		case strings.Contains(q, "resourceActions"):
			_, _ = w.Write([]byte(`{"data":{"actions":["approve","reject"]}}`))
		case strings.Contains(q, "items: resources("):
			filter, _ := json.Marshal(req.Variables["filter"])
			if strings.Contains(string(filter), `"name":"missing"`) {
				_, _ = w.Write([]byte(`{"data":{"items":[],"total":0}}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"items":[{"id":"1063a79d-c17e-41c3-b576-52e21dbb7239","name":"redis","state":"PROVISION","status":"APPROVAL_PENDING","template":{"name":"Redis"},"createdAt":"2026-01-01T00:00:00Z"}],"total":7}}`))
		case strings.Contains(q, "item: resource("):
			_, _ = w.Write([]byte(`{"data":{"item":{"id":"1063a79d-c17e-41c3-b576-52e21dbb7239","name":"redis","status":"APPROVAL_PENDING"}}}`))
		case strings.Contains(q, "resourceAction("):
			_, _ = w.Write([]byte(`{"data":{"item":{"id":"1063a79d-c17e-41c3-b576-52e21dbb7239","name":"redis","state":"PROVISION","status":"QUEUED"}}}`))
		case strings.Contains(q, "deleteResource"):
			_, _ = w.Write([]byte(`{"data":{"deleted":true}}`))
		case strings.Contains(q, "serverInfo"):
			_, _ = w.Write([]byte(`{"data":{"serverInfo":{"version":"v9.9.9"}}}`))
		default:
			_, _ = w.Write([]byte(`{"errors":[{"message":"unexpected query","extensions":{"code":"INTERNAL"}}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

type result struct {
	code   int
	stdout string
	stderr string
}

// run executes ikctl with an isolated config file and environment.
func run(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	f := NewFactory(IOStreams{In: strings.NewReader(stdin), Out: &out, ErrOut: &errOut})
	code := Execute(context.Background(), f, args)
	return result{code: code, stdout: out.String(), stderr: errOut.String()}
}

func setup(t *testing.T) (server, cfgPath string) {
	t.Helper()
	srv := fakeAPI(t)
	cfgPath = filepath.Join(t.TempDir(), "ikctl", "config.yaml")
	t.Setenv(config.EnvConfig, cfgPath)
	t.Setenv(config.EnvServer, "")
	t.Setenv(config.EnvToken, "")
	t.Setenv(config.EnvContext, "")
	return srv.URL, cfgPath
}

func login(t *testing.T, server string) {
	t.Helper()
	if r := run(t, testToken+"\n", "login", "--server", server, "--token-stdin", "--context", "test"); r.code != 0 {
		t.Fatalf("login failed: %+v", r)
	}
}

func TestLoginSavesContext(t *testing.T) {
	server, cfgPath := setup(t)
	r := run(t, testToken+"\n", "login", "--server", server, "--token-stdin", "--context", "test")
	if r.code != 0 || !strings.Contains(r.stdout, "alice <alice@example.com>") {
		t.Fatalf("login: %+v", r)
	}
	st, err := os.Stat(cfgPath)
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("config file: %v, %v", st, err)
	}
	cfg, _ := config.Load(cfgPath)
	if cfg.CurrentContext != "test" || cfg.Contexts[0].Token != testToken {
		t.Errorf("config = %+v", cfg)
	}

	r = run(t, "", "whoami", "-o", "json")
	var who LoginResult
	if err := json.Unmarshal([]byte(r.stdout), &who); err != nil || who.User.Identifier != "alice" || who.Context != "test" {
		t.Errorf("whoami: %+v (%v)", r, err)
	}

	r = run(t, "", "config", "view")
	if strings.Contains(r.stdout, "secret") || !strings.Contains(r.stdout, "REDACTED") {
		t.Errorf("config view leaked token: %s", r.stdout)
	}

	r = run(t, "", "logout")
	cfg, _ = config.Load(cfgPath)
	if r.code != 0 || cfg.Contexts[0].Token != "" {
		t.Errorf("logout: %+v %+v", r, cfg)
	}
	if r := run(t, "", "whoami"); r.code != ExitAuth {
		t.Errorf("whoami after logout exit = %d", r.code)
	}
}

func TestLoginRejectsBadToken(t *testing.T) {
	server, _ := setup(t)
	if r := run(t, "garbage", "login", "--server", server, "--token-stdin"); r.code != ExitUsage {
		t.Errorf("malformed token exit = %d (%s)", r.code, r.stderr)
	}
	if r := run(t, "ik_ffffffffffffffffffffffffffffffff_x", "login", "--server", server, "--token-stdin"); r.code != ExitAuth {
		t.Errorf("invalid token exit = %d (%s)", r.code, r.stderr)
	}
}

func TestGetOutputs(t *testing.T) {
	server, _ := setup(t)
	login(t, server)

	r := run(t, "", "get", "resources")
	if r.code != 0 || !strings.Contains(r.stdout, "NAME") || !strings.Contains(r.stdout, "redis") || !strings.Contains(r.stderr, "of 7") {
		t.Errorf("table: %+v", r)
	}

	r = run(t, "", "get", "rs", "-o", "json")
	var list sdk.List
	if err := json.Unmarshal([]byte(r.stdout), &list); err != nil || list.Kind != "ResourceList" || list.Total != 7 {
		t.Errorf("json: %+v (%v)", r, err)
	}

	r = run(t, "", "get", "rs", "-o", "yaml")
	if !strings.Contains(r.stdout, "kind: ResourceList") {
		t.Errorf("yaml: %s", r.stdout)
	}

	r = run(t, "", "get", "rs", "-o", "name")
	if r.stdout != "resource/redis\n" {
		t.Errorf("name: %q", r.stdout)
	}

	r = run(t, "", "get", "resource", "redis", "-o", "json")
	var obj map[string]any
	if err := json.Unmarshal([]byte(r.stdout), &obj); err != nil || obj["name"] != "redis" {
		t.Errorf("single: %+v", r)
	}
}

func TestExitCodes(t *testing.T) {
	server, _ := setup(t)
	login(t, server)

	tests := []struct {
		args []string
		code int
	}{
		{[]string{"get"}, ExitUsage},
		{[]string{"get", "bogus"}, ExitUsage},
		{[]string{"get", "rs", "--nope"}, ExitUsage},
		{[]string{"get", "rs", "-o", "xml"}, ExitUsage},
		{[]string{"get", "resource", "missing"}, ExitNotFound},
		{[]string{"get", "rs", "--token", "ik_ffffffffffffffffffffffffffffffff_x"}, ExitAuth},
		{[]string{"delete", "resource", "redis"}, ExitUsage}, // non-interactive without --yes
		{[]string{"action", "resource", "redis", "destroy", "--yes"}, ExitConflict},
		{[]string{"create", "-f", "/does/not/exist"}, ExitError},
	}
	for _, tt := range tests {
		if r := run(t, "", tt.args...); r.code != tt.code {
			t.Errorf("%v: exit %d, want %d (stderr: %s)", tt.args, r.code, tt.code, r.stderr)
		}
	}
}

func TestStructuredError(t *testing.T) {
	server, _ := setup(t)
	login(t, server)
	r := run(t, "", "get", "resource", "missing", "-o", "json")
	var doc struct {
		Error struct {
			Code     string `json:"code"`
			ExitCode int    `json:"exitCode"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(r.stderr), &doc); err != nil || doc.Error.Code != "NOT_FOUND" || doc.Error.ExitCode != ExitNotFound {
		t.Errorf("structured error: %q (%v)", r.stderr, err)
	}
}

func TestActionAndDelete(t *testing.T) {
	server, _ := setup(t)
	login(t, server)

	r := run(t, "", "action", "resource", "redis")
	if r.stdout != "approve\nreject\n" {
		t.Errorf("list actions: %+v", r)
	}
	r = run(t, "", "action", "resource", "redis", "approve", "--yes")
	if r.code != 0 || !strings.Contains(r.stdout, `action "approve" submitted (state=PROVISION, status=QUEUED)`) {
		t.Errorf("approve: %+v", r)
	}
	r = run(t, "", "action", "resource", "redis", "approve", "--dry-run", "-o", "json")
	if r.code != 0 || !strings.Contains(r.stdout, "resourceAction(") {
		t.Errorf("dry-run: %+v", r)
	}
	r = run(t, "y\n", "delete", "resource", "redis", "--yes")
	if r.code != 0 || r.stdout != "resource/redis deleted\n" {
		t.Errorf("delete: %+v", r)
	}
}

func TestCreateDryRun(t *testing.T) {
	server, _ := setup(t)
	login(t, server)
	manifest := "kind: resource\nspec:\n  name: demo\n  template_id: t1\n  variables:\n    my_var: 1\n"
	r := run(t, manifest, "create", "-f", "-", "--dry-run", "-o", "json")
	var res DryRunResult
	if err := json.Unmarshal([]byte(r.stdout), &res); err != nil {
		t.Fatalf("dry-run: %+v (%v)", r, err)
	}
	input, _ := json.Marshal(res.Request.Variables["input"])
	if res.Operation != "create" || string(input) != `{"name":"demo","templateId":"t1","variables":{"my_var":1}}` {
		t.Errorf("dry-run result: %+v input=%s", res, input)
	}

	if r := run(t, "kind: resource\n", "create", "-f", "-"); r.code != ExitUsage {
		t.Errorf("missing spec exit = %d", r.code)
	}
}

func TestSkillPrint(t *testing.T) {
	r := run(t, "", "skill", "print")
	if r.code != 0 || !strings.Contains(r.stdout, "name: ikctl") {
		t.Errorf("skill print: %+v", r)
	}
	dir := t.TempDir()
	if r := run(t, "", "skill", "install", "--dir", dir); r.code != 0 {
		t.Fatalf("skill install: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(dir, "ikctl", "SKILL.md")); err != nil {
		t.Error(err)
	}
}
