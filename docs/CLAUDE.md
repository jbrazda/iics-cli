# CLAUDE.md - IICS CLI Project Guide

This file is the authoritative reference for Claude when working on `github.com/jbrazda/iics-cli`.
Read it before making any changes. It describes conventions, patterns, and rules that **must** be followed.

---

## Project Overview

A Go CLI for the Informatica Intelligent Cloud Services (IICS) REST API.
Supports CI/CD and interactive use for managing IICS resources: objects, connections, exports,
imports, users, roles, schedules, agents, runtime environments, and more.

**Module:** `github.com/jbrazda/iics-cli`
**Go version:** 1.25.0
**Go binary on this machine:** `/opt/local/bin/go`

---

## Directory Layout

```text
iics_cli/
├── main.go                        # Entry point; injects version via ldflags
├── go.mod / go.sum
├── Makefile
├── cmd/                           # Cobra command definitions (thin orchestration layer)
│   ├── root.go                    # Root command, global flags, init(), helpers
│   ├── login.go / logout.go
│   ├── agent.go                   # Secure agents (v2 API)
│   ├── connection.go
│   ├── export.go
│   ├── import_.go                 # Named import_.go to avoid Go keyword conflict
│   ├── folder.go
│   ├── lookup.go
│   ├── metering.go
│   ├── objects.go
│   ├── permission.go
│   ├── privilege.go
│   ├── project.go
│   ├── role.go
│   ├── runtime.go
│   ├── schedule.go
│   ├── securitylog.go
│   ├── sourcecontrol.go
│   ├── profile.go
│   ├── state.go
│   ├── tag.go
│   ├── user.go
│   └── usergroup.go
├── internal/
│   ├── client/                    # HTTP client + API data types + tests
│   │   ├── client.go              # Core Client struct, HTTP logic, session management
│   │   ├── auth.go                # Login/Logout, LoginResponse, UserInfo, Product
│   │   ├── errors.go              # APIError, SessionExpiredError, exit codes
│   │   ├── helpers.go             # parseJSON helper
│   │   ├── state.go               # Global client state utilities
│   │   ├── agents.go
│   │   ├── connections.go
│   │   ├── export.go              # ExportJob, JobStatus, JobObject (shared with imports)
│   │   ├── folders.go
│   │   ├── imports.go
│   │   ├── lookup.go
│   │   ├── metering.go
│   │   ├── objects.go
│   │   ├── permissions.go
│   │   ├── privileges.go
│   │   ├── projects.go
│   │   ├── roles.go
│   │   ├── runtimes.go
│   │   ├── schedules.go
│   │   ├── securitylogs.go
│   │   ├── sourcecontrol.go
│   │   ├── tags.go
│   │   ├── users.go
│   │   ├── usergroups.go
│   │   └── *_test.go              # One test file per client module
│   ├── config/
│   │   ├── config.go              # Config/Profile structs, Viper loading
│   │   ├── pods.go                # Region -> login URL mapping
│   │   ├── prompt.go              # Interactive profile prompting (IsTerminal, PromptProfile)
│   │   ├── prompt_test.go         # Tests for prompt logic
│   │   └── session.go             # Session cache (~/.iics/sessions.yaml)
│   └── output/
│       ├── formatter.go           # Format enum + Formatter interface + Column struct
│       ├── table.go               # lipgloss table renderer
│       ├── json.go                # JSON renderer
│       └── csv.go                 # CSV renderer
├── testdata/
│   └── imports/                   # Sample ZIP files for import testing
└── docs/
    ├── CLAUDE.md                  # This file
    ├── DESIGN.md
    ├── ChangeRequests/
    └── issues/
```

---

## Key Dependencies

| Package                             | Version | Purpose                          |
| ----------------------------------- | ------- | -------------------------------- |
| `github.com/spf13/cobra`            | v1.10.2 | CLI framework                    |
| `github.com/spf13/viper`            | v1.21.0 | Config file + env var management |
| `github.com/charmbracelet/lipgloss` | v1.1.0  | Table output and terminal styles |
| `github.com/mattn/go-isatty`        | v0.0.20 | TTY detection for color fallback |
| `gopkg.in/yaml.v3`                  | v3.0.1  | Session cache serialization      |

No external HTTP library - uses standard `net/http` only.

---

## Table Output

Table rendering uses a custom lipgloss-based renderer in `internal/output/table.go`.
Do **not** add new `tablewriter` imports - the dependency has been removed.

To render a table, use `output.New()` with an `output.TableStyle`:

```go
f := output.New(output.FormatTable, w, output.TableStyle{})
cols := []output.Column{
    {Header: "ID",   Field: "id",   Width: 24},
    {Header: "NAME", Field: "name"},
}
_ = f.Format(data, cols)
```

`TableStyle.NoColor = true` forces plain ASCII rendering (no color, no Unicode borders).
When the output writer is not a TTY the renderer automatically falls back to plain style.

---

## API Versioning: V2 vs V3

Two API versions are in use. The session header **differs** between them.

|                    | V2                           | V3                                 |
| ------------------ | ---------------------------- | ---------------------------------- |
| Base path constant | `BaseAPIPathV2 = "api/v2"`   | `BaseAPIPathV3 = "public/core/v3"` |
| Session header     | `icSessionId`                | `INFA-SESSION-ID`                  |
| Used for           | connections, agents, lookups | everything else                    |

**Auto-detection in `client.go`:** `do()` checks whether the URL path contains `/v2/` and sets the appropriate session header automatically. No manual header management needed in resource files.

### Resources by API version

| V2 (`api/v2`) | V3 (`public/core/v3`)                        |
| ------------- | -------------------------------------------- |
| `agent`       | `users`, `userGroups`, `roles`, `privileges` |
| `connection`  | `export`, `import`                           |
| `lookup`      | `runtimeEnvironments`                        |
|               | `folders`, `projects`                        |
|               | `objects`, `schedules`, `tags`               |
|               | `permissions`, `securityLogs`, `metering`    |
|               | `sourceControl`                              |

---

## Architecture: The Two-Layer Rule

**`cmd/` is thin.** Command files only:

1. Define flags and local variables
2. Call `getClient(cmd)` and `getFormatter()`
3. Call a single client method
4. Format and print the result

**`internal/client/` is thick.** All API logic, structs, request building, and response parsing lives here.

Never put API logic, URL construction, or JSON handling in `cmd/`.
Never put Cobra or output logic in `internal/client/`.

---

## Adding a New Resource - Required Pattern

### 1. Client file: `internal/client/<resource>s.go`

```go
package client

import (
    "context"
    "fmt"
    "net/http"
    "strconv"
)

// Widget represents an IICS widget.
type Widget struct {
    ID          string `json:"id,omitempty"`
    OrgID       string `json:"orgId,omitempty"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    CreateTime  string `json:"createTime,omitempty"`
    UpdateTime  string `json:"updateTime,omitempty"`
    CreatedBy   string `json:"createdBy,omitempty"`
    UpdatedBy   string `json:"updatedBy,omitempty"`
}

// WidgetListOptions holds query parameters for listing widgets.
type WidgetListOptions struct {
    Limit int
    Skip  int
}

func (c *Client) ListWidgets(ctx context.Context, opts WidgetListOptions) ([]Widget, error) {
    query := make(map[string]string)
    if opts.Limit > 0 {
        query["limit"] = strconv.Itoa(opts.Limit)
    }
    if opts.Skip > 0 {
        query["skip"] = strconv.Itoa(opts.Skip)
    }
    var resp []Widget
    if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/widgets", BaseAPIPathV3), query, nil, &resp); err != nil {
        return nil, err
    }
    return resp, nil
}

func (c *Client) GetWidget(ctx context.Context, id string) (*Widget, error) {
    var resp Widget
    if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/widgets/%s", BaseAPIPathV3, id), nil, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *Client) CreateWidget(ctx context.Context, w *Widget) (*Widget, error) {
    var resp Widget
    if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("%s/widgets", BaseAPIPathV3), w, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *Client) UpdateWidget(ctx context.Context, id string, w *Widget) (*Widget, error) {
    var resp Widget
    // Use PATCH when the API requires partial update; PUT for full replacement
    if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("%s/widgets/%s", BaseAPIPathV3, id), w, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *Client) DeleteWidget(ctx context.Context, id string) error {
    return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/widgets/%s", BaseAPIPathV3, id), nil, nil)
}
```

### 2. Test file: `internal/client/<resource>s_test.go`

```go
package client

import (
    "context"
    "encoding/json"
    "net/http"
    "testing"
)

func TestListWidgets(t *testing.T) {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            t.Errorf("expected GET, got %s", r.Method)
        }
        widgets := []Widget{
            {ID: "w1", Name: "Widget One"},
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(widgets)
    })

    c := newTestClient(handler)
    widgets, err := c.ListWidgets(context.Background(), WidgetListOptions{})
    if err != nil {
        t.Fatalf("ListWidgets() error: %v", err)
    }
    if len(widgets) != 1 {
        t.Errorf("expected 1 widget, got %d", len(widgets))
    }
}
```

The `newTestClient` helper is defined in `internal/client/objects_test.go`:

```go
func newTestClient(handler http.Handler) *Client {
    srv := httptest.NewServer(handler)
    c := NewClient(srv.URL+"/login", "user", "pass")
    c.SetSession("test-session", srv.URL)
    return c
}
```

### 3. Command file: `cmd/<resource>.go`

```go
package cmd

import (
    "context"
    "fmt"

    "github.com/jbrazda/iics-cli/internal/client"
    "github.com/jbrazda/iics-cli/internal/output"
    "github.com/spf13/cobra"
)

func newWidgetCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "widget",
        Short: "Manage widgets",
    }
    cmd.AddCommand(newWidgetListCmd())
    cmd.AddCommand(newWidgetGetCmd())
    cmd.AddCommand(newWidgetCreateCmd())
    cmd.AddCommand(newWidgetUpdateCmd())
    cmd.AddCommand(newWidgetDeleteCmd())
    return cmd
}

func newWidgetListCmd() *cobra.Command {
    var opts client.WidgetListOptions
    cmd := &cobra.Command{
        Use:   "list",
        Short: "List widgets",
        RunE: func(cmd *cobra.Command, args []string) error {
            c, err := getClient(cmd)
            if err != nil {
                return err
            }
            widgets, err := c.ListWidgets(context.Background(), opts)
            if err != nil {
                return err
            }
            f, err := getFormatter()
            if err != nil {
                return err
            }
            columns := []output.Column{
                {Header: "ID", Field: "id", Width: 24},
                {Header: "NAME", Field: "name", Width: 30},
                {Header: "UPDATED", Field: "updateTime", Width: 22},
            }
            return f.Format(widgets, columns)
        },
    }
    cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
    cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
    return cmd
}
```

### 4. Register in `cmd/root.go`

Add inside the `init()` function:

```go
rootCmd.AddCommand(newWidgetCmd())
```

### 5. Documentation: `docs/documentation/<resource>.md`

Create a command reference page with:

- Synopsis block
- Subcommands table
- Flags table per subcommand (type, default, description)
- Output columns table (JSON tag name + description)
- Usage examples

### 6. Update `README.md` Commands table

Add a row to the Commands table linking to the new doc page.

**Documentation is mandatory.** Every time a command is added or updated, steps 5 and 6 must be
completed before the work is considered done.

### 7. Regenerate shell completions

```bash
make completions
```

Commit the updated files in `completions/` together with the code change.

---

## Struct Field Rules

### JSON tags must match the actual API field names exactly

Always verify against the API documentation before coding structs.
Wrong JSON tags are the most common source of bugs (e.g., `emails` vs `email`, `name` vs `userName`).

### Use `omitempty` on all optional fields

```go
Name        string `json:"name"`           // required - no omitempty
Description string `json:"description,omitempty"`  // optional
```

### Nested objects must be proper structs - never `[]string` for API object arrays

```go
// WRONG - groups is an array of objects, not strings
Groups []string `json:"groups,omitempty"`

// CORRECT
type UserGroupRef struct {
    ID            string `json:"id,omitempty"`
    UserGroupName string `json:"userGroupName,omitempty"`
}
Groups []UserGroupRef `json:"groups,omitempty"`
```

### Match Go type to JSON type precisely

If the API returns a number as a JSON string (e.g., `"maxLoginAttempts": "5"`), use `string` in Go, not `int`.
If the API returns `true`/`false`, use `bool`.

### Common field name mappings (v3 API conventions)

| Go field     | JSON tag       |
| ------------ | -------------- |
| `ID`         | `id`           |
| `OrgID`      | `orgId`        |
| `CreateTime` | `createTime`   |
| `UpdateTime` | `updateTime`   |
| `CreatedBy`  | `createdBy`    |
| `UpdatedBy`  | `updatedBy`    |
| `UserName`   | `userName`     |
| `TimeZoneID` | `timeZoneId`   |
| `AgentHost`  | `agentHost`    |
| `GroupID`    | `agentGroupId` |

---

## HTTP Method Rules

Always use the method the API specifies. Do not assume:

- **GET** - retrieve (list or single)
- **POST** - create
- **PUT** - full replacement update
- **PATCH** - partial update (e.g., folder update uses PATCH)
- **DELETE** - delete

When in doubt, consult the Informatica docs URL referenced in the relevant issue/CR.

---

## Request Body Rules

Only include fields the API accepts in POST/PUT/PATCH bodies.
Use a dedicated unexported request struct when the response struct has extra read-only fields:

```go
// folderRequest only contains fields the API accepts for create/update
type folderRequest struct {
    Name        string `json:"name,omitempty"`
    Description string `json:"description,omitempty"`
}
```

---

## Output Column Field Names

`Column.Field` must match the **JSON tag** of the struct, not the Go field name.
The table formatter marshals the struct to JSON then looks up keys by the field string.

Supports dot notation for nested fields:

```go
{Header: "STATUS", Field: "status.state", Width: 12},
{Header: "MESSAGE", Field: "status.message"},
```

---

## Global Flags (from `cmd/root.go`)

| Flag               | Variable    | Default               | Description            |       |      |
| ------------------ | ----------- | --------------------- | ---------------------- | ----- | ---- |
| `--config`         | `cfgFile`   | `~/.iics/config.yaml` | Config file path       |       |      |
| `--profile` / `-p` | `profile`   | from config           | Profile name           |       |      |
| `--output` / `-o`  | `outputFmt` | `"table"`             | Output format: `table\ | json\ | csv` |
| `--verbose` / `-v` | `verbose`   | `false`               | Verbose output         |       |      |
| `--no-color`       | `noColor`   | `false`               | Disable color          |       |      |
| `--http-timeout`   | `httpTimeoutFlag` | `0` (resolves to `120`) | Per-HTTP-request timeout in seconds; overrides config `httpTimeout` / `IICS_HTTP_TIMEOUT`. Independent of `--max-wait-time` (job polling only). |

`verbose` is a package-level `bool` in the `cmd` package - access it directly in `RunE` closures.

---

## Configuration System

**Config file:** `~/.iics/config.yaml`

```yaml
defaultProfile: dev
httpTimeout: 120        # Optional: per-HTTP-request timeout in seconds (default 120)
profiles:
  dev:
    name: "Development Org"
    region: "US"           # Maps to login URL via pods.go
    username: "user@example.com"
    password: ""
    loginUrl: ""           # Optional: overrides region lookup
```

**Environment variable overrides (highest precedence):**

| Env var          | Overrides        |
| ---------------- | ---------------- |
| `IICS_PROFILE`   | `--profile` flag |
| `IICS_USERNAME`  | profile username |
| `IICS_PASSWORD`  | profile password |
| `IICS_REGION`    | profile region   |
| `IICS_LOGIN_URL` | profile loginUrl |
| `IICS_HTTP_TIMEOUT` | `httpTimeout` config value (per-request timeout in seconds; `--http-timeout` flag wins if explicitly set) |

**Session cache:** `~/.iics/sessions.yaml`

- Expires after 30 minutes
- Never commit this file
- Loaded automatically before every command; saves after successful login

---

## Error Handling

- **All client methods** return `(result, error)` - wrap errors with `fmt.Errorf("context: %w", err)`
- **Command `RunE`** returns errors - `cmd/root.go` handles formatting and exit codes
- **`APIError`** is used for all HTTP error responses - access `.StatusCode`, `.Code`, `.Message`
- **Exit codes:** `ExitOK=0`, `ExitError=1`, `ExitUsageError=2` (from `internal/client/errors.go`)

Never call `os.Exit()` directly. Return an error from `RunE` instead.

---

## Delete Command Pattern

All delete commands must prompt for confirmation unless `--yes` / `-y` is provided:

```go
if !yes {
    fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete %s? [y/N]: ", id)
    var confirm string
    fmt.Scanln(&confirm)
    if confirm != "y" && confirm != "Y" {
        fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
        return nil
    }
}
```

---

## Complex Create/Update Pattern

For resources with many fields, accept a JSON file instead of dozens of flags:

```go
cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with resource definition (required)")

// In RunE:
data, err := os.ReadFile(fromFile)
var resource client.Resource
if err := json.Unmarshal(data, &resource); err != nil {
    return fmt.Errorf("parsing JSON: %w", err)
}
```

---

## Async Job Pattern (Export / Import)

For long-running jobs, poll with a ticker until the terminal state is reached:

```go
// Terminal states: SUCCESSFUL, FAILED, ERROR, WARNINGS
// In-progress states: IN_PROGRESS, QUEUED, STARTING
for isInProgress(job.Status.State) {
    time.Sleep(interval)
    job, err = c.GetStatus(ctx, job.ID, expand)
    if err != nil {
        return err
    }
}
```

The `import run` command (in `cmd/import_.go`) is the reference implementation for:

- Combined upload → start → poll → final output
- `--polling-interval`, `--max-wait-time`, `--detailed-polling`, `--print-import-log`
- Verbose timestamped progress output
- Automatic log download on failure

---

## Testing Rules

- Tests live in `internal/client/<resource>_test.go` (same package: `package client`)
- Use `newTestClient(handler)` - creates a real `httptest.Server` and returns a configured `*Client`
- Always assert the HTTP method and URL path in the handler
- Always verify the response fields you care about
- Do **not** write tests for `cmd/` layer - test the client layer only

```go
func TestGetWidget(t *testing.T) {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            t.Errorf("expected GET, got %s", r.Method)
        }
        if r.URL.Path != "/public/core/v3/widgets/w1" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        widget := Widget{ID: "w1", Name: "Test Widget"}
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(widget)
    })

    c := newTestClient(handler)
    widget, err := c.GetWidget(context.Background(), "w1")
    if err != nil {
        t.Fatalf("GetWidget() error: %v", err)
    }
    if widget.ID != "w1" {
        t.Errorf("expected ID w1, got %s", widget.ID)
    }
}
```

Run tests with: `/opt/local/bin/go test ./...`
Build with: `/opt/local/bin/go build ./...`

---

## What NOT to Do

- **Do not refactor unrelated code** when fixing a bug or implementing a CR
- **Do not add comments or docstrings** to code you did not change
- **Do not add error handling** for impossible scenarios
- **Do not create abstractions** for patterns used only once
- **Do not use `os.Exit()`** - return errors from `RunE`
- **Do not add features** beyond what the issue/CR explicitly requests
- **Do not add `tablewriter` imports** - the dependency was removed; use `output.New()` with lipgloss renderer
- **Do not guess JSON field names** - always verify against the API docs
- **Do not use em dashes** (`-`) in generated Markdown; use a regular hyphen (`-`) instead

---

## Informatica API Documentation

Base URL for docs:
`https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/`

Key sections:

- Platform REST API Version 3: `platform-rest-api-version-3-resources/`
- Platform REST API Version 2: `platform-rest-api-version-2-resources/`

Login endpoints (all return same token, use v3):

- V3: `POST /saas/public/core/v3/login` → returns `sessionId` + `baseApiUrl`
- V2: `POST /ma/api/v2/user/login` → returns `icSessionId` + `serverUrl`
