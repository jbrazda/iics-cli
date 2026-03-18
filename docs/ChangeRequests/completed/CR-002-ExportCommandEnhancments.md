# CR: Export Command Enhancements

**Status:** PENDING - initial implementation complete; blocked on API validation issue (HTTP 400).
See [Bug-Export-InvalidRequest.md](../../issues/new/Bug-Export-InvalidRequest.md).

## Scope

- Files changed:
  - `cmd/export.go` - added `export start` and `export run` subcommands
  - `internal/client/export.go` - new structs, `StartExport`, `DownloadExportLog`, artifact parsing
  - `internal/client/export_test.go` - tests for all new client functions
  - `cmd/root.go` - added global `--debug` flag
  - `internal/client/client.go` - added `WithDebug` option; prints request body on API error

## Problem
IICS Export is used to backup, migrate, and version control IICS resources. It requires multiple
steps which are desired to be automated.

## References

- [IICS Export API documentation](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/exporting-objects.html)
- [IICS AssetManagement CLI](https://knowledge.informatica.com/s/article/DOC-18245?language=en_US)

## Desired Change

- `export run` - full pipeline: read input → resolve IDs → start job → poll → download ZIP
- `export start` - steps 1-5 only: resolve and start, return job ID

Export steps:

1. Read input from stdin or `--artifacts-file`
2. Extract IDs or lookup IDs via the [lookup API](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/lookup.html)
3. Construct the export request
4. In `--verbose` mode print the list of objects to be exported
5. Start export job
6. Poll for job status until complete or `--max-wait-time` exceeded
7. Download export archive to `--export-file-path`
8. If `--print-file-contents` or `--verbose`, print archive contents as a tree list

## Run Options

```text
--artifacts-file string
  File with artifacts list (.txt/.json/.yaml/.csv); omit to read from stdin.
  Format auto-detected from extension:
    .txt  - one Explore/path.TYPE per line (iics asset management CLI format)
    .json - objects list JSON format; minimal fields: id+type or location
    .yaml - objects list YAML format; same minimal fields
    .csv  - CSV with header row; columns: id, path, type, or location

--export-file-path string  (required for export run)
  Output ZIP file path (relative or absolute).

--name / -n string
  Export job name. Default: 'iics-cli(version:{version}) yyyy-mm-dd hh-mm-ss'

--max-wait-time int
  Maximum seconds to wait for completion. Default: 300.

--polling-interval int
  Seconds between status polls. Default: 10.

--print-file-contents
  Print ZIP file contents listing after download.

--expand-status
  Fetch and print the exported object list after completion (expand=objects).

--include-tags
  Adds includeTagInformation=true query param to the export POST.

--exclude-dependencies
  When set, each object is exported without dependent objects.

--download-export-log [path]
  Download the export job log. If path omitted, uses <export-file-path>.log.
```

Expected API request body (per-object `includeDependencies` only - no top-level field):

```json
POST /public/core/v3/export
{
    "name": "testJob1",
    "objects": [
        { "id": "l7bgB85m5oGiXObDxwnvK9", "includeDependencies": true },
        { "id": "1MW0GDAE1sFgnvWkvom7mK", "includeDependencies": true }
    ]
}
```

Command examples:

```text
./iics export run --artifacts-file ./config/export_list.txt --export-file-path ./backup.zip

./iics export run --artifacts-file ./config/export_list.json --export-file-path ./backup.zip --expand-status --verbose

./iics objects list -q "location==ZZ_TEST_CLI" -o csv | ./iics export run --export-file-path ./myexport.zip
```

## Implementation Notes

- `export start` and `export run` commands are implemented, compile, and pass unit tests.
- **Bug fixed (2026-03-10):** removed top-level `includeDependencies` from `ExportRequest`; the
  API does not accept that field and returns HTTP 400. Only per-object `includeDependencies` is valid.
- **Added `--debug` flag (2026-03-10):** global flag that prints the JSON request body to stderr
  on any API error, enabling faster diagnosis of future payload issues.
- Live end-to-end testing against ZZ_TEST_CLI still required to confirm the API accepts the current
  payload and that the full download flow works correctly.

## Acceptance Criteria

- [ ] New and existing tests pass
- [ ] `export run` successfully completes against ZZ_TEST_CLI (downloads a valid ZIP)
- [ ] `export start` returns a valid job ID
- [ ] No new dependencies added

## Do NOT

- Refactor unrelated code
- Change unrelated function signatures
