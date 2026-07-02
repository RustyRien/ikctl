# AGENTS.md

`ikctl`: a kubectl-style CLI + k9s-inspired Go TUI over the InfraKitchen GraphQL API (`POST /api/graphql`, `Authorization: Bearer <JWT>`).

## Commands

- Build: `make build` -> `bin/ikctl` (plain `go build .` produces `ik-tui`, the module name; always use the Makefile target or `-o bin/ikctl`).
- Test all: `make test` or `go test ./...`. Only `cmd`, `internal/client`, `internal/config` have tests.
- Single test: `go test ./internal/client -run TestName`.
- Run TUI: `go run .` (or `go run . <entity>` to seed an entity).
- No linter, formatter config, or CI exists. Use `gofmt`/`go vet` before finishing.
- `version`/`commit`/`date` in `cmd/root.go` are `-ldflags "-X"` targets but nothing injects them, so builds report `dev`.

## Naming gotcha

Module path is `github.com/electrolux-oss/ik-tui`; user-facing binary and config are `ikctl` / `~/.config/ikctl/config.yaml`. This mismatch is intentional — do not "fix" it.

## Architecture

One typed **entity registry** (`internal/resource/registry.go` + `descriptors.go`) drives BOTH the CLI and TUI. Entities: `resources`, `templates`, `integrations`. Add new entity behavior there, not per-command.

Mode dispatch (`cmd/root.go`): bare `ikctl` -> TUI; `ikctl <entity>` -> TUI seeded to entity; `ikctl get|describe|log|enable|disable|delete` -> one-shot.

Layers:
- `cmd/` — Cobra commands. Global flags are persistent and inherited by subcommands.
- `internal/app/app.go` — TUI controller: refresh loop, entity switching, async load-more, log/audit drill-down.
- `internal/ui/` — tview views (`app.go`, `table.go`, `header.go`, `styles.go`).
- `internal/client/` — GraphQL over HTTP (`client.go`) + websocket subscription (`subscription.go`).
- `internal/render/`, `internal/tabledata/`, `internal/printer/` — headers/rows per entity, shared table types, output formats.

## TUI threading (deadlock-prone)

Uses `github.com/rivo/tview` + `github.com/gdamore/tcell/v2` (mainstream, updated from previously-forked versions).

Never call `QueueUpdateDraw()` or `app.Draw()` from inside a tview input handler or header-state setter — this has caused hard hangs / `all goroutines are asleep - deadlock`. Open overviews/logs synchronously on the UI thread; use `QueueUpdateDraw()` only from background-fetch completion.

## Backend-contract quirks (verified against InfraKitchen frontend, not guessable)

- Sorting issues backend queries via `tabledata.Header.SortField`; it is not a local reorder.
- Single-item name lookup uses backend filter `name__like` over range `[0,100]` then exact-match confirms — not a client-side scan of the first page (`descriptors.go` `ResolveByName`).
- Resource logs are execution-scoped: find latest execution, then `LogsForResource` -> `LogsForAudit(entity_id, audit_log_id)`. Audit drill-down passes `execution_start = 0`; do not guess timestamps.
- `--since` filters only within the already-fetched history window (client-side), not a backend time filter.
- Live follow (`log ... -f` only) uses subscription `logStream`, singular `entityName: "resource"`, subprotocol `graphql-transport-ws`, token in the `connection_init` payload. Subscription URL is the GraphQL URL with scheme swapped `http->ws` / `https->wss`.
- Integration actions: `integrationAction(id: UUID!, input: { action })` with strings `enable`/`disable`; delete via `deleteIntegration(id: UUID!)`.

## Config precedence

flags > env (`IK_*`) > config file > defaults. `--no-colors` / `IK_NO_COLORS` / `no_colors` disable the color palette; CLI color output writes raw ANSI (not `tview.TranslateANSI`).

## Deep context

`.opencode/agents/session-compaction.md` holds detailed decision history and a per-file map. Consult it before large changes; keep it in sync when the above contracts change.
