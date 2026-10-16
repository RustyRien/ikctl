# ikctl

`ikctl` is the command-line client and Go SDK for [InfraKitchen](https://github.com/electrolux-oss/infrakitchen).

- **CLI** - `ikctl VERB KIND [NAME|ID]` commands to list, inspect, create, update,
  approve, provision, destroy and delete InfraKitchen objects.
- **Machine-friendly** - every data command supports `-o json` / `-o yaml`,
  errors are structured, and exit codes are stable, so it works well in scripts
  and as a tool for LLM agents.
- **Go SDK** - the CLI is a thin wrapper around the public
  [`sdk`](./sdk) package, which you can import in your own programs.
- **Agent skill** - ships with a [`SKILL.md`](./skills/ikctl/SKILL.md) that
  teaches LLM agents how to use it (`ikctl skill install`).

## Install

```bash
go install github.com/rustyrien/ikctl/cmd/ikctl@latest
# or from a checkout
make build && ./bin/ikctl version --client
```

## Quick start

Create a personal access token in the InfraKitchen UI, then:

```bash
ikctl login --server https://infrakitchen.example.com    # prompts for the token (hidden)
# non-interactive:
printf '%s' "$IK_TOKEN" | ikctl login --server http://localhost:7777 --token-stdin

ikctl whoami
ikctl get resources
ikctl describe resource my-resource
```

The token is validated against the server and saved as a *context* in
`~/.config/ikctl/config.yaml` (`$XDG_CONFIG_HOME` is honored), created with
`0600` permissions. Run `ikctl logout` to remove it.

## Commands

| Command | Description |
|---------|-------------|
| `get KIND [NAME\|ID...]` | List objects (filters, sorting, paging) or get specific ones |
| `describe KIND NAME\|ID` | All details plus currently allowed actions |
| `create -f FILE` | Create objects from YAML/JSON manifests |
| `apply -f FILE` | Create or update objects from manifests |
| `delete KIND NAME\|ID...` | Delete objects (asks for confirmation; `--yes` in scripts) |
| `action KIND NAME\|ID [ACTION]` | List allowed actions or run one (approve, execute, destroy, retry, ...) |
| `graphql [QUERY]` | Run a raw GraphQL query or mutation |
| `api-resources` | Kinds, short names and supported verbs |
| `explain KIND[.FIELD] [--input]` | Fields and input types from the live GraphQL schema |
| `login` / `logout` / `whoami` | Authentication |
| `config view\|get-contexts\|use-context\|current-context\|delete-context` | Manage contexts |
| `version` | Client and server versions |
| `skill print\|install` | The LLM agent skill |
| `completion bash\|zsh\|fish\|powershell` | Shell completion |

Kinds: `resource (rs)`, `template (tpl)`, `integration (int)`, `source-code (sc)`,
`source-code-version (scv)`, `workspace (ws)`, `storage`, `secret`,
`executor (exec)`, `project (proj)`, `service (svc)`, `blueprint (bp)`,
`workflow (wf)`, `tool`, `task`, `audit-log`, `worker`, `user`, `permission`.

### Examples

```bash
# Filtering, sorting and paging
ikctl get rs --name-like redis
ikctl get rs --field-selector status=ERROR --field-selector template__name__like=eks
ikctl get rs --filter '{"or":[{"status":"ERROR"},{"state":"DESTROY"}]}'
ikctl get templates -l aws,eks --sort-by createdAt:desc --limit 20
ikctl get rs --all -o json

# Output formats
ikctl get rs -o wide
ikctl get rs -o yaml
ikctl get rs -o name
ikctl get rs -o custom-columns=NAME:name,TEMPLATE:template.name,STATUS:status
ikctl get rs --select "id name outputs" -o json

# Lifecycle
ikctl action resource my-resource                # allowed actions
ikctl action resource my-resource approve
ikctl action resource my-resource execute --yes

# Manifests
ikctl explain resource --input
ikctl create -f resource.yaml --dry-run
ikctl apply -f resource.yaml

# Anything else
ikctl graphql '{ serverInfo { version } }'
```

A manifest looks like this (top-level keys may be snake_case or camelCase):

```yaml
kind: resource
spec:
  name: my-redis
  description: Redis for team A
  template_id: 011740dc-004c-4fb8-8026-72b1d706943a
  integration_ids: []
  secret_ids: []
  parents: []
  children: []
  labels: [team-a]
  variables: []
```

See [`skills/ikctl/references/filters.md`](skills/ikctl/references/filters.md)
for the full filter syntax.

## Output and exit codes

`-o json|yaml` lists are printed as
`{"kind": "ResourceList", "items": [...], "total": N, "offset": 0, "limit": 100}`;
single objects are printed as-is. Data goes to stdout and messages to stderr.
With `-o json|yaml`, errors are printed to stderr as
`{"error": {"code": "NOT_FOUND", "message": "...", "exitCode": 4}}`.

| Exit code | Meaning |
|-----------|---------|
| 0 | success |
| 1 | error (network, server, ...) |
| 2 | usage error (bad flags or kind, missing `--yes`) |
| 3 | authentication or authorization error |
| 4 | not found |
| 5 | conflict: wrong state, validation, ambiguous name, action not allowed |

## Configuration

Settings are resolved in this order: flags, environment variables, then the config file.

| Flag | Environment | Description |
|------|-------------|-------------|
| `--server` | `IKCTL_SERVER` | Server URL |
| `--token` | `IKCTL_TOKEN` | API token (prefer `login`/env over the flag) |
| `--context` | `IKCTL_CONTEXT` | Context name in the config file |
| `--config` | `IKCTL_CONFIG` | Config file path |
| `--timeout` | | Request timeout (default 30s) |
| `-v` | | Log requests to stderr (tokens are never logged) |

```yaml
# ~/.config/ikctl/config.yaml
current-context: prod
contexts:
- name: prod
  server: https://infrakitchen.example.com
  token: ik_...
  user: alice@example.com
```

## Go SDK

```go
import (
    "github.com/rustyrien/ikctl/sdk"
    "github.com/rustyrien/ikctl/sdk/config"
)

// Explicit configuration ...
c, err := sdk.New(sdk.WithServer("http://localhost:7777"), sdk.WithToken(token))
// ... or reuse 'ikctl login' credentials and IKCTL_* variables.
c, err = sdk.NewFromConfig(config.Overrides{})

me, err := c.Auth.Me(ctx)

list, err := c.Resources.List(ctx, sdk.ListOptions{
    Filter: sdk.Like("name", "redis").And(sdk.Eq("status", "DONE")),
    Sort:   &sdk.Sort{Field: "createdAt", Desc: true},
    Limit:  20,
})
for obj, err := range c.Templates.ListAll(ctx, sdk.ListOptions{}) { ... }

res, err := c.Resources.Resolve(ctx, "my-resource")       // by name or id
actions, err := c.Resources.Actions(ctx, res.ID())
_, err = c.Resources.Action(ctx, res.ID(), "approve")      // validated against allowed actions

ec, err := c.Entities("scv")                                // any kind by name/alias
obj, err := ec.Create(ctx, map[string]any{"name": "x", ...})

var out map[string]any
err = c.GraphQL(ctx, `{ serverInfo { version } }`, nil, &out)

if sdk.IsNotFound(err) || sdk.IsUnauthorized(err) || sdk.IsConflict(err) { ... }
```

Every operation also has a `...Request` builder (e.g. `CreateRequest`) that
returns the GraphQL request without sending it. See
[`examples/list-resources`](examples/list-resources/main.go).

## LLM agents

`ikctl` is designed to be driven by agents: discoverable (`api-resources`,
`explain`), structured output and errors, `--dry-run` previews, and
confirmation required for destructive operations (`--yes`).

```bash
ikctl skill install                       # ~/.claude/skills/ikctl
ikctl skill install --dir .claude/skills  # project-local
ikctl skill print                         # print SKILL.md
```

## Development

```bash
make test    # unit tests with -race
make lint    # go vet + golangci-lint
make build   # bin/ikctl with version information
```

Project layout:

```
cmd/ikctl/         entry point
internal/cli/      cobra commands (thin wrappers around the SDK)
internal/printers/ table/json/yaml/name output
sdk/               public Go SDK: client, auth, entities, filters, errors, schema
sdk/registry/      entity kinds and their GraphQL operations
sdk/config/        config file and context resolution
skills/ikctl/      LLM agent skill (embedded in the binary)
examples/          SDK examples
```

Adding a new kind means adding one entry to `sdk/registry/registry.go`.

## License

Apache License 2.0 - see [LICENSE](LICENSE).
