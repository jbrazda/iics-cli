---
id: BUG-0011
title: release plan target validation fails on missing connections and output is inconsistent
status: completed
priority: high
affects: cmd/release.go, internal/release/plan.go
---

# BUG-0011: release plan target validation fails on missing connections and output is inconsistent

## Summary

`iics release plan` can fail during target dependency validation when a missing connection lookup returns `APP_13436` (HTTP 403), does not consistently emit target-validation HTTP debug traces, prints dependency status rows in non-deterministic order, and needs connector package generation behavior aligned to manifest connector and connection include settings.

## Observed behavior

1. Missing `Connection` lookups may return `APP_13436` with HTTP 403 and message text indicating the connection does not exist, but planning treats this as a hard failure instead of a missing dependency.
2. `--debug` does not emit request and response traces for target-org validation calls used by release planning.
3. Dependency status tables are not sorted by `LOCATION`.
4. Connector package generation behavior for full and tag modes needs explicit consistency with `Include Connectors` and `Include Connections`.

## Expected behavior

1. `APP_13436` missing-connection responses are classified as `missing` and planning continues.
2. Target-org validation clients inherit global debug and verbose behavior.
3. Dependency status table rows are sorted by `LOCATION`.
4. `connectors.package.<ext>` is generated from dependency sets according to manifest include settings:
   - Include Connectors controls connector types.
   - Include Connections controls connection types.

## Root cause

- Connection existence checks only treated HTTP 404 as missing.
- Target validation clients were constructed without debug and verbose client options.
- Dependency status rendering used input order without location sorting.
- Connector package filtering was not explicitly policy-driven in one centralized helper.

## Fix implemented

- Added missing-connection classification for `APP_13436` 403 responses with connection-not-found signature.
- Propagated debug and verbose options into target-resolution clients.
- Sorted dependency status table input by location before rendering.
- Updated connector package selection to respect connector and connection include flags independently.
- Added release planning regression tests for missing classification, connector selection policy, and location sorting.
