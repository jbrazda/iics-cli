# BUG: import run object status table shows empty ID, NAME, and TYPE columns

## Symptoms

When running `iics import run` with `--expand` or `--detailed-polling`, the "Imported Objects"
table prints rows with blank ID, NAME, and TYPE columns. Only STATUS and MESSAGE are populated:

```
Imported Objects:
┌────┬──────┬──────┬────────────┬───────────────────────────┐
│ ID │ NAME │ TYPE │   STATUS   │          MESSAGE          │
├────┼──────┼──────┼────────────┼───────────────────────────┤
│    │      │      │ SUCCESSFUL │ Overwrite existing object │
│    │      │      │ SUCCESSFUL │ Reuse existing object     │
└────┴──────┴──────┴────────────┴───────────────────────────┘
```

Expected output (illustrative):

```
Imported Objects:
┌────────────────────────┬──────────────────────────────┬───────────────────────────────┬──────────────────────┬────────────┬───────────────────────────┐
│       SOURCE ID        │        SOURCE PATH           │         SOURCE NAME           │    TARGET NAME       │   STATE    │          MESSAGE          │
├────────────────────────┼──────────────────────────────┼───────────────────────────────┼──────────────────────┼────────────┼───────────────────────────┤
│ 0KfTorzrNwihXfV38FliA2 │ /ZZ_TEST_CLI/Connections     │ TestServiceConnection1        │ TestServiceConnection1 │ SUCCESSFUL │ Overwrite existing object │
│ 0pXCTRhtjrcdwOCIS5P7S5 │ /ZZ_TEST_CLI/Mappings        │ m_Test_Git                    │ m_Test_Git           │ SUCCESSFUL │ Overwrite existing object │
└────────────────────────┴──────────────────────────────┴───────────────────────────────┴──────────────────────┴────────────┴───────────────────────────┘
```

## Root Cause

The `ImportJob.Objects` field is typed as `[]JobObject`, which is a flat struct shared with the
export API:

```go
// internal/client/export.go
type JobObject struct {
    ID     string    `json:"id"`
    Name   string    `json:"name"`
    Type   string    `json:"type"`
    Status JobStatus `json:"status"`
}
```

However, the import status API (`GET /public/core/v3/import/{jobId}?expand=objects`) returns
a different nested structure for each object:

```json
{
  "sourceObject": {
    "id": "0KfTorzrNwihXfV38FliA2",
    "name": "TestServiceConnection1",
    "path": "/ZZ_TEST_CLI/Connections",
    "type": "AI_CONNECTION",
    "description": "Test connection"
  },
  "targetObject": {
    "id": null,
    "name": "TestServiceConnection1",
    "path": "/ZZ_TEST_CLI/Connections",
    "type": "AI_CONNECTION",
    "description": null,
    "status": null
  },
  "status": {
    "state": "SUCCESSFUL",
    "message": "Overwrite existing object"
  }
}
```

Because no top-level `id`, `name`, or `type` fields exist on the object entry, JSON
unmarshalling leaves those fields empty. The `status` field does unmarshal correctly because
its JSON tag matches.

The column field paths used in `printImportObjects` (`"id"`, `"name"`, `"type"`) are therefore
wrong for import objects.

## Affected Files

- `internal/client/imports.go` - `ImportJob.Objects` uses wrong type; needs `ImportJobObject`
  struct matching the actual API response
- `cmd/import_.go` - `printImportObjects` uses wrong column fields; needs new columns and
  `--object-status-fields` flag; needs duplicate-name warning

## Fix

### 1. Add new structs in `internal/client/imports.go`

Add `ImportObjectRef`, `ImportJobObject`, and `FlatImportObject` structs:

```go
// ImportObjectRef is a source or target object reference within an import job object entry.
type ImportObjectRef struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Path        string `json:"path"`
    Type        string `json:"type"`
    Description string `json:"description"`
}

// ImportJobObject is a single entry in the ImportJob.Objects list.
// The import status API returns a nested sourceObject/targetObject structure,
// unlike the export API which returns flat JobObject entries.
type ImportJobObject struct {
    SourceObject ImportObjectRef `json:"sourceObject"`
    TargetObject ImportObjectRef `json:"targetObject"`
    Status       JobStatus       `json:"status"`
}

// FlatImportObject is a display-ready flattened view of an ImportJobObject.
// All fields from sourceObject, targetObject, and status are exposed so that
// any combination can be selected via --object-status-fields.
type FlatImportObject struct {
    // sourceObject fields
    SourceID          string `json:"sourceId"`
    SourceName        string `json:"sourceName"`
    SourcePath        string `json:"sourcePath"`
    SourceType        string `json:"sourceType"`
    SourceDescription string `json:"sourceDescription"`
    // targetObject fields
    TargetID          string `json:"targetId"`
    TargetName        string `json:"targetName"`
    TargetPath        string `json:"targetPath"`
    TargetType        string `json:"targetType"`
    TargetDescription string `json:"targetDescription"`
    // status fields
    State   string `json:"state"`
    Message string `json:"message"`
}

// FlattenImportObjects converts a slice of ImportJobObject to a flat display slice.
func FlattenImportObjects(objects []ImportJobObject) []FlatImportObject {
    flat := make([]FlatImportObject, len(objects))
    for i, o := range objects {
        flat[i] = FlatImportObject{
            SourceID:          o.SourceObject.ID,
            SourceName:        o.SourceObject.Name,
            SourcePath:        o.SourceObject.Path,
            SourceType:        o.SourceObject.Type,
            SourceDescription: o.SourceObject.Description,
            TargetID:          o.TargetObject.ID,
            TargetName:        o.TargetObject.Name,
            TargetPath:        o.TargetObject.Path,
            TargetType:        o.TargetObject.Type,
            TargetDescription: o.TargetObject.Description,
            State:             o.Status.State,
            Message:           o.Status.Message,
        }
    }
    return flat
}
```

Change `ImportJob.Objects` to use the new type:

```go
type ImportJob struct {
    ...
    Objects []ImportJobObject `json:"objects,omitempty"`
}
```

### 2. Update `printImportObjects` in `cmd/import_.go`

- Accept a `fields []string` parameter (derived from `--object-status-fields` flag, falling
  back to the default column set).
- Flatten the objects with `client.FlattenImportObjects` before formatting.
- Print a warning line to stderr for any row where `sourceName != targetName` (indicates
  a potential rename or duplicate).

Default columns (field names match `FlatImportObject` JSON tags):

```
sourceId, sourcePath, sourceName, targetName, state, message
```

### 3. Add `--object-status-fields` flag to `import run` and `import status`

```go
cmd.Flags().StringVar(&objectStatusFields, "object-status-fields",
    "sourceId,sourcePath,sourceName,targetName,state,message",
    "comma-separated list of FlatImportObject fields to display in the object status table")
```

All available field names (JSON tags of `FlatImportObject`):

| Field name          | Source                          |
| ------------------- | ------------------------------- |
| `sourceId`          | sourceObject.id                 |
| `sourceName`        | sourceObject.name               |
| `sourcePath`        | sourceObject.path               |
| `sourceType`        | sourceObject.type               |
| `sourceDescription` | sourceObject.description        |
| `targetId`          | targetObject.id                 |
| `targetName`        | targetObject.name               |
| `targetPath`        | targetObject.path               |
| `targetType`        | targetObject.type               |
| `targetDescription` | targetObject.description        |
| `state`             | status.state                    |
| `message`           | status.message                  |

### 4. Update tests in `internal/client/imports_test.go`

Add a test that unmarshals a response containing nested `sourceObject`/`targetObject` entries
and verifies that `FlattenImportObjects` produces the correct flat records.

## Acceptance Criteria

- [ ] `import run --expand` prints a table with non-empty SOURCE ID, SOURCE PATH, SOURCE NAME,
  TARGET NAME, STATE, MESSAGE columns for every imported object
- [ ] `import run --detailed-polling` prints the same table on each poll interval when objects
  are returned
- [ ] `import status --id <id> --expand` prints the same object table
- [ ] A warning line is printed to stderr for each object where `sourceName != targetName`
- [ ] `--object-status-fields` overrides the displayed columns (comma-separated JSON tag names)
- [ ] Existing import tests still pass
- [ ] `testdata/imports/importStatusResponse.json` is used as the basis for the new test
