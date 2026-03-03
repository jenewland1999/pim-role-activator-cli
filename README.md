# PIM Role Activator CLI

Activate Azure PIM (Privileged Identity Management) roles from your terminal — no portal clicks required.

## Why?

The Azure PIM portal is slow. Activating a single role means navigating through multiple pages, waiting for spinners, entering justification, and confirming. Multiply that by several roles across subscriptions and it becomes a daily time sink.

This CLI replaces that workflow with a fast, keyboard-driven experience:

- **Select multiple roles at once** from an interactive list
- **One justification** applies to all selected roles
- **Parallel activation** — all roles activate simultaneously
- **24-hour cache** — eligible roles are fetched once and reused
- **Status view** — see what's active and when it expires

What takes minutes in the portal takes seconds here.

## Quick Start

```bash
# Install (macOS example — see install guides for all platforms)
go install github.com/jenewland1999/pim-role-activator-cli/cmd/pim@latest

# Log in to Azure
az login

# First run — interactive setup wizard configures subscriptions + identity
pim

# Activate roles
pim activate
```

## Install

Detailed installation guides for each platform:

- [macOS](docs/user-guides/install/macos.md)
- [Windows](docs/user-guides/install/windows.md)
- [Linux](docs/user-guides/install/linux.md)

**Prerequisites:**

| Tool                                                                              | Purpose             |
| --------------------------------------------------------------------------------- | ------------------- |
| [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) (`az`) | Authentication      |
| Active Azure session (`az login`)                                                 | Token for API calls |
| [Go 1.25+](https://go.dev/dl/) _(build from source only)_                         | Compile the binary  |

## Usage

```text
pim                 Show currently active PIM roles
pim status          Show currently active PIM roles
pim activate        Activate eligible PIM roles interactively
pim setup           Re-run the configuration wizard
pim version         Print version, Go runtime, and platform
pim activate [flags]

Flags:
  --dry-run       Walk through prompts without activating
  --no-cache      Bypass the 24-hour eligible role cache
  --timeout       Timeout for Azure API calls (e.g. 30s, 2m, 5m)
  --debug-timings Emit debug timing logs for command stages
  -h, --help      Show help
```

### Check Active Roles

```bash
pim
```

Displays a table of currently active roles with remaining time and the justification you entered.

### Activate Roles

```bash
pim activate
```

Interactive 3-step flow:

1. **Select roles** — keyboard-driven multi-select list with search and group selection
2. **Enter justification** — free text, sent to Azure audit logs
3. **Choose duration** — configurable via `durations` in config (defaults: 30m, 1h, 2h, 4h)

A summary is shown before anything is activated. Roles are then activated in parallel.

### First-Time Setup

On first run (or via `pim setup`), an interactive wizard configures:

- Your Azure subscriptions (auto-detected from `az login`)
- Your principal (object) ID (auto-detected)
- Optional group-select patterns for bulk role selection
- Optional scope pattern for App/Env column decoding

Configuration is saved to `~/.pim/config.json`.

### Dry Run

```bash
pim activate --dry-run
```

Walks through the full flow without sending any API requests. Useful for testing or demonstrating the workflow.

### Bypass Cache

```bash
pim activate --no-cache
```

Forces a fresh fetch of eligible roles. Normally roles are cached for 24 hours in `~/.pim/`.

## Role Selector Keys

| Key         | Action                                       |
| ----------- | -------------------------------------------- |
| `↑` / `↓`   | Navigate the list                            |
| `Space`     | Toggle selection on the highlighted role     |
| `a`         | Select all visible roles                     |
| `n`         | Deselect all visible roles                   |
| `g`         | Group select — auto-selects by scope pattern |
| `/`         | Search / filter roles by name or scope       |
| `Backspace` | Clear the current search filter              |
| `Enter`     | Confirm selection                            |
| `c`         | Cancel and exit                              |

## Configuration

All configuration is managed through `pim setup` and stored in `~/.pim/config.json`:

```jsonc
{
  "subscriptions": [
    { "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", "name": "My Subscription" },
  ],
  "principal_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "quick_select_patterns": ["APP2", "APP4", "APP5", "APP3"],
  "cache_ttl_hours": 24,
  "scope_pattern": "^.(?P<env>[PQTD]).{5}(?P<app>.{4})",
  "env_labels": { "P": "Prod", "D": "Dev", "Q": "QA", "T": "Test" },
  "durations": [
    { "label": "30 minutes", "iso8601": "PT30M", "minutes": 30 },
    { "label": "1 hour", "iso8601": "PT1H", "minutes": 60 },
  ],
}
```

- `subscriptions` — Azure subscriptions to scan for eligible roles
- `principal_id` — your Azure AD object ID
- `quick_select_patterns` — scope substrings for the `g` group-select key
- `cache_ttl_hours` — how long to cache eligible roles (default: 24)
- `scope_pattern` — regexp with `env` and `app` named groups to decode RG names
- `env_labels` — map single-character environment codes to display labels
- `durations` — optional activation duration options shown in Step 3

## Data Storage

All local data lives in `~/.pim/`:

- `config.json` — user configuration (subscriptions, principal ID, etc.)
- `eligible-roles-data.json` — cached eligible role assignments
- `eligible-roles-meta.json` — cache metadata (`written_at` timestamp)
- `active-roles-data.json` — cached active roles used by status mode
- `active-roles-meta.json` — active-role cache metadata (`written_at` timestamp)
- `activations.json` — local record of activations (for justification display)

## Documentation

- [Architecture](docs/architecture.md) — system design, control flow, component inventory
- [Data Formats](docs/data-formats.md) — cache file schemas, RG naming convention, activation payloads
- [Active Roles Cache](docs/active-roles-cache.md) — status cache design and refresh behavior
- [User Guide](docs/user-guide.md) — commands, key bindings, troubleshooting
- [Implementation Notes](docs/go-migration.md) — architecture decisions, dependency choices, type mappings

## Troubleshooting

| Problem                       | Fix                                                                            |
| ----------------------------- | ------------------------------------------------------------------------------ |
| Authentication errors         | Run `az login` (or `az login --use-device-code` without a browser)             |
| No eligible roles found       | Check `az account show`, try `--no-cache`, verify subscriptions in `pim setup` |
| Role activation fails         | Role may already be active, require approval, or exceed policy limits          |
| Stale data after role changes | Run `pim activate --no-cache` to refresh                                       |
| Wrong identity detected       | Run `pim setup` to update your principal ID                                    |

## Development

To run the CLI directly from source without installing a binary:

```bash
# Run the default command (status)
go run ./cmd/pim

# Run a specific subcommand
go run ./cmd/pim status
go run ./cmd/pim activate
go run ./cmd/pim setup
go run ./cmd/pim activate --dry-run --no-cache

# Run tests
go test -race ./...

# Build a local binary (outputs to current directory)
go build -o pim ./cmd/pim
./pim
```

`go run` compiles and executes in one step — no install required. You need Go 1.25+ and an active `az login` session.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

To report a security vulnerability, see [SECURITY.md](SECURITY.md).

## About This Project

This is my first Go project. It was built with the help of AI tools — GitHub Copilot for in-editor assistance and cloud-hosted models (Claude, ChatGPT) for architecture decisions and code review. The AI helped accelerate development but all code was reviewed, tested, and iterated on by hand.

## Licence

Licensed under the GNU General Public License v3.0 only (GPL-3.0-only). See
the `LICENSE` file.

### Note on redistribution

If you redistribute this program (including modified versions), GPLv3 requires
that you also provide the corresponding source code under the same licence.

### Third-party notices

This project depends on open-source libraries under their own licences
(primarily Apache-2.0, MIT, and BSD). A `THIRD_PARTY_NOTICES.txt` file is
attached to each [GitHub Release](https://github.com/jenewland1999/pim-role-activator-cli/releases/latest) with full attribution.

To generate it locally:

```bash
go install github.com/google/go-licenses@latest
go-licenses report ./... > THIRD_PARTY_NOTICES.txt
```
