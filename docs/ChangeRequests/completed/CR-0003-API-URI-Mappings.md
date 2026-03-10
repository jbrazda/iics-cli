# CR: Update all request bindings Request URL mappings

## Scope

- Files to change: All Command that invoke API

## Problem

Setup of the endpoints are inconsistent
Some Endpoints do not match the actual URL

## Desired Change

Update request URI to match this pattern using the BaseAPIPathV3 or BaseAPIPathV2 respectively instead of hard-codded base uri foe each api Version

```go
c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/widgets/%s", BaseAPIPathV3, id), nil, &resp); err != nil 
```

Check each API Endpoint against the documentation and correct URI if needed:

- [API Documentation (PDF)](https://docs.informatica.com/content/dam/source/GUID-B/GUID-BA8ED0B4-AB32-46B1-A7B8-358FAB844D6B/64/en/IICS_November2025_(REST-API)Reference_en.pdf)
- [API Documentation (html)](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/preface.html)

## Acceptance Criteria

- [x] Make sure all tests pass
- [x] Fix any lint issues

## Changes Made

- Replaced all hardcoded `"public/core/v3/..."` path strings with `BaseAPIPathV3` constant across all client files
- Fixed `agents.go`: `StartAgentService`/`StopAgentService` were incorrectly using `BaseAPIPathV3`; corrected to `BaseAPIPathV2`
- Fixed `connections.go`: moved from V2 `api/v2/connection` (singular) to V3 `public/core/v3/connections` (plural), confirmed by existing tests
- Updated `docs/CLAUDE.md` resources table to reflect that `connections` and `lookup` are V3 resources
