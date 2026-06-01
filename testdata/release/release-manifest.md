# Release Manifest

- Schema Version: `v1`
- Generated At (UTC): `2026-05-07T18:08:26Z`
- Source: `testdata/release/manifest.md`
- Mode: `tag-based`
- Tag: `PSAR`
- Targets: `PROD, QA, TST`
- Include Connectors: `true`

## PR Details

> TODO: The PR details will be populated when the manifest is generated as part of the CI Pipeline execution. This section is intended to provide context on the source PR that triggered the release generation and may include details such as PR author, link to the PR, summary of changes, etc.

- PR Author: [John Doe](mailto:john.doe@example.com)
- PR Link: [123479](<PR URL>)

### Commits

> this section will be written during the CI Pipeline execution by fetching the commit history from the source PR. The commit hashes, messages, and links will be included to provide traceability on the changes that are part of this release.

- [Commit Hash 1](<Commit URL 1>): <Commit Message 1>
- [Commit Hash 2](<Commit URL 2>): <Commit Message 2>
- [Commit Hash 3](<Commit URL 3>): <Commit Message 3>

### Description

> TODO: Add description from PR when available (This will be appended by the CI Pipeline after initial generation of the manifest)

## Errors and Warnings

> Generate Errors and warnings when the Manifest parsing fails or when there are issues with the provided input data. This section will be populated during the CI Pipeline execution.

## Included Items

Included items with transitive dependencies:

| TYPE                 | COUNT |
|----------------------|------:|
| AI_CONNECTION        |    10 |
| AI_SERVICE_CONNECTOR |     9 |
| GUIDE                |     6 |
| PROCESS              |    20 |
| PROCESS_OBJECT       |    14 |
| TOTAL                |    59 |

## Excluded Items

> TODO: Add excluded items when implemented

## Deployment Dependency Status Summary

| LOCATION                                                                                 | DEPENDENCY | STATUS (PROD) | STATUS (QA) | STATUS (TST) |
|------------------------------------------------------------------------------------------|:----------:|:-------------:|:-----------:|:------------:|
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieve-API-v1.AI_CONNECTION             |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearchConnector.AI_CONNECTION               |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/RiskRetrieve-API-v1.AI_CONNECTION                 |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/LocationSearch-API-v1.AI_CONNECTION                 |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/RiskSerach-API-v1.AI_CONNECTION                     |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/VehicleSearch-API-v1.AI_CONNECTION                  |  explicit  |    missing    |    found    |    found     |
| Explore/Connections/Natl APIs/EntityMeshServiceConnection.AI_CONNECTION                  | transitive |    missing    |    found    |    found     |
| Explore/Connections/Salesforce/SalesforceMetadata.AI_CONNECTION                          | transitive |     found     |    found    |    found     |
| Explore/DAS/Connections/DataAccessService.AI_CONNECTION                                  | transitive |     found     |    found    |    found     |
| Explore/Tools/Connections/IPaaS-Configuration-DB.AI_CONNECTION                           | transitive |     found     |    found    |    found     |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieve-API-v1.AI_SERVICE_CONNECTOR      |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearchServiceConnector.AI_SERVICE_CONNECTOR |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/RiskRetrieve-API-v1.AI_SERVICE_CONNECTOR          |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/LocationSearch-API-v1.AI_SERVICE_CONNECTOR          |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/RiskSearch-API-v1.AI_SERVICE_CONNECTOR              |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/VehicleSearch-API-v1.AI_SERVICE_CONNECTOR           |  explicit  |    missing    |    found    |    found     |
| Explore/Connectors/Natl APIs/EntityMeshService.AI_SERVICE_CONNECTOR                      | transitive |    missing    |    found    |    found     |
| Explore/Connectors/Salesforce/SalesforceMetadata.AI_SERVICE_CONNECTOR                    | transitive |     found     |    found    |    found     |
| Explore/DAS/Connectors/DataAccessService.AI_SERVICE_CONNECTOR                            | transitive |     found     |    found    |    found     |
| Explore/ClaimCenter_GW/Guides/TestPolicyRetrieve.GUIDE                                   |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/Guides/TestPolicySearch.GUIDE                                     |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/Guides/TestRiskSearch.GUIDE                                       |  explicit  |    missing    |    found    |    found     |
| Explore/Logging/Guides/SP Show Process Links.GUIDE                                       | transitive |     found     |    found    |    found     |
| Explore/Logging/Guides/iPaaS Job View DB.GUIDE                                           | transitive |     found     |    found    |    found     |
| Explore/Tools/Guides/iPaaS Configuration Manager.GUIDE                                   | transitive |     found     |    found    |    found     |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/MP-PolicyRetrieve-CAI-v1.PROCESS                |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieveWIP.PROCESS                       |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieve_v1.PROCESS                       |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/SP-SF-PolicyRetrieve-CAI-v1.PROCESS             |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicySearch_v1/MP-PolicySearch-CAI-v1.PROCESS                    |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearch_v1.PROCESS                           |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicySearch_v1/SP-PolicySearch-CAI-v1.PROCESS                    |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/LocationRiskRetrieve_v1.PROCESS                   |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/MP-RiskRetrieve_v1-CAI-v1.PROCESS                 |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/SP-SF-RiskRetrieve_v1-CAI-v1.PROCESS              |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/VehicleRiskRetrieve_v1.PROCESS                    |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/LocationRiskSearch_v1.PROCESS                       |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/MP-RiskSearch-CAI-v1.PROCESS                        |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/SP-RiskSearch-CAI-v1.PROCESS                        |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/SP-SF-RiskSearch-CAI-vOLD.PROCESS                   |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/VehicleRiskSearch_v1.PROCESS                        |  explicit  |    missing    |    found    |    found     |
| Explore/DAS/Processes/execMultiSQLProxy.PROCESS                                          | transitive |     found     |    found    |    found     |
| Explore/Tools/Processes/SP-Import-Configuration.PROCESS                                  | transitive |     found     |    found    |    found     |
| Explore/Tools/ProxyProcesses/SP-IPaaS-Encrypt-NA.PROCESS                                 | transitive |     found     |    found    |    found     |
| Explore/Tools/ProxyProcesses/SP-ReadConfiguration.PROCESS                                | transitive |     found     |    found    |    found     |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieveParameters.PROCESS_OBJECT         |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearchParameters.PROCESS_OBJECT             |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearchRequest.PROCESS_OBJECT                |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/RiskRetrieveParameters.PROCESS_OBJECT             |  explicit  |    missing    |    found    |    found     |
| Explore/ClaimCenter_GW/RiskSearch_v1/RiskSearchParameters.PROCESS_OBJECT                 |  explicit  |    missing    |    found    |    found     |
| Explore/Logging/ProcessObjects/CreateJobLogEventRequest.PROCESS_OBJECT                   | transitive |     found     |    found    |    found     |
| Explore/Logging/ProcessObjects/FaultInfo.PROCESS_OBJECT                                  | transitive |     found     |    found    |    found     |
| Explore/Logging/ProcessObjects/JobEvent.PROCESS_OBJECT                                   | transitive |     found     |    found    |    found     |
| Explore/Logging/ProcessObjects/JobEventView.PROCESS_OBJECT                               | transitive |     found     |    found    |    found     |
| Explore/Logging/ProcessObjects/JobLogRecord.PROCESS_OBJECT                               | transitive |     found     |    found    |    found     |
| Explore/Logging/ProcessObjects/ProcessExecutionContext.PROCESS_OBJECT                    | transitive |     found     |    found    |    found     |
| Explore/Tools/ProcessObjects/CachePutItem.PROCESS_OBJECT                                 | transitive |     found     |    found    |    found     |
| Explore/Tools/ProcessObjects/CachePutRequest.PROCESS_OBJECT                              | transitive |     found     |    found    |    found     |
| Explore/Tools/ProcessObjects/ConfigurationProperty.PROCESS_OBJECT                        | transitive |     found     |    found    |    found     |

## Release Plan

> Note: Following files were generated for each target environment as part of the release package. The content of the files may differ based on the target-specific inclusions/exclusions and resolved dependencies.

```text
TODO: Add tree of files generated for each target i.e.
target
└── iics
    └── import
        ├── conf
        │   └── release_manifest.yaml
        ├── connectors.package.csv
        ├── logs
        │   └── release_manifest.md
        ├── prod
        │   ├── publish_assets.csv
        │   └── tag_build.package.csv
        ├── qa
        │   ├── publish_assets.csv
        │   └── tag_build.package.csv
        └── tst
            ├── publish_assets.csv
            └── tag_build.package.csv
```

## Package Content per Target

| TYPE                 | COUNT (PROD) | COUNT (QA) | COUNT (TST) |
|----------------------|-------------:|-----------:|------------:|
| AI_CONNECTION        |            7 |          6 |           6 |
| AI_SERVICE_CONNECTOR |            7 |          6 |           6 |
| GUIDE                |            3 |          3 |           3 |
| PROCESS              |           16 |         16 |          16 |
| PROCESS_OBJECT       |            5 |          5 |           5 |
| TOTAL                |           38 |         36 |          36 |

> NOTE: Package content includes all items that are included in the release package for each target + any transitive dependencies that are not explicitly included but missing in the target Environment. The counts may differ from the total included items due to explicit exclusions or differences in dependencies. See the generated `tag_build.package.csv` or `publish_assets.csv` files for details on the assets included for each target.

## Publishable Assets per Target

 | TYPE                 | COUNT (PROD) | COUNT (QA) | COUNT (TST) |
 |----------------------|-------------:|-----------:|------------:|
 | AI_CONNECTION        |            7 |          6 |           6 |
 | AI_SERVICE_CONNECTOR |            7 |          6 |           6 |
 | GUIDE                |            3 |          3 |           3 |
 | PROCESS              |           16 |         16 |          16 |
 | TOTAL                |           33 |         31 |          31 |

 > NOTE: Publishable assets are those that are included in the release package for each target or resolved as a transitive dependency. The counts may differ from the total included items due to target-specific exclusions or differences in dependencies. See the generated `publish_assets.csv` files for details on the assets included for each target.

## Backup and Rollback Plan

> Generate  list of backup items (export command outputs) invoked by CI Pipeline before deploying the release package to each target environment. This section will be generated by invoking ics export command with --log-file option to print list of exported items and their details to a log file. The log file will be parsed to extract the relevant information and populate this section of the manifest. This will provide traceability on what items were backed up before deployment and can be used for rollback if needed.

### Exported Objects

| ID                   | PATH                                                              | TYPE                 | STATUS     |
|----------------------|-------------------------------------------------------------------|----------------------|------------|
| 1a2b3c4d5e6f7g8h9i0j | ZZ_TEST_CLI/Connections/TestServiceConnection1.AI_CONNECTION      | AI_CONNECTION        | SUCCESSFUL |
| 2b3c4d5e6f7g8h9i0j1a | ZZ_TEST_CLI/Connectors/TestServiceConnector1.AI_SERVICE_CONNECTOR | AI_SERVICE_CONNECTOR | SUCCESSFUL |
| 3c4d5e6f7g8h9i0j1a2b | ZZ_TEST_CLI/Guides/Test Conversion Utility.GUIDE                  | GUIDE                | SUCCESSFUL |
| 4d5e6f7g8h9i0j1a2b3  | ZZ_TEST_CLI/Processes/MP-SampleJob-v1.PROCESS                     | PROCESS              | SUCCESSFUL |

n rows

## Import Report

> The following table summarizes the results of the import process for each item included in the release package. It indicates whether the item was found or missing in each target environment (PROD, QA, TST) and whether it was included explicitly in the release package or resolved as a transitive dependency.
> print contents of the generated `import_report.csv` file which includes details on each item imported, its type, and the status of the import for each target environment. This report provides traceability on what items were included in the release package and their import status across environments.

### Import Summary

| FIELD      | VALUE                    |
|------------|--------------------------|
| Job ID     | 2afCEsYYRXSgv08gkRywC6   |
| State      | SUCCESSFUL               |
| Start Date | 2026-05-29T17:10:58.000Z |
| End Date   | 2026-05-29T17:11:01.000Z |
| Duration   | 28s                      |
| Total      | 10                       |
| Published  | 10                       |
| Errors     | 0                        |

### Imported Objects

| SOURCE ID              | SOURCE PATH                            | SOURCE NAME               | TARGET NAME               | SOURCE TYPE          | STATE      | MESSAGE                   |
|------------------------|----------------------------------------|---------------------------|---------------------------|----------------------|------------|---------------------------|
| 0KfTorzrNwihXfV38FliA2 | /ZZ_TEST_CLI/Connections               | TestServiceConnection1    | TestServiceConnection1    | AI_CONNECTION        | SUCCESSFUL | Overwrite existing object |
| 1yeVVpAOUcHioT3sSTii2G | /ZZ_TEST_CLI                           | Processes                 | Processes                 | Folder               | SUCCESSFUL | Reuse existing object     |
| 3cnlhCxf4I5kGPJQoIZSMZ | /ZZ_TEST_CLI/Processes                 | MockupEcho                | MockupEcho                | PROCESS              | SUCCESSFUL | Overwrite existing object |
| 3N4yMZtC2GBhTfdAhT3yG0 | /ZZ_TEST_CLI                           | ProcessObjects            | ProcessObjects            | Folder               | SUCCESSFUL | Reuse existing object     |
| 3OrYJxTOzUJdwgHskwZCcj | /ZZ_TEST_CLI                           | Guides                    | Guides                    | Folder               | SUCCESSFUL | Reuse existing object     |
| 3sSitJsDn5odECT8qJMYut | /ZZ_TEST_CLI/Guides                    | Test Conversion Utility   | Test Conversion Utility   | GUIDE                | SUCCESSFUL | Overwrite existing object |
| 4ayib7p2V5xlJ1CHmRiTA0 | /ZZ_TEST_CLI/Connectors                | TestServiceConnector1     | TestServiceConnector1     | AI_SERVICE_CONNECTOR | SUCCESSFUL | Overwrite existing object |
| 5NdzLW2s8S6ghkAjo8384O | /ZZ_TEST_CLI/Processes                 | SP-SampleJob-v1           | SP-SampleJob-v1           | PROCESS              | SUCCESSFUL | Overwrite existing object |
| 7fDEP5uXWTWdA9vmHsb7Ex | /ZZ_TEST_CLI                           | Tasks                     | Tasks                     | Folder               | SUCCESSFUL | Reuse existing object     |
| 8aJAqowEF2phOFeUtU8TuS | /ZZ_TEST_CLI                           | Connections               | Connections               | Folder               | SUCCESSFUL | Reuse existing object     |
| 8XK79DgSovVblOcwrOWF62 | /                                      | ZZ_TEST_CLI               | ZZ_TEST_CLI               | Project              | SUCCESSFUL | Reuse existing object     |
| 94w3YRfYqTwkXDMISNqemA | /ZZ_TEST_CLI/Processes                 | MP-SampleJob-v1           | MP-SampleJob-v1           | PROCESS              | SUCCESSFUL | Overwrite existing object |
| 9k4OaSJhnYyd4g0NUDlE6s | /ZZ_TEST_CLI                           | IssueGenerateRandomString | IssueGenerateRandomString | Folder               | SUCCESSFUL | Reuse existing object     |
| 9WuPOQqwGVfi4M1RQ1OVUZ | /ZZ_TEST_CLI/IssueGenerateRandomString | TestRandomString          | TestRandomString          | GUIDE                | SUCCESSFUL | Overwrite existing object |
| aLX7qnviqxJdmqpVsd17SG | /ZZ_TEST_CLI                           | Connectors                | Connectors                | Folder               | SUCCESSFUL | Reuse existing object     |
| aN1Pyo4KVAfltahTnFdX6O | /ZZ_TEST_CLI/Processes                 | SP-SimulateFault          | SP-SimulateFault          | PROCESS              | SUCCESSFUL | Overwrite existing object |
| d6dJpLA2O4keMYSbHodOEU | /ZZ_TEST_CLI/Processes                 | SCH-SampleJob-v1          | SCH-SampleJob-v1          | PROCESS              | SUCCESSFUL | Overwrite existing object |
| h2zy4tla02Gf0L72vmIo6L | /ZZ_TEST_CLI/ProcessObjects            | TestConfigurationProperty | TestConfigurationProperty | PROCESS_OBJECT       | SUCCESSFUL | Overwrite existing object |
| kUBJTuKdfe3fZCZfsibhme | /ZZ_TEST_CLI/Tasks                     | mt_Test_Git               | mt_Test_Git               | MTT                  | SUCCESSFUL | Overwrite existing object |
| lFlQBdLdygSfyFK3TZh4IZ | /ZZ_TEST_CLI/IssueGenerateRandomString | Test_GenRandomString      | Test_GenRandomString      | PROCESS              | SUCCESSFUL | Overwrite existing object |

> If import fails print downloaded import log file content here for debugging

```txt
TODO: Add contents of the import log file here if import fails for debugging purposes. This will help identify the root cause of the failure and provide insights on what went wrong during the import process.
```

## Publish Report

Published items are listed in the table below, along with their type and the status of the publish operation

### Publish Summary

| FIELD      | VALUE                        |
|------------|------------------------------|
| Job ID     | 1245061829462720512          |
| State      | SUCCESS                      |
| Start Date | 2026-05-29T17:11:12.000+0000 |
| End Date   | 2026-05-29T17:11:40.000+0000 |
| Duration   | 28s                          |

### Publish Items

| INDEX | GUID                   | ASSET PATH                                                                     | STATE   | START DATE                   | END DATE                     | DURATION |
| ----- | ---------------------- | ------------------------------------------------------------------------------ | ------- | ---------------------------- | ---------------------------- | -------- |
| 0     | hvOcvUe6mmybXDfA0IYI6N | Explore/ZZ_TEST_CLI/Connectors/TestServiceConnector1.AI_SERVICE_CONNECTOR.xml  | SUCCESS | 2026-05-29T17:11:12.000+0000 | 2026-05-29T17:11:13.000+0000 | 1s       |
| 1     | a7KamTdOe1Xj2FJ5cVR1au | Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1.AI_CONNECTION.xml       | SUCCESS | 2026-05-29T17:11:13.000+0000 | 2026-05-29T17:11:17.000+0000 | 4s       |
| 2     | 7hJcq7CTjarjv58Lk1kAC2 | Explore/ZZ_TEST_CLI/IssueGenerateRandomString/Test_GenRandomString.PROCESS.xml | SUCCESS | 2026-05-29T17:11:17.000+0000 | 2026-05-29T17:11:19.000+0000 | 2s       |
| 3     | 147KD6djM7dkGmBIOio8KK | Explore/ZZ_TEST_CLI/Processes/MP-SampleJob-v1.PROCESS.xml                      | SUCCESS | 2026-05-29T17:11:19.000+0000 | 2026-05-29T17:11:26.000+0000 | 7s       |
| 4     | fqQhNPlAtcrkkUsOZtzhcJ | Explore/ZZ_TEST_CLI/Processes/MockupEcho.PROCESS.xml                           | SUCCESS | 2026-05-29T17:11:26.000+0000 | 2026-05-29T17:11:28.000+0000 | 2s       |
| 5     | e9kCvKSRYyne1OBgQUfgKF | Explore/ZZ_TEST_CLI/Processes/SCH-SampleJob-v1.PROCESS.xml                     | SUCCESS | 2026-05-29T17:11:28.000+0000 | 2026-05-29T17:11:35.000+0000 | 7s       |
| 6     | 4Bn4ptoDKJJgS0saak93MU | Explore/ZZ_TEST_CLI/Processes/SP-SampleJob-v1.PROCESS.xml                      | SUCCESS | 2026-05-29T17:11:35.000+0000 | 2026-05-29T17:11:36.000+0000 | 1s       |
| 7     | 607zNkzugz3djuQJBiqlE5 | Explore/ZZ_TEST_CLI/Processes/SP-SimulateFault.PROCESS.xml                     | SUCCESS | 2026-05-29T17:11:36.000+0000 | 2026-05-29T17:11:36.000+0000 | < 1s     |
| 8     | 0kVVAoiy2zGjpXDkEIQzqX | Explore/ZZ_TEST_CLI/Guides/Test Conversion Utility.GUIDE.xml                   | SUCCESS | 2026-05-29T17:11:36.000+0000 | 2026-05-29T17:11:39.000+0000 | 3s       |
| 9     | 8xxqLVprDSCaZUYZhO2KgB | Explore/ZZ_TEST_CLI/IssueGenerateRandomString/TestRandomString.GUIDE.xml       | SUCCESS | 2026-05-29T17:11:39.000+0000 | 2026-05-29T17:11:40.000+0000 | 1s       |

10 rows

if errors > 0 then include the following section in the manifest with details on the failed publish operations

## Publish Errors

List error details for any failed publish operations, including the asset path, error message, and any relevant information to help diagnose the issue.