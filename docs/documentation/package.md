# package

Work with IICS export package files locally. No authentication or API calls are required - all
operations are performed on the local file system.

## Synopsis

```bash
iics package <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
| ---------- | ----------- |
| `expand`   | Extract an IICS export ZIP package to a directory |
| `create`   | Assemble a new IICS export ZIP package from a directory |

---

## package expand

Extracts the contents of an IICS export ZIP package to a target directory. JSON files are
pretty-printed (2-space indent) to produce clean diffs in version control. Nested ZIP files
(e.g. `m_Test_Git.DTEMPLATE.zip`) can be recursively expanded into directories.

The typical workflow for version-controlling IICS assets is:

1. `iics export run` - export assets to a ZIP package
2. `iics package expand` - extract the package to a versioned directory
3. Commit the directory to git
4. `iics package create` - reassemble the ZIP from the directory
5. `iics import run` - import the package to a target org

### Flags

| Flag | Short | Type | Required | Default | Description |
| ---- | ----- | ---- | -------- | ------- | ----------- |
| `--file` | `-f` | string | yes | | Path to the source ZIP package |
| `--target` | `-t` | string | yes | | Destination directory; created if absent |
| `--recursive` | `-r` | bool | | false | Recursively expand nested ZIPs into `<name>.zip/` directories |
| `--clean` | `-c` | bool | | false | Delete target contents before expanding |
| `--yes` | `-y` | bool | | false | Suppress the `--clean` confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Behavior details

**Target directory**

- Created automatically with `os.MkdirAll` if it does not exist.
- If the target contains files and `--clean` is not set, the command returns an error. This
  is intentional: in CI/CD pipelines the source directory should be empty so that deleted or
  renamed assets are detected by `git status`.
- With `--clean`, the command counts and removes existing contents, prompting for confirmation
  unless `--yes` is also given.

**JSON pretty-printing**

All `.json` files are reformatted with 2-space indentation on extraction. This produces
readable, diff-friendly output in git. It also means the SHA-256 hashes stored in
`exportPackage.chksum` will differ from the originals; `package create` always regenerates
the checksum from current file contents.

**Recursive expand (`--recursive`)**

Nested ZIP files (e.g. `Mappings/m_Test_Git.DTEMPLATE.zip`) are expanded into a directory
with the same name including the `.zip` extension
(e.g. `Mappings/m_Test_Git.DTEMPLATE.zip/`). The source zip file is not written to disk.
When `package create` encounters a directory whose name ends in `.zip`, it re-packs it into
a nested ZIP automatically.

**Verbose output (`--verbose`)**

Lists the ZIP entries before extraction and lists the target directory contents after.

### Examples

```bash
# Basic expand
iics package expand \
  --file exports/my-project.zip \
  --target src/iics

# Expand and overwrite an existing directory (with confirmation)
iics package expand \
  --file exports/my-project.zip \
  --target src/iics \
  --clean

# Non-interactive expand (CI/CD)
iics package expand \
  --file exports/my-project.zip \
  --target src/iics \
  --clean --yes

# Recursively expand all nested ZIPs (e.g. DTEMPLATE, MTT assets)
iics package expand \
  --file exports/my-project.zip \
  --target src/iics \
  --recursive --verbose
```

---

## package create

Assembles a new IICS export ZIP package from the contents of a source directory and
regenerates `exportPackage.chksum` to match. The resulting ZIP is suitable for import
via `iics import run`.

Dot files (names starting with `.`) and any existing `exportPackage.chksum` are
excluded. Directories whose name ends in `.zip` (produced by `package expand --recursive`)
are re-packed into nested ZIP entries automatically.

### Flags

| Flag | Short | Type | Required | Default | Description |
| ---- | ----- | ---- | -------- | ------- | ----------- |
| `--source` | `-s` | string | yes | | Source directory to package |
| `--target` | `-t` | string | yes | | Output ZIP file path |
| `--force` | `-f` | bool | | false | Overwrite existing output file |

All [global flags](../../README.md#global-flags) apply.

### Checksum file

`exportPackage.chksum` is always regenerated from the actual files in the source directory.
The format matches the Informatica CLI v2 package format:

- One `path=SHA256` entry per file, sorted alphabetically
- Spaces in paths escaped as `\` followed by a space
- SHA-256 hash in uppercase hex
- No header or timestamp line

### Verbose output (`--verbose`)

Prints the list of files added to the ZIP before the summary line.

### Examples

```bash
# Basic create
iics package create \
  --source src/iics \
  --target dist/my-project-reimported.zip

# Overwrite existing output
iics package create \
  --source src/iics \
  --target dist/my-project.zip \
  --force --verbose
```

---

## Round-trip workflow

```bash
# 1. Export from source org
iics export run \
  --artifacts-file objects.txt \
  --export-file-path dist/export.zip \
  --profile dev

# 2. Expand to versioned directory
iics package expand \
  --file dist/export.zip \
  --target src/iics \
  --clean --yes

# 3. Commit to git
git add src/iics
git commit -m "Update IICS assets"

# 4. Reassemble package from versioned sources
iics package create \
  --source src/iics \
  --target dist/import.zip \
  --force

# 5. Import to target org
iics import run \
  --zip-file dist/import.zip \
  --profile prod
```

---

## Package file format

An IICS export package is a ZIP archive with the following structure:

```text
<package-name>.zip
├── exportMetadata.v2.json        # Asset metadata with GUIDs, objectRefs, repoHandles
├── exportPackage.chksum          # SHA-256 checksums for all files
├── ContentsofExportPackage_*.csv # Asset inventory (present in server exports)
├── Explore/
│   └── <Project>/
│       ├── <Project>.Project.json
│       ├── <Folder>.Folder.json
│       └── <Folder>/
│           └── <Asset>.<TYPE>.xml  # or .zip for binary asset types
└── SYS/                          # Present when exported with dependencies
    └── <Connection>.<TYPE>.zip
```

Key files:

- **`exportMetadata.v2.json`** - mandatory for import; contains per-asset metadata
  including `repoInfo.repoHandle` and `objectRefs` arrays
- **`exportPackage.chksum`** - always regenerated by `package create`; IICS validates
  this on import

---

## See also

- [export](export.md) - export assets from IICS to a ZIP package
- [import](import.md) - import a ZIP package into IICS
