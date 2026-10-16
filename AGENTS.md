# AGENTS.md

Guidance for LLM coding agents working on this repository. For *using* the
CLI against an InfraKitchen server, read [`skills/ikctl/SKILL.md`](skills/ikctl/SKILL.md)
instead (also available as `ikctl skill print`).

## Project overview

`ikctl` is a Go command-line client and public Go SDK for
[InfraKitchen](https://github.com/electrolux-oss/infrakitchen). It talks to the
InfraKitchen GraphQL API (`/api/graphql`) with personal access tokens
(`ik_<id>_<secret>`).

- Module: `github.com/rustyrien/ikctl`, Go 1.26.
- Command shape: `ikctl VERB KIND [NAME|ID] [flags]`.
- The CLI is a thin wrapper: all API logic lives in the `sdk` package, and
  `internal/cli` only parses flags, calls the SDK and prints results.

## Repository layout

| Path | Purpose |
|------|---------|
| `cmd/ikctl/main.go` | Entry point; calls `cli.Execute` and exits with its code |
| `internal/cli/` | Cobra commands, one file per command group (`get.go`, `describe.go`, `apply.go`, `action.go`, `delete.go`, `auth.go`, `config.go`, `discovery.go`, `graphql.go`, `skill.go`, `version.go`); shared code in `root.go` (Factory, global flags), `helpers.go`, `errors.go` (exit codes, structured errors) |
| `internal/printers/` | table / wide / json / yaml / name / custom-columns output, AGE column formatting |
| `internal/version/` | Version info set via `-ldflags` |
| `sdk/` | Public SDK: `client.go` (`New`, options, HTTP), `graphql.go` (`Do`, `GraphQL`), `entity.go` (`EntityClient`: List/ListAll/Get/GetByName/Resolve/Count/Actions/Action/Create/Update/Delete and `*Request` builders), `auth.go` (token parsing/redaction, `Auth.Me`, `Server.Info`), `filter.go`, `errors.go`, `schema.go` (introspection), `fromconfig.go` |
| `sdk/registry/` | The entity kinds (`registry.go`) and their GraphQL operation names, fields and table columns |
| `sdk/config/` | `~/.config/ikctl/config.yaml` contexts; precedence flags > `IKCTL_*` env > file |
| `skills/ikctl/` | `SKILL.md` + `references/` for LLM agents; embedded into the binary by `skills/embed.go` |
| `examples/` | SDK usage examples (must compile) |

## Build, test, lint

```bash
make build   # -> bin/ikctl (CGO disabled, version ldflags)
make test    # go test -race -count=1 ./...
make lint    # go vet + golangci-lint v2.5.0 (config: .golangci.yml)
make fmt     # gofmt -s
```

Before finishing a change, run `make lint test build` and make sure both
report no issues. Imports are grouped by `goimports` with
`github.com/rustyrien/ikctl` as the local prefix.

## Conventions

- **SDK first.** New API behavior goes into `sdk/` with a doc comment on every
  exported identifier; the CLI command only wires flags to it. Keep `sdk/`
  free of cobra, printing and `os.Exit`.
- **Registry-driven kinds.** Every kind follows the same GraphQL naming
  (`x`, `xs`, `xsCount`, `xActions`, `createX`, `updateX`, `xAction`,
  `deleteX`). Adding a kind = one entry in `sdk/registry/registry.go`
  (name, plural, aliases, GraphQL names, fields, columns, verbs). Update the
  kind list in `README.md` and `skills/ikctl/SKILL.md` too.
- **Output.** Data goes to stdout, messages and warnings to stderr. Every data
  command supports `-o json|yaml`; lists use the envelope
  `{"kind": "<Title>List", "items": [...], "total", "offset", "limit"}`.
  Use `addOutputFlag` / `parseOutput` from `helpers.go`.
- **Errors and exit codes** (`internal/cli/errors.go`): 0 ok, 1 error,
  2 usage (`usageErrorf`, `usageArgs`), 3 auth, 4 not found, 5 conflict.
  SDK errors are classified with `sdk.IsNotFound` / `IsUnauthorized` /
  `IsConflict`; with `-o json|yaml` errors are printed as
  `{"error": {"code", "message", "exitCode"}}`. Keep these stable - scripts
  and agents depend on them.
- **Safety.** Destructive or state-changing commands (`delete`, `action`)
  require confirmation via `confirm(...)` and accept `--yes`; mutating
  commands support `--dry-run`, which prints the GraphQL request without
  sending it.
- **Secrets.** Never log or print tokens; use `sdk.RedactToken`. The config
  file is written atomically with `0600` permissions.
- **Naming.** Do not reference other CLI tools by name in code, docs or
  comments; describe behavior on its own terms.
- **Tests.** Table-driven tests next to the code. SDK tests use `httptest`
  servers (`sdk/sdk_test.go`); CLI tests use the `fakeAPI` server in
  `internal/cli/cli_test.go` and assert on output and exit codes. Tests must
  not need a real InfraKitchen server or network access.

## Working against a live server

A local InfraKitchen may run at `http://localhost:7777`. Read-only commands
(`get`, `describe`, `explain`, `api-resources`, `whoami`, `action KIND NAME`
without an action) are safe for manual checks. **Do not run mutating commands
(`create`, `apply`, `delete`, `action ... <action>`, `graphql` mutations)
against a live server unless the user explicitly asks** - `execute` and
`destroy` actions provision or tear down real infrastructure. Prefer
`--dry-run` to verify request shapes.

Never commit tokens, `~/.config/ikctl/config.yaml` contents or `bin/`.

## Keeping docs in sync

When you change commands, flags, kinds, output formats or exit codes, update:

1. `README.md` (user docs),
2. `skills/ikctl/SKILL.md` and `skills/ikctl/references/` (agent docs, embedded
   in the binary - rebuild after editing),
3. this file, if conventions or layout change.
