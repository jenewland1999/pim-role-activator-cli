# Features

## User Experience

- Keyboard-driven role selection with search and group-select shortcuts
- Guided setup wizard on first run (`pim` auto-runs setup when config is missing)
- Three-step activation flow: role selection, justification, duration
- Status mode as default command (`pim`)
- Dry-run support for safe rehearsal (`pim activate --dry-run`)

## Performance and Reliability

- Parallel subscription fetch for eligible and active role queries
- Bounded activation worker pool to reduce throttling tail latency
- Active-role cache with dynamic TTL based on soonest expiry
- Eligible-role cache with configurable TTL (`cache_ttl_hours`)
- Timeouts for Azure API calls (`--timeout`)
- Graceful signal handling (`Ctrl+C` / SIGTERM)

## Observability and Diagnostics

- Structured logs via Go `slog`
- Optional stage timing output (`--debug-timings`)
- Clear terminal status, summary, and activation result reporting

## Security and Data Handling

- Uses Azure SDK credential chain (`DefaultAzureCredential`)
- Restrictive local file permissions for config/state/cache
- Validation of required config fields and justification input
- Local activation history for justification lookup in status mode

## Platform and Distribution

- Cross-platform binaries released for:
  - macOS (`amd64`, `arm64`)
  - Linux (`amd64`, `arm64`)
  - Windows (`amd64`)
- Source installation via `go install`
- CI/CD pipelines for lint, test, build, docs, and release
