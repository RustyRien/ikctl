# ik-tui

`ik-tui` is a k9s-inspired terminal UI for browsing InfraKitchen resources.

## Features

- Live resource list backed by InfraKitchen GraphQL
- Default columns: `NAME`, `TEMPLATE`, `STATE`, `STATUS`, `WORKSPACE`, `AGE`
- Periodic refresh (default `2s`)
- Sorting and fuzzy filtering

## Requirements

- Go `1.25+`
- Access to an InfraKitchen instance
- A bearer token for `Authorization: Bearer <token>`

## Build

```bash
go build ./...
```

## Run

```bash
go run . --endpoint http://localhost:8000 --token "$IK_TOKEN"
```

## Config

Default config path: `~/.config/ik-tui/config.yaml`

```yaml
endpoint: http://localhost:8000
token: your-jwt-token
refresh_seconds: 2
insecure_skip_tls_verify: false
```

Overrides:

- Env: `IK_ENDPOINT`, `IK_TOKEN`, `IK_REFRESH_SECONDS`, `IK_INSECURE_SKIP_TLS_VERIFY`
- Flags: `--config`, `--endpoint`, `--token`, `--refresh`, `--insecure-skip-tls-verify`

Precedence: flags > env > config file > defaults.

## Keybindings

- `q`, `Ctrl-C`: quit
- `/`: enter filter mode, `Enter` to apply, `Esc` to cancel
- `1`..`6`: sort by a visible column, repeat to toggle direction

## InfraKitchen API

`ik-tui` queries `POST /api/graphql` and uses the `resources` query with `resourcesCount`.
