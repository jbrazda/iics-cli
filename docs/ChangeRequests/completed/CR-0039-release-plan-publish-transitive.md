# CR-0039: `release plan` publish-only transitive inclusion and location excludes

## CR Type

- [x] Enhancement to existing command

## Problem

`release plan` generates a `package` file and a `publish` file per target,
both derived from the same asset list. `--include-found-transitive` controls
transitive-dependency inclusion for both files identically: by default,
transitive dependencies already found in the target are dropped from both;
explicit and missing-transitive assets are always kept.

There is no way to give the publish file a different transitive policy than
the package file, and no way to exclude specific assets from the publish file
only. Operators running `release plan` manually sometimes want the publish
file to include found-transitive assets (e.g. to force a republish) while
leaving the package file's deployment scope untouched, and to exclude a few
specific assets from publish by location pattern.

## Proposed Solution

Add three CLI-only flags to `release plan` (no manifest schema change - this
is for manual, ad hoc runs, not automated pipeline manifests):

- `--publish-include-found-transitive` (bool, default `false`): include
  transitive dependencies already found in the target in the generated
  publish file. Package file is unaffected; independent of
  `--include-found-transitive`.
- `--publish-exclude-regex <pattern>`: regex matched against an asset's
  `location`, excluding matches from the publish file only.
- `--publish-exclude-file <path>`: path to a regex-patterns file (one per
  line, `#` comments), same format as the existing `release.LoadExcludePatterns`
  used by the manifest's `excludeFile`. Combined (OR) with
  `--publish-exclude-regex` when both are given.

Excludes always apply to the publish file regardless of whether
`--publish-include-found-transitive` is set. Default behavior (no new flags)
is byte-identical to today - this is strictly additive and non-breaking.

`iics publish` (the execution command) is untouched; it already has
`--from-file`/stdin to control exactly what gets published.

## Implementation

- `internal/release/plan.go` - new `ExcludeAssetsByLocation(assets, patterns)`
  helper (`ApplyPolicies` refactored to reuse it, no behavior change).
- `internal/release/build.go` - `PlanOptions.PublishIncludeFoundTransitive`,
  `PlanOptions.PublishExcludePatterns`; `BuildPlan` computes publish assets
  from the unfiltered per-plan asset list when the flag is set, then applies
  the exclude patterns.
- `cmd/release.go` - three new flags on `newReleasePlanCmd`, wired into
  `PlanOptions` at both the full-deployment and selective/tag-mode call
  sites.
- Tests: `internal/release/plan_test.go`, `internal/release/build_test.go`.
- Docs: `docs/documentation/release.md`.
