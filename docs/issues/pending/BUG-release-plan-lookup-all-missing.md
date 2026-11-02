# Bug: `release plan` marks all non-Connection assets missing

## Problem

Since v0.5.5 (PR #27), `iics release plan` reports every non-Connection asset
as `missing` for a target when at least one asset in the plan does not exist
in that target. Only Connection assets show `found`. The missing-transitive
filter then keeps transitive assets that already exist in the target, so
package and publish lists are inflated (865 instead of 849 in the
ClaimCenter_Conversion pipeline).

## Root cause

`validateAssetsExistInTarget` in `internal/release/plan.go` sends all
non-Connection assets in one Lookup call. The V3 Lookup API fails the whole
batch with `V3API_LookupError_012` when any item is missing, and the code
treated that error as "every asset missing". The test fake returned partial
results instead of the batch error, so the tests did not catch it.

## Fix

- Lookups are sent in chunks of 200 (`lookupChunkSize`).
- A batch that fails with `V3API_LookupError_012` is split in half and each
  half retried. A single-item batch that fails is `missing`.
- A single-item lookup that returns any object is `found` (some types return
  a different path format). Assets a successful multi-item batch does not
  match by path and type are rechecked one by one.
- Other errors stay `unknown` with a warning.
- One Info log line per target reports found, missing and unknown counts and
  the number of Lookup calls.

## Tests

`internal/release/validate_test.go` uses a fake Lookup API that fails a batch
with `V3API_LookupError_012` like the real one: all exist, one missing among
50, all missing, mixed types with Connections, HTTP 500, chunking, and the
missing-transitive filter. The shared fake in `build_test.go` now behaves the
same way.
