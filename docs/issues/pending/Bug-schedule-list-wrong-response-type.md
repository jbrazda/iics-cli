# BUG: schedule list fails with JSON unmarshal error

---

## Symptoms

`iics schedule list` exits with error: `parsing response: json: cannot unmarshal object into Go
value of type []client.Schedule`. No schedules are displayed.

---

## Command / Reproduction Steps

```bash
iics schedule list --profile dev
```

---

## Expected Behaviour

A table of schedules is printed.

---

## Actual Behaviour

```text
Error: parsing response: json: cannot unmarshal object into Go value of type []client.Schedule
exit status 1
```

---

## Environment

| Field                      | Value                         |
| -------------------------- | ----------------------------- |
| OS                         | macOS 25.3.0                  |
| `iics --version`           | dev build                     |
| Go version                 | 1.25.0                        |
| IICS region                | US                            |
| Output format (`--output`) | table                         |

---

## Architecture Layer

- [x] **`internal/client/`** - HTTP logic, API structs, request/response handling

---

## Likely Affected Files

```text
internal/client/schedules.go
```

---

## API Details

| Field          | Value                           |
| -------------- | ------------------------------- |
| API version    | V3 (`public/core/v3`)           |
| HTTP method    | GET                             |
| Endpoint path  | `public/core/v3/schedule`       |
| Session header | `INFA-SESSION-ID`               |

The API returns a wrapper object `{"schedules": [...]}`, not a plain array.
Docs state: "If you request the details for all schedules, the schedules object contains details
for each schedule in the organization."

---

## Fix (filled in after resolution)

**Root cause:**

Two issues in `internal/client/schedules.go`:

1. `ListSchedules` decoded into `[]Schedule` but the API wraps the array in `{"schedules": [...]}`.
2. `Schedule` struct had incorrect/missing fields compared to the API docs:
   - `Timezone` with tag `json:"timezone"` should be `TimeZoneID` with `json:"timeZoneId"`
   - Missing fields: `ScheduleFederatedID`, `Sun`, `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat`, `WeekDay`
   - `DayOfMonth` typed as `string` should be `int`

**Files changed:**

```text
internal/client/schedules.go:11-48 - added scheduleListResponse wrapper struct, corrected Schedule fields
```

**Test added / updated:**

None - no test file existed for schedules. Test coverage can be added as a follow-up.
