---
id: BUG-0012
title: package create --exclude-found-transitive produces import-failing package in v0.4.10
status: pending
priority: high
affects: cmd/package.go, docs/documentation/package.md
---

# BUG-0012: package create --exclude-found-transitive produces import-failing package in v0.4.10

## Summary

`iics package create` with `--manifest-file` and `--exclude-found-transitive`
regressed in v0.4.10. Packages created for `ZZ_TEST_CLI` fail to deploy via
`iics import run` in target orgs where v0.4.8-generated packages deploy
successfully.

## Observed behavior

1. The same project flow that succeeds with v0.4.8 fails with v0.4.10.
2. The failure occurs on packages generated with:
   - `--manifest-file .../tag_build.package.csv`
   - `--exclude-found-transitive`
3. Import failure is correlated with v0.4.10 metadata generation behavior.

## Expected behavior

1. Package import should succeed for the same project and manifest flow used in
   v0.4.8.
2. `--exclude-found-transitive` should exclude transitive payload assets that
   are already found in target orgs without producing invalid import metadata.

## Root cause

v0.4.10 changed selective metadata behavior to regenerate
`exportMetadata.v2.json` from selected-only objects for
`--exclude-found-transitive`, instead of preserving the full metadata graph used
previously. This can remove metadata dependencies required for import
compatibility.

## Fix implemented

The fix now aligns dependency ownership and package behavior:

- Build regenerated `exportMetadata.v2.json` from selected assets plus
  dependency closure and required container metadata for
  `--exclude-found-transitive`.
- Remove hidden payload parsing from `package create`.
- Retain required CDI `Connection` and `AgentGroup` references for selected CDI
  carrier assets using source `exportMetadata.v2.json` `objectRefs`.
- Make release planning output authoritative by default with missing-transitive
  filtering enabled (opt-out flag available).
- Update command documentation and changelog to match corrected behavior.
