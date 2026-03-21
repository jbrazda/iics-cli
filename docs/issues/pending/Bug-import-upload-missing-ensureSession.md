# BUG: import run fails with "unsupported protocol scheme" when no cached session exists

## Symptoms

Running `import run` without a prior `iics login` (or after the 30-minute session cache expires)
fails immediately with:

```shell
$ ./iics import run --zip-file testdata/imports/ZZ_TEST_CLI.zip --verbose --conflict-resolution OVERWRITE
[21:20:47] Uploading testdata/imports/ZZ_TEST_CLI.zip...
Error: upload failed: Post "/public/core/v3/import/package": unsupported protocol scheme ""
```

The same command succeeds after running `iics login` first. This indicates the URL is being
constructed before the session (and therefore `baseAPIURL`) is initialized.

## Root Cause

`UploadImportPackage` in `internal/client/imports.go` constructs the full request URL before
calling `c.do()`:

```go
url := c.apiURL(fmt.Sprintf("%s/import/package", BaseAPIPathV3))
req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
...
resp, err := c.do(ctx, req)
```

`c.apiURL()` reads `c.baseAPIURL`, which is empty when no session has been established yet.
With an empty `baseAPIURL`, `apiURL()` returns a path-only string like
`/public/core/v3/import/package`, which `net/http` rejects with "unsupported protocol scheme".

All other HTTP helpers (`doJSON`, `doJSONWithQuery`, `doRaw`) call `c.ensureSession()` before
`c.apiURL()`, which triggers an auto-login if no session exists, populating `baseAPIURL` before
the URL is built. `UploadImportPackage` is the only method that skips this call.

## Affected Files

- `internal/client/imports.go` - `UploadImportPackage`

## Fix

Add `ensureSession()` before `apiURL()` in `UploadImportPackage`, matching the pattern used by
all other HTTP helpers:

```go
func (c *Client) UploadImportPackage(ctx context.Context, filename string, reader io.Reader) (*ImportUploadResponse, error) {
    if err := c.ensureSession(ctx); err != nil {
        return nil, err
    }
    // ... multipart body construction ...
    url := c.apiURL(fmt.Sprintf("%s/import/package", BaseAPIPathV3))
    ...
}
```

## Acceptance Criteria

- [ ] `import run` succeeds without a prior `iics login` (auto-logins using profile credentials)
- [ ] `import run` succeeds after the 30-minute session cache expires (auto-logins on expiry)
- [ ] Existing `TestUploadImportPackage` test still passes
- [ ] No other `UploadImportPackage` callers are broken
