# CR-0004: Add `package` Command (expand / create)

---

## CR Type

- [x] **New resource** - add a brand-new `iics package` command tree

> Note: this command operates on local files only. No IICS API calls are made.
> Only `cmd/package.go` is required; no `internal/client/` files are needed.

---

## Problem

The `iics export download` command produces a ZIP package file, but there is no built-in way
to inspect or modify its contents, or to assemble a new package from a local directory.

Informatica Export/Import package contains proprietary content and it is not sufficient to simply zip/unzip the package to be able to commit its contents to git as generated source code assets for version control and later re-assemble the package for deployment via import command.

The package contains following structure shown in the expanded example in the [testdata/imports/ZZ_TEST_CLI_Unpacked](../../../testdata/imports/ZZ_TEST_CLI_Unpacked) folder of this project.

```text
testdata/imports/ZZ_TEST_CLI_Unpacked  # root of the package
├── ContentsofExportPackage_ZZ_TEST_CLI-1772990451616.csv #scv containing the Content of the package
├── Explore # Exported Assets in two layer hierarchy Project/Folder/Assets
│   ├── ZZ_TEST_CLI # project folder
│   │   ├── Components.Folder.json # Folder metadata
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
├── exportPackage.chksum ## checksum file of all included files
└── SYS # contains nested zips with Environment Connection metadata that the assets depend on, this is only present when exporting with dependencies
    ├── CDI-G01.AgentGroup.zip
    └── Telematics.Connection.zip
```

In order to be able to implement the expand and create sub-commands, we need to reverse-engineer and replicate replicate the functionality expand and package command of the IICS management Utility available on github https://github.com/InformaticaCloudApplicationIntegration/Tools/tree/master/IICS%20Asset%20Management%20CLI/v2.0.1

>Note: Informatic asset organization has only three layer

A `package` command would provide first-class support for working with IICS package files
directly from the CLI, enabling local inspection, editing, and re-packaging workflows.

Analyze the contents of the file

- it contains json, xml, csv, nested zip files and checksum file  in the root
- checksum file is mandatory to be able to import package to IICS cloud platform

Current informatica API and cli tools has some limitations

- Only xml asset files are pretty-printed, providing a good diff in git
- Only some json files are pretty printed
- Original iic cli seem to have been developed before informatica added export csv containing each exported asset path and id which can be likely is later used to generate the checksum file. It instead does generates additional metadata file `.assetName.json` which is used to assemble the package zip later. The json contains metadata extracted from asset designs (XML/json/nested.zip).
- See provided extracted example using original cli `testdata/imports/ZZ_TEST_CLI_Extracted`
- I believe that the dot files are not necessary to build valid package and we only need included csv with the asset id and type to create a checksum file.
- I'm not sure what type of checksum informatica uses please verify which checksum to use (SHA, MD5) to generate matching checksum file
- Our package checksum will differ as we want to store all json file in the pretty printed form in cvs as well as in the newly assembled package

we need to verify that this still produces valid importable package

---

## Desired Change

Two subcommands under `iics package`:

### `iics package expand`

Extracts the contents of an IICS export ZIP package to a target directory.

```bash
iics package expand --file <package.zip> --target <directory> --recursive
```

- `--file` (required) short `-f` - path to the source ZIP file (output of `iics export download`)
- `--target` (required) short `-t` - destination directory; created if it does not exist
- `--recursive` (optiona)  short `-r` - recursively expand nested zips to folder named as `nested.nzip` and removing the source zip from the file hierarchy
- `--clean` (optional) short `-c` - we assume that the ci/cd script deletes the content of target directory (typically `repositoryroot/src/iics`) is clean and empty in order to be able to recognize file renames, moves or deletions. If this option is provided delete recursively count the files first and prompt user with warning if --yes is provided in addition supress the prompt and warning
- if target directory contains files consider it error state and report error
- in `--verbose` mode print contents of the directory before taking any action
- in `--verbose` mode print contents of the packaged zip file before expanding
  
The original iics utility expands only first level and does not expand nested zips

### `iics package create`

Creates a new IICS export ZIP package from the contents of a source directory.

```bash
iics package create --source <directory> --output <package.zip>
```

- `--source` string (required) short `-s` - path to the directory whose contents are zipped
- `--target` string (required) short `-t` - path to write the resulting ZIP file
- `--force` (optional) short `-f` - overwrites an existing output file without prompting
- `--include` (optional) short `-i` - string array regex expression to include files or folders multiple occurrences supported
- `--exlude` (optional) short `-e` - string array regex expression to exclude files or folders multiple occurrences supported
- `--match-file` (optional) - relative or absolute path to a file which contains set of include/exclude expressions, propose an efficient way to these expression, the file should support comments in a form of `# comment`
When path/foldername.Folder is included in the match list autmatically include all files in the folder and Folder metadata json
- `--workdir` (optional) short `-w` - path to working directory where you generate content of the new package by copying by applying `--include`, `--exclude` -contents rules

---

## Scope

### Files to CREATE

```text
cmd/package.go          # expand and create subcommands (local file ops only)
```

### Files to MODIFY

```text
cmd/root.go             # add rootCmd.AddCommand(newPackageCmd()) in init()
```

### Files to READ (context only - do NOT modify)

```text
docs/CLAUDE.md                    # mandatory: patterns and rules
cmd/root.go                       # getFormatter(), global flags
cmd/export.go                     # reference: how export download writes ZIP files
```

### Forbidden (do NOT touch)

```text
internal/client/    # no API calls involved
internal/config/    # no config changes needed
internal/output/    # output layer is correct as-is
```

---

## API Details

Not applicable - this command performs local file system operations only.
No IICS REST API calls are made.

---

## Implementation Instructions

### Step 1 - Command layer (`cmd/package.go`)

1. `newPackageCmd()` returns a parent `*cobra.Command` with `Use: "package"`.
2. Add `newPackageExpandCmd()` and `newPackageCreateCmd()` as subcommands.
3. Neither subcommand calls `getClient()` - there is no authentication needed.

#### `newPackageExpandCmd()`

```text
Flags:
  --file    string   path to the source ZIP package (required)
  --target  string   destination directory (required; created if absent)
  --file    string   path to the source ZIP package (required)
  --target  string   destination directory (required; created if absent)
```

`RunE` logic:

1. Validate `--file` exists and is readable.
2. Create `--target` directory (and any parents) if it does not exist (`os.MkdirAll`).
3. Open the ZIP with `archive/zip.OpenReader`.
4. For each entry, create the destination path, write file contents.
5. Preserve directory structure from the ZIP
6. Print a summary line to stdout: `Expanded N files to <target>`.
7. in `--verbose` mode print
8. Return any error encountered.

#### `newPackageCreateCmd()`

```text
Flags:
  --source  string   source directory to zip (required)
  --output  string   output ZIP file path (required)
```

`RunE` logic:

1. Validate `--source` exists and is a directory.
2. Create (or truncate) the output file.
3. Walk `--source` with `filepath.WalkDir`, adding each file to a `archive/zip.Writer`.
   only include source directory contents
4. Close the writer (flushes the ZIP central directory).
5. Print a summary line to stdout: `Created <output> (N files)`.
6. In `--verbose` mode print contents of content report cvs file contained in the package 
7. Return any error encountered.

### Step 2 - Register (`cmd/root.go`)

```go
rootCmd.AddCommand(newPackageCmd())
```

### Step 3 - Verify

```bash
/opt/local/bin/go build ./...
/opt/local/bin/go test ./...
/opt/local/bin/go vet ./...
golangci-lint run ./...
```

---

## Output Columns

Contents of the generated `ContentsofExportPackage_ZZ_TEST_CLI.csv`as a table

---

## Acceptance Criteria

- [x] `iics package expand --file <zip> --target <dir>` extracts all files
- [x] `iics package create --source <dir> --target <zip>` produces a valid ZIP (flag is `--target/-t`; `--output` was not used to avoid conflict with the global `--output/-o` persistent flag)
- [x] Round-trip: expand then create produces a ZIP with correct format and self-consistent checksums
- [x] Missing required flags return a clear error message
- [x] `--target` directory is created automatically if absent
- [x] `go build ./...` succeeds with no errors
- [x] `go test ./...` passes with no failures
- [x] `go vet ./...` reports no issues
- [x] `golangci-lint run ./...` reports no new issues
- [x] No `internal/client/` files were created or modified
- [x] No unrelated code was modified

---

## Do NOT

- Call `getClient()` or any IICS API from this command
- Refactor, reformat, or add comments to code outside the CR scope
- Add error handling for scenarios that cannot happen
- Use `os.Exit()` - return errors from `RunE`
- Add `Co-Authored-By` trailers to commit messages

---

## Example Original package Command Output 

```txt
 ~/bin/iics package -z ~/workspace/claude/iics-cli/testdata/imports/ZZ_TEST_CLI_packaged.zip -w ~/workspace/claude/iics-cli/testdata/imports/ZZ_TEST_CLI_Extracted -a 'Explore/ZZ_TEST_CLI.Project'
INFO[0000] IICS CLI Version                              Version=2.0.0
INFO[0000] Gathered artifacts                            Artifacts="[Explore/ZZ_TEST_CLI.Project]"
INFO[0000] Packaging artifacts                           Workspace Directory=/Users/jbrazda/workspace/claude/iics-cli/testdata/imports/ZZ_TEST_CLI_Extracted
INFO[0000] Artifact verification complete                Result=true
INFO[0000] Copying artifacts to temp folder
INFO[0000] Creating checksum file
INFO[0000] Generated checksum for artifact               Artifact="{Project Explore/ZZ_TEST_CLI.Project.json Explore/.ZZ_TEST_CLI.Project.json Explore/ZZ_TEST_CLI.Project }" Checksum="Explore/ZZ_TEST_CLI.Project.json=C27F1C2010C8CA7E6FC01BE8978EFFA35F0D666AF1853AFEB45518833C426142\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/Components.Folder.json Explore/ZZ_TEST_CLI/.Components.Folder.json Explore/ZZ_TEST_CLI/Components.Folder }" Checksum="Explore/ZZ_TEST_CLI/Components.Folder.json=12DEBDE2E6FEF9A9AF6B9EF51B782EA616CEFA20E88C53513056184FF590C063\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/Connections.Folder.json Explore/ZZ_TEST_CLI/.Connections.Folder.json Explore/ZZ_TEST_CLI/Connections.Folder }" Checksum="Explore/ZZ_TEST_CLI/Connections.Folder.json=592DC8C812F91F52B6DE78BBAEA11F6AD10F6FED470329CD01FEA0D16C8C54A2\n"
INFO[0000] Generated checksum for artifact               Artifact="{AI_CONNECTION Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1.AI_CONNECTION.xml Explore/ZZ_TEST_CLI/Connections/.TestServiceConnection1.AI_CONNECTION.json Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1.AI_CONNECTION }" Checksum="Explore/ZZ_TEST_CLI/Connections/TestServiceConnection1.AI_CONNECTION.xml=7ED08257D6EB77CA0AA783D9D7E13486A41F4286F447F7DCB07412D2CE0A6907\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/Connectors.Folder.json Explore/ZZ_TEST_CLI/.Connectors.Folder.json Explore/ZZ_TEST_CLI/Connectors.Folder }" Checksum="Explore/ZZ_TEST_CLI/Connectors.Folder.json=01F7DE1034AFFE6376CBB8A37C0B75057194DE9259C93C175E4190FB23209427\n"
INFO[0000] Generated checksum for artifact               Artifact="{AI_SERVICE_CONNECTOR Explore/ZZ_TEST_CLI/Connectors/TestServiceConnector1.AI_SERVICE_CONNECTOR.xml Explore/ZZ_TEST_CLI/Connectors/.TestServiceConnector1.AI_SERVICE_CONNECTOR.json Explore/ZZ_TEST_CLI/Connectors/TestServiceConnector1.AI_SERVICE_CONNECTOR }" Checksum="Explore/ZZ_TEST_CLI/Connectors/TestServiceConnector1.AI_SERVICE_CONNECTOR.xml=5C776BD2AA3D164EAF2284090EC8C62DC57F077CF9E09150768D870A69E7B52F\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/DataQuality.Folder.json Explore/ZZ_TEST_CLI/.DataQuality.Folder.json Explore/ZZ_TEST_CLI/DataQuality.Folder }" Checksum="Explore/ZZ_TEST_CLI/DataQuality.Folder.json=1956F7F3157F3979E29E8C9ACE5F4A36855C6CD0899750F9CA2B55B9B545630F\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/Guides.Folder.json Explore/ZZ_TEST_CLI/.Guides.Folder.json Explore/ZZ_TEST_CLI/Guides.Folder }" Checksum="Explore/ZZ_TEST_CLI/Guides.Folder.json=8DE49ACD0942046B5AF1F173702755CF565E80E80C458B4B0FAF28A70025EBB8\n"
INFO[0000] Generated checksum for artifact               Artifact="{GUIDE Explore/ZZ_TEST_CLI/Guides/Test Conversion Utility.GUIDE.xml Explore/ZZ_TEST_CLI/Guides/.Test Conversion Utility.GUIDE.json Explore/ZZ_TEST_CLI/Guides/Test Conversion Utility.GUIDE }" Checksum="Explore/ZZ_TEST_CLI/Guides/Test\\ Conversion\\ Utility.GUIDE.xml=FCE7EDDF981EF420CEB06E31C743FF9B3CA82D389649CEB9B7877C2F787A1D64\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/IssueGenerateRandomString.Folder.json Explore/ZZ_TEST_CLI/.IssueGenerateRandomString.Folder.json Explore/ZZ_TEST_CLI/IssueGenerateRandomString.Folder }" Checksum="Explore/ZZ_TEST_CLI/IssueGenerateRandomString.Folder.json=7F90959C61F51DE7C32A9A35ABE4401C26D3D102EC8E2CF6C3A0C860BB1E14B6\n"
INFO[0000] Generated checksum for artifact               Artifact="{GUIDE Explore/ZZ_TEST_CLI/IssueGenerateRandomString/TestRandomString.GUIDE.xml Explore/ZZ_TEST_CLI/IssueGenerateRandomString/.TestRandomString.GUIDE.json Explore/ZZ_TEST_CLI/IssueGenerateRandomString/TestRandomString.GUIDE }" Checksum="Explore/ZZ_TEST_CLI/IssueGenerateRandomString/TestRandomString.GUIDE.xml=427C699D39BD6350780BD34735043D80204296CF74D6317B6611D5A1BF8C2506\n"
INFO[0000] Generated checksum for artifact               Artifact="{PROCESS Explore/ZZ_TEST_CLI/IssueGenerateRandomString/Test_GenRandomString.PROCESS.xml Explore/ZZ_TEST_CLI/IssueGenerateRandomString/.Test_GenRandomString.PROCESS.json Explore/ZZ_TEST_CLI/IssueGenerateRandomString/Test_GenRandomString.PROCESS }" Checksum="Explore/ZZ_TEST_CLI/IssueGenerateRandomString/Test_GenRandomString.PROCESS.xml=4370803D48E033F7D924810AE34CA768120109B0465F75C87476007DA3632F04\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/Mappings.Folder.json Explore/ZZ_TEST_CLI/.Mappings.Folder.json Explore/ZZ_TEST_CLI/Mappings.Folder }" Checksum="Explore/ZZ_TEST_CLI/Mappings.Folder.json=C2D6639D2ADCBB696A33037A9C5E11C88DF71212D41A0B935B8BDE76647E4E29\n"
INFO[0000] Generated checksum for artifact               Artifact="{DTEMPLATE Explore/ZZ_TEST_CLI/Mappings/m_Test_Git.DTEMPLATE.zip Explore/ZZ_TEST_CLI/Mappings/.m_Test_Git.DTEMPLATE.json Explore/ZZ_TEST_CLI/Mappings/m_Test_Git.DTEMPLATE }" Checksum="Explore/ZZ_TEST_CLI/Mappings/m_Test_Git.DTEMPLATE.zip=1CB5FD2E362F282139B811411059B399D9A5CE5A9B92B7B8F32BA2D0FAB02E53\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/Mapplets.Folder.json Explore/ZZ_TEST_CLI/.Mapplets.Folder.json Explore/ZZ_TEST_CLI/Mapplets.Folder }" Checksum="Explore/ZZ_TEST_CLI/Mapplets.Folder.json=A329FCBFD7CF575DCAB566642B73A143AC6F6843E58B6ED37BF8C279FC9C531B\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/MassIngestion.Folder.json Explore/ZZ_TEST_CLI/.MassIngestion.Folder.json Explore/ZZ_TEST_CLI/MassIngestion.Folder }" Checksum="Explore/ZZ_TEST_CLI/MassIngestion.Folder.json=123DE64D9E90EC60137FD7E5E42B4F49E233D27748676B7B1CC3E774EF36DFD5\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/ProcessObjects.Folder.json Explore/ZZ_TEST_CLI/.ProcessObjects.Folder.json Explore/ZZ_TEST_CLI/ProcessObjects.Folder }" Checksum="Explore/ZZ_TEST_CLI/ProcessObjects.Folder.json=0CA59C7DA6BE5F9C6CD7229BC66FC9C267D774F19EFFE3A922F384C6D186D155\n"
INFO[0000] Generated checksum for artifact               Artifact="{PROCESS_OBJECT Explore/ZZ_TEST_CLI/ProcessObjects/TestConfigurationProperty.PROCESS_OBJECT.xml Explore/ZZ_TEST_CLI/ProcessObjects/.TestConfigurationProperty.PROCESS_OBJECT.json Explore/ZZ_TEST_CLI/ProcessObjects/TestConfigurationProperty.PROCESS_OBJECT }" Checksum="Explore/ZZ_TEST_CLI/ProcessObjects/TestConfigurationProperty.PROCESS_OBJECT.xml=C2871E61ADD1338B982A645FC367DF9087DEDD26E2219BBB8E6DB55D8754BEC1\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/Processes.Folder.json Explore/ZZ_TEST_CLI/.Processes.Folder.json Explore/ZZ_TEST_CLI/Processes.Folder }" Checksum="Explore/ZZ_TEST_CLI/Processes.Folder.json=F5A36A1A4C919420EA1216B40F26370FCF9EE7F3029D3D9358FF7F06DE1E8205\n"
INFO[0000] Generated checksum for artifact               Artifact="{PROCESS Explore/ZZ_TEST_CLI/Processes/MockupEcho.PROCESS.xml Explore/ZZ_TEST_CLI/Processes/.MockupEcho.PROCESS.json Explore/ZZ_TEST_CLI/Processes/MockupEcho.PROCESS }" Checksum="Explore/ZZ_TEST_CLI/Processes/MockupEcho.PROCESS.xml=96A832EDF39B662399CC576A156848B70A1CCCA4CA713182A64EDD31E3729469\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/TaskFlows.Folder.json Explore/ZZ_TEST_CLI/.TaskFlows.Folder.json Explore/ZZ_TEST_CLI/TaskFlows.Folder }" Checksum="Explore/ZZ_TEST_CLI/TaskFlows.Folder.json=A59C4EE6EA1208D35414DD379C9282E76BB37DB623C842D8AE3A455C972A3330\n"
INFO[0000] Generated checksum for artifact               Artifact="{Folder Explore/ZZ_TEST_CLI/Tasks.Folder.json Explore/ZZ_TEST_CLI/.Tasks.Folder.json Explore/ZZ_TEST_CLI/Tasks.Folder }"
```


## Implementation Notes

### Files created / modified

- `cmd/package.go` - new file (~300 lines); contains `newPackageCmd`, `newPackageExpandCmd`,
  `newPackageCreateCmd`, and private helpers `extractZIPEntries`, `expandNestedZIP`,
  `prettyJSON`, `createNestedZIP`, `generatePackageChecksum`
- `cmd/root.go` - added `rootCmd.AddCommand(newPackageCmd())` in `init()`

### Checksum format

The `exportPackage.chksum` generated by `create` matches the format produced by the
reference Informatica CLI (confirmed via `testdata/imports/ZZ_TEST_CLI_Packaged/`):

- Algorithm: SHA-256, uppercase hex
- Format: sorted `path=HASH` pairs, one per line, no header/timestamp
- Spaces in file paths escaped as backslash-space (`\` followed by a space)
- The checksum file itself is not hashed

The IICS server-generated packages include additional obfuscated ID entries in the chksum
whose derivation algorithm is unknown. The reference CLI packages do not include them, and
IICS accepts the reference CLI format for import - so our format is correct.

### JSON pretty-printing

All JSON files are pretty-printed (2-space indent) during `expand`. This changes their
SHA-256 hashes from the originals. `create` always regenerates `exportPackage.chksum`
from the actual file contents, so the round-trip ZIP is self-consistent.

Validation with a real IICS instance is needed to confirm IICS accepts the import. If
IICS rejects due to checksum differences, a `--no-pretty-print` flag can be added to
`expand` as a minimal fix.

### Dot files

The reference Informatica CLI generates `.asset.json` sidecar files during expand and
reads them during package to reconstruct `exportMetadata.v2.json`. Our implementation
does not generate dot files - instead it preserves the existing `exportMetadata.v2.json`
from the IICS export during expand and includes it unchanged during create. This is
simpler and equally correct for round-trip workflows.

Dot files found in the source directory during `create` are silently skipped.

### Overwrite behavior

- `expand`: non-empty target directory is an error unless `--clean` is given; with
  `--clean`, prompts for confirmation (suppressed by `--yes`)
- `create`: existing output file is an error unless `--force` is given

### Future selective packaging

The future filtering CR (`--include`/`--exclude`/`--match-file`) can use
`exportMetadata.v2.json` as the sole metadata source. One pass builds:

- `guid -> entry` map for O(1) dependency resolution via `objectRefs`
- `(path, objectName, objectType) -> entry` map for O(1) reverse-lookup from filenames

Dot files are not needed. See the architecture note in the implementation plan for details.
