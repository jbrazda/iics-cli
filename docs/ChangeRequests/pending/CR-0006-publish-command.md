# CR-0006: Implement `publish` and `unpublish` Commands for CAI Asset Deployment

## CR Type

- [x] **New resource** - add brand-new `iics publish` and `iics unpublish` command trees

---

## Problem

The legacy Informatica Asset Management CLI V2 (KB_DOC-18245) included `publish` and
`publish status` commands for deploying Cloud Application Integration (CAI) assets
(processes, service connectors, connections, guides, mappings, process objects) to the
IICS runtime. These commands are absent from `iics-cli`.

Deploying CAI assets after import or development is a critical CI/CD step. Without CLI
support, users must invoke the REST API manually via curl or Postman. There is also no
equivalent unpublish capability. Both operations need to work in automated pipelines with
consistent flags, polling behavior, and verbose logging matching the existing `import run`
and `export run` commands.

---

## Desired Change

Add two new top-level command trees that share client-layer code:

**`iics publish`**

- `iics publish start` - submit a publish job; accept assets from flag, file, or stdin;
  print job ID and exit (fire-and-forget)
- `iics publish status` - retrieve or poll status of a publish job by ID
- `iics publish run` - full workflow: resolve inputs, auto-batch into 199-asset chunks,
  submit, poll to completion, print detailed summary (mirrors `import run` pattern)

**`iics unpublish`**

- `iics unpublish start` - same as publish start but POSTs to `/unpublish`
- `iics unpublish status`
- `iics unpublish run`

**CAI URL auto-detection:** The publish API uses a CAI-specific service host. The host is
derived automatically from the login response `products[].baseApiUrl` field (strip the path,
keep scheme+host). An optional `caiUrl` config field and `IICS_CAI_URL` env var allow override.

**Reference:** Official Informatica docs:
`https://docs.informatica.com/ipaas/application-integration/current-version/invoke/publishing-application-integration-assets-in-bulk.html`
`https://docs.informatica.com/ipaas/application-integration/current-version/invoke/unpublishing-application-integration-assets-in-bulk.html`

---

## Scope

### Files to CREATE

```text
internal/client/publish.go        # all publish/unpublish structs and client methods
internal/client/publish_test.go   # tests for each client method
cmd/publish.go                    # iics publish start/status/run
cmd/unpublish.go                  # iics unpublish start/status/run (thin wrapper over publish client code)
docs/documentation/publish.md     # command reference page
docs/documentation/unpublish.md   # command reference page
```

### Files to MODIFY

```text
internal/client/client.go         # add caiURL field, WithCAIURL option, CAIURL() accessor, doCAIJSON method
internal/client/auth.go           # auto-set caiURL from login response products[].baseApiUrl
internal/config/config.go         # add CaiURL field to Profile + IICS_CAI_URL env var
cmd/root.go                       # expose CaiURL from getClient(), register newPublishCmd() and newUnpublishCmd()
README.md                         # add publish and unpublish rows to commands table
```

### Files to READ (context only - do NOT modify)

```text
docs/CLAUDE.md                          # mandatory: patterns and rules
internal/client/client.go               # doJSON / doCAIJSON / ensureSession signatures
internal/client/auth.go                 # Login() - where to inject caiURL auto-detection
internal/client/objects_test.go         # newTestClient() helper definition
internal/output/formatter.go            # Column struct, Formatter interface
internal/config/config.go               # Profile struct, ResolveProfile, env var override pattern
cmd/root.go                             # getClient(), getFormatter(), resolveProfile(), global flags
cmd/import_.go                          # reference for run/polling/verbose/detailed-polling pattern
cmd/export.go                           # reference for readArtifacts/stdin/--from-file input pattern
```

### Forbidden (do NOT touch)

```text
internal/output/       # output layer is correct as-is
internal/config/pods.go
```

---

## API Details

This API uses a **CAI-specific service URL** and **JSON:API content type** (`application/vnd.api+json`),
different from all other IICS resources. The base URL is automatically derived from the login
response (see "CAI URL auto-detection" section below).

### Start publish

| Field          | Value                                                          |
| -------------- | -------------------------------------------------------------- |
| HTTP method    | POST                                                           |
| Endpoint       | `<cai-base-url>/active-bpel/asset/v1/publish`                  |
| Content-Type   | `application/vnd.api+json`                                     |
| Session header | `INFA-SESSION-ID` (same as V3 - set automatically by `do()`)   |
| Request body   | JSON:API wrapper with `assetPaths` array (max 199 per request) |
| Response       | HTTP 202 with JSON:API publish job object                      |

Request body:

```json
{
  "data": {
    "type": "publish",
    "attributes": {
      "assetPaths": [
        "Explore/<location>/<name>.PROCESS.xml",
        "Explore/<location>/<name>.AI_SERVICE_CONNECTOR.xml",
        "Explore/<location>/<name>.AI_CONNECTION.xml"
      ]
    }
  }
}
```

Response (HTTP 202):

```json
{
  "data": {
    "type": "publish",
    "id": "<publish-id>",
    "attributes": {
      "jobState": "NOT_STARTED",
      "jobStatusDetail": {},
      "startedBy": "<username>",
      "startDate": "<ISO-8601-timestamp>",
      "totalCount": 3,
      "processedCount": 0,
      "assetPaths": ["..."]
    }
  },
  "links": {
    "self": "<status-url>",
    "status": "<status-detail-url>"
  }
}
```

**Known `jobState` values:** `NOT_STARTED`, `PROCESSING`, `COMPLETED`, and likely `FAILED`.

**Constraint:** Maximum 199 assets per request. If more are provided, split into sequential
batches of up to 199 and run each batch.

### Get publish status

`GET <cai-base-url>/active-bpel/asset/v1/publish/<id>/Status` - returns status fields only

`GET <cai-base-url>/active-bpel/asset/v1/publish/<id>` - returns full job including asset list

Both return the same JSON structure as the start response (with current `jobState` and counts).

### Start unpublish

| Field          | Value                                                              |
| -------------- | ------------------------------------------------------------------ |
| HTTP method    | POST                                                               |
| Endpoint       | `<cai-base-url>/active-bpel/asset/v1/unpublish`                    |
| Content-Type   | `application/vnd.api+json`                                         |
| Session header | `INFA-SESSION-ID`                                                  |

Request body: same structure as publish, with `"type": "unpublish"`.

Status endpoints: `GET .../unpublish/<id>/Status` and `GET .../unpublish/<id>`

### Supported asset path suffixes

```text
.PROCESS.xml              - processes
.AI_SERVICE_CONNECTOR.xml - service connectors
.AI_CONNECTION.xml        - application integration connections
.DTEMPLATE.xml            - mappings
.GUIDE.xml                - guides
.PROCESS_OBJECT.xml       - process objects
```

**Path format:** `Explore/<folder-path>/<asset-name>.<type-suffix>`

When reading from `iics objects list -o csv`, convert: `Explore/<path>.<type>.xml`

**Known bug ICAI-41690:** The `links.self` and `links.status` fields in the response use
`http://` instead of `https://`. Do NOT follow these links. Always construct the status URL
manually from the job ID.

---

## Implementation Instructions

### Step 0 - Config layer (`internal/config/config.go`)

Add `CaiURL` field to the `Profile` struct:

```go
type Profile struct {
    Name     string `yaml:"name" mapstructure:"name"`
    Region   string `yaml:"region" mapstructure:"region"`
    Username string `yaml:"username" mapstructure:"username"`
    Password string `yaml:"password" mapstructure:"password"`
    LoginURL string `yaml:"loginUrl,omitempty" mapstructure:"loginUrl"`
    CaiURL   string `yaml:"caiUrl,omitempty" mapstructure:"caiUrl"`
}
```

Add env var override in `ResolveProfile()` after the existing `IICS_LOGIN_URL` block:

```go
if v := os.Getenv("IICS_CAI_URL"); v != "" {
    profile.CaiURL = v
}
```

### Step 1 - Client struct additions (`internal/client/client.go`)

1. Add `caiURL string` field to the `Client` struct.

2. Add `WithCAIURL` option:

   ```go
   func WithCAIURL(url string) ClientOption {
       return func(c *Client) { c.caiURL = url }
   }
   ```

3. Add `CAIURL()` accessor:

   ```go
   func (c *Client) CAIURL() string {
       c.mu.RLock()
       defer c.mu.RUnlock()
       return c.caiURL
   }
   ```

4. Add `doCAIJSON` method. It sends a JSON:API request to an absolute URL, sets
   `application/vnd.api+json` headers, and decodes the response. It bypasses `c.apiURL()`:

   ```go
   func (c *Client) doCAIJSON(ctx context.Context, method, absoluteURL string, reqBody, respBody interface{}) error {
       if err := c.ensureSession(ctx); err != nil {
           return err
       }
       var body io.Reader
       var reqData []byte
       if reqBody != nil {
           var err error
           reqData, err = json.Marshal(reqBody)
           if err != nil {
               return fmt.Errorf("marshaling request: %w", err)
           }
           body = bytes.NewReader(reqData)
       }
       req, err := http.NewRequestWithContext(ctx, method, absoluteURL, body)
       if err != nil {
           return fmt.Errorf("creating request: %w", err)
       }
       req.Header.Set("Content-Type", "application/vnd.api+json")
       req.Header.Set("Accept", "application/vnd.api+json")
       resp, err := c.do(ctx, req)
       if err != nil {
           return err
       }
       defer func() { _ = resp.Body.Close() }()
       respData, err := io.ReadAll(resp.Body)
       if err != nil {
           return fmt.Errorf("reading response: %w", err)
       }
       if resp.StatusCode < 200 || resp.StatusCode >= 300 {
           if c.debug && len(reqData) > 0 {
               _, _ = fmt.Fprintf(os.Stderr, "DEBUG request body (%s %s):\n%s\n", method, absoluteURL, reqData)
           }
           return newAPIError(resp, respData)
       }
       if respBody != nil && len(respData) > 0 {
           if err := json.Unmarshal(respData, respBody); err != nil {
               return fmt.Errorf("parsing response: %w", err)
           }
       }
       return nil
   }
   ```

   Note: `doWithSession` already sets `INFA-SESSION-ID` for all non-`/v2/` paths. The
   explicit `Content-Type` set here overrides the default in `doWithSession` (which only
   sets it when not already present).

### Step 2 - CAI URL auto-detection (`internal/client/auth.go`)

In `Login()`, after setting `c.baseAPIURL`, also set `c.caiURL` from the products base URL.
The CAI service host is the scheme+host portion of the product's `baseApiUrl`:

```go
// Auto-detect CAI base URL from the product base URL (scheme+host only).
// The login response products[].baseApiUrl contains the POD-specific URL,
// e.g. "https://na1.ai.dm-us.informaticacloud.com/saas/public".
// The CAI service is at the root of that host.
if baseURL != "" {
    if u, err := url.Parse(baseURL); err == nil && u.Host != "" {
        c.mu.Lock()
        c.caiURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
        c.mu.Unlock()
    }
}
```

Add `"net/url"` to the imports in `auth.go`.

If `WithCAIURL` was called with a non-empty value (from config/flag), it takes precedence
over the auto-detected value. Implement this by only setting `c.caiURL` in Login() when
the field is empty:

```go
c.mu.Lock()
if c.caiURL == "" {
    if u, err := url.Parse(baseURL); err == nil && u.Host != "" {
        c.caiURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
    }
}
c.mu.Unlock()
```

### Step 3 - Client file (`internal/client/publish.go`)

Define structs matching the JSON:API schema exactly, shared by both publish and unpublish.
Use a single `PublishJobResponse` type for both operations since the schemas are identical:

```go
package client

import (
    "context"
    "fmt"
    "net/http"
    "strings"
)

// publishRequest is the JSON:API request body for start publish/unpublish.
type publishRequest struct {
    Data publishRequestData `json:"data"`
}

type publishRequestData struct {
    Type       string                   `json:"type"`
    Attributes publishRequestAttributes `json:"attributes"`
}

type publishRequestAttributes struct {
    AssetPaths []string `json:"assetPaths"`
}

// PublishJobResponse is the JSON:API response from start publish/unpublish
// and from the status endpoints.
type PublishJobResponse struct {
    Data  PublishJobData  `json:"data"`
    Links PublishLinks    `json:"links,omitempty"`
}

// PublishJobData holds the publish/unpublish job fields.
type PublishJobData struct {
    Type       string               `json:"type"`
    ID         string               `json:"id"`
    Attributes PublishJobAttributes `json:"attributes"`
}

// PublishJobAttributes holds the status and progress fields.
type PublishJobAttributes struct {
    JobState        string   `json:"jobState"`
    JobStatusDetail interface{} `json:"jobStatusDetail,omitempty"`
    StartedBy       string   `json:"startedBy,omitempty"`
    StartDate       string   `json:"startDate,omitempty"`
    TotalCount      int      `json:"totalCount,omitempty"`
    ProcessedCount  int      `json:"processedCount,omitempty"`
    AssetPaths      []string `json:"assetPaths,omitempty"`
}

// PublishLinks holds the self and status link URLs.
// Note: known bug ICAI-41690 - these links may use http:// instead of https://.
// Always construct status URLs manually from the job ID.
type PublishLinks struct {
    Self   string `json:"self,omitempty"`
    Status string `json:"status,omitempty"`
}

// PublishIsTerminal returns true when the jobState is a terminal state.
func PublishIsTerminal(jobState string) bool {
    switch jobState {
    case "COMPLETED", "FAILED", "ERROR":
        return true
    }
    return false
}

// PublishIsInProgress returns true when the jobState means still running.
func PublishIsInProgress(jobState string) bool {
    return jobState == "NOT_STARTED" || jobState == "PROCESSING"
}

const publishMaxBatchSize = 199

// StartPublish submits one batch of CAI asset paths for publishing.
// caiURL is the CAI-specific base URL (e.g. https://na1.ai.dm-us.informaticacloud.com).
// If caiURL is empty, c.CAIURL() is used (auto-detected from login response).
func (c *Client) StartPublish(ctx context.Context, caiURL string, assetPaths []string) (*PublishJobResponse, error) {
    return c.startPublishOp(ctx, caiURL, "publish", assetPaths)
}

// StartUnpublish submits one batch of CAI asset paths for unpublishing.
func (c *Client) StartUnpublish(ctx context.Context, caiURL string, assetPaths []string) (*PublishJobResponse, error) {
    return c.startPublishOp(ctx, caiURL, "unpublish", assetPaths)
}

func (c *Client) startPublishOp(ctx context.Context, caiURL, opType string, assetPaths []string) (*PublishJobResponse, error) {
    base := caiURL
    if base == "" {
        base = c.CAIURL()
    }
    if base == "" {
        return nil, fmt.Errorf("CAI URL not configured; set caiUrl in profile config, IICS_CAI_URL env var, or --cai-url flag")
    }
    url := strings.TrimRight(base, "/") + "/active-bpel/asset/v1/" + opType
    req := publishRequest{
        Data: publishRequestData{
            Type: opType,
            Attributes: publishRequestAttributes{AssetPaths: assetPaths},
        },
    }
    var resp PublishJobResponse
    if err := c.doCAIJSON(ctx, http.MethodPost, url, req, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

// GetPublishStatus retrieves the current status of a publish job.
// Note: Do NOT use the status URL from the response (ICAI-41690 http:// bug).
// Set full=true to retrieve the full job object including asset list.
func (c *Client) GetPublishStatus(ctx context.Context, caiURL, publishID string, full bool) (*PublishJobResponse, error) {
    return c.getPublishOpStatus(ctx, caiURL, "publish", publishID, full)
}

// GetUnpublishStatus retrieves the current status of an unpublish job.
func (c *Client) GetUnpublishStatus(ctx context.Context, caiURL, publishID string, full bool) (*PublishJobResponse, error) {
    return c.getPublishOpStatus(ctx, caiURL, "unpublish", publishID, full)
}

func (c *Client) getPublishOpStatus(ctx context.Context, caiURL, opType, jobID string, full bool) (*PublishJobResponse, error) {
    base := caiURL
    if base == "" {
        base = c.CAIURL()
    }
    if base == "" {
        return nil, fmt.Errorf("CAI URL not configured")
    }
    var u string
    if full {
        u = fmt.Sprintf("%s/active-bpel/asset/v1/%s/%s", strings.TrimRight(base, "/"), opType, jobID)
    } else {
        u = fmt.Sprintf("%s/active-bpel/asset/v1/%s/%s/Status", strings.TrimRight(base, "/"), opType, jobID)
    }
    var resp PublishJobResponse
    if err := c.doCAIJSON(ctx, http.MethodGet, u, nil, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}
```

### Step 4 - Tests (`internal/client/publish_test.go`)

Write tests for `StartPublish`, `StartUnpublish`, `GetPublishStatus`, `GetUnpublishStatus`:

- Assert HTTP method and full URL path in the handler
- For start operations: decode request body; verify `data.type` and `assetPaths`
- For start operations: return HTTP 202 with job ID and initial `jobState: "NOT_STARTED"`
- For status operations: verify URL includes job ID and `/Status`; verify `full=true` omits `/Status`
- Assert at least one field on each returned struct
- Test `PublishIsTerminal` and `PublishIsInProgress` helper functions

Use `newTestClient(handler)` from `internal/client/objects_test.go`.

### Step 5 - Asset input helpers

Add a helper in `internal/client/publish.go` (or a separate `internal/client/publish_input.go`)
that converts objects-list output to asset paths:

```go
// AssetPathFromObject builds a CAI asset path from an Object's path and type fields.
// The object type must be one of the supported CAI asset types.
// Returns an error if the type is not supported.
func AssetPathFromObject(obj Object) (string, error) {
    switch obj.Type {
    case "PROCESS", "AI_SERVICE_CONNECTOR", "AI_CONNECTION", "DTEMPLATE", "GUIDE", "PROCESS_OBJECT":
        return fmt.Sprintf("Explore/%s.%s.xml", obj.Path, obj.Type), nil
    default:
        return "", fmt.Errorf("asset type %q is not publishable", obj.Type)
    }
}

// SplitIntoBatches splits a slice of asset paths into batches of at most batchSize.
func SplitIntoBatches(paths []string, batchSize int) [][]string {
    var batches [][]string
    for len(paths) > 0 {
        end := batchSize
        if end > len(paths) {
            end = len(paths)
        }
        batches = append(batches, paths[:end])
        paths = paths[end:]
    }
    return batches
}
```

### Step 6 - Command layer (`cmd/publish.go`)

The command reads asset paths from three sources (priority order):

1. `--asset` repeatable string flag
2. `--from-file <file>` - one path per line (`.txt`), JSON array (`[...]`), or CSV with
   `path` and `type` columns (output of `iics objects list -o csv`); uses `AssetPathFromObject`
   for CSV conversion
3. stdin (when `--from-file` is absent and stdin is not a terminal) - same format detection
   as export's `readArtifactsFromStdin`

#### `publish start`

Flags:

- `--asset` ([]string, repeatable) - explicit asset path(s)
- `--from-file` (string) - asset list file
- `--cai-url` (string) - CAI base URL override
- `--name` (string) - optional job label for verbose output

Behavior: resolve inputs → auto-batch if > 199 → call `c.StartPublish` for each batch →
print batch job IDs. On multi-batch, print each batch ID as it completes.

#### `publish status`

Flags:

- `--id` (string, required) - publish job ID
- `--cai-url` (string) - CAI base URL override
- `--full` (bool, default false) - fetch full job object including asset list

Behavior: call `c.GetPublishStatus`, print using `getFormatter()`.

#### `publish run`

Flags (all matching `import run` naming):

- `--asset` ([]string, repeatable) - explicit asset path(s)
- `--from-file` (string) - asset list file; omit to read from stdin
- `--cai-url` (string) - CAI base URL override
- `--name` (string) - optional job label
- `--polling-interval` (int, default 10) - seconds between status polls
- `--max-wait-time` (int, default 300) - max seconds to wait before timeout
- `--detailed-polling` (bool) - print totalCount/processedCount on each poll

Behavior mirrors `import run`:

1. Resolve asset paths from inputs; if zero paths, return error
2. If `verbose`: print `[HH:MM:SS] Publishing N assets in B batch(es)...`
3. For each batch:
   a. `c.StartPublish(ctx, caiURL, batch)` - print job ID
   b. Poll with `c.GetPublishStatus(ctx, caiURL, id, false)` at interval
   c. If `verbose`: print `[HH:MM:SS] Status: PROCESSING (X/Y processed) elapsed: Xs`
   d. If `detailed-polling`: print totalCount/processedCount each poll
   e. On timeout: return error with last state
   f. On `FAILED`: return error
4. Print final summary using `getFormatter()` - table with ID, state, totalCount, processedCount

#### `cmd/unpublish.go`

Identical structure to `cmd/publish.go` but:

- Uses `c.StartUnpublish` and `c.GetUnpublishStatus`
- Verb text in verbose output says "Unpublishing" not "Publishing"
- Command name `unpublish`, short description updated

Factor out shared logic (input reading, batch loop) into unexported helpers in
`cmd/publish.go` that both files can call.

### Step 7 - Register in `cmd/root.go`

1. In `getClient()`, pass `WithCAIURL` when the profile has a configured CAI URL:

   ```go
   if p.CaiURL != "" {
       opts = append(opts, client.WithCAIURL(p.CaiURL))
   }
   ```

2. In `init()`:

   ```go
   rootCmd.AddCommand(newPublishCmd())
   rootCmd.AddCommand(newUnpublishCmd())
   ```

### Step 8 - Documentation

Create `docs/documentation/publish.md` and `docs/documentation/unpublish.md` following
the project template. Each page must include:

- Synopsis
- Subcommands table (`start`, `status`, `run`)
- Flags table per subcommand (Type, Default, Description, Required)
- Output columns table for `status` and `run`
- Note explaining CAI URL auto-detection and the optional `caiUrl` profile config field
- Note about the 199-asset batch limit and auto-batching
- Supported asset path suffixes table
- Usage examples including stdin piping from `iics objects list`

### Step 9 - Update `README.md`

Add two rows to the Commands table:

```markdown
| [publish](docs/documentation/publish.md) | | `start`, `status`, `run` | Publish CAI assets to the runtime |
| [unpublish](docs/documentation/unpublish.md) | | `start`, `status`, `run` | Unpublish CAI assets from the runtime |
```

### Step 10 - Verify

```bash
/opt/local/bin/go build ./...
/opt/local/bin/go test ./...
/opt/local/bin/go vet ./...
golangci-lint run ./...
iics publish run --help
iics unpublish run --help
```

All must pass with zero new errors.

---

## Output Columns

### `publish status` / `unpublish status` / final output of `run`

Flatten the nested JSON:API structure into a display struct before calling `f.Format`:

| Header    | Source field                       | Width | Notes                              |
| --------- | ---------------------------------- | ----- | ---------------------------------- |
| ID        | `data.id`                          | 24    | job ID                             |
| TYPE      | `data.type`                        | 12    | "publish" or "unpublish"           |
| STATE     | `data.attributes.jobState`         | 12    | NOT_STARTED, PROCESSING, COMPLETED |
| TOTAL     | `data.attributes.totalCount`       | 8     | total assets in job                |
| PROCESSED | `data.attributes.processedCount`   | 10    | assets processed so far            |
| STARTED   | `data.attributes.startDate`        | 22    | ISO timestamp                      |
| BY        | `data.attributes.startedBy`        | 20    | username                           |

Because the output formatter uses JSON-tag dot notation and the JSON:API nesting is deep,
create a flat display struct in `cmd/publish.go` and populate it from `PublishJobResponse`
before calling `f.Format`. Do not attempt dot-notation through the `data.attributes.*` path.

---

## Acceptance Criteria

- [ ] `iics publish run --asset <path>` submits, polls, and prints final status
- [ ] `iics publish run --from-file <file>` works for txt, JSON, and CSV formats
- [ ] `iics objects list -o csv | iics publish run` works end-to-end
- [ ] Auto-batching: >199 assets splits into sequential batches automatically
- [ ] `iics publish status --id <id>` prints current job state
- [ ] `iics unpublish run` and `iics unpublish status` work identically with unpublish endpoints
- [ ] CAI URL auto-detected from login response - no config required in normal use
- [ ] `caiUrl` in profile config and `IICS_CAI_URL` env var override auto-detection
- [ ] `--cai-url` flag also overrides auto-detection
- [ ] `--polling-interval`, `--max-wait-time`, `--detailed-polling` all work correctly
- [ ] Verbose `[HH:MM:SS]` progress printed to stdout when `--verbose` set
- [ ] FAILED terminal state returns a non-zero exit code (via error return from `RunE`)
- [ ] Timeout returns a non-zero exit code with clear message
- [ ] Output works in table, json, and csv formats
- [ ] All new client methods have passing tests
- [ ] `go build ./...` succeeds with no errors
- [ ] `go test ./...` passes with no failures
- [ ] `go vet ./...` reports no issues
- [ ] `golangci-lint run ./...` reports no new issues
- [ ] No unrelated code was modified
- [ ] Two-layer rule respected: no API logic in `cmd/`, no Cobra in `internal/client/`
- [ ] `docs/documentation/publish.md` and `docs/documentation/unpublish.md` created
- [ ] `README.md` commands table updated with both commands

---

## Do NOT

- Refactor, reformat, or add comments to code outside the CR scope
- Follow the `links.self` or `links.status` URL from the response (ICAI-41690 - use `http://`)
- Use `os.Exit()` - return errors from `RunE`
- Use tablewriter v0.x API
- Hard-code the CAI base URL - always derive from login response or config/flag
- Exceed the 199-asset batch limit in a single API call
- Add features beyond what is explicitly described above
- Modify `internal/config/pods.go` or `internal/output/`
- Add `Co-Authored-By` trailers to commit messages

---

## Implementation Notes (filled in by Claude during implementation)

-
