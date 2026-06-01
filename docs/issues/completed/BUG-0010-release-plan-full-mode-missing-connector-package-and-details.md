---
id: BUG-0010
title: release plan full mode omits connector package and verbose composition details
status: new
priority: high
affects: cmd/release.go
---

# BUG-0010: release plan full mode omits connector package and verbose composition details

## Summary

`iics release plan` in full deployment mode does not generate
`target/iics/import/connectors.package.csv` and does not print the same detailed
dependency and package composition information that tag-based mode prints with
verbose logging.

## Observed Behavior

Running:

```bash
iics release plan \
   --manifest /home/jbrazda/git/ado/natl/ZZ_TEST_CLI/target/iics/import/conf/release_manifest.properties \
   --output-root target/iics/import \
   --add-missing-transitive-deps \
   --full-package-config /home/jbrazda/git/ado/natl/ZZ_TEST_CLI/conf/full_build.package.csv \
   --verbose --log-file
```

in full deployment mode writes per-environment package and publish files, but
does not write `target/iics/import/connectors.package.csv`. Verbose output also
lacks the dependency status table and per-environment package and publish type
composition tables that tag-based mode prints.

## Expected Behavior

- Full mode writes `connectors.package.<ext>` when either connectors or
  connections are included in the release options.
- Full mode keeps per-environment `full_build.package.<ext>` and
  `publish_assets.<ext>` generation unchanged.
- Full mode prints dependency status and package composition details with
  `--verbose`, matching tag-based mode.
- `--log-file` continues to append the Markdown release plan report.

## Root Cause

The full-mode branch in `cmd/release.go` resolves and writes per-environment
full package and publish files, but it does not accumulate connector assets into
the global connector package. It also skips the verbose dependency status and
type-count tables that are rendered in the tag-based branch.

## Fix

Mirror the tag-based planning behavior in the full-mode branch by collecting the
target-specific connector asset union, writing `connectors.package.<ext>` when
connector or connection inclusion is requested, and rendering the same verbose
dependency and composition tables.
