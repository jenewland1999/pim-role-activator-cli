# Codebase Audit — PIM Role Activator CLI

**Date:** 2026-02-26

---

## CRITICAL

### 1. No unit or integration tests exist

There are zero `_test.go` files in the entire repository. This is the single biggest risk to reliability and correctness. Every package — `azure`, `cache`, `config`, `model`, `state`, `setup`, `tui` — is completely untested. Regressions can ship silently, and the CI pipeline (`.github/workflows/ci.yml`) has no `go test` step.

### 2. `pimDir()` uses `$HOME` — breaks on Windows

`cmd/pim/main.go:33`: `os.Getenv("HOME")` returns empty on Windows. The standard library's `os.UserHomeDir()` handles all platforms correctly:

```go
// current (broken on Windows)
dir := filepath.Join(os.Getenv("HOME"), ".pim")

// fix
home, err := os.UserHomeDir()
```

Since the CI matrix builds for Windows, this is a shipped-to-users bug.

### 3. JWT decoded without signature verification

`internal/azure/identity.go:44-60`: `GetTokenClaims` manually base64-decodes the JWT payload and trusts it unconditionally. While the comment says "we trust our own token", if this code were reused or the token source changed, there would be no integrity check. At minimum, document this trust boundary explicitly and ensure the function is never called with tokens from untrusted sources. Using `golang-jwt/jwt/v5` (already an indirect dependency) for proper parsing would be safer.

### 4. Sensitive data written to disk with world-readable permissions

- `internal/cache/cache.go:56`: `os.WriteFile(..., 0o644)` — the eligible-roles cache and meta file are readable by all users on the system.
- `internal/config/config.go:83`: `os.WriteFile(..., 0o644)` — `config.json` contains `principal_id` (the user's Entra Object ID) and subscription UUIDs.
- `internal/state/state.go:52`: `os.WriteFile(..., 0o644)` — `activations.json` contains justification text and role scopes.
- `cmd/pim/main.go:34`: `os.MkdirAll(dir, 0o755)` — the `.pim` directory itself is world-readable.

All of these should use `0o600` for files and `0o700` for directories.

---

## HIGH

### 5. No context timeout/cancellation on Azure API calls

Throughout `cmd/pim/main.go`, `context.Background()` is used with no deadline or timeout. If Azure APIs are slow or unresponsive, the CLI hangs indefinitely. All network-bound operations should have a `context.WithTimeout`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()
```

### 6. `http.DefaultClient` used for subscription listing — no timeout

`internal/azure/subscriptions.go:39`: `http.DefaultClient.Do(req)` has no timeout. A hung connection will block forever. Create a client with explicit timeout:

```go
client := &http.Client{Timeout: 30 * time.Second}
```

### 7. Unbounded response body decoding

`internal/azure/subscriptions.go:48`: `json.NewDecoder(resp.Body).Decode(...)` reads the entire response body with no size limit. A misbehaving server could send an arbitrarily large payload causing OOM. Wrap with `io.LimitReader`:

```go
json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(...)
```

### 8. Race condition in `ActivateRoles` — shared slice write from goroutines

`internal/azure/activate.go:30-32`: `results[i] = ...` writes to distinct indices, which is safe in Go. However, the loop variable capture uses the legacy `i, role := i, role` pattern. With Go 1.22+ (the module uses Go 1.25), loop variables are per-iteration by default, making this redundant but harmless. No actual race, but the pattern should be cleaned up.

### 9. Panic-on-empty in activation client creation

`cmd/pim/main.go:133`: `azure.NewClients(cfg.Subscriptions[0].ID, cred)` will panic with an index-out-of-range if `cfg.Subscriptions` is empty (e.g. corrupted config). No bounds check is performed.

---

## MEDIUM

### 10. `version` variable referenced in ldflags but never declared

The release workflow injects `-X main.version=...` via ldflags, but there is no `var version string` in `cmd/pim/main.go`. The linker silently drops the flag, so no version information is available at runtime. Add:

```go
var version = "dev"
```

and expose it via a `--version` flag or `pim version` subcommand.

### 11. Dead code: `formatDuration` function

`internal/tui/status.go:76`: `func formatDuration(d model.ActiveRole) string { return "" }` is entirely unused dead code with a misleading comment about import cycles. It should be removed.

### 12. `truncate` function defined twice with different semantics

- `cmd/pim/main.go:364`: operates on `[]rune` (Unicode-safe).
- `internal/tui/selector.go:349`: operates on `[]byte` via `len(s)` (byte-length, breaks on multi-byte characters).

The TUI version will corrupt multi-byte scope names. Unify to the rune-based version in a shared location.

### 13. Cache is not atomic — partial writes can corrupt state

`internal/cache/cache.go:52-58`: `Set()` writes the data file and then the meta file sequentially. A crash between the two writes leaves them out of sync. Use write-to-temp-then-rename (`os.Rename` is atomic on POSIX):

```go
tmp := path + ".tmp"
os.WriteFile(tmp, data, 0o600)
os.Rename(tmp, path)
```

### 14. State file `Load()` silently swallows corruption

`internal/state/state.go:33-35`: If `activations.json` contains malformed JSON, `Load()` returns `nil, nil` with no warning. The user loses all activation history silently. At minimum, log a warning.

### 15. No input sanitisation on justification text

`cmd/pim/main.go:270-283`: The justification string is sent verbatim to the Azure API and persisted to disk. While Azure likely sanitises on its end, there is no local validation for length limits, control characters, or injection patterns.

### 16. No `go vet` or `staticcheck` in CI

`.github/workflows/ci.yml` runs `golangci-lint` but does not explicitly run `go test ./...`. The pipeline should include both `go vet ./...` and `go test ./...`.

---

## LOW

### 17. `GroupSelectPatterns` has a JSON tag typo (inconsistent casing)

`internal/config/config.go:28`: The field is `GroupSelectPatterns` with JSON tag `"quick_select_patterns"` (snake_case, fine), but it uses a non-standard Go field name with `[]string` — the actual JSON name is correct but inconsistent with how users might expect it.

### 18. No graceful signal handling

The CLI does not handle `SIGINT`/`SIGTERM` gracefully. If the user presses Ctrl+C during an API call (outside the bubbletea program), the process terminates without cleanup. Consider registering a signal handler to cancel the context.

### 19. `errgroup` context cancellation not leveraged properly

`internal/azure/activate.go:28`: `g, ctx := errgroup.WithContext(ctx)` creates a child context, but every goroutine returns `nil` unconditionally. The `errgroup` cancellation feature is effectively unused. A plain `sync.WaitGroup` would express intent more clearly, or the function should propagate at least one error and cancel siblings on fatal failures (e.g. auth token expired).

### 20. Pager responses are not parallelised across subscriptions

`cmd/pim/main.go:80-92`: Eligible and active role fetches loop over subscriptions sequentially. For users with many subscriptions, this could be slow. Consider parallelising across subscriptions.

### 21. Hardcoded API version constant is unused

`internal/config/config.go:12`: `APIVersion = "2020-10-01"` is declared but never referenced anywhere in the codebase — the Azure SDK manages its own API versions.

### 22. No `--version` flag or version subcommand

Users have no way to check which version of the CLI they are running, making support and debugging harder.

### 23. `go.sum` and `.gitignore` not checked

No `.gitignore` was found in the workspace listing. Ensure the compiled binary (`main`) at the root (visible in the file tree) is not committed.

### 24. Binary committed to repo

The file `main` at the repository root appears to be a compiled binary. This should be in `.gitignore` and removed from version control.

### 25. No error wrapping context in several return paths

Several error returns (e.g. in `config.Load`, `state.Save`) propagate raw `os` or `json` errors without additional context, making debugging harder. Use `fmt.Errorf("loading state: %w", err)` consistently.

---

## INFORMATIONAL / BEST PRACTICES

### 26. Consider using `slog` for structured logging

The codebase uses `fmt.Fprintf(os.Stderr, ...)` for warnings. Go 1.21+ provides `log/slog` for structured, levelled logging which would improve observability.

### 27. Config validation is minimal

`config.Load()` deserialises JSON but does not validate that required fields (`principal_id`, `subscriptions`) are present or well-formed. A post-load validation step would prevent confusing runtime errors.

### 28. Consider dependency injection for testability

Azure client calls, file I/O, and UI rendering are all tightly coupled. Introducing interfaces (e.g. `RoleFetcher`, `StateStore`) would enable mocking and testing without live Azure credentials.

### 29. `ScopePattern` regex is compiled on every invocation

`ParsedScopePattern()` in `internal/config/config.go:43-48` compiles the regex every time it is called. While called infrequently, caching it (via `sync.Once` or a field) is cleaner.

### 30. Duration options are hardcoded

`internal/model/role.go:30-36`: The 4 duration options (30m, 1h, 2h, 4h) are hardcoded. Consider making them configurable via `config.json` to accommodate different Azure PIM policies that allow up to 8 or 24 hours.

---

## Summary

| Criticality       | Count | Key Themes                                              |
| ----------------- | ----- | ------------------------------------------------------- |
| **Critical**      | 4     | No tests, Windows breakage, world-readable secrets, JWT |
| **High**          | 5     | No timeouts, unbounded reads, potential panic           |
| **Medium**        | 7     | Dead code, non-atomic writes, no version info, CI gaps  |
| **Low**           | 9     | Signal handling, parallelisation, unused constants      |
| **Informational** | 5     | Structured logging, DI, config validation               |

The highest-impact improvements would be: **(1)** adding tests + `go test` to CI, **(2)** fixing file permissions to `0o600`/`0o700`, **(3)** replacing `os.Getenv("HOME")` with `os.UserHomeDir()`, and **(4)** adding context timeouts to all network operations.
