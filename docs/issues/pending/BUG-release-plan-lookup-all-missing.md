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
- A batch that fails with `V3API_LookupError_012` uses the error message,
  which lists every missing object, to mark those assets missing; the rest of
  the batch is looked up again. When the message cannot be matched to the
  batch, the batch is split in half and each half retried. A single-item
  batch that fails is `missing`.
- A single-item lookup that returns any object is `found` (some types return
  a different path format). Assets a successful multi-item batch does not
  match by path and type are rechecked one by one.
- HTTP 429 is retried up to 3 times, waiting as `Retry-After` or
  `ratelimit-reset` says, or 2, 4 and 8 seconds plus jitter without them. A
  lookup still rate limited is `missing` with an `APIRateLimitError` warning,
  so the asset stays in the package.
- Read-only requests (GET, Lookup) are resent up to 2 times when the
  connection drops (`EOF` or connection reset).
- Other errors stay `unknown` with a warning.
- One Info log line per target reports found, missing, unknown and
  rate-limited counts and the number of Lookup calls.

## Tests

`internal/release/validate_test.go` uses a fake Lookup API that fails a batch
with `V3API_LookupError_012` like the real one: all exist, one missing among
50, all missing, mixed types with Connections, HTTP 500, chunking, and the
missing-transitive filter. The shared fake in `build_test.go` now behaves the
same way.

## Verification against real orgs

Full plan for project `ClaimCenter_GW` from `dev` to `qa` and `tst` (761
assets: 546 explicit, 215 transitive), compared with v0.5.3:

| Run           | qa found/missing/unknown | tst found/missing/unknown | Lookup calls (qa/tst) |
| ------------- | ------------------------ | ------------------------- | --------------------- |
| v0.5.3        | 660 / 100 / 1            | 394 / 366 / 1             | one per asset         |
| v0.6.2 (fix)  | 661 / 100 / 0            | 395 / 366 / 0             | 20 / 25               |

Every asset has the same status in both runs except the two v0.5.3 reported
`unknown`, which the fix reports `found`. The package and publish files are
identical (qa 556/263, tst 575/268 rows).

The same comparison on `ClaimCenter_Conversion` (1311 assets): v0.5.3 took
21 min 15 s, the fix 4 min 54 s (target checks: about 17.5 min against 36 s,
21 Lookup calls). Package rows and publish files are the same. v0.5.3 left 3
assets `unknown` in its status table after dropped connections (`EOF`,
connection reset), which the fix resolved. Neither run hit a rate limit.
