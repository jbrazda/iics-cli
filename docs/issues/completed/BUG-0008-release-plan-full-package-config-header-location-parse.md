---
id: BUG-0008
title: release plan fails parsing full package config header row as location
status: completed
priority: high
affects: internal/release/plan.go
---

# BUG-0008: release plan fails parsing full package config header row as location

## Summary

`iics release plan` fails when `--full-package-config` points to a valid CSV that
includes a header row. The parser attempts to parse the literal header value
`LOCATION` as an asset location and exits with an invalid location error.

## Observed Behavior

Running:

```bash
iics release plan \
  --manifest /home/jbrazda/git/ado/natl/target/iics/import/conf/release_manifest.properties \
  --output-root target/iics/import \
  --add-missing-transitive-deps \
  --verbose \
  --full-package-config /home/jbrazda/git/ado/natl/ZZ_TEST_CLI/conf/full_build.package.csv
```

fails with:

```text
Error: parsing full package config /home/jbrazda/git/ado/natl/ZZ_TEST_CLI/conf/full_build.package.csv: parsing location "LOCATION": invalid location "LOCATION": expected Explore/path.TYPE or SYS/path.TYPE format
```

## Expected Behavior

- CSV header rows are recognized and skipped.
- Valid full package entries such as `Explore/ZZ_TEST_CLI.Project` are accepted.
- `release plan` resolves nested assets for project pointers and generates
  environment specific build configurations.

## Root Cause

The full package config parser is not skipping or normalizing the header row
before location parsing, so `LOCATION` is treated as a data row and passed into
location validation.

## Impact

- `release plan` cannot run with valid header based full package CSV files.
- Nested asset resolution and environment specific build config generation are
  blocked for project pointer inputs.

## Reproduction Data

- Manifest: `/home/jbrazda/git/ado/natl/target/iics/import/conf/release_manifest.properties`
- Full package config: `/home/jbrazda/git/ado/natl/ZZ_TEST_CLI/conf/full_build.package.csv`
- Error timestamp: `2026-05-29 09:04:57`
