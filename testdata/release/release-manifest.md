# Release Manifest

- Schema Version: `v1`
- Generated At (UTC): `2026-05-07T18:08:26Z`
- Source: `testdata/release/manifest.md`
- Mode: `tag-based`
- Tag: `PSAR`
- Targets: `PROD, QA, TST`
- Include Connectors: `true`
- Connectors Only: `false`

## PR Details

- PR Author: [Jaroslav Brazda](mailto:john.doe@example.com)
- PR Link: [<PR Link>](<PR URL>)

### Commits

- [Commit Hash 1](<Commit URL 1>): <Commit Message 1>
- [Commit Hash 2](<Commit URL 2>): <Commit Message 2>
- [Commit Hash 3](<Commit URL 3>): <Commit Message 3>

### Description

> TODO: Add description from PR when available (This will be appended by the CI Pipeline after initial generation of the manifest)

## Errors and Warnings

Generate Errors and warnings when the Manifest parsing fails or when there are issues with the provided input data. This section will be populated during the CI Pipeline execution.

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
|------------------------------------------------------------------------------------------|------------|---------------|-------------|--------------|
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieve-API-v1.AI_CONNECTION             | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearchConnector.AI_CONNECTION               | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/RiskRetrieve-API-v1.AI_CONNECTION                 | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/LocationSearch-API-v1.AI_CONNECTION                 | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/RiskSerach-API-v1.AI_CONNECTION                     | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/VehicleSearch-API-v1.AI_CONNECTION                  | explicit   | missing       | found       | found        |
| Explore/Connections/Natl APIs/EntityMeshServiceConnection.AI_CONNECTION                  | transitive | missing       | found       | found        |
| Explore/Connections/Salesforce/SalesforceMetadata.AI_CONNECTION                          | transitive | found         | found       | found        |
| Explore/DAS/Connections/DataAccessService.AI_CONNECTION                                  | transitive | found         | found       | found        |
| Explore/Tools/Connections/IPaaS-Configuration-DB.AI_CONNECTION                           | transitive | found         | found       | found        |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieve-API-v1.AI_SERVICE_CONNECTOR      | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearchServiceConnector.AI_SERVICE_CONNECTOR | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/RiskRetrieve-API-v1.AI_SERVICE_CONNECTOR          | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/LocationSearch-API-v1.AI_SERVICE_CONNECTOR          | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/RiskSearch-API-v1.AI_SERVICE_CONNECTOR              | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/VehicleSearch-API-v1.AI_SERVICE_CONNECTOR           | explicit   | missing       | found       | found        |
| Explore/Connectors/Natl APIs/EntityMeshService.AI_SERVICE_CONNECTOR                      | transitive | missing       | found       | found        |
| Explore/Connectors/Salesforce/SalesforceMetadata.AI_SERVICE_CONNECTOR                    | transitive | found         | found       | found        |
| Explore/DAS/Connectors/DataAccessService.AI_SERVICE_CONNECTOR                            | transitive | found         | found       | found        |
| Explore/ClaimCenter_GW/Guides/TestPolicyRetrieve.GUIDE                                   | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/Guides/TestPolicySearch.GUIDE                                     | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/Guides/TestRiskSearch.GUIDE                                       | explicit   | missing       | found       | found        |
| Explore/Logging/Guides/SP Show Process Links.GUIDE                                       | transitive | found         | found       | found        |
| Explore/Logging/Guides/iPaaS Job View DB.GUIDE                                           | transitive | found         | found       | found        |
| Explore/Tools/Guides/iPaaS Configuration Manager.GUIDE                                   | transitive | found         | found       | found        |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/MP-PolicyRetrieve-CAI-v1.PROCESS                | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieveWIP.PROCESS                       | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieve_v1.PROCESS                       | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/SP-SF-PolicyRetrieve-CAI-v1.PROCESS             | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicySearch_v1/MP-PolicySearch-CAI-v1.PROCESS                    | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearch_v1.PROCESS                           | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicySearch_v1/SP-PolicySearch-CAI-v1.PROCESS                    | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/LocationRiskRetrieve_v1.PROCESS                   | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/MP-RiskRetrieve_v1-CAI-v1.PROCESS                 | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/SP-SF-RiskRetrieve_v1-CAI-v1.PROCESS              | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/VehicleRiskRetrieve_v1.PROCESS                    | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/LocationRiskSearch_v1.PROCESS                       | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/MP-RiskSearch-CAI-v1.PROCESS                        | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/SP-RiskSearch-CAI-v1.PROCESS                        | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/SP-SF-RiskSearch-CAI-vOLD.PROCESS                   | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/VehicleRiskSearch_v1.PROCESS                        | explicit   | missing       | found       | found        |
| Explore/DAS/Processes/execMultiSQLProxy.PROCESS                                          | transitive | found         | found       | found        |
| Explore/Tools/Processes/SP-Import-Configuration.PROCESS                                  | transitive | found         | found       | found        |
| Explore/Tools/ProxyProcesses/SP-IPaaS-Encrypt-NA.PROCESS                                 | transitive | found         | found       | found        |
| Explore/Tools/ProxyProcesses/SP-ReadConfiguration.PROCESS                                | transitive | found         | found       | found        |
| Explore/ClaimCenter_GW/PolicyRetrieve_v1/PolicyRetrieveParameters.PROCESS_OBJECT         | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearchParameters.PROCESS_OBJECT             | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/PolicySearch_v1/PolicySearchRequest.PROCESS_OBJECT                | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskRetrieve_v1/RiskRetrieveParameters.PROCESS_OBJECT             | explicit   | missing       | found       | found        |
| Explore/ClaimCenter_GW/RiskSearch_v1/RiskSearchParameters.PROCESS_OBJECT                 | explicit   | missing       | found       | found        |
| Explore/Logging/ProcessObjects/CreateJobLogEventRequest.PROCESS_OBJECT                   | transitive | found         | found       | found        |
| Explore/Logging/ProcessObjects/FaultInfo.PROCESS_OBJECT                                  | transitive | found         | found       | found        |
| Explore/Logging/ProcessObjects/JobEvent.PROCESS_OBJECT                                   | transitive | found         | found       | found        |
| Explore/Logging/ProcessObjects/JobEventView.PROCESS_OBJECT                               | transitive | found         | found       | found        |
| Explore/Logging/ProcessObjects/JobLogRecord.PROCESS_OBJECT                               | transitive | found         | found       | found        |
| Explore/Logging/ProcessObjects/ProcessExecutionContext.PROCESS_OBJECT                    | transitive | found         | found       | found        |
| Explore/Tools/ProcessObjects/CachePutItem.PROCESS_OBJECT                                 | transitive | found         | found       | found        |
| Explore/Tools/ProcessObjects/CachePutRequest.PROCESS_OBJECT                              | transitive | found         | found       | found        |
| Explore/Tools/ProcessObjects/ConfigurationProperty.PROCESS_OBJECT                        | transitive | found         | found       | found        |

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

> NOTE: Package content includes all items that are included in the release package for each target + any transitive dependencies that are not explicitly included but missing in the target Environment. The counts may differ from the total included items due to explicit exclusions or differences in dependencies.

## Publishable Assets per Target

 | TYPE                 | COUNT (PROD) | COUNT (QA) | COUNT (TST) |
 |----------------------|-------------:|-----------:|------------:|
 | AI_CONNECTION        |            7 |          6 |           6 |
 | AI_SERVICE_CONNECTOR |            7 |          6 |           6 |
 | GUIDE                |            3 |          3 |           3 |
 | PROCESS              |           16 |         16 |          16 |
 | TOTAL                |           33 |         31 |          31 |

 > NOTE: Publishable assets are those that are included in the release package for each target or resolved as a transitive dependency. The counts may differ from the total included items due to target-specific exclusions or differences in dependencies.
