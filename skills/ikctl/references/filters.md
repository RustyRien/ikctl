# ikctl filter reference

List commands (`ikctl get KIND`) translate flags into the InfraKitchen GraphQL
`filter` argument. All conditions are combined with AND.

## Flags

| Flag | Meaning | Example |
|------|---------|---------|
| `--name-like S` | case-insensitive substring of the name | `--name-like redis` |
| `-l, --selector a,b` | object has **all** labels | `-l aws,prod` |
| `--field-selector k=v` | field condition (repeatable) | `--field-selector status=ERROR` |
| `--field-selector k!=v` | not equal | `--field-selector state!=DESTROYED` |
| `--filter JSON` | raw filter object | `--filter '{"or":[{"status":"ERROR"},{"status":"CANCELLED"}]}'` |
| `--sort-by f[:asc\|desc]` | sort (camelCase or snake_case) | `--sort-by createdAt:desc` |
| `--limit N --offset M` | paging (default limit 100) | `--limit 20 --offset 40` |
| `--all` | fetch all pages | |

## Field names

Filter keys are **snake_case** database column names: `name`, `status`,
`state`, `created_at`, `template_id`, `workspace_id`, `entity_id`,
`integration_provider`, `source_code_url`, ... Use `ikctl explain KIND` to see
the fields (shown in camelCase; convert to snake_case for filters).

Enum values (status, state) are matched case-insensitively, e.g. `status=done`
and `status=DONE` are equivalent.

## Operators

Append an operator to the key with a double underscore:

| Operator | Meaning | Example |
|----------|---------|---------|
| *(none)* / `__eq` | equals (a list value means "any of") | `status=DONE` |
| `__not_eq` | not equal | `state__not_eq=DESTROYED` |
| `__in` | value in list (comma-separated) | `status__in=ERROR,CANCELLED` |
| `__like` | case-insensitive substring | `name__like=prod` |
| `__not_like` | does not contain | `name__not_like=test` |
| `__is_none` | is null (`true`) / not null (`false`) | `storage_id__is_none=true` |
| `__any` | to-many relation contains any id | `parents__any=<uuid>` |
| `__contains_all` | JSON array contains all values | `labels__contains_all=aws,eks` |

## Relation fields

Filter on a related object's column with `relation__column[__op]`:

```bash
ikctl get resources --field-selector template__name__like=eks
ikctl get resources --field-selector template__template=dummy_redis
```

## OR

Use `--filter` with an `or` list (each element is a filter object):

```bash
ikctl get resources --filter '{"or": [{"status": "ERROR"}, {"state": "DESTROY"}]}'
```

`--filter` can be combined with the other flags (AND).

## Values

Values are parsed as JSON when possible (`true`, `false`, `null`, numbers,
`["a","b"]`), otherwise as strings. For `__like` and `__not_like` the value is
always a string.
