# User Guide

> Complete guide for using the PIM Role Activator CLI.

## Quick Start

```bash
# Check what roles are currently active
pim

# Activate roles
pim activate

# Test the activation flow without actually activating
pim activate --dry-run

# Force-refresh the role cache
pim activate --no-cache
```

---

## Installation

### Prerequisites

| Tool             | Install                                                                              | Purpose           |
| ---------------- | ------------------------------------------------------------------------------------ | ----------------- |
| Azure CLI (`az`) | [Install guide](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli-macos) | Authentication    |
| Go 1.25+         | [go.dev/dl](https://go.dev/dl/)                                                      | Build from source |

### Setup

1. Clone the repository and build the binary:

```bash
git clone https://github.com/jenewland1999/pim-role-activator-cli
cd pim-role-activator-cli
go build -o pim ./cmd/pim
sudo mv pim /usr/local/bin/pim
```

1. Log in to Azure:

```bash
az login
```

1. Verify it works:

```bash
pim
```

---

## Commands

### `pim` — Show Active Roles

Running `pim` with no arguments shows a table of currently active PIM roles:

```text
╔══════════════════════════════════════╗
║     PIM Role Activator CLI           ║
╚══════════════════════════════════════╝

  ✔ 3 active PIM role(s):

  App  │ Env  │ Scope                    │ Role                           │ Expires In   │ Justification
  ──────────────────────────────────────────────────────────────────────────────────────────────────────
  APP1 │ Prod │ RG-PRD-APP1-001          │ Contributor                    │ 45m          │ Deploy hotfix
  APP2 │ Prod │ RG-PRD-APP2-001          │ Contributor                    │ 1h 20m       │ Deploy hotfix
  APP3 │ Prod │ RG-PRD-APP3-001          │ Reader                         │ 3h 55m       │ Investigating issue

  Run pim activate to activate more roles.
```

**Columns:**

- **App** — 4-character application code from characters 8–11 of the RG name
- **Env** — Environment decoded from the RG name (P=Prod, Q=QA, T=Test, D=Dev)
- **Scope** — Resource group or resource name
- **Role** — Azure role definition name
- **Expires In** — Time remaining until activation expires
- **Justification** — The text entered when activating (from local state)

> **Note:** The justification column only shows text for roles activated through
> this CLI. Roles activated via the Azure Portal will show "—".

---

### `pim activate` — Activate Roles

Interactive 3-step workflow:

#### Step 1: Select Roles

An interactive list of all eligible PIM roles appears. Navigate with your
keyboard:

| Key         | Action                                                                                 |
| ----------- | -------------------------------------------------------------------------------------- |
| `↑` / `↓`   | Move cursor up/down (wraps around)                                                     |
| `Space`     | Toggle selection on the highlighted role                                               |
| `a`         | Select **all** visible roles                                                           |
| `n`         | Deselect **all** visible roles                                                         |
| `g`         | **Group select** — auto-selects roles with APP2, APP4, APP5, or APP3 in the scope name |
| `/`         | Enter **search mode** — type to filter roles by name or scope                          |
| `Backspace` | Clear the current search filter                                                        |
| `Enter`     | **Confirm** selection (need at least 1 selected)                                       |
| `c`         | **Cancel** and exit                                                                    |

The list scrolls automatically when there are more roles than fit on screen.
Scroll indicators ("↑ N more above" / "↓ N more below") show when content
is above or below the viewport.

**Search:** Press `/`, type your search term, and press Enter. The list
filters to matching roles (case-insensitive). Your selections are preserved
even when filtering. Press Backspace to clear the filter.

#### Step 2: Enter Justification

Type a justification that will be sent to Azure (appears in audit logs) and
stored locally. This field is required.

```text
Step 2: Enter justification
  Justification: Deploying hotfix for critical bug
```

#### Step 3: Choose Duration

Select how long the roles should be active:

```text
Step 3: Select activation duration

  [1] 30 minutes
  [2] 1 hour
  [3] 2 hours
  [4] 4 hours

  Duration [1-4]:
```

#### Confirmation

A summary is shown before any API calls are made:

```text
─── Summary ────────────────────────────────────────────────────────────
  Roles:
    ▸ APP1  Prod  RG-PRD-APP1-001           Contributor
    ▸ APP2  Prod  RG-PRD-APP2-001           Contributor
  Justification: Deploying hotfix for critical bug
  Duration:      1 hour
────────────────────────────────────────────────────────────────────────
```

---

### Flags

#### `--dry-run`

Walks through the full interactive flow (role selection, justification,
duration) but **does not send any API requests**. Shows the summary and
exits with "Dry run complete."

```bash
pim activate --dry-run
```

Useful for:

- Testing the CLI without side effects
- Practising the workflow
- Verifying which roles would be selected

#### `--no-cache`

Forces a fresh fetch of eligible roles from the Azure API, bypassing the
24-hour cache.

```bash
pim activate --no-cache
```

Useful when:

- New roles have been assigned to you
- Roles have been removed
- Cache data seems stale

#### `--help` / `-h`

Shows usage information:

```bash
pim --help
```

#### `--debug-timings`

Emits structured timing logs for performance diagnosis of key command stages
(for example: auth, eligible-role fetch, active-role refresh, activation submit).

```bash
pim --debug-timings
pim activate --debug-timings
```

---

## Caching

Eligible roles are cached for **24 hours** in `~/.pim/`:

- `eligible-roles-data.json` — serialised eligible roles (JSON)
- `eligible-roles-meta.json` — cache metadata with `written_at`

The cache is automatically used when:

- The cache file exists
- The cache is less than 24 hours old
- `--no-cache` was not passed

When the cache is used, you'll see:

```text
Using cached roles (1234m until refresh). Use --no-cache to bypass.
```

To manually clear the cache:

```bash
rm -rf ~/.pim/eligible-roles-data.json ~/.pim/eligible-roles-meta.json
```

---

## Local State

Activation records are stored in `~/.pim/activations.json`. This file:

- Records the justification, scope, role, and expiry for each activation
- Is read by `pim` (status mode) to display justifications
- Is automatically pruned of expired entries on each activation
- Only contains records from activations done through this CLI

To clear the state:

```bash
rm ~/.pim/activations.json
```

---

## Troubleshooting

### "Not logged in" / authentication errors

The CLI uses `DefaultAzureCredential` from the Azure SDK, which reads the
same credentials that `az login` creates. If you see an authentication error:

```bash
az login
```

If you're on a machine without a browser:

```bash
az login --use-device-code
```

### "No eligible PIM role assignments found"

This means the API returned no results. Possible causes:

- You're logged in as the wrong account (`az account show`)
- Your eligibility has expired
- Config is stale or points to the wrong subscriptions (`pim setup`)
- Try `--no-cache` to bypass a stale cache

### "Failed to activate"

Per-role failures can happen when:

- The role is already active
- The PIM policy requires approval
- The maximum activation count is exceeded
- The requested duration exceeds the policy maximum

The CLI continues activating remaining roles even if one fails.

### Wrong subscription or principal ID

Re-run setup to refresh your saved config:

```bash
pim setup
```

Find your object ID with:

```bash
az ad signed-in-user show --query "id" -o tsv
```
