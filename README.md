# ikctl

`ikctl` is a kubectl-style CLI and k9s-inspired terminal UI for browsing InfraKitchen resources.

## Features

- Kubectl-style `get` commands for InfraKitchen entities
- Live `ikctl log resources <name-or-id>` streaming via GraphQL subscription
- `ikctl edit <entity> <name-or-id>` opens YAML in your default editor and updates via GraphQL mutation
- Live TUI backed by the same InfraKitchen GraphQL entity layer
- Default columns: `NAME`, `TEMPLATE`, `STATE`, `STATUS`, `WORKSPACE`, `AGE`
- Periodic refresh (default `2s`)
- Sorting and fuzzy filtering
- Optional `--no-colors` mode for plain output

## Requirements

- Go `1.25+`
- Access to an InfraKitchen instance
- Either a bearer token for `Authorization: Bearer <token>` or a stored refresh token via `ikctl login`

## Build

```bash
make build
```

This produces `bin/ikctl`.

## Run

```bash
go run . --endpoint http://localhost:8000 --token "$IK_TOKEN"
go run . login --provider github --refresh-token "$IK_REFRESH_TOKEN"
go run . login --provider github
```

Launch the TUI directly:

```bash
go run .
go run . templates
```

Kubectl-style one-shot commands:

```bash
go run . get resources
go run . get resources -o wide
go run . get resources redis-prod
go run . get templates --sort name --sort-order asc
go run . get integrations -o json
go run . describe resource r1
go run . log resources r1
go run . log resources r1 -f
go run . log resources r1 --since 1h
go run . log resources r1 --since 1h -f
go run . log resources r1 --since 2026-07-02T10:30:00Z
go run . login --provider github --refresh-token "$IK_REFRESH_TOKEN"
go run . login --provider github
go run . login --provider microsoft
go run . login --provider guest --scope infra
go run . logout
go run . disable integrations aws-prod
go run . enable integrations aws-prod
go run . delete integrations aws-prod
go run . disable templates aws_redis
go run . enable templates aws_redis
go run . delete templates aws_redis
go run . edit resources redis-prod
go run . edit templates aws_redis
go run . edit integrations aws-prod
```

## Config

Default config path: `~/.config/ikctl/config.yaml`

Stored auth credentials path: `~/.config/ikctl/credentials.yaml`

```yaml
endpoint: http://localhost:8000
token: your-jwt-token
refresh_seconds: 2
insecure_skip_tls_verify: false
no_colors: false
```

Overrides:

- Env: `IK_ENDPOINT`, `IK_TOKEN`, `IK_REFRESH_SECONDS`, `IK_INSECURE_SKIP_TLS_VERIFY`, `IK_NO_COLORS`
- Flags: `--config`, `--endpoint`, `--token`, `--refresh`, `--insecure-skip-tls-verify`, `--no-colors`

Precedence: flags > env > config file > defaults.

## CLI

- `ikctl get <entity>` prints a table and exits.
- `ikctl login --provider <github|microsoft|guest> [--refresh-token <token>]` stores a manually provided refresh token in `~/.config/ikctl/credentials.yaml` with `0600` permissions, then refreshes a bearer token automatically for later commands.
- If `--refresh-token` is omitted, `ikctl login` prompts for it interactively and hides the input like a password.
- `ikctl logout` clears the stored credentials for the active endpoint.
- `ikctl get <entity> <name-or-id>` fetches a single item.
- `ikctl describe <entity> <name-or-id>` fetches a single item and prints YAML by default.
- `ikctl log resources <name-or-id>` shows recent logs and exits.
- `ikctl log resources <name-or-id> -f` shows recent logs, then follows live logs for a resource until `Ctrl-C`.
- `ikctl log resources <name-or-id> --since <duration|rfc3339>` filters the initial log history by time.
- `ikctl log resources <name-or-id> --since <duration|rfc3339> -f` filters the initial log history, then follows live output.
- `ikctl edit <entity> <name-or-id>` opens the entity YAML in `IK_EDITOR`, `K9S_EDITOR`, `KUBE_EDITOR`, or `EDITOR`, then sends the corresponding update mutation.
- `ikctl disable integrations <name-or-id>` sends a disable action for an integration.
- `ikctl enable integrations <name-or-id>` sends an enable action for an integration.
- `ikctl delete integrations <name-or-id>` deletes an integration.
- `ikctl disable templates <name-or-id>` sends a disable action for a template.
- `ikctl enable templates <name-or-id>` sends an enable action for a template.
- `ikctl delete templates <name-or-id>` deletes a template.
- Supported entities: `resources`, `templates`, `integrations`.
- Output formats: `table`, `wide`, `json`, `yaml`, `name`.
- Common flags: `-o`, `--sort`, `--sort-order`, `--limit`, `--filter key=value`.
- Global flags are inherited by subcommands: `--config`, `--endpoint`, `--token`, `--refresh`, `--insecure-skip-tls-verify`, `--no-colors`.
- Token precedence: `--token` > `IK_TOKEN` > stored login credentials in `~/.config/ikctl/credentials.yaml`.
- Live log follow mode uses the InfraKitchen GraphQL `logStream(entityName, entityId)` subscription over `graphql-ws` at `/api/graphql`.
- `--since` accepts either a Go duration like `1h30m` or an RFC3339 timestamp like `2026-07-02T10:30:00Z`.
- Entity-specific filters:
  - resources: `--state`, `--status`, `--label`
  - integrations: `--provider`, `--type`

Auth notes:

- `--refresh-token` should be the exact InfraKitchen auth cookie value for the selected provider.
- Use `guest-token` for `guest`, `github-refresh-token` for `github`, and `microsoft-refresh-token` for `microsoft`.
- `ikctl` stores the refresh token and uses `refreshAuthToken` to mint short-lived bearer tokens for API and websocket requests.

## Keybindings

- `q`, `Ctrl-C`: quit
- `Ctrl-U`, `Ctrl-D`: move up/down by half a page
- `/`: enter filter mode, `Enter` to apply, `Esc` to cancel
- `Enter`: open selected resource overview
- `l`: open selected resource logs
- `x`: disable selected template/integration
- `X`: enable selected template/integration
- `D`: delete selected template/integration
- `E`: edit selected resource/template/integration in your editor
- `Esc`, `q`: close detail view
- `s`: enter sort mode, press a highlighted column number, then `a` for ascending or `d` for descending to fetch sorted results from the backend, `Esc` to cancel
- `e`: choose entity (`r` resources, `t` templates, `i` integrations)

Command mode also supports `:enable`, `:disable`, `:delete`, and `:edit` for the currently selected entity.

The main list shows `Shown / Total` in the status bar and loads more rows from the backend automatically as you scroll near the bottom.

## InfraKitchen API

`ikctl` queries `POST /api/graphql` and uses the InfraKitchen GraphQL API.
