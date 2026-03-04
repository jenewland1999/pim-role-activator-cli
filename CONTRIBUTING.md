# Contributing

Thanks for your interest in contributing to PIM Role Activator CLI.

## Getting Started

1. Fork the repository
2. Clone your fork:

   ```bash
   git clone https://github.com/<your-username>/pim-role-activator-cli.git
   cd pim-role-activator-cli
   ```

3. Install dependencies:

   ```bash
   go mod download
   ```

4. Build:

   ```bash
   go build -o pim ./cmd/pim
   ```

## Development Workflow

1. Create a branch from `main`:

   ```bash
   git checkout -b feat/my-feature
   ```

2. Make your changes
3. Format your code:

   ```bash
   gofmt -w .
   ```

4. Ensure the project builds and passes checks:

   ```bash
   go vet ./...
   go test -race -count=1 ./...
   go build ./...
   ```

5. Commit using [Conventional Commits](https://www.conventionalcommits.org/):

   ```text
   feat: add new duration option
   fix: handle empty role list gracefully
   docs: update install guide for Linux ARM64
   chore: update dependencies
   ```

6. Push and open a pull request against `main`

## Commit Messages

This project uses **semantic commits** to automate versioning and release notes. Every commit message should follow the format:

```text
<type>: <description>

[optional body]
```

| Type       | When to use                                 |
| ---------- | ------------------------------------------- |
| `feat`     | A new feature (bumps minor version)         |
| `fix`      | A bug fix (bumps patch version)             |
| `docs`     | Documentation changes only                  |
| `chore`    | Maintenance tasks, dependency updates       |
| `refactor` | Code restructuring without behaviour change |
| `style`    | Formatting, whitespace (no logic changes)   |
| `ci`       | CI/CD pipeline changes                      |

Add `BREAKING CHANGE:` in the commit body (or `!` after the type) for breaking changes — this bumps the major version.

## Code Style

- Run `gofmt` before committing — CI will reject unformatted code
- Follow standard Go conventions and idioms
- Keep functions focused and well-named
- Add comments for exported types and functions

## CI Expectations

CI runs on every push/PR to `main` and includes:

- Formatting + lint checks
- `go vet`
- Test execution with race detection
- Coverage generation and test result artifact upload
- Cross-platform build verification

See [docs/ci-cd.md](docs/ci-cd.md) for full pipeline details.

## Reporting Issues

Use the [issue templates](https://github.com/jenewland1999/pim-role-activator-cli/issues/new/choose) to report bugs or request features.

## Security

If you discover a security vulnerability, **do not open a public issue**. See [SECURITY.md](SECURITY.md) for reporting instructions.
