# CR-0015: Add `auditlog` command to retrieve audit log entries

---

## CR Type

- [x] **New resource** - add new command and client file (create new files, register in root)
- [ ] New subcommand
- [ ] Enhancement - change behaviour of an existing command
- [ ] Output change - add/remove/rename columns, change default format, fix display
- [ ] Flag / config change - add/rename/remove a CLI flag or config field

---

## Problem

There is no CLI command to retrieve audit log entries from IICS. Audit logs track all actions
performed in the organization (logins, object creation/deletion, publish events, etc.) and are
essential for compliance and troubleshooting. Users must currently call the REST API manually.

---

## Desired Change

**Informatica API docs:**
`https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-2-resources/audit-logs.html`

Add a new `auditlog` top-level command with a single `list` subcommand that retrieves audit log
entries from the V2 `GET /api/v2/auditlog` endpoint with optional batch pagination.

### Command shape

```text
iics auditlog list [flags]
iics auditlog list --limit 50
iics auditlog list --limit 100 --skip 2 --output json
iics auditlog list --fields id,username,category,event,entryTimeUTC,objectName
```

### Pagination

The underlying API uses `batchSize` + `batchId` parameters. The CLI exposes them as `--limit`
and `--skip` to stay consistent with other commands:

- `--limit` maps to the API `batchSize` query parameter (number of entries per page).
- `--skip` maps to the API `batchId` query parameter (0-based page/batch number; 0 = most recent).
- When neither flag is provided, no pagination parameters are sent and the API returns the most
  recent 200 entries.
- When `--limit` is set, `--skip` defaults to 0 (most recent batch). Setting `--skip` without
  `--limit` is a no-op; the client only sends pagination parameters when `--limit > 0`.

### `--fields` flag

Accepts a comma-separated list of JSON tag names to display in table/CSV output.
JSON and YAML output always include all struct fields regardless of `--fields`.

Default fields: `id,username,category,event,entryTimeUTC,objectName`

---

## Scope

### Files to CREATE

```text
internal/client/auditlogs.go         # AuditLog struct + ListAuditLogs method
internal/client/auditlogs_test.go    # TestListAuditLogs, TestListAuditLogsPaginated
cmd/auditlog.go                      # newAuditlogCmd, newAuditlogListCmd
docs/documentation/auditlog.md       # command reference page
```

### Files to MODIFY

```text
cmd/root.go                          # register newAuditlogCmd() in init()
README.md                            # add auditlog row to Commands table
completions/                         # regenerate via make completions
```

### Files to READ (context only - do NOT modify)

```text
docs/CLAUDE.md
internal/client/client.go
internal/client/securitylogs.go
internal/output/formatter.go
cmd/root.go
```

### Forbidden (do NOT touch)

```text
internal/output/    # output layer is correct as-is
```

---

## API Details

| Field          | Value                                  |
| -------------- | -------------------------------------- |
| API version    | V2 (`api/v2`)                          |
| HTTP method    | GET                                    |
| Endpoint path  | `api/v2/auditlog`                      |
| Session header | auto-detected (v2 path: `icSessionId`) |
| Request body   | none                                   |
| Response type  | array of `auditLogEntry` objects       |

### Query parameters

| API parameter | CLI flag  | Description                                        |
| ------------- | --------- | -------------------------------------------------- |
| `batchSize`   | `--limit` | Number of entries per batch                        |
| `batchId`     | `--skip`  | Batch number; 0 = most recent, 1 = next older batch |

When neither parameter is supplied, the API returns the most recent 200 entries.

### Response field inventory

| JSON tag        | Go type | Description                                         |
| --------------- | ------- | --------------------------------------------------- |
| `id`            | string  | Audit log entry identifier                          |
| `version`       | int     | Version number                                      |
| `orgId`         | string  | Organization identifier                             |
| `username`      | string  | User who performed the action                       |
| `entryTime`     | string  | Action timestamp (Eastern Time)                     |
| `entryTimeUTC`  | string  | Action timestamp (UTC)                              |
| `objectId`      | string  | Identifier of the affected object                   |
| `objectName`    | string  | Name of the affected object                         |
| `category`      | string  | Audit category (AGENT, AUTH, CONNECTION, USER, ...) |
| `event`         | string  | Action type (CREATE, DELETE, UPDATE, RUN, ...)      |
| `eventParam`    | string  | Related objects (max 1024 characters)               |
| `message`       | string  | Additional context information                      |

> **Note:** Verify the exact JSON tag for `entryTimeUTC` against a live API response before
> writing struct code. The official docs show `entryTimeUTC` (uppercase `UTC`) but confirm the
> exact casing from an actual response before finalizing the struct.

---

## Implementation Instructions

> Read `docs/CLAUDE.md` and `internal/client/securitylogs.go` before starting.

### Step 1 - Create `internal/client/auditlogs.go`

```go
package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// AuditLog represents a single audit log entry returned by GET /api/v2/auditlog.
type AuditLog struct {
	ID           string `json:"id,omitempty"`
	Version      int    `json:"version,omitempty"`
	OrgID        string `json:"orgId,omitempty"`
	Username     string `json:"username,omitempty"`
	EntryTime    string `json:"entryTime,omitempty"`
	EntryTimeUTC string `json:"entryTimeUTC,omitempty"`
	ObjectID     string `json:"objectId,omitempty"`
	ObjectName   string `json:"objectName,omitempty"`
	Category     string `json:"category,omitempty"`
	Event        string `json:"event,omitempty"`
	EventParam   string `json:"eventParam,omitempty"`
	Message      string `json:"message,omitempty"`
}

// AuditLogListOptions holds optional query parameters for listing audit logs.
// Limit maps to the API batchSize parameter; Skip maps to the API batchId parameter.
type AuditLogListOptions struct {
	Limit int // 0 = not set; API returns most recent 200 when omitted
	Skip  int // 0-based batch number; only sent when Limit > 0
}

// ListAuditLogs retrieves audit log entries.
// When opts.Limit is 0, no pagination parameters are sent and the API returns
// the most recent 200 entries.
func (c *Client) ListAuditLogs(ctx context.Context, opts AuditLogListOptions) ([]AuditLog, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["batchSize"] = strconv.Itoa(opts.Limit)
		query["batchId"] = strconv.Itoa(opts.Skip)
	}
	var resp []AuditLog
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/auditlog", BaseAPIPathV2), query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
```

Verify the exact JSON tag for `entryTimeUTC` against a real API response before committing.
If the API uses `entryTimeUtc` (lowercase `tc`), update the tag and Go field name accordingly,
and update the column map in `cmd/auditlog.go` to match.

### Step 2 - Create `internal/client/auditlogs_test.go`

```go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListAuditLogs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/auditlog" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		entries := []AuditLog{
			{
				ID:           "al1",
				Username:     "admin@example.com",
				Category:     "AUTH",
				Event:        "LOGIN",
				EntryTimeUTC: "2025-01-15T10:00:00.000Z",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	})

	c := newTestClient(handler)
	logs, err := c.ListAuditLogs(context.Background(), AuditLogListOptions{})
	if err != nil {
		t.Fatalf("ListAuditLogs() error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(logs))
	}
	if logs[0].ID != "al1" {
		t.Errorf("expected ID al1, got %s", logs[0].ID)
	}
	if logs[0].Category != "AUTH" {
		t.Errorf("expected category AUTH, got %s", logs[0].Category)
	}
}

func TestListAuditLogsPaginated(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("batchSize") != "50" {
			t.Errorf("expected batchSize=50, got %s", r.URL.Query().Get("batchSize"))
		}
		if r.URL.Query().Get("batchId") != "1" {
			t.Errorf("expected batchId=1, got %s", r.URL.Query().Get("batchId"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]AuditLog{})
	})

	c := newTestClient(handler)
	_, err := c.ListAuditLogs(context.Background(), AuditLogListOptions{Limit: 50, Skip: 1})
	if err != nil {
		t.Fatalf("ListAuditLogs() error: %v", err)
	}
}
```

### Step 3 - Create `cmd/auditlog.go`

```go
package cmd

import (
	"context"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

var auditLogColumnMap = map[string]output.Column{
	"id":           {Header: "ID", Field: "id", Width: 24},
	"username":     {Header: "USERNAME", Field: "username", Width: 30},
	"category":     {Header: "CATEGORY", Field: "category", Width: 16},
	"event":        {Header: "EVENT", Field: "event", Width: 16},
	"entryTimeUTC": {Header: "TIME (UTC)", Field: "entryTimeUTC", Width: 24},
	"entryTime":    {Header: "TIME (ET)", Field: "entryTime", Width: 24},
	"objectId":     {Header: "OBJECT ID", Field: "objectId", Width: 24},
	"objectName":   {Header: "OBJECT NAME", Field: "objectName", Width: 30},
	"eventParam":   {Header: "EVENT PARAM", Field: "eventParam", Width: 40},
	"message":      {Header: "MESSAGE", Field: "message", Width: 40},
	"orgId":        {Header: "ORG ID", Field: "orgId", Width: 24},
	"version":      {Header: "VERSION", Field: "version", Width: 8},
}

const auditLogDefaultFields = "id,username,category,event,entryTimeUTC,objectName"

func buildAuditLogColumns(fields string) []output.Column {
	names := strings.Split(fields, ",")
	cols := make([]output.Column, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if col, ok := auditLogColumnMap[name]; ok {
			cols = append(cols, col)
		}
	}
	return cols
}

func newAuditlogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auditlog",
		Short: "Retrieve audit log entries",
	}
	cmd.AddCommand(newAuditlogListCmd())
	return cmd
}

func newAuditlogListCmd() *cobra.Command {
	var opts client.AuditLogListOptions
	var fields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			logs, err := c.ListAuditLogs(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			return f.Format(logs, buildAuditLogColumns(fields))
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "number of entries per page (maps to API batchSize; 0 = return most recent 200)")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "page number to retrieve, 0-based (maps to API batchId; only used when --limit > 0)")
	cmd.Flags().StringVar(&fields, "fields", auditLogDefaultFields, "comma-separated list of fields to display (table/csv only)")
	return cmd
}
```

Update all `entryTimeUTC` occurrences in the column map if the verified JSON tag differs.

### Step 4 - Register in `cmd/root.go`

Inside the `init()` function, add:

```go
rootCmd.AddCommand(newAuditlogCmd())
```

Follow the alphabetical order of existing `AddCommand` calls.

### Step 5 - Create `docs/documentation/auditlog.md`

Create a command reference page following the same structure as other command docs.
Include:

- Synopsis
- Subcommands table
- Flags table for `list` (document that `--limit`/`--skip` map to API `batchSize`/`batchId`)
- Output columns table (all 12 JSON tag names with descriptions)
- Usage examples (default, paginated, custom fields, JSON output)

### Step 6 - Update `README.md`

Add a row to the Commands table:

```markdown
| `auditlog` | Retrieve audit log entries | [docs](docs/documentation/auditlog.md) |
```

Follow the alphabetical order of existing rows.

### Step 7 - Regenerate completions

```bash
make completions
```

Commit updated `completions/` files together with the code change.

### Step 8 - Verify

```bash
/opt/local/bin/go build ./...
/opt/local/bin/go test ./internal/client/... -v -run TestListAuditLogs
/opt/local/bin/go test ./internal/client/... -v -run TestListAuditLogsPaginated
/opt/local/bin/go test ./...
/opt/local/bin/go vet ./...
```

All must pass with zero new errors or warnings.

---

## Output Columns

### `auditlog list` (selectable via `--fields`)

| Header      | Field (JSON tag) | Width | Notes                                     |
| ----------- | ---------------- | ----- | ----------------------------------------- |
| ID          | `id`             | 24    | default                                   |
| USERNAME    | `username`       | 30    | default                                   |
| CATEGORY    | `category`       | 16    | default; AGENT, AUTH, CONNECTION, USER... |
| EVENT       | `event`          | 16    | default; CREATE, DELETE, UPDATE, RUN...   |
| TIME (UTC)  | `entryTimeUTC`   | 24    | default                                   |
| OBJECT NAME | `objectName`     | 30    | default                                   |
| TIME (ET)   | `entryTime`      | 24    | optional; Eastern Time                    |
| OBJECT ID   | `objectId`       | 24    | optional                                  |
| EVENT PARAM | `eventParam`     | 40    | optional; related objects (max 1024 ch)   |
| MESSAGE     | `message`        | 40    | optional                                  |
| ORG ID      | `orgId`          | 24    | optional                                  |
| VERSION     | `version`        | 8     | optional                                  |

---

## Acceptance Criteria

- [ ] `iics auditlog list` returns the most recent 200 entries in table format
- [ ] `iics auditlog list --limit 50` sends `batchSize=50&batchId=0` to the API
- [ ] `iics auditlog list --limit 50 --skip 2` sends `batchSize=50&batchId=2` to the API
- [ ] `iics auditlog list --fields id,username,event` shows exactly those 3 columns in table output
- [ ] `iics auditlog list --output json` includes all struct fields regardless of `--fields`
- [ ] `iics auditlog list --output csv` respects `--fields`
- [ ] `go build ./...` succeeds with no errors
- [ ] `go test ./...` passes with no failures
- [ ] `go vet ./...` reports no issues
- [ ] No unrelated code was modified
- [ ] Two-layer rule respected: no API logic in `cmd/`, no Cobra in `internal/client/`
- [ ] `docs/documentation/auditlog.md` created with full flag and column reference
- [ ] `README.md` Commands table updated
- [ ] Completions regenerated

---

## Do NOT

- Refactor, reformat, or add comments to code outside the CR scope
- Modify any file not listed in the Scope section
- Add error handling for scenarios that cannot happen
- Use `os.Exit()` - return errors from `RunE`
- Hard-code base API paths - use `BaseAPIPathV2`
- Guess JSON field names - verify the exact `entryTimeUTC` casing from a real API response
- Add `Co-Authored-By` trailers to commit messages
