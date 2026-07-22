---
id: BUG-0013
title: package create leaks unselected Mass Ingestion (.dat) assets past selective filtering
status: pending
priority: high
affects: internal/dependencies/checksum.go, cmd/package.go
---

# BUG-0013: package create leaks unselected Mass Ingestion (.dat) assets past selective filtering

## Summary

`iics package create --manifest-file ... --exclude-found-transitive` (the
tag-based selective packaging path) includes Mass Ingestion asset files
(`MI_TASK`, `MI_FILE_LISTENER`, etc.) in the output ZIP even when those assets
are not present anywhere in the selection manifest - not as `explicit`, not as
`transitive`, and not even as an excluded found-transitive row.

## Observed behavior

A tag-based build for tag `PSAR_AggLimits` ran:

```bash
iics package create \
  --source ".../target/transform/qa/src" \
  --target ".../NATL_ClaimCenter_GW_tag_PSAR_AggLimits_7f7c1f31.zip" \
  --force \
  --manifest-file "./target/iics/import/qa/tag_build.package.csv" \
  --name "NATL_ClaimCenter_GW_tag_PSAR_AggLimits_7f7c1f31" \
  --exclude-found-transitive \
  --status-target "qa" \
  --verbose
```

`tag_build.package.csv` contained exactly 5 rows, all `explicit`, all under
`ClaimCenter_GW/AggLimits_v1`:

```csv
LOCATION,TYPE,PATH,DEPENDENCY,STATUS (QA)
Explore/ClaimCenter_GW/AggLimits_v1/AggLimitsConnector.AI_CONNECTION,AI_CONNECTION,ClaimCenter_GW/AggLimits_v1/AggLimitsConnector,explicit,found
Explore/ClaimCenter_GW/AggLimits_v1/AggLimitsServiceConnector.AI_SERVICE_CONNECTOR,AI_SERVICE_CONNECTOR,ClaimCenter_GW/AggLimits_v1/AggLimitsServiceConnector,explicit,found
Explore/ClaimCenter_GW/AggLimits_v1/GetAggLimits_v1.PROCESS,PROCESS,ClaimCenter_GW/AggLimits_v1/GetAggLimits_v1,explicit,missing
Explore/ClaimCenter_GW/AggLimits_v1/MP-GetAggLimits-CAI-v1.PROCESS,PROCESS,ClaimCenter_GW/AggLimits_v1/MP-GetAggLimits-CAI-v1,explicit,missing
Explore/ClaimCenter_GW/AggLimits_v1/SP-GetAggLimits-CAI-v1.PROCESS,PROCESS,ClaimCenter_GW/AggLimits_v1/SP-GetAggLimits-CAI-v1,explicit,missing
```

Despite this, the produced package included ~40 unrelated
`*.MI_TASK.dat` / `*.MI_FILE_LISTENER.dat` files from sibling folders that
have no relationship to `AggLimits_v1` (`CCC_v1`, `Corvel_v1`, `ESubro`,
`LSS_v1`, `NCCI_V1`, `Optum_V1`, `SmartComm_v1`, `Verisk`,
`Connectors/LexisNexis`).

## Expected behavior

Only assets present in the selection manifest (plus their required container
hierarchy and reference closure, per existing selective packaging rules)
should be included in the package ZIP, regardless of asset type or file
extension.

## Root cause

`internal/dependencies/checksum.go` → `ObjectChecksumCandidates` only
generated `.xml`, `.zip`, and `.json` candidate file names for matching an
exported object to its file in the package payload. Mass Ingestion asset
types (`MI_TASK`, `MI_FILE_LISTENER`, `MI_SERVICE_CONNECTOR`, etc.) serialize
as `<Name>.<TYPE>.dat`, not `.xml`/`.json`.

`cmd/package.go` → `filterPackageFilesForSelection` uses
`ObjectChecksumCandidates` to determine whether a file in the package
corresponds to a tracked object. Files that cannot be matched to any object
fall through a fallback branch that keeps them unconditionally (this
fallback is intentional for package-level metadata files that are not
tracked as objects, e.g. `exportPackage.chksum`). Because `.dat` files never
matched, every Mass Ingestion asset file was treated as untracked package
metadata and always kept, completely bypassing tag/dependency selection.

The same missing `.dat` candidate also affected
`dependencies.IsObjectChecksumBacked`, used by the `package objects`
dependency-report command to compute `explicitInPackage`, causing Mass
Ingestion assets to be misreported as not checksum-backed even when their
`.dat` file was physically present in the package.

## Fix implemented

- Added `.dat` as a fourth generic candidate in `ObjectChecksumCandidates`
  (alongside the existing type-agnostic `.xml`/`.zip`/`.json` candidates),
  so Mass Ingestion assets are correctly matched and subjected to the same
  selective filtering as every other asset type.
- No changes were needed in `cmd/package.go` - both
  `filterPackageFilesForSelection` and the `package objects`
  `IsObjectChecksumBacked` call site automatically pick up correct behavior
  once the candidate list is complete.
- Added test coverage in `internal/dependencies/checksum_test.go` for
  `MI_TASK`/`MI_FILE_LISTENER` assets backed only by a `.dat` file.
