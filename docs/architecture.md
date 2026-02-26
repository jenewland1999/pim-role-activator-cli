# Architecture

> Describes the system's structure, control flow, data flow, and component
> responsibilities for the Go implementation.

## Overview

PIM Role Activator CLI is a compiled Go binary (`pim`) that provides two
modes of operation:

| Mode         | Command        | Purpose                                         |
| ------------ | -------------- | ----------------------------------------------- |
| **Status**   | `pim`          | Display currently active PIM role assignments   |
| **Activate** | `pim activate` | Interactive workflow to activate eligible roles |

The CLI uses the **Azure SDK for Go** to communicate with the Azure Resource
Manager REST API, and stores local cache/state in `~/.pim/`.

## Project Structure

```text
pim-role-activator-cli/
├── cmd/
│   └── pim/
│       └── main.go              # Entry point, cobra commands, top-level glue
├── internal/
│   ├── azure/
│   │   ├── client.go            # Azure SDK client setup + DefaultAzureCredential
│   │   ├── eligible.go          # Fetch eligible PIM roles (asTarget() filter)
│   │   ├── active.go            # Fetch active role assignments (Activated filter)
│   │   └── activate.go          # Self-activate a role (SelfActivate PUT)
│   ├── cache/
│   │   └── cache.go             # File-based eligible-role cache with 24h TTL
│   ├── config/
│   │   └── config.go            # Constants: subscription ID, principal ID, scope
│   ├── model/
│   │   ├── role.go              # Role, ActiveRole, DurationOption types
│   │   ├── activation.go        # ActivationRecord + ActivationResult types
│   │   └── rgname.go            # DecodeEnv() + DecodeAppCode() from RG name
│   ├── state/
│   │   └── state.go             # activations.json read/write/prune
│   └── tui/
│       ├── selector.go          # Bubbletea role selector with row render cache
│       ├── duration.go          # Bubbletea duration picker
│       ├── loader.go            # Spinner wrapper for blocking API calls
│       ├── status.go            # Status table + summary + results display
│       └── styles.go            # Lipgloss style definitions + helper funcs
├── docs/
├── go.mod
└── go.sum
```

## System Diagram

```text
┌──────────────────────────────────────────────────────────┐
│                     User Terminal                        │
│                                                          │
│  pim              → Status Mode (read-only)              │
│  pim activate     → Activate Mode (interactive + write)  │
│  pim activate --dry-run  → Activate Mode (no API calls)  │
│  pim activate --no-cache → Activate Mode (skip cache)    │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│                  pim (Go binary)                         │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │ cobra        │  │ azidentity   │  │ model/       │   │
│  │ rootCmd      │  │ DefaultAzure │  │ rgname.go    │   │
│  │ activateCmd  │  │ Credential   │  │ Env + App    │   │
│  │ --dry-run    │  │              │  │ Code decoder │   │
│  │ --no-cache   │  │              │  │              │   │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘   │
│         │                 │                              │
│         ▼                 ▼                              │
│  ┌─────────────────────────────────────────────────────┐ │
│  │              Mode Router (cobra)                    │ │
│  │  rootCmd.RunE  → Status Flow                        │ │
│  │  activateCmd.RunE → Activate Flow                   │ │
│  └──────────┬──────────────────────┬───────────────────┘ │
│             │                      │                     │
│    ┌────────▼────────┐    ┌────────▼────────┐            │
│    │  Status Mode    │    │  Activate Mode  │            │
│    │                 │    │                 │            │
│    │  azure/active   │    │  1. cache/get   │            │
│    │  .FetchActive   │    │     or azure/   │            │
│    │  Roles()        │    │     eligible    │            │
│    │  tui.PrintStatus│    │  2. tui/select  │            │
│    │                 │    │  3. huh justify │            │
│    └────────┬────────┘    │  4. tui/duration│            │
│             │             │  5. tui/summary │            │
│             │             │  6. azure/activ │            │
│             │             │     ate (parallel)│          │
│             │             │  7. state/save  │            │
│             │             └────────┬────────┘            │
└─────────────┼──────────────────────┼─────────────────────┘
              │                      │
              ▼                      ▼
┌─────────────────────┐  ┌───────────────────────┐
│  Azure ARM REST API │  │  Local Storage        │
│  (management.azure  │  │  (~/.pim/)            │
│  .com) via SDK      │  │                       │
│                     │  │  eligible-roles.json  │
│  • roleEligibility  │  │  cache-meta           │
│    ScheduleInstances│  │  activations.json     │
│  • roleAssignment   │  │                       │
│    ScheduleInstances│  └───────────────────────┘
│  • roleAssignment   │
│    ScheduleRequests │
└─────────────────────┘
```

## Control Flow

### Status Mode (`pim`)

```text
1. cobra rootCmd.RunE → runStatus()
2. azidentity.DefaultAzureCredential — reads ~/.azure/ from az login
3. azure.FetchActiveRoles() — GET roleAssignmentScheduleInstances?$filter=asTarget()
   └─ filters: assignmentType == "Activated"
4. state.LookupJustification() — reads ~/.pim/activations.json
5. For each active assignment:
   a. Extract role name, scope, type from expandedProperties (typed SDK structs)
   b. DecodeEnv() + DecodeAppCode() from RG name (case-insensitive)
   c. time.Until(endDateTime) → remaining duration
   d. justification lookup by composite key (scope + "|" + roleDefinitionId)
6. tui.PrintStatus() — renders table to stdout
7. Exit
```

### Activate Mode (`pim activate`)

```text
1. cobra activateCmd.RunE → runActivate()
2. azidentity.DefaultAzureCredential
3. cache.Get():
   a. If hit → json.Unmarshal into []model.Role
   b. If miss → azure.FetchEligibleRoles() then cache.Set()
4. tui.RunSelector() — bubbletea alt-screen TUI
   a. Pre-renders all 4 row states into rowRender cache at startup
   b. Handles ↑/↓ (cursor only), Space (toggle + single cache rebuild),
      a/n/g (bulk toggle + full cache rebuild), / (search + rebuildVisible)
   c. Returns selected []model.Role or cancelled
5. huh.NewForm() — justification text input with validation
6. tui.RunDurationSelector() — bubbletea duration picker
7. tui.PrintSummary()
8. If --dry-run → exit
9. huh confirm prompt (y/N)
10. azure.ActivateRoles() — parallel with errgroup.Group
    a. uuid.New() for each request ID
    b. PUT roleAssignmentScheduleRequests/{uuid}
11. state.Save() — prune expired + append new entries → activations.json
12. tui.PrintResults()
```

## Component Inventory

| Package             | File(s)         | Responsibility                                            |
| ------------------- | --------------- | --------------------------------------------------------- |
| `cmd/pim`           | `main.go`       | cobra wiring, flag definitions, top-level orchestration   |
| `internal/azure`    | `client.go`     | Azure SDK client factory, DefaultAzureCredential          |
| `internal/azure`    | `eligible.go`   | Fetch eligible roles, map → model.Role, decode RG name    |
| `internal/azure`    | `active.go`     | Fetch active assignments, map → model.ActiveRole          |
| `internal/azure`    | `activate.go`   | SelfActivate PUT, parallel via errgroup                   |
| `internal/cache`    | `cache.go`      | File-based 24h TTL cache (eligible-roles.json + meta)     |
| `internal/config`   | `config.go`     | Subscription ID, principal ID, scope, cache TTL           |
| `internal/model`    | `role.go`       | Role, ActiveRole, DurationOption, ActivationResult types  |
| `internal/model`    | `activation.go` | ActivationRecord type                                     |
| `internal/model`    | `rgname.go`     | DecodeEnv(), DecodeAppCode() — case-insensitive           |
| `internal/state`    | `state.go`      | activations.json read / prune-expired / append / write    |
| `internal/tui`      | `selector.go`   | Bubbletea role selector with rowRender pre-render cache   |
| `internal/tui`      | `duration.go`   | Bubbletea duration picker                                 |
| `internal/tui`      | `loader.go`     | Spinner wrapper (RunWithSpinner) for blocking API calls   |
| `internal/tui`      | `status.go`     | PrintStatus, PrintSummary, PrintResults, PrintBanner      |
| `internal/tui`      | `styles.go`     | Lipgloss styles, helper render funcs (Bold, Dim, etc.)    |

## Data Flow

```text
Eligible Roles API Response
    │
    ├─ json.Marshal → ~/.pim/eligible-roles.json (24h TTL)
    │
    ▼
[]model.Role (typed Go structs)
    │
    ├─ RoleName, ScopeName  → TUI display + search
    ├─ Environment, AppCode → Decoded from ScopeName (DecodeEnv/DecodeAppCode)
    ├─ RoleDefinitionID     → API activation request body
    └─ Scope                → API activation request URL

User Selections ([]model.Role where Selected == true)
    │
    ├─ + justification (string)
    ├─ + model.DurationOption{ISO8601, Duration, Label}
    │
    ▼
Parallel azure.ActivateRoles() (errgroup.Group)
    │
    ▼
[]model.ActivationResult → tui.PrintResults()
    │
    ▼
[]model.ActivationRecord → ~/.pim/activations.json
```

## Terminal UI Architecture

The interactive selector is built on [bubbletea](https://github.com/charmbracelet/bubbletea)
and [lipgloss](https://github.com/charmbracelet/lipgloss):

| Technique              | Purpose                                          |
| ---------------------- | ------------------------------------------------ |
| `tea.WithAltScreen()`  | Full-screen mode, restored automatically on exit |
| `tea.KeyMsg`           | Keystroke handling (↑/↓/space/a/n/g/c/enter//)   |
| `tea.WindowSizeMsg`    | Terminal resize → viewport recalculation         |
| `lipgloss.Reverse()`   | Highlight cursor row uniformly                   |
| `lipgloss.Faint()`     | Dim unselected rows                              |
| `lipgloss.Green()`     | Selected (non-cursor) rows                       |
| `bubbles/textinput`    | Search input with `Blink` command                |

### Rendering Strategy — Row Render Cache

Because `View()` is called on every keypress, all lipgloss rendering is done
**once at startup** and stored in a `rowRender` cache (one entry per role,
four pre-built strings per entry):

| Field         | When used                                |
| ------------- | ---------------------------------------- |
| `normalUnsel` | Cursor elsewhere, role not selected      |
| `normalSel`   | Cursor elsewhere, role selected          |
| `cursorUnsel` | Cursor on this row, role not selected    |
| `cursorSel`   | Cursor on this row, role selected        |

`View()` for navigation (↑/↓) is a pure integer increment + slice lookups.
Only selection changes (`Space`, `a`, `n`, `g`) trigger cache rebuilds
(single entry or full rebuild respectively).

## Error Handling

| Scenario                  | Behaviour                                    |
| ------------------------- | -------------------------------------------- |
| Not authenticated         | SDK returns error → "run az login" message   |
| API call fails            | Wrapped error returned to cobra → exit 1     |
| No eligible roles         | Info message + exit 0                        |
| No active roles (status)  | Info message + exit 0                        |
| Empty justification       | huh validation rejects empty input           |
| Activation fails (1 role) | Per-role error logged, remaining continue    |
| User cancels TUI          | cancelled = true → "Cancelled." + exit 0     |
