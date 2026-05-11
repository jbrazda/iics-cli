---
id: BUG-0007
title: location prefix incorrectly uses Explore for Connection and AgentGroup assets
status: pending
priority: medium
affects: cmd/objects.go, cmd/package.go, internal/client/objects.go, internal/release/plan.go
---

# BUG-0007: location prefix incorrectly uses Explore for Connection and AgentGroup assets

## Summary

Location strings were being built with a hardcoded `Explore/` prefix even for
asset types that live under the `SYS/` root in expanded package structures.

This impacted outputs from:

- `objects list`
- `objects dependencies`
- `package dependencies`
- release planning asset resolution

## Observed behavior

- `Connection` and `AgentGroup` rows were emitted as `Explore/<path>.<type>`.
- Filtering and reporting workflows that rely on location roots could not
  distinguish these system rooted assets correctly.

## Expected behavior

- `Connection` and `AgentGroup` use `SYS/<path>.<type>`.
- Other asset types continue to use `Explore/<path>.<type>`.

## Root cause

Multiple code paths independently constructed location values with
`"Explore/" + path + "." + type`, without type-aware root handling.

## Fix implemented

- Added a shared location builder in `internal/client/location.go`.
- Replaced hardcoded location construction in object, package, and release
  dependency flows.
- Updated parsing and documentation to support both `Explore/` and `SYS/`
  location roots.

