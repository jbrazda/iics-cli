# CR: Add Listing export to file for List commands

- Files to change: [internal/client/objects.go](CR-AddListFileToListCommands.md)

## Problem

We want to be able to list IICS objects to generate the declarative Configuration files to be able to drive export, import publish operations

## Desired Change

- Add --outputFile flag representing a path to a file that would contain object id and path in yaml format to represent the target location of the asset in the IICS Repository or export Import package
- Add --outputFileFormat supporting yaml, json, table, csv
- Add --outputFieldsFields as a comma separated list of fields in the  object defaults to id,path,type,description,updatedBy,updateTime
- Add --outputFields to manage output fields to console also defaulting to id,path,type,description,updatedBy,updateTime
- Add support for yaml also withing the --output parameter
- This should allow to have a different output format for console display vs output file
- Update the completion and documentation accordingly
- --verbose mode should print the steps of the file listing paging progress and status, location and size of the produced file
- add calculated location field to the output when requested that would be composed as follows this can be used to drive the packaging and publishing steps

```text
Explore/<path from the list output>.<type>
```

### Example Yaml output structure

Structure when when the id,path,type,location fields ar requested 

```yaml
objects:
    - id: "1vRc0Rb7ua4kT8ccjL6sKf"
      path: "ZZ_TEST_CLI/Components "
      type: "Folder"
      location: "Explore/ZZ_TEST_CLI/Components"
    - path: "ZZ_TEST_CLI/Connections"
      type: "Folder"
      location: "Explore/ZZ_TEST_CLI/Connections.Folder"
    # rest of the objects follows
```

Example file structure of exported assets in the export/import zip is following

example list command output

```text
┌────────────────────────┬─────────┬───────────────────────────────────────┬──────────────────────────────┬──────────────────────┐
│           ID           │  TYPE   │                 PATH                  │          UPDATED BY          │       UPDATED        │
├────────────────────────┼─────────┼───────────────────────────────────────┼──────────────────────────────┼──────────────────────┤
│ 1vRc0Rb7ua4kT8ccjL6sKf │ Folder  │ ZZ_TEST_CLI/Components                │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ 8aJAqowEF2phOFeUtU8TuS │ Folder  │ ZZ_TEST_CLI/Connections               │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ aLX7qnviqxJdmqpVsd17SG │ Folder  │ ZZ_TEST_CLI/Connectors                │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ guoHBxV3L8GdMprXqqSdKH │ Folder  │ ZZ_TEST_CLI/DataQuality               │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ 3OrYJxTOzUJdwgHskwZCcj │ Folder  │ ZZ_TEST_CLI/Guides                    │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ 9k4OaSJhnYyd4g0NUDlE6s │ Folder  │ ZZ_TEST_CLI/IssueGenerateRandomString │ jaroslav.brazda.dev@natl.com │ 2024-07-08T21:17:44Z │
│ 2evb0eclhUEfvPKOyj794R │ Folder  │ ZZ_TEST_CLI/Mappings                  │ jaroslav.brazda.dev@natl.com │ 2026-03-08T17:12:05Z │
│ 6diNoBL2NEqlCYHhYgc68O │ Folder  │ ZZ_TEST_CLI/Mapplets                  │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ 191ZGSrZLeTgtPaXNRQpWV │ Folder  │ ZZ_TEST_CLI/MassIngestion             │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ 1yeVVpAOUcHioT3sSTii2G │ Folder  │ ZZ_TEST_CLI/Processes                 │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ 3N4yMZtC2GBhTfdAhT3yG0 │ Folder  │ ZZ_TEST_CLI/ProcessObjects            │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ 2mGokcuTPCjdbdeyCmzFKM │ Folder  │ ZZ_TEST_CLI/TaskFlows                 │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:28:20Z │
│ 7fDEP5uXWTWdA9vmHsb7Ex │ Folder  │ ZZ_TEST_CLI/Tasks                     │ jaroslav.brazda.dev@natl.com │ 2026-03-08T17:12:05Z │
│ 8XK79DgSovVblOcwrOWF62 │ Project │ ZZ_TEST_CLI                           │ jaroslav.brazda.dev@natl.com │ 2026-03-08T16:27:44Z │
└────────────────────────┴─────────┴───────────────────────────────────────┴──────────────────────────────┴──────────────────────┘
```

```text
testdata/imports/ZZ_TEST_CLI_Unpacked
├── ContentsofExportPackage_ZZ_TEST_CLI-1772990451616.csv
├── Explore
│   ├── ZZ_TEST_CLI
│   │   ├── Components.Folder.json
│   │   ├── Connections
│   │   │   └── TestServiceConnection1.AI_CONNECTION.xml
│   │   ├── Connections.Folder.json
│   │   ├── Connectors
│   │   │   └── TestServiceConnector1.AI_SERVICE_CONNECTOR.xml
│   │   ├── Connectors.Folder.json
│   │   ├── DataQuality.Folder.json
│   │   ├── Guides
│   │   │   └── Test Conversion Utility.GUIDE.xml
│   │   ├── Guides.Folder.json
│   │   ├── IssueGenerateRandomString
│   │   │   ├── Test_GenRandomString.PROCESS.xml
│   │   │   └── TestRandomString.GUIDE.xml
│   │   ├── IssueGenerateRandomString.Folder.json
│   │   ├── Mappings
│   │   │   └── m_Test_Git.DTEMPLATE.zip
│   │   ├── Mappings.Folder.json
│   │   ├── Mapplets.Folder.json
│   │   ├── MassIngestion.Folder.json
│   │   ├── Processes
│   │   │   └── MockupEcho.PROCESS.xml
│   │   ├── Processes.Folder.json
│   │   ├── ProcessObjects
│   │   │   └── TestConfigurationProperty.PROCESS_OBJECT.xml
│   │   ├── ProcessObjects.Folder.json
│   │   ├── TaskFlows.Folder.json
│   │   ├── Tasks
│   │   │   └── mt_Test_Git.MTT.zip
│   │   └── Tasks.Folder.json
│   └── ZZ_TEST_CLI.Project.json
├── exportMetadata.v2.json
├── exportPackage.chksum
└── SYS
    ├── CDI-G01.AgentGroup.zip
    └── Telematics.Connection.zip
```

## Acceptance Criteria

- [ ] Generate configuration file for expoir and import
- [ ] Existing passing tests still pass

## Do NOT

- Refactor unrelated code
