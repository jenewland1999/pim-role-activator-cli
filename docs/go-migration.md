# Implementation Notes

> Architecture decisions, dependency choices, type definitions, and
> component-by-component implementation details for the Go CLI.

## Why Go?

| Concern             | Bash                        | Go                      |
| ------------------- | --------------------------- | ----------------------- |
| Startup time        | ~200ms (az rest cold start) | <50ms (compiled binary) |
| TUI rendering       | Manual stty/tput hacks      | bubbletea / huh library |
| JSON handling       | jq subprocesses             | Native `encoding/json`  |
| Parallel activation | Sequential loop             | Goroutines + errgroup   |
| Error handling      | Exit codes + stderr         | Typed errors, wrapping  |
| Distribution        | Symlink script + deps       | Single static binary    |
| Auth token caching  | `az rest` handles it        | Azure SDK handles it    |

---

## Recommended Dependencies

| Dependency                                                                                | Purpose                                 | Install  |
| ----------------------------------------------------------------------------------------- | --------------------------------------- | -------- |
| `github.com/Azure/azure-sdk-for-go/sdk/azidentity`                                        | Authentication (DefaultAzureCredential) | `go get` |
| `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2` | PIM API client                          | `go get` |
| `github.com/charmbracelet/bubbletea`                                                      | TUI framework                           | `go get` |
| `github.com/charmbracelet/huh`                                                            | Form prompts (justification, duration)  | `go get` |
| `github.com/charmbracelet/lipgloss`                                                       | Terminal styling                        | `go get` |
| `github.com/google/uuid`                                                                  | UUID generation for request IDs         | `go get` |
| `github.com/spf13/cobra`                                                                  | CLI argument parsing                    | `go get` |

---

## Project Structure

```text
pim-role-activator-cli/
├── cmd/
│   └── pim/
│       └── main.go              # Entry point, cobra root command
├── internal/
│   ├── azure/
│   │   ├── client.go            # Azure SDK client setup + auth
│   │   ├── eligible.go          # Fetch eligible roles
│   │   ├── active.go            # Fetch active roles
│   │   └── activate.go          # Self-activate role
│   ├── cache/
│   │   └── cache.go             # File-based cache with TTL
│   ├── config/
│   │   └── config.go            # Constants (subscription ID, principal ID)
│   ├── model/
│   │   ├── role.go              # Role struct definitions
│   │   ├── activation.go        # Activation state record
│   │   └── rgname.go            # RG name parser (env + app code)
│   ├── state/
│   │   └── state.go             # Activation state persistence
│   └── tui/
│       ├── selector.go          # Role selector (bubbletea model)
│       ├── status.go            # Status table display
│       └── styles.go            # Lipgloss style definitions
├── docs/                        # (existing docs)
├── go.mod
├── go.sum
└── README.md
```

---

## Core Types

### Role

```go
// Role represents an eligible PIM role assignment.
// Maps directly to one entry in the eligibility API response.
type Role struct {
    // Display fields (from expandedProperties)
    RoleName  string // e.g. "Contributor"
    ScopeName string // e.g. "RG-PRD-APP1-001"
    ScopeType string // "resourcegroup", "subscription", etc.

    // API fields (needed for activation request)
    RoleDefinitionID string // Full ARM resource ID
    Scope            string // Full ARM resource ID

    // Derived fields (from RG name)
    Environment string // "Prod", "QA", "Test", "Dev", "—"
    AppCode     string // 4-char code or "—"

    // UI state
    Selected bool
}
```

### Activation Record

```go
// ActivationRecord represents a locally stored activation.
// Written after successful activation, read by status display.
type ActivationRecord struct {
    Scope            string `json:"scope"`
    RoleDefinitionID string `json:"roleDefinitionId"`
    RoleName         string `json:"roleName"`
    ScopeName        string `json:"scopeName"`
    Justification    string `json:"justification"`
    Duration         string `json:"duration"`
    ActivatedAt      string `json:"activatedAt"`
    ExpiresEpoch     int64  `json:"expiresEpoch"`
}
```

### ActiveRole (for status display)

```go
// ActiveRole combines API data with local state for display.
type ActiveRole struct {
    RoleName      string
    ScopeName     string
    ScopeType     string
    Environment   string
    AppCode       string
    ExpiresIn     time.Duration
    Justification string // From local state, or "—" if not found
}
```

### Duration Option

```go
type DurationOption struct {
    Label    string        // "30 minutes", "1 hour", etc.
    ISO8601  string        // "PT30M", "PT1H", etc.
    Duration time.Duration // 30*time.Minute, time.Hour, etc.
}

var DurationOptions = []DurationOption{
    {Label: "30 minutes", ISO8601: "PT30M", Duration: 30 * time.Minute},
    {Label: "1 hour",     ISO8601: "PT1H",  Duration: time.Hour},
    {Label: "2 hours",    ISO8601: "PT2H",  Duration: 2 * time.Hour},
    {Label: "4 hours",    ISO8601: "PT4H",  Duration: 4 * time.Hour},
}
```

---

## Component-by-Component Mapping

### 1. CLI Argument Parsing

| Bash                           | Go                                                     |
| ------------------------------ | ------------------------------------------------------ |
| `for arg in "$@"; do case ...` | `spf13/cobra` commands + flags                         |
| `ACTIVATE_MODE=false`          | Separate `rootCmd` (status) + `activateCmd` subcommand |
| `DRY_RUN`, `NO_CACHE`          | `cobra.Command.Flags().Bool()`                         |

```go
var rootCmd = &cobra.Command{
    Use:   "pim",
    Short: "PIM Role Activator CLI",
    RunE:  runStatus,  // Default: status mode
}

var activateCmd = &cobra.Command{
    Use:   "activate",
    Short: "Activate eligible PIM roles",
    RunE:  runActivate,
}

func init() {
    activateCmd.Flags().Bool("dry-run", false, "Walk through prompts without activating")
    activateCmd.Flags().Bool("no-cache", false, "Bypass the 24-hour role cache")
    rootCmd.AddCommand(activateCmd)
}
```

### 2. Preflight / Authentication

| Bash              | Go                                                                            |
| ----------------- | ----------------------------------------------------------------------------- |
| `command -v az`   | Not needed (Go uses SDK directly)                                             |
| `az account show` | `azidentity.NewDefaultAzureCredential()` — fails if not logged in             |
| `az login`        | Prompt user to run `az login` manually, or use `InteractiveBrowserCredential` |

```go
func newAzureClient() (*armauthorization.ClientFactory, error) {
    cred, err := azidentity.NewDefaultAzureCredential(nil)
    if err != nil {
        return nil, fmt.Errorf("not authenticated — run 'az login' first: %w", err)
    }
    return armauthorization.NewClientFactory(subscriptionID, cred, nil)
}
```

### 3. Fetch Eligible Roles

| Bash                                                                                  | Go                                                              |
| ------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `az rest --method GET --url "...roleEligibilityScheduleInstances?$filter=asTarget()"` | `RoleEligibilityScheduleInstancesClient.NewListForScopePager()` |
| `jq '.value[].properties.expandedProperties...'`                                      | Typed struct access                                             |

```go
func fetchEligibleRoles(ctx context.Context, client *armauthorization.RoleEligibilityScheduleInstancesClient) ([]Role, error) {
    filter := "asTarget()"
    scope := "/subscriptions/" + subscriptionID
    pager := client.NewListForScopePager(scope, &armauthorization.RoleEligibilityScheduleInstancesClientListForScopeOptions{
        Filter: &filter,
    })

    var roles []Role
    for pager.More() {
        page, err := pager.NextPage(ctx)
        if err != nil {
            return nil, err
        }
        for _, instance := range page.Value {
            roles = append(roles, mapToRole(instance))
        }
    }
    return roles, nil
}
```

### 4. Cache

| Bash                               | Go                                                               |
| ---------------------------------- | ---------------------------------------------------------------- |
| `cat "$CACHE_FILE"`                | `os.ReadFile()`                                                  |
| `echo "$RESPONSE" > "$CACHE_FILE"` | `os.WriteFile()`                                                 |
| `date +%s > "$CACHE_META"`         | `os.WriteFile([]byte(strconv.FormatInt(time.Now().Unix(), 10)))` |
| `age=$((now - cached_at))`         | `time.Since(cachedAt)`                                           |

```go
type Cache struct {
    Dir string // ~/.pim/
    TTL time.Duration
}

func (c *Cache) Get() ([]byte, bool) {
    metaBytes, err := os.ReadFile(filepath.Join(c.Dir, "cache-meta"))
    if err != nil {
        return nil, false
    }
    epoch, _ := strconv.ParseInt(strings.TrimSpace(string(metaBytes)), 10, 64)
    if time.Since(time.Unix(epoch, 0)) > c.TTL {
        return nil, false
    }
    data, err := os.ReadFile(filepath.Join(c.Dir, "eligible-roles.json"))
    return data, err == nil
}

func (c *Cache) Set(data []byte) error {
    os.MkdirAll(c.Dir, 0755)
    if err := os.WriteFile(filepath.Join(c.Dir, "eligible-roles.json"), data, 0644); err != nil {
        return err
    }
    return os.WriteFile(filepath.Join(c.Dir, "cache-meta"),
        []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)
}
```

### 5. RG Name Decoder

| Bash                                 | Go                                      |
| ------------------------------------ | --------------------------------------- |
| `decode_env()` + `decode_app_code()` | Methods on Role or standalone functions |

```go
func DecodeEnv(rgName string) string {
    if len(rgName) < 2 {
        return "—"
    }
    switch rgName[1] {
    case 'P', 'p':
        return "Prod"
    case 'Q', 'q':
        return "QA"
    case 'T', 't':
        return "Test"
    case 'D', 'd':
        return "Dev"
    default:
        return "—"
    }
}

func DecodeAppCode(rgName string) string {
    if len(rgName) >= 11 {
        return rgName[7:11]
    }
    return "—"
}
```

### 6. Interactive Selector (TUI)

| Bash                       | Go                                   |
| -------------------------- | ------------------------------------ |
| `stty -echo -icanon`       | bubbletea handles raw mode           |
| `dd bs=1 count=1`          | bubbletea `tea.KeyMsg`               |
| `tput cuu1 + tput el`      | bubbletea `View()` re-render         |
| `REVERSE` highlight        | lipgloss `Reverse(true)`             |
| Arrow key escape sequences | bubbletea `tea.KeyUp`, `tea.KeyDown` |
| Search mode                | Filter model state + text input      |

```go
// bubbletea model for the role selector
type SelectorModel struct {
    roles     []Role
    cursor    int
    search    string
    searching bool
    visible   []int // indices matching search
    quitting  bool
    cancelled bool
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            m.cursor = (m.cursor - 1 + len(m.visible)) % len(m.visible)
        case "down", "j":
            m.cursor = (m.cursor + 1) % len(m.visible)
        case " ":
            idx := m.visible[m.cursor]
            m.roles[idx].Selected = !m.roles[idx].Selected
        case "a":
            for _, i := range m.visible {
                m.roles[i].Selected = true
            }
        case "g":
            for i, r := range m.roles {
                if containsAny(r.ScopeName, "APP2", "APP4", "APP5", "APP3") {
                    m.roles[i].Selected = true
                }
            }
        case "c":
            m.cancelled = true
            return m, tea.Quit
        case "enter":
            if m.selectedCount() > 0 {
                return m, tea.Quit
            }
        case "/":
            m.searching = true
        }
    }
    return m, nil
}
```

### 7. Activation

| Bash                                    | Go                                              |
| --------------------------------------- | ----------------------------------------------- |
| `uuidgen \| tr '[:upper:]' '[:lower:]'` | `uuid.New().String()`                           |
| `jq -n ...` body construction           | `json.Marshal(request)`                         |
| `az rest --method PUT`                  | `RoleAssignmentScheduleRequestsClient.Create()` |
| Sequential loop                         | `errgroup.Group` for parallel activation        |

```go
func activateRoles(ctx context.Context, client *armauthorization.RoleAssignmentScheduleRequestsClient,
    roles []Role, justification string, duration DurationOption) ([]ActivationResult, error) {

    g, ctx := errgroup.WithContext(ctx)
    results := make([]ActivationResult, len(roles))

    for i, role := range roles {
        i, role := i, role
        g.Go(func() error {
            requestID := uuid.New().String()
            startTime := time.Now().UTC().Format(time.RFC3339)

            request := armauthorization.RoleAssignmentScheduleRequest{
                Properties: &armauthorization.RoleAssignmentScheduleRequestProperties{
                    PrincipalID:      &principalID,
                    RoleDefinitionID: &role.RoleDefinitionID,
                    RequestType:      to.Ptr(armauthorization.RequestTypeSelfActivate),
                    Justification:    &justification,
                    ScheduleInfo: &armauthorization.RoleAssignmentScheduleRequestPropertiesScheduleInfo{
                        StartDateTime: &startTime,
                        Expiration: &armauthorization.RoleAssignmentScheduleRequestPropertiesScheduleInfoExpiration{
                            Type:     to.Ptr(armauthorization.TypeAfterDuration),
                            Duration: &duration.ISO8601,
                        },
                    },
                },
            }

            _, err := client.Create(ctx, role.Scope, requestID, request, nil)
            results[i] = ActivationResult{Role: role, Err: err}
            return nil // Don't fail the group on individual errors
        })
    }

    g.Wait()
    return results, nil
}
```

### 8. State Persistence

| Bash                                                                    | Go                                        |
| ----------------------------------------------------------------------- | ----------------------------------------- |
| `jq --argjson now "$NOW_EPOCH" '[.[] \| select(.expiresEpoch > $now)]'` | Filter with `time.Now().Unix()`           |
| `jq '. + [$e]'` append                                                  | `append()`                                |
| `echo "$ALL_STATE" \| jq '.' > "$STATE_FILE"`                           | `json.MarshalIndent()` + `os.WriteFile()` |

```go
type StateStore struct {
    Path string // ~/.pim/activations.json
}

func (s *StateStore) Load() ([]ActivationRecord, error) {
    data, err := os.ReadFile(s.Path)
    if err != nil {
        return nil, nil // File doesn't exist yet
    }
    var records []ActivationRecord
    json.Unmarshal(data, &records)

    // Prune expired
    now := time.Now().Unix()
    filtered := records[:0]
    for _, r := range records {
        if r.ExpiresEpoch > now {
            filtered = append(filtered, r)
        }
    }
    return filtered, nil
}

func (s *StateStore) Save(records []ActivationRecord) error {
    data, _ := json.MarshalIndent(records, "", "  ")
    return os.WriteFile(s.Path, data, 0644)
}
```

---

## Build & Distribution

```bash
# Build for macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o pim ./cmd/pim

# Build for macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o pim ./cmd/pim

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o pim ./cmd/pim

# Install to PATH
go install ./cmd/pim
```

With Go, the binary is fully self-contained — no need for `az`, `jq`, or
any other runtime dependency. Authentication uses the Azure SDK which reads
the same `~/.azure/` credentials that `az login` creates.

---

## Testing Strategy

### Unit Tests

| Component               | Test Approach                            |
| ----------------------- | ---------------------------------------- |
| `DecodeEnv()`           | Table-driven tests with edge cases       |
| `DecodeAppCode()`       | Table-driven tests with short/long names |
| Cache TTL logic         | Mock file system or temp dir             |
| State load/save         | Temp file + JSON round-trip              |
| Activation request body | Marshal and validate JSON structure      |

### Integration Tests

| Test       | Approach                                |
| ---------- | --------------------------------------- |
| API client | Mock HTTP server or recorded responses  |
| TUI        | bubbletea testing framework (`teatest`) |
| End-to-end | `--dry-run` mode with captured output   |

---

## Implementation Checklist

- [x] Set up Go module (`go mod init`)
- [x] Implement config constants
- [x] Implement Azure SDK client factory
- [x] Implement eligible roles fetch + `asTarget()` filter
- [x] Implement active roles fetch + `Activated` filter
- [x] Implement role activation (SelfActivate PUT)
- [x] Implement RG name decoder (env + app code, case-insensitive)
- [x] Implement file-based cache with 24h TTL
- [x] Implement activation state persistence
- [x] Implement TUI role selector (bubbletea) with row render cache
- [x] Implement justification prompt (huh)
- [x] Implement duration selector (huh)
- [x] Implement summary display
- [x] Implement dry-run mode
- [x] Implement status table display
- [x] Wire up cobra commands (root = status, activate = activate)
- [x] Add `--dry-run` and `--no-cache` flags
- [x] Add parallel activation with errgroup
- [x] Add group select (`g` key) for APP2/APP4/APP5/APP3
- [x] Build + test on macOS
- [ ] Write unit tests
- [ ] Write integration tests
