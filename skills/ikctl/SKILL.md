---
name: ikctl
description: Operate an InfraKitchen platform with the ikctl CLI - list, inspect, create, update, approve, provision, destroy and delete InfraKitchen objects (resources, templates, integrations, source code versions, workspaces, storages, secrets, executors, projects, services, blueprints, workflows, tasks, audit logs) and run raw GraphQL against the InfraKitchen API. Use when the user mentions InfraKitchen, ikctl, IK resources/templates, or asks about the state of infrastructure managed by InfraKitchen.
---

# ikctl - InfraKitchen CLI

`ikctl` is a command-line client for the InfraKitchen API. Commands follow a
`ikctl VERB KIND [NAME|ID] [flags]` pattern. Objects can be referenced by
name or UUID.

## Rules for agents

1. **Always pass `-o json`** when you need to read results. JSON goes to stdout;
   errors go to stderr (as JSON when `-o json` is set) with a stable `code`.
2. **Discover before guessing.** Use `ikctl api-resources -o json` for kinds and
   verbs, and `ikctl explain KIND [--input] -o json` for fields and input types.
3. **Preview mutations first.** Run `create`/`apply`/`action` with `--dry-run`,
   show the user what will happen, and only then run it for real.
4. **Mutations need explicit consent.** `delete` and `action` refuse to run
   non-interactively without `--yes`. Only add `--yes` after the user has agreed
   to that specific change. Never destroy or delete without being asked.
5. **Never print or log the token.** Do not echo `IKCTL_TOKEN`, do not pass the
   token with `--token` on the command line, and do not `cat` the config file
   (`ikctl config view` redacts tokens).
6. New resources usually stop in status `APPROVAL_PENDING`; a human (or the
   user, explicitly) must `approve` them before `execute` provisions them.

## Authentication

Check first: `ikctl whoami -o json`. Exit code 3 means not logged in or token invalid.

```bash
# The user (not the agent) should provide the token, e.g. from a secret store:
printf '%s' "$IK_TOKEN" | ikctl login --server https://infrakitchen.example.com --token-stdin
# or, without a config file:
export IKCTL_SERVER=https://infrakitchen.example.com IKCTL_TOKEN=...   # set by the user
```

Contexts: `ikctl config get-contexts`, `ikctl config use-context NAME`, or
`--context NAME` per command. Config file: `~/.config/ikctl/config.yaml` (0600).

## Reading

```bash
ikctl api-resources -o json                      # kinds, short names, verbs
ikctl get resources -o json                      # first 100, with "total"
ikctl get rs --name-like redis -o json           # substring match on name
ikctl get rs --field-selector status=ERROR -o json
ikctl get rs --field-selector template__name__like=eks --sort-by createdAt:desc --limit 20 -o json
ikctl get templates -l aws,eks -o json           # has all labels
ikctl get rs --all -o json                       # every page
ikctl get rs --select "id name outputs" -o json  # custom GraphQL selection
ikctl get resource my-resource -o json           # one object (bare, not a list)
ikctl describe resource my-resource -o json      # {"kind","item","allowedActions"}
ikctl explain resource -o json                   # fields of ResourceType
ikctl get tasks --field-selector entity_id=<uuid> --sort-by createdAt:desc -o json
ikctl get workflows --sort-by createdAt:desc --limit 5 -o json
ikctl get audit-logs --field-selector entity_id=<uuid> -o json
```

List output shape: `{"kind": "ResourceList", "items": [...], "total": N, "offset": 0, "limit": 100}`.
If `offset + len(items) < total`, more pages exist (use `--offset` or `--all`).

Filter syntax (snake_case field names) is in [references/filters.md](references/filters.md)
(`ikctl skill print references/filters.md`).

## Lifecycle actions

```bash
ikctl action resource my-resource -o json             # {"allowedActions": [...]}
ikctl action resource my-resource approve --dry-run   # preview
ikctl action resource my-resource approve --yes -o json
ikctl action resource my-resource execute --yes -o json   # provision / apply
ikctl action resource my-resource destroy --yes -o json   # destroy infrastructure
```

Typical resource flow: create -> `APPROVAL_PENDING` -> approve -> `dryrun` (optional)
-> `execute` -> state `PROVISIONED`, status `DONE`. Poll with
`ikctl get resource NAME -o json` and check `state` / `status`
(states: PROVISION, PROVISIONED, DESTROY, DESTROYED; statuses include QUEUED,
IN_PROGRESS, DONE, ERROR, APPROVAL_PENDING, PENDING, REJECTED, READY).
An action not in `allowedActions` fails with exit code 5 and lists the allowed ones.

## Creating and updating

1. `ikctl explain resource --input -o json` - read `createInput.inputFields` (`required: true` must be set).
2. Look up referenced ids (`ikctl get templates --name-like ... -o json`, etc.).
3. Write a manifest and preview it:

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

```bash
ikctl create -f my-redis.yaml --dry-run -o json   # shows the GraphQL request
ikctl create -f my-redis.yaml -o json
ikctl apply -f my-redis.yaml -o json              # update if id given or name exists, else create
```

Top-level spec keys may be snake_case or camelCase; nested values are sent unchanged.
Use `-f -` to read from stdin. Multiple documents can be separated with `---`.

## Deleting

```bash
ikctl delete resource my-resource --dry-run -o json
ikctl delete resource my-resource --yes -o json
```

Provisioned resources normally have to be destroyed (and reach `DESTROYED`) before deletion.

## Anything else: raw GraphQL

```bash
ikctl graphql '{ serverInfo { version } }'
ikctl graphql -q 'query($id: UUID!) { resourceTree(id: $id, direction: "children") { id name children { id name } } }' --var id=<uuid>
ikctl graphql -q 'query($f: JSON) { resourcesCount(filter: $f) }' --vars '{"f": {"status": "ERROR"}}'
ikctl graphql '{ __schema { queryType { fields { name } } mutationType { fields { name } } } }'
```

Conventions: `x(id)`, `xs(filter, sort, range)`, `xsCount(filter)`, `xActions(id)`,
`createX(input)`, `updateX(id, input)`, `xAction(id, input: {action})`, `deleteX(id)`.

## Exit codes

| Code | Meaning | `error.code` examples |
|------|---------|-----------------------|
| 0 | success | |
| 1 | other error (network, server) | `INTERNAL`, `HTTP_ERROR`, `TIMEOUT` |
| 2 | usage error (bad flags/kind, missing `--yes`) | `USAGE` |
| 3 | not logged in / invalid token / permission denied | `UNAUTHORIZED`, `ACCESS_DENIED`, `NOT_CONFIGURED` |
| 4 | object not found | `NOT_FOUND` |
| 5 | conflict: wrong state, validation, ambiguous name, action not allowed | `WRONG_STATE`, `VALIDATION`, `AMBIGUOUS`, `ACTION_NOT_ALLOWED` |

On an ambiguous name (exit 5, `AMBIGUOUS`), retry with the UUID.

## Troubleshooting

- `-v` logs HTTP and GraphQL requests to stderr (tokens are never logged).
- `ikctl version -o json` shows client and server versions.
- `--timeout 2m` for slow servers; `--insecure-skip-tls-verify` only for test servers.
