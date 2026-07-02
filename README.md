# ikctl

`ikctl` is a kubectl-style CLI and k9s-inspired terminal UI for browsing InfraKitchen resources.

## Features

- Kubectl-style `get` commands for InfraKitchen entities
- Live `ikctl log resources <name-or-id>` streaming via GraphQL subscription
- Live TUI backed by the same InfraKitchen GraphQL entity layer
- Default columns: `NAME`, `TEMPLATE`, `STATE`, `STATUS`, `WORKSPACE`, `AGE`
- Periodic refresh (default `2s`)
- Sorting and fuzzy filtering
- Optional `--no-colors` mode for plain output

## Requirements

- Go `1.25+`
- Access to an InfraKitchen instance
- A bearer token for `Authorization: Bearer <token>`

## Build

```bash
make build
```

This produces `bin/ikctl`.

## Run

```bash
go run . --endpoint http://localhost:8000 --token "$IK_TOKEN"
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
```

## Config

Default config path: `~/.config/ikctl/config.yaml`

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
- `ikctl get <entity> <name-or-id>` fetches a single item.
- `ikctl describe <entity> <name-or-id>` fetches a single item and prints YAML by default.
- `ikctl log resources <name-or-id>` streams live logs for a resource until `Ctrl-C`.
- Supported entities: `resources`, `templates`, `integrations`.
- Output formats: `table`, `wide`, `json`, `yaml`, `name`.
- Common flags: `-o`, `--sort`, `--sort-order`, `--limit`, `--filter key=value`.
- Global flags are inherited by subcommands: `--config`, `--endpoint`, `--token`, `--refresh`, `--insecure-skip-tls-verify`, `--no-colors`.
- Live log streaming uses the InfraKitchen GraphQL `logStream(entityName, entityId)` subscription over `graphql-ws` at `/api/graphql`.
- Entity-specific filters:
  - resources: `--state`, `--status`, `--label`
  - integrations: `--provider`, `--type`

## Keybindings

- `q`, `Ctrl-C`: quit
- `Ctrl-U`, `Ctrl-D`: move up/down by half a page
- `/`: enter filter mode, `Enter` to apply, `Esc` to cancel
- `Enter`: open selected resource overview
- `l`: open selected resource logs
- `Esc`, `q`: close detail view
- `s`: enter sort mode, press a highlighted column number, then `a` for ascending or `d` for descending to fetch sorted results from the backend, `Esc` to cancel
- `e`: choose entity (`1` resources, `2` templates, `3` integrations)

The main list shows `Shown / Total` in the status bar and loads more rows from the backend automatically as you scroll near the bottom.

## InfraKitchen API

`ikctl` queries `POST /api/graphql` and uses the InfraKitchen GraphQL API.
