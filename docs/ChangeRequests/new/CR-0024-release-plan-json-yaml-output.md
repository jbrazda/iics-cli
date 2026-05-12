# CR-0024: Release Plan JSON and YAML Output Formats

## CR Type

- [ ] New resource
- [x] Enhancement to existing command

## Problem

The `iics release plan` command currently writes asset lists exclusively as CSV files. For each
deployment target it produces:

- `<output-root>/<env>/tag_build.package.csv` - assets to package
- `<output-root>/<env>/publish_assets.csv` - assets to publish
- `<output-root>/connectors.package.csv` - connector union (when `--include-connectors` is set)

CSV is sufficient for simple CI consumption but is limiting in several scenarios:

- JSON output is required when downstream tooling (Python scripts, jq pipelines, REST clients)
  needs structured access to all asset fields including `id`, `type`, and `dependency`.
- YAML output matches the idiom of the release manifest itself (`release_manifest.yaml`) and is
  easier to read and hand-edit when debugging or writing exclusion policies.
- CSV has no standard way to express nested or typed data; adding `id` to a CSV file is possible
  but the format is then less interoperable than JSON or YAML.

There is currently no flag to change the output format of the generated plan files.

## Proposed Solution

Add a `--plan-format` flag to `iics release plan`. The flag controls the serialization format of
the generated asset list files. The flag accepts the values `csv`, `json`, and `yaml`. The default
is `csv` to preserve backward compatibility.

The file names change with the format:

| Format | Package file                   | Publish file              | Connectors file              |
|--------|--------------------------------|---------------------------|------------------------------|
| `csv`  | `tag_build.package.csv`        | `publish_assets.csv`      | `connectors.package.csv`     |
| `json` | `tag_build.package.json`       | `publish_assets.json`     | `connectors.package.json`    |
| `yaml` | `tag_build.package.yaml`       | `publish_assets.yaml`     | `connectors.package.yaml`    |

Full-deployment mode also uses `--plan-format` for the generated `publish_assets.<ext>` file. The
full-package config file (copied from `--full-package-config`) keeps its original `.csv` extension
regardless of `--plan-format`, because it is a static file that is copied verbatim from the
repository, not generated from the asset list.

## CLI Changes

### New flag on `iics release plan`

```text
--plan-format string   output format for generated plan files: csv, json, yaml (default "csv")
```

Flag declaration in `newReleasePlanCmd()`:

```go
var planFormat string
cmd.Flags().StringVar(&planFormat, "plan-format", "csv", "output format for generated plan files: csv, json, yaml")
```

### Validated values

The flag must be validated before writing any files. Accepted values (case-insensitive): `csv`,
`json`, `yaml`. Any other value returns an error before any file I/O occurs.

## Output Format Details

### JSON

The JSON file is a JSON array. Each element is an object with the fields selected by
`--package-fields` or `--publish-fields`. Field names in the JSON object use lowercase keys
matching the field names already known to `WriteAssetsCSV` (`location`, `dependency`, `id`,
`type`, `path`).

Example for `tag_build.package.json` with `--package-fields location,dependency,type,path`:

```json
[
  {
    "location": "Explore/MyProject/MyProcess.PROCESS",
    "dependency": "explicit",
    "type": "PROCESS",
    "path": "MyProject/MyProcess"
  },
  {
    "location": "Explore/MyProject/SharedGuide.GUIDE",
    "dependency": "transitive",
    "type": "GUIDE",
    "path": "MyProject/SharedGuide"
  }
]
```

An empty asset list must produce a valid empty JSON array `[]`, not an empty file or `null`.

### YAML

The YAML file is a YAML sequence. Each element is a mapping with the same fields and lowercase
keys as the JSON output.

Example for `tag_build.package.yaml`:

```yaml
- location: Explore/MyProject/MyProcess.PROCESS
  dependency: explicit
  type: PROCESS
  path: MyProject/MyProcess
- location: Explore/MyProject/SharedGuide.GUIDE
  dependency: transitive
  type: GUIDE
  path: MyProject/SharedGuide
```

An empty asset list must produce a valid YAML empty sequence `[]\n`.

### Field selection

`--package-fields` and `--publish-fields` continue to control which fields appear in the output
for all three formats. The field order in JSON objects and YAML mappings must follow the order
specified in `--package-fields` / `--publish-fields`. Unknown field names produce an empty string
value, matching the existing CSV behavior.

## Acceptance Criteria

1. `iics release plan --plan-format csv` produces the same files as today with no behavioral
   change. This is the default when `--plan-format` is not specified.
2. `iics release plan --plan-format json` produces `.json` files in place of `.csv` files for all
   generated asset lists (package, publish, connectors).
3. `iics release plan --plan-format yaml` produces `.yaml` files in place of `.csv` files for all
   generated asset lists.
4. JSON output is a valid JSON array. An empty asset list produces `[]`.
5. YAML output is a valid YAML sequence. An empty asset list produces `[]\n`.
6. Field selection via `--package-fields` and `--publish-fields` is honored in all three formats.
7. An invalid `--plan-format` value returns an error before any files are written.
8. Full-deployment mode uses `--plan-format` for the generated `publish_assets.<ext>` file; the
   static package config file retains its original extension.
9. The `slog` log lines that report generated file paths reflect the actual file names including
   the correct extension.
10. Unit tests cover `WriteAssets` (or equivalent) for JSON and YAML output with at least one
    non-empty and one empty input case for each format.

## Files to Change

- `internal/release/plan.go` - add `WriteAssetsJSON` and `WriteAssetsYAML` functions alongside
  the existing `WriteAssetsCSV`. All three share the same field selection logic. The field-value
  extraction should be refactored into a small unexported helper (`assetFieldValue(a Asset, field string) string`)
  that all three writers call, replacing the `switch` block duplicated across them.
- `cmd/release.go` - add the `--plan-format` flag to `newReleasePlanCmd()`, validate its value,
  determine the output file extension from the format, and call the appropriate write function
  (`WriteAssetsCSV`, `WriteAssetsJSON`, or `WriteAssetsYAML`) when generating each plan file.
- `internal/release/plan_test.go` - add tests for `WriteAssetsJSON` and `WriteAssetsYAML`
  covering non-empty input, empty input, and field selection.

## Notes / Considerations

- Do not add a new import for a JSON library; `encoding/json` from the standard library is
  sufficient and is already available in the module. `gopkg.in/yaml.v3` is already a direct
  dependency (used in `manifest.go`), so no new dependencies are required.
- The `internal/output` package (`Formatter` interface) is intentionally not used here. The plan
  file writers operate on file paths, not `io.Writer` streams tied to terminal output. Keep the
  existing file-writing pattern: open a file, write content, close.
- Do not change the signature of `WriteAssetsCSV`. Add `WriteAssetsJSON` and `WriteAssetsYAML`
  with identical signatures so call sites in `cmd/release.go` can select the function with a
  simple variable or switch.
- The `--plan-format` flag only controls the format of the generated asset list files. It does
  not affect the `slog` log tables rendered to stderr, the `release_manifest.yaml`, or the
  static full-package config file that is copied in full-deployment mode.
- Validate `--plan-format` early in `RunE`, before the manifest is read or any API calls are
  made, so the user receives a clear error for a typo without incurring network cost.
- Consider lowercasing the flag value before validation so `CSV`, `Json`, and `YAML` are accepted
  in addition to the canonical lowercase forms.
