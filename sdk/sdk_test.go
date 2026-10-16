package sdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rustyrien/ikctl/sdk"
)

const testToken = "ik_0123456789abcdef0123456789abcdef_s3cr_et-value"

// fakeServer answers GraphQL requests with handler(query, variables).
type fakeServer struct {
	t        *testing.T
	handler  func(query string, vars map[string]any) (status int, body string)
	requests []sdk.Request
}

func newFakeServer(t *testing.T, handler func(query string, vars map[string]any) (int, string)) (*sdk.Client, *fakeServer) {
	t.Helper()
	fs := &fakeServer{t: t, handler: handler}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"Invalid authentication credentials"}`))
			return
		}
		switch r.URL.Path {
		case sdk.MePath:
			_, _ = w.Write([]byte(`{"id":"u1","email":"a@example.com","identifier":"alice","provider":"guest"}`))
		case sdk.GraphQLPath:
			var req sdk.Request
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode request: %v", err)
			}
			fs.requests = append(fs.requests, req)
			status, body := handler(req.Query, req.Variables)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := sdk.New(sdk.WithServer(srv.URL), sdk.WithToken(testToken))
	if err != nil {
		t.Fatal(err)
	}
	return c, fs
}

func TestListBuildsRequestAndDecodes(t *testing.T) {
	c, fs := newFakeServer(t, func(q string, _ map[string]any) (int, string) {
		if !strings.Contains(q, "items: resources(") || !strings.Contains(q, "total: resourcesCount(") {
			t.Errorf("unexpected query: %s", q)
		}
		return 200, `{"data":{"items":[{"id":"1","name":"a"},{"id":"2","name":"b"}],"total":42}}`
	})
	list, err := c.Resources.List(context.Background(), sdk.ListOptions{
		Filter: sdk.Like("name", "redis").And(sdk.Labels("prod")),
		Sort:   &sdk.Sort{Field: "createdAt", Desc: true},
		Offset: 10, Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if list.Kind != "ResourceList" || list.Total != 42 || len(list.Items) != 2 || list.Items[1].ID() != "2" {
		t.Errorf("unexpected list: %+v", list)
	}
	vars := fs.requests[0].Variables
	if got, _ := json.Marshal(vars["range"]); string(got) != "[10,15]" {
		t.Errorf("range = %s", got)
	}
	if got, _ := json.Marshal(vars["sort"]); string(got) != `["createdAt","DESC"]` {
		t.Errorf("sort = %s", got)
	}
	if got, _ := json.Marshal(vars["filter"]); string(got) != `{"labels__contains_all":["prod"],"name__like":"redis"}` {
		t.Errorf("filter = %s", got)
	}
}

func TestListAllPaginates(t *testing.T) {
	calls := 0
	c, _ := newFakeServer(t, func(_ string, vars map[string]any) (int, string) {
		calls++
		r := vars["range"].([]any)
		switch r[0].(float64) {
		case 0:
			return 200, `{"data":{"items":[{"id":"1"},{"id":"2"}],"total":3}}`
		default:
			return 200, `{"data":{"items":[{"id":"3"}],"total":3}}`
		}
	})
	var ids []string
	for obj, err := range c.Templates.ListAll(context.Background(), sdk.ListOptions{Limit: 2}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, obj.ID())
	}
	if strings.Join(ids, ",") != "1,2,3" || calls != 2 {
		t.Errorf("ids=%v calls=%d", ids, calls)
	}
}

func TestGetNotFound(t *testing.T) {
	c, _ := newFakeServer(t, func(string, map[string]any) (int, string) {
		return 200, `{"data":{"item":null}}`
	})
	_, err := c.Resources.Get(context.Background(), "1063a79d-c17e-41c3-b576-52e21dbb7239")
	if !sdk.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	// Non-UUID ids never reach the server.
	if _, err := c.Resources.Get(context.Background(), "not-a-uuid"); !sdk.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetByNameUsesDescribeFields(t *testing.T) {
	c, fs := newFakeServer(t, func(string, map[string]any) (int, string) {
		return 200, `{"data":{"items":[{"id":"a"}],"total":1}}`
	})
	if _, err := c.Resources.Resolve(context.Background(), "by-name"); err != nil {
		t.Fatal(err)
	}
	if q := fs.requests[0].Query; !strings.Contains(q, "variables") || !strings.Contains(q, "parents {") {
		t.Errorf("name lookup should fetch describe fields, got: %s", q)
	}
}

func TestGetByNameAmbiguous(t *testing.T) {
	c, _ := newFakeServer(t, func(string, map[string]any) (int, string) {
		return 200, `{"data":{"items":[{"id":"a"},{"id":"b"}],"total":2}}`
	})
	_, err := c.Resources.Resolve(context.Background(), "dup")
	var amb *sdk.AmbiguousError
	if !errors.As(err, &amb) || len(amb.Matches) != 2 || !sdk.IsConflict(err) {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
}

func TestGraphQLErrorCodes(t *testing.T) {
	c, _ := newFakeServer(t, func(string, map[string]any) (int, string) {
		return 200, `{"data":null,"errors":[{"message":"bad state","path":["resourceAction"],"extensions":{"code":"WRONG_STATE"}}]}`
	})
	var out map[string]any
	err := c.GraphQL(context.Background(), "mutation { x }", nil, &out)
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != sdk.CodeWrongState || apiErr.Path[0] != "resourceAction" {
		t.Fatalf("unexpected error %#v", err)
	}
	if !sdk.IsConflict(err) {
		t.Error("WRONG_STATE should be a conflict")
	}
}

func TestUnauthorized(t *testing.T) {
	c, _ := newFakeServer(t, nil)
	bad, err := sdk.New(sdk.WithServer(c.ServerURL()), sdk.WithToken("ik_ffffffffffffffffffffffffffffffff_x"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = bad.Auth.Me(context.Background())
	if !sdk.IsUnauthorized(err) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	u, err := c.Auth.Me(context.Background())
	if err != nil || u.Identifier != "alice" {
		t.Fatalf("Me() = %+v, %v", u, err)
	}
}

func TestActionNotAllowed(t *testing.T) {
	c, fs := newFakeServer(t, func(q string, _ map[string]any) (int, string) {
		if strings.Contains(q, "resourceActions") {
			return 200, `{"data":{"actions":["approve","reject"]}}`
		}
		return 200, `{"data":{"item":{"id":"x","status":"QUEUED"}}}`
	})
	ctx := context.Background()
	_, err := c.Resources.Action(ctx, "x", "destroy")
	var na *sdk.ActionNotAllowedError
	if !errors.As(err, &na) || strings.Join(na.Allowed, ",") != "approve,reject" {
		t.Fatalf("expected not allowed, got %v", err)
	}
	if _, err := c.Resources.Action(ctx, "x", "APPROVE"); err != nil {
		t.Fatal(err)
	}
	last := fs.requests[len(fs.requests)-1]
	if got, _ := json.Marshal(last.Variables["input"]); string(got) != `{"action":"APPROVE"}` {
		t.Errorf("input = %s", got)
	}
}

func TestCreateCamelizesTopLevelKeysOnly(t *testing.T) {
	c, _ := newFakeServer(t, nil)
	req, err := c.Resources.CreateRequest(map[string]any{
		"template_id": "t",
		"variables":   map[string]any{"snake_key": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(req.Variables["input"])
	if string(got) != `{"templateId":"t","variables":{"snake_key":1}}` {
		t.Errorf("input = %s", got)
	}
	if _, err := c.Tasks.CreateRequest(nil); err == nil {
		t.Error("tasks should not support create")
	}
}

func TestParseToken(t *testing.T) {
	info, err := sdk.ParseToken(testToken)
	if err != nil || info.ID != "0123456789abcdef0123456789abcdef" || info.Prefix != "ik_01234567" {
		t.Fatalf("ParseToken = %+v, %v", info, err)
	}
	for _, bad := range []string{"", "ik_", "ik_abc_def", "xx_0123456789abcdef0123456789abcdef_s", "ik_0123456789abcdef0123456789abcdef_"} {
		if _, err := sdk.ParseToken(bad); err == nil {
			t.Errorf("ParseToken(%q) should fail", bad)
		}
	}
	if r := sdk.RedactToken(testToken); strings.Contains(r, "s3cr") {
		t.Errorf("RedactToken leaked secret: %s", r)
	}
}

func TestParseServerURL(t *testing.T) {
	tests := map[string]string{
		"http://localhost:7777/":      "http://localhost:7777",
		"ik.example.com":              "https://ik.example.com",
		"https://x.example.com/ik/?a": "https://x.example.com/ik",
	}
	for in, want := range tests {
		u, err := sdk.ParseServerURL(in)
		if err != nil || u.String() != want {
			t.Errorf("ParseServerURL(%q) = %v, %v; want %s", in, u, err, want)
		}
	}
	if _, err := sdk.ParseServerURL("ftp://x"); err == nil {
		t.Error("ftp scheme should be rejected")
	}
	if _, err := sdk.New(); !errors.Is(err, sdk.ErrNotConfigured) {
		t.Errorf("New() without server = %v", err)
	}
}

func TestToCamel(t *testing.T) {
	for in, want := range map[string]string{
		"template_id": "templateId", "source-code-version-id": "sourceCodeVersionId",
		"name": "name", "templateId": "templateId", "_x": "x",
	} {
		if got := sdk.ToCamel(in); got != want {
			t.Errorf("ToCamel(%q) = %q, want %q", in, got, want)
		}
	}
}
