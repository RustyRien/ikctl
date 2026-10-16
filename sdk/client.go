// Package sdk is a Go client for the InfraKitchen API.
//
// It talks to the GraphQL endpoint (/api/graphql) and a small number of REST
// endpoints using a personal access token or service-account token.
//
//	c, err := sdk.New(sdk.WithServer("http://localhost:7777"), sdk.WithToken(token))
//	list, err := c.Resources.List(ctx, sdk.ListOptions{Filter: sdk.Like("name", "redis")})
package sdk

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/rustyrien/ikctl/sdk/registry"
)

const (
	// GraphQLPath is the path of the GraphQL endpoint.
	GraphQLPath = "/api/graphql"
	// MePath is the REST endpoint returning the authenticated user.
	MePath = "/api/auth/me"

	defaultTimeout = 30 * time.Second
	maxErrorBody   = 4096
)

// Client is an InfraKitchen API client. It is safe for concurrent use.
type Client struct {
	server    *url.URL
	token     string
	http      *http.Client
	userAgent string
	log       *slog.Logger

	Auth   *AuthService
	Server *ServerService
	Schema *SchemaService

	Resources          *ResourceClient
	Templates          *EntityClient
	Integrations       *EntityClient
	SourceCodes        *EntityClient
	SourceCodeVersions *EntityClient
	Workspaces         *EntityClient
	Storages           *EntityClient
	Secrets            *EntityClient
	Executors          *EntityClient
	Projects           *EntityClient
	Services           *EntityClient
	Blueprints         *EntityClient
	Workflows          *EntityClient
	Tools              *EntityClient
	Tasks              *EntityClient
	AuditLogs          *EntityClient
	Workers            *EntityClient
	Users              *EntityClient
	Permissions        *EntityClient
}

// Option configures a Client.
type Option func(*options) error

type options struct {
	server     string
	token      string
	httpClient *http.Client
	timeout    time.Duration
	userAgent  string
	logger     *slog.Logger
	insecure   bool
}

// WithServer sets the base URL of the InfraKitchen server, e.g. http://localhost:7777.
func WithServer(server string) Option {
	return func(o *options) error { o.server = server; return nil }
}

// WithToken sets the bearer token (personal access token or service-account JWT).
func WithToken(token string) Option {
	return func(o *options) error { o.token = strings.TrimSpace(token); return nil }
}

// WithHTTPClient sets a custom HTTP client. Timeout and TLS options are not applied to it.
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) error { o.httpClient = c; return nil }
}

// WithTimeout sets the per-request timeout (default 30s).
func WithTimeout(d time.Duration) Option {
	return func(o *options) error {
		if d < 0 {
			return fmt.Errorf("timeout must not be negative")
		}
		o.timeout = d
		return nil
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(o *options) error { o.userAgent = ua; return nil }
}

// WithLogger sets a logger for debug output. Tokens are never logged.
func WithLogger(l *slog.Logger) Option {
	return func(o *options) error { o.logger = l; return nil }
}

// WithInsecureSkipTLSVerify disables TLS certificate verification.
func WithInsecureSkipTLSVerify(skip bool) Option {
	return func(o *options) error { o.insecure = skip; return nil }
}

// New creates a Client.
func New(opts ...Option) (*Client, error) {
	o := options{timeout: defaultTimeout}
	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return nil, err
		}
	}
	if o.server == "" {
		return nil, ErrNotConfigured
	}
	u, err := ParseServerURL(o.server)
	if err != nil {
		return nil, err
	}

	hc := o.httpClient
	if hc == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if o.insecure {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicitly requested by the user
		}
		hc = &http.Client{Transport: transport, Timeout: o.timeout}
	}
	if o.userAgent == "" {
		o.userAgent = fmt.Sprintf("infrakitchen-go-sdk (%s/%s)", runtime.GOOS, runtime.GOARCH)
	}
	if o.logger == nil {
		o.logger = slog.New(slog.DiscardHandler)
	}

	c := &Client{server: u, token: o.token, http: hc, userAgent: o.userAgent, log: o.logger}
	c.Auth = &AuthService{c: c}
	c.Server = &ServerService{c: c}
	c.Schema = &SchemaService{c: c}

	mustEntity := func(name string) *EntityClient {
		k, ok := registry.Lookup(name)
		if !ok {
			panic("sdk: unknown builtin kind " + name)
		}
		return &EntityClient{c: c, kind: k}
	}
	c.Resources = &ResourceClient{EntityClient: mustEntity("resource")}
	c.Templates = mustEntity("template")
	c.Integrations = mustEntity("integration")
	c.SourceCodes = mustEntity("source-code")
	c.SourceCodeVersions = mustEntity("source-code-version")
	c.Workspaces = mustEntity("workspace")
	c.Storages = mustEntity("storage")
	c.Secrets = mustEntity("secret")
	c.Executors = mustEntity("executor")
	c.Projects = mustEntity("project")
	c.Services = mustEntity("service")
	c.Blueprints = mustEntity("blueprint")
	c.Workflows = mustEntity("workflow")
	c.Tools = mustEntity("tool")
	c.Tasks = mustEntity("task")
	c.AuditLogs = mustEntity("audit-log")
	c.Workers = mustEntity("worker")
	c.Users = mustEntity("user")
	c.Permissions = mustEntity("permission")
	return c, nil
}

// ParseServerURL validates and normalizes a server URL. A missing scheme defaults to https.
func ParseServerURL(s string) (*url.URL, error) {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL %q: %w", s, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid server URL %q: scheme must be http or https", s)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid server URL %q: missing host", s)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery, u.Fragment = "", ""
	return u, nil
}

// ServerURL returns the normalized server base URL.
func (c *Client) ServerURL() string { return c.server.String() }

// Entities returns a client for the given kind. The kind may be a name, plural,
// short alias or GraphQL name (e.g. "resource", "resources", "rs").
func (c *Client) Entities(kind string) (*EntityClient, error) {
	k, ok := registry.Lookup(kind)
	if !ok {
		return nil, fmt.Errorf("unknown kind %q (see 'ikctl api-resources')", kind)
	}
	return &EntityClient{c: c, kind: k}, nil
}

func (c *Client) endpoint(path string) string {
	u := *c.server
	u.Path += path
	return u.String()
}

// doHTTP sends a request with auth headers and returns the response body.
// Non-2xx responses are converted to *APIError.
func (c *Client) doHTTP(ctx context.Context, method, path string, body any) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, c.endpoint(path), err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	c.log.Debug("http request", "method", method, "url", req.URL.String(), "status", resp.StatusCode,
		"duration", time.Since(start).Round(time.Millisecond), "bytes", len(data))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, httpError(resp.StatusCode, data)
	}
	return data, nil
}

func httpError(status int, body []byte) error {
	e := &APIError{Code: CodeHTTP, HTTPStatus: status}
	switch status {
	case http.StatusUnauthorized:
		e.Code = CodeUnauthorized
	case http.StatusForbidden:
		e.Code = CodeAccessDenied
	case http.StatusNotFound:
		e.Code = CodeNotFound
	}
	var payload struct {
		Message string `json:"message"`
		Detail  any    `json:"detail"`
	}
	if json.Unmarshal(body, &payload) == nil {
		switch {
		case payload.Message != "":
			e.Message = payload.Message
		case payload.Detail != nil:
			e.Message = fmt.Sprint(payload.Detail)
		}
	}
	if e.Message == "" {
		msg := strings.TrimSpace(string(body))
		if len(msg) > maxErrorBody {
			msg = msg[:maxErrorBody] + "..."
		}
		if msg == "" {
			msg = http.StatusText(status)
		}
		e.Message = msg
	}
	if status == http.StatusUnauthorized {
		e.Message += " (token is invalid, expired or revoked; run 'ikctl login')"
	}
	return e
}

// decodeJSON decodes data preserving numbers as json.Number.
func decodeJSON(data []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

var errNilOut = errors.New("sdk: out must not be nil")
