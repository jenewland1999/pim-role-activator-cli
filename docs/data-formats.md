# Data Formats

This document describes the local persisted data used by the CLI and the activation payload shape sent to Azure.

## Local Storage Root

All local files are stored in `~/.pim/`.

```text
~/.pim/
├── config.json
├── eligible-roles-data.json
├── eligible-roles-meta.json
├── active-roles-data.json
├── active-roles-meta.json
└── activations.json
```

## config.json

Persisted user configuration.

```jsonc
{
  "$schema": "https://github.com/jenewland1999/pim-role-activator-cli/docs/config.schema.json",
  "subscriptions": [
    { "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", "name": "Production" },
  ],
  "principal_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "cache_ttl_hours": 24,
  "scope_pattern": "^.(?P<env>[PQTD]).{5}(?P<app>.{4})",
  "env_labels": { "P": "Prod", "D": "Dev", "Q": "QA", "T": "Test" },
  "durations": [
    { "label": "30 minutes", "iso8601": "PT30M", "minutes": 30 },
    { "label": "1 hour", "iso8601": "PT1H", "minutes": 60 },
  ],
}
```

Notes:

- `subscriptions` and `principal_id` are required for normal operation.
- `durations` is optional; when omitted, built-in defaults are used.
- `$schema` is optional and intended for editor tooling.

## eligible-roles-meta.json

Metadata for eligible-role cache freshness.

```json
{
  "written_at": "2026-03-04T10:30:00Z"
}
```

## eligible-roles-data.json

Cached array of eligible roles (internal projection, not raw ARM response).

```jsonc
[
  {
    "role_name": "Contributor",
    "scope_name": "example-scope",
    "scope_type": "scope",
    "subscription_name": "Production",
    "role_definition_id": "/subscriptions/.../providers/Microsoft.Authorization/roleDefinitions/...",
    "scope": "/subscriptions/.../providers/Microsoft.Authorization/...",
    "environment": "Prod",
    "app_code": "APP1",
    "selected": false,
  },
]
```

## active-roles-meta.json

Metadata for active-role cache freshness.

```json
{
  "written_at": "2026-03-04T10:45:00Z"
}
```

## active-roles-data.json

Cached array of active roles with absolute expiry timestamps.

```jsonc
[
  {
    "role_name": "Contributor",
    "scope_name": "example-scope",
    "scope_type": "scope",
    "subscription_name": "Production",
    "environment": "Prod",
    "app_code": "APP1",
    "expires_at": "2026-03-04T11:30:00Z",
    "justification": "Deploying hotfix",
  },
]
```

Notes:

- `expires_at` is converted back to relative `ExpiresIn` at read time.
- Cache write TTL is dynamically bounded by the soonest-expiring active role (and capped by `cache_ttl_hours`).

## activations.json

Local activation history used for justification lookup in status output.

```jsonc
[
  {
    "scope": "/subscriptions/.../providers/Microsoft.Authorization/...",
    "roleDefinitionId": "/subscriptions/.../providers/Microsoft.Authorization/roleDefinitions/...",
    "roleName": "Contributor",
    "scopeName": "example-scope",
    "justification": "Deploying hotfix",
    "duration": "1 hour",
    "activatedAt": "2026-03-04T10:30:00Z",
    "expiresEpoch": 1771935600,
  },
]
```

Lookup key used internally:

```text
scope + "|" + roleDefinitionId
```

## Azure Activation Request Body

The CLI submits `SelfActivate` requests to `RoleAssignmentScheduleRequests`.

```jsonc
{
  "properties": {
    "principalId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "roleDefinitionId": "/subscriptions/.../providers/Microsoft.Authorization/roleDefinitions/...",
    "requestType": "SelfActivate",
    "justification": "Deploying hotfix",
    "scheduleInfo": {
      "startDateTime": "2026-03-04T10:30:00Z",
      "expiration": {
        "type": "AfterDuration",
        "duration": "PT1H",
      },
    },
  },
}
```

## Duration Mapping

Default durations when `config.json` does not define custom `durations`:

| Label      | ISO 8601 |
| ---------- | -------- |
| 30 minutes | `PT30M`  |
| 1 hour     | `PT1H`   |
| 2 hours    | `PT2H`   |
| 4 hours    | `PT4H`   |
