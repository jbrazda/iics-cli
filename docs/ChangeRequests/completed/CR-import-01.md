# CR: Update Import to be user friendly for CI/CD

## Scope

- Files to change
  - internal/client/imports.go
  - related tests and documentation

## Problem

- using the asynchronous import job polling for job import is too difficult in the ci/cd scenarios

## Desired Change

- use the [Documentation](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/importing-objects.html)
- add downloadLog sub-command to getLog use the --logPath path and optional --logName with default to name_id_status.log
- add import sub-command combining existing upload start and status, import will do following
  - upload the desired .zip package to iics using the --zipFilePath string with shorthand -z
  - check status until the import is complete
    - allow override of the polling interval via --pollingInterval flag int defaulting to 10 s
    - allow to set --maxWaitTime in seconds for polling defaulting to 300 s
  - for --verbose option
    - print each step of the process and its results
    - print import job id
    - print start time
    - print polling results including the timestamp and job status
    - print total job duration
  - add --detailedPolling Print list of imported objects object details on each poll and overall progress of imported objects otherwise print just timestamp status and duration
  - add --printImportLog  which the import job via /public/core/v3/import/<id>/log log and print to console
    - for table print job summary table and table with objects
  - Always download and print importLog in case of import errors
  - Always print final status of the job formatted according to --output and --expand flags Print job summary and object list table
  - Update the documentation and completion accordingly
  - Use the Provided zip sample for import testing: `testdata/imports/ZZ_TEST_CLI.zip`

## Acceptance Criteria

- [x] Import is successful
- [x] Verbose logging outputs details
- [x] Existing passing tests still pass

## Do NOT

- Refactor unrelated code
