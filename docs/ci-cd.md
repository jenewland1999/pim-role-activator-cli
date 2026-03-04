# CI/CD

This repository uses GitHub Actions for validation, documentation quality checks, semantic versioning, release publishing, and artifact generation.

## Workflows

## CI (`.github/workflows/ci.yml`)

Triggers:

- Push to `main`
- Pull requests targeting `main`

Jobs:

1. `lint`
   - Verifies `gofmt` compliance
   - Runs `golangci-lint`

2. `test`
   - Runs `go vet ./...`
   - Runs tests with race detector and coverage:
     - `go test -race -count=1 -covermode=atomic -coverprofile=coverage.out -json ./...`
   - Generates summary:
     - `go tool cover -func=coverage.out > coverage.txt`
   - Uploads artifacts:
     - `coverage.out`
     - `coverage.txt`
     - `test-results.json`

3. `build`
   - Cross-platform compile matrix:
     - `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`
   - Uploads compiled binaries as workflow artifacts

## Docs (`.github/workflows/docs.yml`)

Triggers on markdown/workflow changes.

Jobs:

- Markdown lint (`markdownlint-cli2`)
- Markdown link validation (`markdown-link-check`)

## Release (`.github/workflows/release.yml`)

Trigger:

- Push to `main`

Release flow:

1. `semantic-release` calculates next version from Conventional Commits
2. Build matrix compiles release binaries (version injected via `-ldflags`)
3. `go-licenses` generates `THIRD_PARTY_NOTICES.txt`
4. Publish job uploads binaries and notices to GitHub Release

## Commit Convention and Versioning

Versioning is automated via Conventional Commits and `.releaserc.json`:

- `feat` => minor release
- `fix`, `perf`, `refactor` => patch release
- `docs`, `chore`, `style`, `ci` => no version bump

Use `BREAKING CHANGE:` (or `type!`) for major releases.

## Local Validation Commands

Before opening a PR:

```bash
gofmt -w .
go vet ./...
go test -race -count=1 ./...
go build ./...
```
