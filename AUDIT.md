# Codebase Audit — PIM Role Activator CLI

**Date:** 2026-03-03 (post-PRD completion)
**Previous audit:** 2026-02-26 (pre-PRD — 30 findings, 0 tests)

---

## Executive Summary

The PRD roadmap (8 stages, 30+ items) has been fully executed. All 30 findings
from the original 2026-02-26 audit have been resolved. The codebase now has
79 unit tests across 5 packages, a CI pipeline with lint/vet/test/build, proper
file permissions, context timeouts, signal handling, structured logging, atomic
cache writes, input validation, config validation, dependency injection
interfaces, and a `--version`/`pim version` subcommand.

This follow-up audit examines the codebase in its current state and identifies
remaining low-severity items and future improvement opportunities. No critical
or high-severity issues remain.

---

## Status of Original Findings

All 30 items from the 2026-02-26 audit have been addressed:

| # | Original Finding | Resolution | PRD Item |
|---|------------------|------------|----------|
| 1 | No tests | 79 tests across 5 packages; CI runs `go test -race` | 3.1–3.6 |
| 2 | `os.Getenv("HOME")` | Replaced with `os.UserHomeDir()` | 1.1 |
| 3 | JWT no signature verification | Using `golang-jwt/jwt/v5` `ParseUnverified`; trust boundary documented | 1.3 |
| 4 | World-readable file permissions | All files `0o600`, directories `0o700` | 1.2 |
| 5 | No context timeouts | `context.WithTimeout` + `--timeout` flag (default 2m) | 2.1 |
| 6 | `http.DefaultClient` no timeout | Local `http.Client{Timeout: 30s}` | 2.2 |
| 7 | Unbounded response body | `io.LimitReader` 10 MiB cap | 2.3 |
| 8 | Loop variable capture (Go 1.22+) | Removed redundant rebinding | 4.4 |
| 9 | Panic on empty subscriptions | Guard clause + `config.Validate()` | 1.4, 5.4 |
| 10 | No `version` variable | `var version = "dev"` + `--version` flag + `pim version` | 6.1, 6.2 |
| 11 | Dead `formatDuration` | Removed | 4.1 |
| 12 | Duplicate `truncate` | Unified to rune-based `tui.Truncate()` | 4.2 |
| 13 | Non-atomic cache writes | `atomicWriteFile` (temp + rename) | 5.1 |
| 14 | Silent state corruption | `slog.Warn` on corrupt JSON | 5.2 |
| 15 | No justification validation | `validateJustification`: non-empty, ≤500 runes, no control chars | 5.3 |
| 16 | No `go vet`/tests in CI | CI has lint, vet, test (race), and multi-platform build jobs | 3.6 |
| 17 | JSON tag casing | Consistent `snake_case` throughout; no change needed | — |
| 18 | No signal handling | `signal.NotifyContext` for SIGINT/SIGTERM; exit 130 | 7.1 |
| 19 | errgroup misuse | `ActivateRoles` now uses `sync.WaitGroup`; errgroup used correctly in parallel subscription fetch | 4.5, 7.2 |
| 20 | Sequential subscription fetches | Parallelised via `errgroup` + `sync.Mutex` | 7.2 |
| 21 | Unused `APIVersion` constant | Removed | 4.3 |
| 22 | No `--version` flag | `--version` flag + `pim version` subcommand | 6.1, 6.2 |
| 23 | No `.gitignore` | Comprehensive `.gitignore` added | 1.5 |
| 24 | Binary in repo | Removed; `.gitignore` prevents reoccurrence | 1.5 |
| 25 | No error wrapping | `fmt.Errorf` with `%w` throughout config, state, cache | 4.6 |
| 26 | No structured logging | `log/slog` used for all warnings/errors | 6.3 |
| 27 | Minimal config validation | `Validate()` checks `principal_id` + `subscriptions`; called by `Load()` | 5.4 |
| 28 | No dependency injection | `Authenticator`, `RoleFetcher`, `RoleActivator`, `StateStore` interfaces | 8.1 |
| 29 | Regex compiled every call | Cached via `sync.Once` in `ParsedScopePattern()` | 7.4 |
| 30 | Hardcoded duration options | Configurable via `config.json` `durations` field | 7.3 |

---

## Current Architecture Assessment

### Strengths

- **Clean package structure**: `cmd/pim` → `internal/{azure,cache,config,model,setup,state,tui}`. No import cycles. Model package has zero external dependencies.
- **Comprehensive test coverage**: 79 tests across `model` (14), `cache` (12), `config` (34), `state` (18), `azure/identity` (12). All use `t.TempDir()` isolation and run with `-race`.
- **Robust CI pipeline**: Lint (gofmt + golangci-lint), vet, test (race), multi-platform build (5 targets), semantic-release.
- **Defence in depth**: Context timeouts + HTTP client timeouts + `io.LimitReader` + atomic writes + file permissions.
- **Good UX**: Interactive TUI with search/filter, cached role display, spinner feedback, dry-run mode, signal handling, configurable durations.
- **Structured observability**: `log/slog` for warnings/errors with structured key-value attributes.
- **Testability foundations**: DI interfaces for all major external dependencies (`Authenticator`, `RoleFetcher`, `RoleActivator`, `StateStore`).

### Metrics

| Metric | Value |
|--------|-------|
| Go version | 1.25.0 |
| Direct dependencies | 10 |
| Source files (`.go`, excl. tests) | 16 |
| Test files | 5 |
| Unit tests | 79 |
| Packages | 8 (`main` + 7 internal) |
| Lines of Go (approx.) | ~2,600 |
| CI jobs | 4 (lint, test, build×5, release) |

---

## Remaining Findings

### LOW

#### 1. `SelectionMarker()` is exported but never called

`internal/tui/styles.go`: `SelectionMarker(selected bool) string` is defined and exported but never referenced anywhere in the codebase. The selector uses pre-rendered `rowRender` structs with inline checkmark strings instead. This is dead code that should be removed.

#### 2. `PruneCachedRoles()` is only used in tests

`internal/model/role.go`: `PruneCachedRoles` is exported but only called from `role_test.go`. The production code uses `FromCachedRoles` (which prunes and converts in one pass). Consider removing `PruneCachedRoles` or marking it as a test helper if it serves no real caller.

#### 3. `Scopes()` method is only used in tests

`internal/config/config.go`: `Scopes()` is exported and tested but never called from production code. The subscription loop in `main.go` constructs scope strings inline. Either wire it into the callers or accept it as a convenience method for future use.

#### 4. State file writes are not atomic

`internal/state/state.go`: `Save()` uses `os.WriteFile` directly, unlike `cache.Set()` which was upgraded to atomic writes (temp + rename) in item 5.1. A crash during `Save()` could leave `activations.json` truncated. The `atomicWriteFile` helper in `cache.go` could be extracted to a shared `internal/fileutil` package and reused.

#### 5. Config file writes are not atomic

`internal/config/config.go`: `Save()` also uses `os.WriteFile` directly. Same risk as state — a crash during write could corrupt `config.json`. Lower impact since config is only written during `pim setup`, but applying the same atomic write pattern would be consistent.

#### 6. No tests for `setup`, `tui`, or `azure` (non-identity) packages

Tests exist for `model`, `cache`, `config`, `state`, and `azure/identity`. The following remain untested:
- `internal/setup/setup.go` — interactive wizard (would need mocked `huh` form inputs)
- `internal/tui/*.go` — bubbletea models (would need `teatest` or similar)
- `internal/azure/activate.go`, `active.go`, `eligible.go`, `subscriptions.go` — ARM API calls (would need mock ARM clients or HTTP record/replay)

These are harder to unit test due to interactive I/O and Azure SDK dependencies, but the DI interfaces from 8.1 lay the groundwork.

#### 7. `gofmt` formatting should be verified locally

CI checks `gofmt` compliance, but developers should also run `gofmt -w .` locally. Consider adding a `Makefile` or `.pre-commit-config.yaml` to automate this.

#### 8. No `golangci-lint` configuration file

The CI runs `golangci-lint` with defaults. A `.golangci.yml` file would make the linter configuration explicit and allow enabling additional linters (e.g. `errcheck`, `gosec`, `exhaustive`, `gocyclo`).

---

### INFORMATIONAL / FUTURE IMPROVEMENTS

#### 9. Consider extracting `atomicWriteFile` to a shared package

The `atomicWriteFile` helper in `internal/cache/cache.go` is well-implemented but private to the cache package. Extracting it to `internal/fileutil` would allow reuse in `config.Save()` and `state.Save()`.

#### 10. Consider `teatest` for TUI component testing

The `charmbracelet/x/exp/teatest` package enables headless bubbletea model testing. The selector and duration picker could be tested without a real terminal.

#### 11. No integration or end-to-end tests

All tests are unit-level. An integration test that exercises `runStatus`/`runActivate` with a mocked Azure backend (via the DI interfaces) would catch wiring bugs. This is a natural next step given the interfaces from 8.1.

#### 12. `config.Save` does not use atomic writes but `cache.Set` does

Inconsistency in write safety across the codebase. Both should behave the same way.

#### 13. Release workflow `ldflags` injection should be verified

The release workflow injects version via `-ldflags "-X main.version=..."`. With `var version = "dev"` now declared, this should work correctly, but the release workflow was not modified as part of the PRD (only `ci.yml` was). Verify the release build produces correct version output.

#### 14. No `CHANGELOG.md` in the repository

The project uses semantic-release which generates GitHub release notes automatically, but there is no `CHANGELOG.md` committed to the repository for offline reference.

#### 15. Documentation is comprehensive but could link to the schema

The user guide docs exist but don't reference `docs/config.schema.json`. Adding a note in the user guide about editor autocompletion via the `$schema` field would improve discoverability.

---

## Summary

| Severity | Count | Themes |
|----------|-------|--------|
| **Critical** | 0 | — |
| **High** | 0 | — |
| **Medium** | 0 | — |
| **Low** | 8 | Dead code (2), unused exports (1), non-atomic writes (2), missing tests (1), tooling (2) |
| **Informational** | 7 | Shared utilities, integration tests, release verification, docs |

The codebase is in good shape. All critical, high, and medium issues from the original audit have been resolved. The remaining items are low-severity cleanup tasks and future improvement opportunities. The most impactful next steps would be:

1. **Extract `atomicWriteFile` to a shared package** and apply it to state and config writes (items 4, 5, 9, 12)
2. **Remove dead code**: `SelectionMarker`, and optionally `PruneCachedRoles` (items 1, 2)
3. **Add integration tests** using the DI interfaces from 8.1 (item 11)
4. **Add a `.golangci.yml`** for explicit linter configuration (item 8)
