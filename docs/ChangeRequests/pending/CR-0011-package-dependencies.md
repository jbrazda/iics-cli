# CR-0011: Implement `package dependencies` Subcommand

## CR Type

- [x] **Enhancement** - add `dependencies` as a new subcommand of the existing `package` command

---

## Problem

After exporting a package from IICS and before publishing it to a target org, operators need to
know what assets the package depends on. Without this information:

- A publish operation may fail silently because required connections, connectors, or process
  objects are absent from the target org
- There is no way to audit the full dependency graph of a package without manually inspecting
  `exportMetadata.v2.json`
- CI/CD pipelines have no automated way to validate a target org before attempting publish

Currently `iics package` only provides `expand` and `create` subcommands for local file
manipulation. There is no command to resolve the dependency graph or validate it against a
target org.

---

## Desired Change

Add `iics package dependencies` as a new subcommand. The command operates in two modes:

### Default mode (no `--publish` flag)

Resolves and lists ALL transitive dependencies of every object in the package, including
non-publishable types (Folder, Project, Connection, AgentGroup, MTT, etc.). This is useful for
general inspection and auditing of the full dependency graph. The `--target-profile` flag is
optional; when provided, each dependency is validated against the target org and missing assets
are flagged in the output.

### Publish mode (`--publish` flag)

Filters the dependency list to publishable types only:
`AI_SERVICE_CONNECTOR`, `AI_CONNECTION`, `PROCESS`, `GUIDE`, `TASKFLOW`.

The `--target-profile` flag is **required** in this mode. The output is fully compatible with
`iics publish run --from-file`, enabling the common CI pattern. The `-o mermaid` output format
renders the dependency graph as a Mermaid `graph TD` diagram for review in GitHub, GitLab, or
VS Code (no extra Go dependencies - pure text generation).

```bash
iics package dependencies -f pkg.zip --publish --target-profile prod -o csv | iics publish run
```

### Example usage

```bash
# Inspect full dependency graph from a ZIP file
iics package dependencies --file mypackage.zip

# Inspect from an expanded workspace
iics package dependencies --workspace ./unpacked

# Publish mode: resolve publishable deps, validate target, pipe to publish
iics package dependencies -f pkg.zip --publish --target-profile prod -o csv | iics publish run

# Exclude system connections and filter to a specific project
iics package dependencies -f pkg.zip --exclude '^/SYS' --filter 'ZZ_TEST_CLI'

# Exclude specific connection type and project prefix
iics package dependencies -f pkg.zip --exclude 'Salesforce\.AI_CONNECTION|DAS'

# Render dependency graph as Mermaid diagram
iics package dependencies -f pkg.zip -o mermaid

# Render publish-only graph for review before deploying
iics package dependencies -f pkg.zip --publish --target-profile prod -o mermaid
```

---

## Scope

### Files to CREATE

None.

### Files to MODIFY

```text
cmd/package.go                  # add dependencies subcommand + local structs + helpers
internal/config/config.go       # add ResolveTargetProfile() with IICS_TARGET_* env vars
docs/documentation/package.md   # add dependencies subcommand section
README.md                       # update package command entry to mention dependencies
```

### Files to READ (context only - do NOT modify)

```text
docs/CLAUDE.md                                              # mandatory: patterns and rules
cmd/package.go                                              # existing expand/create subcommands
internal/client/lookup.go                                   # Lookup(), LookupObject, LookupResult
internal/client/objects.go                                  # GetObjectDependencies(), ObjectReference
internal/client/publish.go                                  # publishable types, AssetPathFromObject
internal/config/config.go                                   # ResolveProfile() as template
cmd/root.go                                                 # getClient(), getFormatter()
testdata/imports/ZZ_TEST_CLI_Unpacked/exportMetadata.v2.json  # structure reference
```

### Forbidden (do NOT touch)

```text
internal/output/
internal/config/pods.go
cmd/root.go    # do NOT modify root.go
```

---

## API Details

This command uses two existing client methods. No new client files are required.

### Lookup by GUID (resolve external dependencies)

| Field        | Value                                   |
| ------------ | --------------------------------------- |
| HTTP method  | POST                                    |
| Endpoint     | `public/core/v3/lookup`                 |
| Session      | `INFA-SESSION-ID` (V3)                  |

Request body: `{"objects": [{"id": "<guid>"}, ...]}`

Response: `{"objects": [{"id": "...", "path": "...", "type": "...", ...}]}`

Already implemented in `internal/client/lookup.go` as `client.Lookup()`.

### Lookup by path+type (target org validation)

Same endpoint, request body: `{"objects": [{"path": "<path>", "type": "<type>"}, ...]}`

Already implemented in `internal/client/lookup.go` as `client.Lookup()`.

### Get object references (transitive deps of external objects)

| Field       | Value                                                   |
| ----------- | ------------------------------------------------------- |
| HTTP method | GET                                                     |
| Endpoint    | `public/core/v3/objects/<id>/references?refType=uses`   |

Already implemented in `internal/client/objects.go` as `client.GetObjectDependencies()`.

---

## Implementation Instructions

### Step 0 - Config layer (`internal/config/config.go`)

Add `ResolveTargetProfile()` next to `ResolveProfile()`. It follows the same pattern but
applies `IICS_TARGET_*` environment variable overrides:

```go
// ResolveTargetProfile returns the named profile with IICS_TARGET_* env var overrides applied.
// Used by `package dependencies --target-profile` for cross-org dependency validation.
func (c *Config) ResolveTargetProfile(profileName string) (*Profile, error) {
    if profileName == "" {
        return nil, fmt.Errorf("target profile name is required")
    }
    p, ok := c.Profiles[profileName]
    if !ok {
        return nil, fmt.Errorf("profile %q not found in config", profileName)
    }
    resolved := p // copy

    if v := os.Getenv("IICS_TARGET_USERNAME"); v != "" {
        resolved.Username = v
    }
    if v := os.Getenv("IICS_TARGET_PASSWORD"); v != "" {
        resolved.Password = v
    }
    if v := os.Getenv("IICS_TARGET_REGION"); v != "" {
        resolved.Region = v
    }
    if v := os.Getenv("IICS_TARGET_LOGIN_URL"); v != "" {
        resolved.LoginURL = v
    }

    if resolved.Username == "" {
        return nil, fmt.Errorf("username not configured for target profile %q; set in config file or IICS_TARGET_USERNAME env var", profileName)
    }
    if resolved.Password == "" {
        return nil, fmt.Errorf("password not configured for target profile %q; set in config file or IICS_TARGET_PASSWORD env var", profileName)
    }
    return &resolved, nil
}
```

### Step 1 - Local structs (`cmd/package.go`)

Add unexported structs for parsing `exportMetadata.v2.json` and for output. These are not
API structs - they live only in `cmd/package.go`.

```go
// exportMetadata represents the exportMetadata.v2.json file in an IICS export package.
type exportMetadata struct {
    Name            string           `json:"name"`
    SourceOrgID     string           `json:"sourceOrgId"`
    SourceOrgName   string           `json:"sourceOrgName"`
    ExportedObjects []exportedObject `json:"exportedObjects"`
}

type exportedObject struct {
    ObjectGUID string          `json:"objectGuid"`
    ObjectName string          `json:"objectName"`
    ObjectType string          `json:"objectType"`
    Path       string          `json:"path"`
    Metadata   exportedObjMeta `json:"metadata"`
}

type exportedObjMeta struct {
    ObjectRefs []string `json:"objectRefs"`
}

// dependencyItem is one row of output from the dependencies command.
type dependencyItem struct {
    Path         string `json:"path"`
    Type         string `json:"type"`
    Source       string `json:"source"`
    TargetStatus string `json:"targetStatus,omitempty"`
    Warning      string `json:"warning,omitempty"`
}

// dependencyEdge represents a directed "depends on" relationship between two items.
// Both FromKey and ToKey are matchKeys (path + "." + type).
type dependencyEdge struct {
    FromKey string
    ToKey   string
}
```

### Step 2 - Helper: `readExportMetadata`

```go
func readExportMetadata(filePath, workspace string) (*exportMetadata, error) {
    if workspace != "" {
        data, err := os.ReadFile(filepath.Join(workspace, "exportMetadata.v2.json"))
        if err != nil {
            return nil, fmt.Errorf("reading exportMetadata.v2.json: %w", err)
        }
        var meta exportMetadata
        return &meta, json.Unmarshal(data, &meta)
    }
    r, err := zip.OpenReader(filePath)
    if err != nil {
        return nil, fmt.Errorf("opening package file: %w", err)
    }
    defer r.Close()
    for _, f := range r.File {
        if f.Name == "exportMetadata.v2.json" {
            rc, err := f.Open()
            if err != nil {
                return nil, fmt.Errorf("opening exportMetadata.v2.json in ZIP: %w", err)
            }
            defer rc.Close()
            data, err := io.ReadAll(rc)
            if err != nil {
                return nil, fmt.Errorf("reading exportMetadata.v2.json from ZIP: %w", err)
            }
            var meta exportMetadata
            return &meta, json.Unmarshal(data, &meta)
        }
    }
    return nil, fmt.Errorf("exportMetadata.v2.json not found in package")
}
```

`io` must be added to the imports in `cmd/package.go` if not already present.

### Step 3 - Helper: `resolveDependencies`

The function signature:

```go
func resolveDependencies(
    ctx context.Context,
    c *client.Client,
    meta *exportMetadata,
    excludePattern string,
    publishOnly bool,
) ([]dependencyItem, []dependencyEdge, error)
```

**Publishable types set**: `DTEMPLATE` and `PROCESS_OBJECT` appear in `AssetPathFromObject`
in `internal/client/publish.go` but are NOT actually publishable via the IICS publish API
despite being mentioned in some Informatica documentation. The correct set is:

```go
var publishableTypes = map[string]bool{
    "AI_SERVICE_CONNECTOR": true,
    "AI_CONNECTION":        true,
    "PROCESS":              true,
    "GUIDE":                true,
    "TASKFLOW":             true,
}
```

**Match key construction**: given an object's `path` field (e.g., `/Explore/ZZ_TEST_CLI/Connections`)
and `objectName` (e.g., `TestServiceConnection1`) and `objectType` (e.g., `AI_CONNECTION`):

```go
matchKey = path + "/" + objectName + "." + objectType
// result: "/Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1.AI_CONNECTION"
```

For `LookupResult` (external deps), `path` already includes the full path to the object
(e.g., `Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1`), so:

```go
matchKey = result.Path + "." + result.Type
```

**Full path for output**: the `path` field in the output `dependencyItem` must be the full path
to the object including the name, without leading slash, without type suffix - e.g.,
`Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1`. This is the format consumed by
`iics publish run --from-file` (via `AssetPathFromObject`).

For package objects:

```go
fullPath = strings.TrimPrefix(obj.Path, "/") + "/" + obj.ObjectName
// path="/Explore/ZZ_TEST_CLI/Connections" + name="TestServiceConnection1"
// result: "Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1"
```

**BFS algorithm**:

```text
1. Build pkgMap: map[string]*exportedObject keyed by objectGuid
2. Initialize:
   - visited: map[string]bool (GUID -> seen)
   - result: map[string]dependencyItem (fullPath+"."+type -> item, to deduplicate)
   - queue: []string of GUIDs

3. Seed the queue with all objectGuids from exportedObjects

4. While queue not empty:
   a. Pop guid; if visited, skip; mark visited
   b. If guid in pkgMap (package object):
      - Compute matchKey and fullPath
      - If excludePattern != "" and regex matches matchKey: mark visited, skip (no recursion)
      - If not publishOnly OR publishableTypes[obj.ObjectType]:
          add to result as source="package"
      - For each objectRef: record potential edge (matchKey, childMatchKey) in a raw edge list
      - Enqueue all objectRefs not yet visited
   c. Else (external guid - not in package):
      - Collect into externalBatch (process in bulk after draining immediate queue)

5. After each queue pass, batch-lookup all collected externalBatch GUIDs via client.Lookup()
   - For each LookupResult found:
     - Compute matchKey (result.Path + "." + result.Type)
     - If excludePattern matches: skip
     - If not publishOnly OR publishableTypes[result.Type]:
         add to result as source="external"
     - Call client.GetObjectDependencies(ctx, result.ID, "uses", 200, 0)
     - Collect all Uses refs (path+type pairs)
   - Batch-lookup those path+type pairs via client.Lookup([{Path, Type}, ...])
   - Enqueue any resulting GUIDs not yet visited

6. Repeat until queue is empty

7. Deduplicate result map to slice; sort by typePriority then path
8. Build edge list: for each (parentKey, childKey) pair recorded during BFS,
   include the edge only when BOTH parentKey and childKey exist in the final result map
9. Return sorted slice + filtered edges
```

Type priority for sorting (lower number = earlier in output):

```go
var typePriority = map[string]int{
    "AI_SERVICE_CONNECTOR": 1,
    "AI_CONNECTION":        2,
    "PROCESS":              3,
    "GUIDE":                4,
    "TASKFLOW":             5,
}
```

Types not in the priority map (non-publishable types shown in default mode) sort after all
publishable types, ordered by type name then path.

### Step 3b - Helper: `renderMermaid`

`renderMermaid` writes a Mermaid `graph TD` diagram to `w`. It requires no external libraries.

```go
func renderMermaid(items []dependencyItem, edges []dependencyEdge, w io.Writer) error
```

Implementation rules:

- Assign short stable node IDs: `n0`, `n1`, ... indexed by position in `items`
- Build a `map[string]string` from matchKey (`item.Path + "." + item.Type`) to node ID
- Display name: `filepath.Base(item.Path)` (last path segment, e.g. `TestServiceConnection1`)
- Node label format: `nN["TYPE: DisplayName"]` - escape any `"` in label as `'`
- Write `graph TD` header followed by one line per node definition, then one line per edge
- Edges: `fromID --> toID` - only emit edges where both endpoints exist in the node map
- Isolated nodes (no edges) are still written as standalone node lines
- Nodes with `targetStatus="missing"` get a distinct style class:
  `nN:::missing` and append `classDef missing fill:#ffcccc,stroke:#cc0000` at the end

Example output for the test package (publish mode):

```text
graph TD
    n0["AI_SERVICE_CONNECTOR: TestServiceConnector1"]
    n1["AI_CONNECTION: TestServiceConnection1"]
    n2["PROCESS: MockupEcho"]
    n3["GUIDE: Test Conversion Utility"]
    n1 --> n0
classDef missing fill:#ffcccc,stroke:#cc0000
```

Add `"path/filepath"` to imports in `cmd/package.go` if not already present (it already is,
since `readExportMetadata` uses `filepath.Join`).

### Step 4 - Helper: `validateTargetDependencies`

```go
func validateTargetDependencies(
    ctx context.Context,
    targetProfileName string,
    deps []dependencyItem,
) error
```

Implementation:

1. Load config via `config.Load("")`
2. Call `cfg.ResolveTargetProfile(targetProfileName)`
3. Get login URL via `targetProfile.GetLoginURL()`
4. Create `client.NewClient(loginURL, targetProfile.Username, targetProfile.Password)`
5. Build `[]client.LookupObject` - one `{Path: item.Path, Type: item.Type}` per dep
6. Batch all in a single `targetClient.Lookup(ctx, lookupObjects)` call
7. Build a set of found paths: `map[string]bool` keyed by `result.Path + "." + result.Type`
8. For each dep NOT in the found set: set `deps[i].TargetStatus = "missing"` and
   `deps[i].Warning = "asset not found in target org"`
9. For deps in the found set: set `deps[i].TargetStatus = "found"`

Use a pointer-to-slice or index-based loop so modifications persist.

### Step 5 - Helper: `applyFilter`

```go
func applyFilter(deps []dependencyItem, pattern string) ([]dependencyItem, error) {
    re, err := regexp.Compile(pattern)
    if err != nil {
        return nil, fmt.Errorf("invalid --filter regex: %w", err)
    }
    var filtered []dependencyItem
    for _, d := range deps {
        if re.MatchString(d.Path + "." + d.Type) {
            filtered = append(filtered, d)
        }
    }
    return filtered, nil
}
```

Same match key pattern as `--exclude`. Also compile the `--exclude` pattern once at the
start of `resolveDependencies` and return an error if the pattern is invalid.

`regexp` must be added to the imports in `cmd/package.go`.

### Step 6 - Command definition (`cmd/package.go`)

```go
func newPackageDependenciesCmd() *cobra.Command {
    var (
        file          string
        workspace     string
        publishMode   bool
        excludeRegex  string
        filterRegex   string
        targetProfile string
    )
    cmd := &cobra.Command{
        Use:   "dependencies",
        Short: "Resolve transitive dependencies from an IICS export package",
        Long: `Resolve and list all transitive dependencies of objects in an IICS export package.

In default mode, all dependency types are listed (Folder, Project, Connection, AgentGroup,
publishable types, etc.) for auditing purposes.

When --publish is set, only publishable types are listed (AI_SERVICE_CONNECTOR, AI_CONNECTION,
PROCESS, GUIDE, TASKFLOW) and --target-profile is required. The
output is compatible with 'iics publish run --from-file' for direct piping.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Validate mutually exclusive input flags
            if file == "" && workspace == "" {
                return fmt.Errorf("one of --file or --workspace is required")
            }
            if file != "" && workspace != "" {
                return fmt.Errorf("--file and --workspace are mutually exclusive")
            }
            // --target-profile is required in publish mode
            if publishMode && targetProfile == "" {
                return fmt.Errorf("--target-profile is required when --publish is set")
            }

            meta, err := readExportMetadata(file, workspace)
            if err != nil {
                return err
            }

            c, err := getClient(cmd)
            if err != nil {
                return err
            }

            deps, edges, err := resolveDependencies(cmd.Context(), c, meta, excludeRegex, publishMode)
            if err != nil {
                return err
            }

            if targetProfile != "" {
                if err := validateTargetDependencies(cmd.Context(), targetProfile, deps); err != nil {
                    return err
                }
            }

            if filterRegex != "" {
                deps, err = applyFilter(deps, filterRegex)
                if err != nil {
                    return err
                }
            }

            // Mermaid output bypasses the standard formatter.
            // outputFmt is the package-level var set by the global --output/-o flag.
            if outputFmt == "mermaid" {
                return renderMermaid(deps, edges, cmd.OutOrStdout())
            }

            f, err := getFormatter()
            if err != nil {
                return err
            }

            columns := []output.Column{
                {Header: "PATH",   Field: "path",   Width: 70},
                {Header: "TYPE",   Field: "type",   Width: 22},
                {Header: "SOURCE", Field: "source", Width: 10},
            }
            if targetProfile != "" {
                columns = append(columns,
                    output.Column{Header: "TARGET",  Field: "targetStatus", Width: 10},
                    output.Column{Header: "WARNING", Field: "warning",      Width: 50},
                )
            }
            return f.Format(deps, columns)
        },
    }
    cmd.Flags().StringVarP(&file,          "file",           "f", "", "Path to IICS export ZIP package (mutually exclusive with --workspace)")
    cmd.Flags().StringVarP(&workspace,     "workspace",      "w", "", "Path to expanded workspace directory (mutually exclusive with --file)")
    cmd.Flags().BoolVar(&publishMode,      "publish",        false, "Resolve publishable types only; requires --target-profile")
    cmd.Flags().StringVarP(&excludeRegex,  "exclude",        "e", "", "Regex matched against path.type to exclude assets from resolution")
    cmd.Flags().StringVar(&filterRegex,    "filter",         "",  "Regex matched against path.type to filter output")
    cmd.Flags().StringVarP(&targetProfile, "target-profile", "t", "", "Profile name for target org validation (required with --publish)")
    return cmd
}
```

Register in `newPackageCmd()`:

```go
cmd.AddCommand(newPackageDependenciesCmd())
```

Add the following import to `cmd/package.go`:

```go
"context"
"io"
"regexp"

"github.com/jbrazda/iics-cli/internal/client"
"github.com/jbrazda/iics-cli/internal/config"
"github.com/jbrazda/iics-cli/internal/output"
```

### Step 7 - Documentation (`docs/documentation/package.md`)

Add a `## package dependencies` section with:

- Purpose and modes description
- Flags table with all six flags (Type, Default, Description, Required column)
- Note that `--file` and `--workspace` are mutually exclusive
- Note that `--target-profile` is required when `--publish` is set
- Output columns table (PATH, TYPE, SOURCE, TARGET, WARNING)
- Environment variables table for target org overrides (`IICS_TARGET_USERNAME`, `IICS_TARGET_PASSWORD`, `IICS_TARGET_LOGIN_URL`, `IICS_TARGET_REGION`)
- Regex match key explanation with example
- Note that `-o mermaid` bypasses the standard formatter and generates a Mermaid diagram
- Usage examples including:
  - Basic dependency inspection
  - Mermaid graph output (`-o mermaid`)
  - Publish mode with target validation
  - Piping to `publish run`
  - Using `--exclude` to skip system objects
  - CI usage with `IICS_TARGET_*` env vars

Update the subcommands table at the top of the file to add the `dependencies` row.

### Step 8 - Update `README.md`

Update the package command row in the commands table to mention the new `dependencies`
subcommand.

### Step 9 - Verify

```bash
/opt/local/bin/go build ./...
/opt/local/bin/go vet ./...
golangci-lint run ./...
iics package dependencies --help
# Smoke test with testdata (no auth needed for metadata parsing):
iics package dependencies -w ./testdata/imports/ZZ_TEST_CLI_Unpacked
```

---

## Output Columns

### Default mode (no `--target-profile`)

| Header | Field  | Width | Notes                                                                           |
| ------ | ------ | ----- | ------------------------------------------------------------------------------- |
| PATH   | path   | 70    | Full object path, e.g. `Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1` |
| TYPE   | type   | 22    | Object type, e.g. `AI_CONNECTION`                                               |
| SOURCE | source | 10    | `package` (in the ZIP) or `external` (resolved via API)                         |

### With `--target-profile` (appended columns)

| Header  | Field        | Width | Notes                                        |
| ------- | ------------ | ----- | -------------------------------------------- |
| TARGET  | targetStatus | 10    | `found` or `missing`                         |
| WARNING | warning      | 50    | `asset not found in target org` when missing |

The `targetStatus` and `warning` fields are included in JSON and CSV output regardless of
`--target-profile` (they will simply be empty strings when target validation was not performed).

The `path` field is in the format expected by `iics publish run --from-file` when using CSV
or JSON input. Piping `-o csv` output to `iics publish run` is fully supported.

### `-o mermaid` graph output

Produces a Mermaid `graph TD` diagram. No additional Go library dependencies.

Each node represents one dependency item. Node ID is a short index (`n0`, `n1`, ...).
Node label format: `TYPE: DisplayName` where `DisplayName` is the last segment of `path`.

Nodes with `targetStatus="missing"` are marked with `:::missing` and a red-fill `classDef`
is appended so the missing assets stand out when rendered.

Edges represent "depends on" relationships derived from `objectRefs` in the package metadata
and from `GetObjectDependencies` for external items. Only edges where both endpoints are in
the filtered result set are emitted.

Example:

```text
graph TD
    n0["AI_SERVICE_CONNECTOR: TestServiceConnector1"]
    n1["AI_CONNECTION: TestServiceConnection1"]
    n2["PROCESS: MockupEcho"]
    n3["GUIDE: Test Conversion Utility"]
    n1 --> n0
classDef missing fill:#ffcccc,stroke:#cc0000
```

---

## Acceptance Criteria

- [ ] `iics package dependencies --file <zip>` resolves and prints all dependencies
- [ ] `iics package dependencies --workspace <dir>` works from an expanded directory
- [ ] `--file` and `--workspace` are mutually exclusive; error returned if both are provided
- [ ] Error returned if neither `--file` nor `--workspace` is provided
- [ ] Default mode lists ALL types including non-publishable (Folder, Connection, AgentGroup, etc.)
- [ ] `--publish` mode limits output to publishable types only
- [ ] `--target-profile` is required when `--publish` is set; error returned if absent
- [ ] `--target-profile` is optional in default mode
- [ ] `--exclude` regex removes matched assets from resolution and stops recursion through them
- [ ] `--filter` regex filters the final output without affecting dependency resolution
- [ ] Regex is matched against `<path>/<objectName>.<objectType>` (e.g., `/Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1.AI_CONNECTION`)
- [ ] Invalid regex in `--exclude` or `--filter` returns a descriptive error
- [ ] External GUIDs (not in package metadata) are resolved via `client.Lookup()` API
- [ ] Transitive deps of external objects are resolved via `GetObjectDependencies()` + `Lookup()`
- [ ] Results are deduplicated (same GUID referenced from multiple objects appears once)
- [ ] Results are sorted by type priority then path
- [ ] Target org validation uses `client.Lookup()` with `{Path, Type}` (not GUIDs)
- [ ] Missing target deps show `targetStatus="missing"` and `warning` field
- [ ] Found target deps show `targetStatus="found"`
- [ ] `IICS_TARGET_USERNAME`, `IICS_TARGET_PASSWORD`, `IICS_TARGET_LOGIN_URL`, `IICS_TARGET_REGION` override target profile values
- [ ] `iics package dependencies -f pkg.zip --publish --target-profile prod -o csv | iics publish run` pipes correctly
- [ ] Output works in table, json, and csv formats
- [ ] `-o mermaid` produces valid Mermaid `graph TD` syntax
- [ ] Mermaid nodes use `TYPE: DisplayName` label format
- [ ] Isolated nodes (no edges) still appear in Mermaid output
- [ ] Mermaid edges are emitted only when both endpoints are in the result set
- [ ] Nodes with `targetStatus="missing"` are styled with `:::missing` class in Mermaid output
- [ ] No new Go module dependencies added for Mermaid support
- [ ] `go build ./...` succeeds with no errors
- [ ] `go vet ./...` reports no new issues
- [ ] `golangci-lint run ./...` reports no new issues
- [ ] `docs/documentation/package.md` updated with `dependencies` subcommand section
- [ ] Subcommands table in `docs/documentation/package.md` updated
- [ ] `README.md` updated
- [ ] No unrelated code was modified
- [ ] Two-layer rule respected: no API logic in `cmd/` beyond what was here before; no Cobra in `internal/`

---

## Do NOT

- Refactor, reformat, or add comments to code outside the CR scope
- Add new external Go module dependencies
- Add a new `internal/client/` file - use existing `client.Lookup()` and `client.GetObjectDependencies()`
- Use `os.Exit()` - return errors from `RunE`
- Modify `internal/output/`, `internal/config/pods.go`, or `cmd/root.go`
- Add `Co-Authored-By` trailers to commit messages
- Use em dashes in documentation
- Add features beyond what is explicitly described above
- Exceed batch sizes or make one-request-per-item API calls when batching is possible
- Modify `internal/output/` to add Mermaid support - handle it directly in `RunE` by checking
  `outputFmt == "mermaid"` before calling `getFormatter()`
- Add new Go module dependencies for Mermaid or ASCII graph rendering

---

## Implementation Notes (filled in by Claude during implementation)

-
