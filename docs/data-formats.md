# Data Formats

> Specification of all data files stored locally and parsed from the API.

## File Locations

All data files live under `~/.pim/` (created automatically on first run):

```text
~/.pim/
├── eligible-roles.json   # Cached API response (24h TTL)
├── cache-meta            # Cache timestamp
└── activations.json      # Local activation records
```

---

## 1. eligible-roles.json

**Source:** Cached response from the Azure ARM eligibility API.
**Written by:** Activate mode (on cache miss or `--no-cache`).
**Read by:** Activate mode (on cache hit).
**TTL:** 24 hours (86400 seconds).

This is the raw JSON response from:

```text
GET .../roleEligibilityScheduleInstances?$filter=asTarget()&api-version=2020-10-01
```

### Schema

```jsonc
{
  "value": [
    {
      "id": "string", // Full ARM resource ID of the schedule instance
      "name": "string", // GUID
      "type": "string", // "Microsoft.Authorization/roleEligibilityScheduleInstances"
      "properties": {
        "principalId": "string", // GUID of the principal (may be a group)
        "principalType": "string", // "Group" | "User"
        "roleDefinitionId": "string", // Full ARM path to role definition
        "scope": "string", // Full ARM path to scope
        "memberType": "string", // "Group" | "Direct"
        "status": "string", // "Provisioned"
        "createdOn": "string", // ISO 8601
        "startDateTime": "string", // ISO 8601
        "endDateTime": "string", // ISO 8601 (eligibility expiry, not activation expiry)
        "expandedProperties": {
          "principal": {
            "displayName": "string",
            "id": "string",
            "type": "string", // "Group" | "User"
          },
          "roleDefinition": {
            "displayName": "string", // e.g. "Contributor", "Reader"
            "id": "string",
            "type": "string", // "BuiltInRole" | "CustomRole"
          },
          "scope": {
            "displayName": "string", // e.g. "RG-PRD-APP1-001" or subscription name
            "id": "string",
            "type": "string", // "resourcegroup" | "subscription" | resource type
          },
        },
      },
    },
  ],
}
```

### Example

```json
{
  "value": [
    {
      "id": "/subscriptions/5a325a18-5032-42f2-8cfb-02eee35d5304/resourceGroups/RG-PRD-APP1-001/providers/Microsoft.Authorization/roleEligibilityScheduleInstances/9a30e31e-3b23-4e21-8c4d-80bc4a27b85d",
      "name": "9a30e31e-3b23-4e21-8c4d-80bc4a27b85d",
      "properties": {
        "principalId": "44b5d3fb-b144-4c11-9cf2-12bfd88a0442",
        "principalType": "Group",
        "roleDefinitionId": "/subscriptions/5a325a18-5032-42f2-8cfb-02eee35d5304/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c",
        "scope": "/subscriptions/5a325a18-5032-42f2-8cfb-02eee35d5304/resourceGroups/RG-PRD-APP1-001",
        "memberType": "Group",
        "status": "Provisioned",
        "expandedProperties": {
          "principal": {
            "displayName": "AD-SEC-ALL-COM-APP1-ADMINS",
            "id": "44b5d3fb-b144-4c11-9cf2-12bfd88a0442",
            "type": "Group"
          },
          "roleDefinition": {
            "displayName": "Contributor",
            "id": "/subscriptions/5a325a18-5032-42f2-8cfb-02eee35d5304/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c",
            "type": "BuiltInRole"
          },
          "scope": {
            "displayName": "RG-PRD-APP1-001",
            "id": "/subscriptions/5a325a18-5032-42f2-8cfb-02eee35d5304/resourceGroups/RG-PRD-APP1-001",
            "type": "resourcegroup"
          }
        }
      }
    }
  ]
}
```

---

## 2. cache-meta

**Source:** Written by the CLI after a successful API fetch.
**Format:** Single line containing Unix epoch timestamp (seconds since 1970-01-01).
**Read by:** Activate mode to check cache freshness.

### Schema

```
{epoch_seconds}
```

### Example

```
1771931980
```

### Cache Logic

```
cache_age = current_epoch - cached_epoch
if cache_age < 86400 (24h) AND --no-cache not set:
    use cached eligible-roles.json
else:
    fetch from API, write both files
```

---

## 3. activations.json

**Source:** Written by the CLI after successful role activations.
**Format:** JSON array of activation records.
**Read by:** Status mode (to display justification for active roles).

### Schema

```jsonc
[
  {
    "scope": "string",              // Full ARM path to the scope
    "roleDefinitionId": "string",   // Full ARM path to the role definition
    "roleName": "string",           // Human-readable role name
    "scopeName": "string",          // Human-readable scope name
    "justification": "string",      // User-provided justification text
    "duration": "string",           // Human-readable duration label
    "activatedAt": "string",        // ISO 8601 UTC timestamp
    "expiresEpoch": number          // Unix epoch when activation expires
  }
]
```

### Example

```json
[
  {
    "scope": "/subscriptions/5a325a18-5032-42f2-8cfb-02eee35d5304/resourceGroups/RG-PRD-APP1-001",
    "roleDefinitionId": "/subscriptions/5a325a18-5032-42f2-8cfb-02eee35d5304/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c",
    "roleName": "Contributor",
    "scopeName": "RG-PRD-APP1-001",
    "justification": "Deploying hotfix to production",
    "duration": "1 hour",
    "activatedAt": "2026-02-24T12:00:00.000Z",
    "expiresEpoch": 1771935600
  }
]
```

### Lifecycle

1. **On activation:** New entries are appended to the array
2. **On next activation:** Expired entries (`expiresEpoch < now`) are pruned
3. **On status display:** Entries are read and matched by composite key

### Lookup Key

Justification is looked up by the composite key:

```
key = scope + "|" + roleDefinitionId
```

The status mode reads active assignments from the API (which don't include
justification), then looks up the justification from this local file using
the composite key. If the role was activated outside this CLI (e.g. Azure
Portal), the justification will show "—".

---

## 4. RG Name Convention

Example resource group naming convention encoding environment and application:

```
XEZZZZCCCCZZNN
│││   ││││
│││   ││││
│││   └┤└┤
│││    │  └── Sequence number + resource type suffix
│││    └───── 4-character application code (positions 8-11, 1-indexed)
│││
││└── Region/location code (positions 3-7)
│└─── Environment (position 2): P=Prod, Q=QA, T=Test, D=Dev
└──── Cloud provider prefix (position 1): B=Azure
```

### Decode Functions

| Function            | Input   | Position                      | Output                              |
| ------------------- | ------- | ----------------------------- | ----------------------------------- |
| `decode_env()`      | RG name | char at index 1 (0-based)     | "Prod", "QA", "Test", "Dev", or "—" |
| `decode_app_code()` | RG name | chars at index 7-10 (0-based) | 4-char code or "—"                  |

### Examples

| RG Name           | Environment | App Code |
| ----------------- | ----------- | -------- |
| `RG-PRD-APP1-001` | Prod        | APP1     |
| `RG-PRD-APP2-001` | Prod        | APP2     |
| `RG-DEV-APP4-001` | Dev         | APP4     |
| `RG-TST-APP5-001` | Test        | APP5     |
| `RG-QA-APP3-001` | QA          | APP3     |

### Edge Cases

- Names shorter than 11 characters → app code returns "—"
- Non-resourcegroup scopes → both env and app code return "—"
- Unrecognised 2nd character → env returns "—"

---

## 5. Activation Request Body

The JSON body sent to the activation PUT endpoint:

### Schema

```jsonc
{
  "properties": {
    "principalId": "string", // Entra ID object ID of the user
    "roleDefinitionId": "string", // Full ARM path to role definition
    "requestType": "SelfActivate", // Literal string, always "SelfActivate"
    "justification": "string", // User-provided text
    "scheduleInfo": {
      "startDateTime": "string", // ISO 8601 UTC (e.g. "2026-02-24T12:00:00.000Z")
      "expiration": {
        "type": "AfterDuration", // Literal string
        "duration": "string", // ISO 8601 duration (PT30M, PT1H, PT2H, PT4H)
      },
    },
  },
}
```

### Duration Mapping

| Label      | ISO 8601 | Seconds |
| ---------- | -------- | ------- |
| 30 minutes | `PT30M`  | 1800    |
| 1 hour     | `PT1H`   | 3600    |
| 2 hours    | `PT2H`   | 7200    |
| 4 hours    | `PT4H`   | 14400   |

### Go Type Mapping

```go
type ActivationRequest struct {
    Properties struct {
        PrincipalID      string `json:"principalId"`
        RoleDefinitionID string `json:"roleDefinitionId"`
        RequestType      string `json:"requestType"` // always "SelfActivate"
        Justification    string `json:"justification"`
        ScheduleInfo     struct {
            StartDateTime string `json:"startDateTime"`
            Expiration    struct {
                Type     string `json:"type"`     // always "AfterDuration"
                Duration string `json:"duration"` // PT30M, PT1H, PT2H, PT4H
            } `json:"expiration"`
        } `json:"scheduleInfo"`
    } `json:"properties"`
}
```
