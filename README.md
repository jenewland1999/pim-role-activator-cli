# PIM Role Activator CLI

Activate Azure PIM (Privileged Identity Management) roles from your terminal.

## Context

At work, we use Azure PIM roles to access production resource groups for our applications.
Because this is company policy, I often need to get into multiple production resource groups at the same time when debugging outages or incidents.
In the Azure portal, you can only activate one role at a time, and for anyone who has used that flow, each activation plus justification typically takes around two minutes.

This CLI exists to make that operational workflow faster, more repeatable, and less frustrating when time matters.

## Build Transparency

I had never written Go before this project and I am still learning the language.
I chose Go because it is a native, type-safe language that I have wanted to learn for a long time, and because it is a strong fit for CLI performance.

I also want to be transparent that I did not hand-write all of the code in this repository.
This project was built using AI-assisted, agentic workflows (sometimes called "vibe coding"), with me steering the architecture, behavior, and quality gates.

## Features

- Interactive multi-select activation flow with search and bulk-selection shortcuts
- Status view for active assignments with remaining time and local justification history
- Eligibility-expiry view for activatable roles across subscriptions
- Parallel role activation with bounded concurrency
- Eligible-role cache with configurable TTL and active-role cache with dynamic expiry
- Dry-run activation mode (`pim activate --dry-run`)
- Configurable duration options (`durations` in config)
- Structured timing diagnostics (`--debug-timings`)
- Cross-platform builds (macOS, Linux, Windows)

## Quick Start

```bash
# Install (source-based)
go install github.com/jenewland1999/pim-role-activator-cli/cmd/pim@latest

# Authenticate
az login

# First run (auto-launches setup if config is missing)
pim

# Activate roles
pim activate
```

## Core Dependencies

- Go 1.25+
- Azure CLI (`az`) for login/session management
- Azure SDK for Go (`azidentity`, `armauthorization/v2`)
- Charmbracelet (`bubbletea`, `huh`, `lipgloss`) for TUI/forms
- Cobra for CLI command routing

## Integrations

- Azure Resource Manager (`management.azure.com`) via Azure SDK:
  - `RoleEligibilityScheduleInstances`
  - `RoleAssignmentScheduleInstances`
  - `RoleAssignmentScheduleRequests`
- GitHub Actions for CI/CD, release automation, and docs checks

## Commands

```text
pim                 Show currently active PIM roles (default)
pim status          Show currently active PIM roles
pim eligible        Show eligible PIM roles ordered by eligibility expiry
pim activate        Activate eligible PIM roles interactively
pim completion      Generate shell completion scripts
pim setup           Re-run the configuration wizard
pim version         Print version, Go runtime, and platform

Global flags:
  --timeout duration   Timeout for Azure API calls (default 2m)
  --debug-timings      Emit debug timing logs for command stages

Activate flags:
  --dry-run            Walk through prompts without activating
  --no-cache           Bypass the eligible-role cache
```

## Platform Guides

- [Install, setup, usage, uninstall — macOS](docs/user-guides/install/macos.md)
- [Install, setup, usage, uninstall — Linux](docs/user-guides/install/linux.md)
- [Install, setup, usage, uninstall — Windows](docs/user-guides/install/windows.md)

## Configuration and Data

- Config file: `~/.pim/config.json`
- Data files: `~/.pim/eligible-roles-*.json`, `~/.pim/active-roles-*.json`, `~/.pim/activations.json`
- Schema: [docs/config.schema.json](docs/config.schema.json)
- Details: [Data Formats](docs/data-formats.md)

## Documentation Index

- [Features](docs/features.md)
- [Architecture](docs/architecture.md)
- [Data Formats](docs/data-formats.md)
- [User Guide](docs/user-guide.md)
- [CI/CD](docs/ci-cd.md)
- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)

## Development

```bash
# Run locally
go run ./cmd/pim

# Tests + vet
go test -race -count=1 ./...
go vet ./...

# Build
go build -o pim ./cmd/pim
```

## License

GPL-3.0-only. See [LICENSE](LICENSE).

Third-party license notices are generated in release CI and attached to GitHub Releases.
