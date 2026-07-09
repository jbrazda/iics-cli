# BUG-0010: Publish fails with AvrEntryNotFoundFault when a batch mixes CAI assets and TaskFlows

## Symptom

`iics publish start|run` (and the equivalent `iics unpublish` commands) fail with
`AvrEntryNotFoundFault` when the input asset list contains a mix of CAI assets
(connections, connectors, processes, guides, process objects) and TaskFlow
assets in the same request.

## Root Cause

Publish/unpublish requests are batched purely by count
(`client.SplitIntoBatches`, max 199 assets per batch) without regard to asset
type. TaskFlow assets must be published/unpublished against the plain base
API URL (with `/saas` stripped), while all other CAI asset types are
published/unpublished against the CAI URL (see `publishBaseURL` /
`hasOnlyTaskflows`). Because `hasOnlyTaskflows` only routes a batch to the
TaskFlow-compatible URL when *every* asset in that batch is a TaskFlow, any
batch containing a mix of CAI assets and TaskFlows was sent entirely to the
CAI URL. The CAI backend does not recognize TaskFlow entries, so it returns
`AvrEntryNotFoundFault` for those items.

This was very likely to happen in practice because `iics publish run`/`start`
is commonly fed a single combined manifest (e.g. from
`iics objects dependencies --publish`) that legitimately mixes CAI assets and
TaskFlows in one invocation.

## Fix

Added `client.SplitPublishBatches`, which:

1. Partitions the (already dependency-sorted) input asset paths into a CAI
   group and a TaskFlow group via `client.PartitionAssetsByKind`.
2. Splits each group independently into batches of at most
   `client.PublishMaxBatchSize` (199) assets, so every batch is homogeneous.
3. Preserves the caller's intended cross-group ordering: if the first asset
   in the input is a TaskFlow (unpublish, reverse-dependency order), TaskFlow
   batches are emitted before CAI batches; otherwise (publish,
   forward-dependency order) CAI batches are emitted first.

`cmd/publish.go` (`newPublishStartCmd`, `runPublishOp`) and
`cmd/unpublish.go` (`newUnpublishStartCmd`) now call `SplitPublishBatches`
instead of `SplitIntoBatches`, so each batch is routed to the correct backend
via the existing `publishBaseURL`/`hasOnlyTaskflows` logic, which now always
sees a homogeneous batch.

Multi-batch reporting (`printMultiBatchSummary`, the manifest log via
`release.PublishBatchLog`) now includes a `GROUP` column (`CAI` or
`TASKFLOW`) so the split is visible in both the batch summary table and the
generated publish manifest log.

## Files Changed

- `internal/client/publish.go`: added `AssetBatchKind`, `AssetBatch`,
  `PartitionAssetsByKind`, `SplitPublishBatches`; refactored
  `hasOnlyTaskflows` to use new `isTaskflowPath` helper.
- `internal/client/publish_test.go`: added
  `TestPartitionAssetsByKind`, `TestSplitPublishBatchesGroupsCAIAndTaskflowSeparately`.
- `cmd/publish.go`: `newPublishStartCmd`, `runPublishOp`,
  `batchResult` (added `kind` field), `publishBatchesToManifestLog`,
  `batchSummaryRow` (added `Group` field), `printMultiBatchSummary`.
- `cmd/unpublish.go`: `newUnpublishStartCmd`.
- `internal/release/manifest_log.go`: `PublishBatchLog` (added `Group`
  field), `RenderPublishRunLog` (added GROUP column).

## Verification

- `go build ./...`, `go vet ./...`, `gofmt -s -l .`, `golangci-lint run ./...`
  all clean.
- `go test -race ./...` passes, including new tests covering mixed
  CAI/TaskFlow batch splitting and per-group batch size limits.
