---
id: CR-0020
title: dependency commands output and file export alignment
status: new
priority: high
affects: cmd/package.go, cmd/objects.go, docs/documentation/package.md, docs/documentation/objects.md, README.md
---

## CR-0020: dependency commands output and file export alignment

## Background

Build and deploy CI/CD pipelines need dependency output that can be consumed
directly by package and publish workflows without additional transformation.

Current `objects dependencies` and `package dependencies` output does not expose
dependency origin (`explicit` or `transitive`), uses inconsistent location
labels, and does not provide consistent output-file options.

## Problem

1. There is no `dependency` field indicating whether an asset is directly
   specified (`explicit`) or discovered by traversal (`transitive`).
2. Output labels and fields are inconsistent for downstream tools.
3. Filtering should target normalized location (`Explore/path.type`) to exclude
   assets from explicit packaging and publishing sets.
4. `--output-file`, `--output-file-format`, and `--output-file-fields` should
   be available consistently on both dependency commands.

## Desired change

Implement the following across both commands:

1. Add `dependency` output field with values `explicit` and `transitive`.
2. Add normalized `location` output field with format `Explore/path.type`.
3. Add or align `--filter` behavior to match regex against `location`.
4. Rename table and report display header from path-type variants to
   `Location`.
5. Add output file flags:
   - `--output-file`
   - `--output-file-format`
   - `--output-file-fields`

## Confirmed rules

### `package dependencies`

- `explicit`: assets present in package or export metadata
- `transitive`: assets resolved externally via API during traversal

### `objects dependencies`

- `explicit`: seed object IDs provided by `--id` or stdin input list
- `transitive`: dependency assets discovered through recursive traversal

### `Location` field

- Use `Explore/path.type` format in output fields and filtering.

### Column order

- Default output leads with `location`, then `dependency`, then existing
  status and warning fields when present.

## Scope

### Files to modify

- `cmd/package.go`
- `cmd/objects.go`
- `docs/documentation/package.md`
- `docs/documentation/objects.md`
- `README.md` (if command option summary needs update)

### Out of scope

- Changes to dependency traversal depth or graph logic beyond output behavior
- Changes to publish parser behavior

## Implementation details

### 1. Data shaping

Both commands should emit rows containing:

- `location`
- `dependency`
- existing identity and validation fields (`path`, `type`, `status`,
  `warning`, profile report fields as applicable)

`location` must be computed and persisted in output rows as:

```text
Explore/<path>.<type>
```

### 2. Filtering

- `--filter` regex runs against `location`
- filtering applies to final output rows, not traversal mechanics
- invalid regex returns a descriptive user error

### 3. Table and report headers

- Standard output uses `Location`
- Report output replaces path-type header variants with `Location`

### 4. Output file support

Add to both dependency commands:

- `--output-file`: write output to a file
- `--output-file-format`: table/json/csv/yaml
- `--output-file-fields`: comma-separated field list for file output selection

Behavior should mirror existing output-file patterns used in other commands.

## Documentation updates required

Update command documentation with:

1. New or updated flags (`--filter`, output-file trio)
2. `dependency` and `location` field definitions
3. `Location` header naming in table and report examples
4. Examples showing location-based filtering and file export usage

## Acceptance criteria

- [ ] `package dependencies` output includes `dependency` and `location`
- [ ] `objects dependencies` output includes `dependency` and `location`
- [ ] `dependency` values follow confirmed command-specific classification rules
- [ ] `location` always uses `Explore/path.type` format
- [ ] `--filter` matches against `location` in both commands
- [ ] Standard and report headers use `Location` naming
- [ ] Both commands support output-file flags
- [ ] Documentation for both commands reflects new fields and usage
- [ ] Output remains compatible with package and publish pipeline inputs
