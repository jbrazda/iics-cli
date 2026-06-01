---
id: BUG-0009
title: release plan full mode writes empty publish assets
status: pending
priority: high
affects: cmd/release.go
---

# BUG-0009: release plan full mode writes empty publish assets

## Summary

`iics release plan` in full deployment mode resolves package assets from
`--full-package-config`, but writes each target environment's
`publish_assets.csv` with no asset rows.

## Observed Behavior

Running:

```bash
iics release plan \
  --manifest /home/jbrazda/git/ado/natl/ZZ_TEST_CLI/target/iics/import/conf/release_manifest.properties \
  --output-root target/iics/import \
  --add-missing-transitive-deps \
  --full-package-config /home/jbrazda/git/ado/natl/ZZ_TEST_CLI/conf/full_build.package.csv \
  --verbose
```

generates correct `full_build.package.csv` composition, but creates empty
`publish_assets.csv` files.

## Expected Behavior

- Full mode recursively expands included `Project` and `Folder` seed rows from
  the full package config.
- Full mode applies target-specific missing transitive dependency handling.
- Full mode writes `publish_assets.<ext>` from all publishable resolved assets
  for each environment, matching tag-based publish composition semantics.

## Root Cause

The full-mode branch in `cmd/release.go` passes `nil` to the asset writer for
`publish_assets.<ext>` instead of deriving publish rows from the resolved
environment asset set.

## Fix

Generate full-mode publish rows with `release.PublishAssets(envAssets)` after
target-specific filtering and write them with the configured publish fields.
