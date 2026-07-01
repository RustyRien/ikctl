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
- `Ctrl-U`, `Ctrl-D`: move up/down by half a page
- `/`: enter filter mode, `Enter` to apply, `Esc` to cancel
- `Enter`: open selected resource overview
- `l`: open selected resource logs
- `Esc`, `q`: close overview/log overlay
- `s`: enter sort mode, press a highlighted column number, then `a` for ascending or `d` for descending to fetch sorted results from the backend, `Esc` to cancel

The main list shows `Shown / Total` in the status bar and loads more rows from the backend automatically as you scroll near the bottom.

## InfraKitchen API

`ik-tui` queries `POST /api/graphql` and uses the `resources` query with `resourcesCount`.
