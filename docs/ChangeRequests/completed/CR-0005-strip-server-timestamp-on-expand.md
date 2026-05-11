# CR-0005: Strip `CurrentServerDateTime` from XML Assets During `package expand`

---

## CR Type

- [ ] **New resource** - brand-new command
- [x] **Enhancement** - add behavior to an existing command (`iics package expand`)
- [ ] **Bug fix**

---

## Status

**Implemented / pending developer confirmation** - code is in `cmd/package.go`.
R1 and R2 complete. Removal is confirmed safe - round-trip import to a live IICS dev org succeeds with zero errors and `checksumValid=true`.

---

## Problem

Every IICS export package contains XML asset files
(`*.PROCESS.xml`, `*.GUIDE.xml`, `*.AI_CONNECTION.xml`, `*.AI_SERVICE_CONNECTOR.xml`,
`*.PROCESS_OBJECT.xml`, etc.). Each file ends with a server-generated element:

```xml
<types1:CurrentServerDateTime>2026-03-08T17:21:02.373Z</types1:CurrentServerDateTime>
```

This timestamp is injected by the IICS server at export time. It reflects the moment the
server served the file, **not** when the asset was last modified. The asset's actual change
dates are captured separately in stable sibling elements:

```xml
<types1:CreationDate>2026-03-08T16:42:30Z</types1:CreationDate>
<types1:ModificationDate>2026-03-08T16:43:16Z</types1:ModificationDate>
<types1:PublicationDate>2026-03-08T16:43:28Z</types1:PublicationDate>
```

Because `CurrentServerDateTime` changes on every export - even when no designer made any
change to the asset - every export marks every XML file as modified in git. On a project
with dozens of assets, code reviewers must sift through hundreds of single-line diffs that
carry no meaningful information, making pull request reviews noisy and error-prone.

### Element location in the file

The element always appears as the last child of `<aetgt:getResponse>`, after the closing
`</types1:Item>` tag, on its own line:

```xml
   </types1:Item>
   <types1:CurrentServerDateTime>2026-03-08T17:21:02.373Z</types1:CurrentServerDateTime>
</aetgt:getResponse>
```

It appears **exactly once per file** in all observed XML asset files in
`testdata/imports/ZZ_TEST_CLI_Unpacked/` and `testdata/imports/ZZ_TEST_CLI_Packaged/`.

---

## Desired Change

During `iics package expand`, after each `.xml` file is read from the ZIP and before it is
written to disk, remove the `<types1:CurrentServerDateTime>` line so that the git snapshot
of the expanded package only reflects genuine design changes.

The removal must be transparent to `iics package create`: the assembled ZIP should be
importable by IICS regardless of whether the element is present (see Research Tasks below).

---

## Approach Analysis

Three approaches were considered.

### Option A - Regex line removal (recommended)

Remove any line that contains a `CurrentServerDateTime` element, regardless of the XML
namespace prefix. The namespace prefix (`types1:` in current exports) is not guaranteed to
remain stable across Informatica versions or export configurations, so the regex matches
`\w+:` to cover any prefix.

```go
// serverTimestampRE matches the CurrentServerDateTime element with any namespace prefix,
// including the surrounding whitespace and line ending, so ReplaceAll leaves no blank line.
var serverTimestampRE = regexp.MustCompile(
    `(?m)^[ \t]*<\w+:CurrentServerDateTime>[^<]*</\w+:CurrentServerDateTime>[ \t]*\r?\n?`,
)

func stripServerTimestamp(data []byte) []byte {
    return serverTimestampRE.ReplaceAll(data, nil)
}
```

**Why this is safe here:**

- The element name `CurrentServerDateTime` is unique in the IICS XML vocabulary; it does not
  appear inside asset designs, only in the outer `getResponse` envelope added by the server.
- The element is always on its own indented line with no sibling content on the same line.
- The regex matches any namespace prefix (`\w+:`), so the code is not sensitive to Informatica
  changing or reordering namespace prefix bindings across export tool versions.
- No XML reformation occurs; every other byte of the file is preserved exactly as the server
  wrote it.
- The regex is compiled once at package level - no per-call overhead.

**Integration point:** `extractZIPEntries` in `cmd/package.go` already applies a
per-extension transformation (JSON pretty-printing). The XML stripping fits the same pattern:

```go
// Strip volatile server timestamp from XML assets (CR-0005)
if strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
    data = stripServerTimestamp(data)
}
```

**Risk:** Low. The transformation is additive to the expand step; `create` re-reads files
from disk, so the assembled ZIP simply omits the element. If IICS rejects an import without
the element a `--no-strip-timestamp` escape hatch can be added as a minimal follow-up.

---

### Option B - Go `encoding/xml` full parse/re-emit

Parse the XML into a generic DOM, remove the matching element, re-serialize.

**Why not recommended:**

- Go's `encoding/xml` does not preserve document formatting on round-trip: attribute order,
  indentation, XML declarations, and namespace declarations are all potentially rewritten.
- Changing the serialized form of every XML file other than the target element would make
  the initial commit noise (one-time churn) nearly as large as the ongoing problem being
  solved.
- The resulting files would be harder to compare to Informatica's original format.

---

### Option C - XSLT identity transform via Apache Ant

Use an XSLT stylesheet with an identity transform that drops the target element, invoked via
Ant from inside the CLI using `exec.Cmd`.

**Why not recommended:**

- Requires Java and Apache Ant installed on every developer machine and CI runner - this
  CLI has no external runtime dependencies and should stay that way.
- JVM startup adds latency to every `expand` invocation.
- Adds substantial complexity (subprocess management, stdout/stderr capture, error mapping)
  for a transformation that Option A solves in ten lines of Go.

---

## Research Tasks (required before implementation)

### R1 - Informatica Knowledge Base search

Search Informatica documentation and Knowledge Base for any mention of
`CurrentServerDateTime` in the context of the export/import XML format:

- Does Informatica's importer read or validate this field during import?
- Is it referenced in any Informatica-provided XSLT or schema (`.xsd`) files?
- Are there any known community reports of import failures related to this element?

Search terms:

- `CurrentServerDateTime` site:docs.informatica.com
- `CurrentServerDateTime` site:knowledge.informatica.com
- `avrepository.xsd CurrentServerDateTime`
- `getResponse CurrentServerDateTime import`


Document findings in the **Research Findings** section below before implementation begins.

### R2 - Round-trip import test

Perform the following end-to-end test against a real IICS development org:

1. Export a package containing at least one asset of each XML type:
   `PROCESS`, `GUIDE`, `PROCESS_OBJECT`, `AI_SERVICE_CONNECTOR`, `AI_CONNECTION`.
2. Run `iics package expand` on the exported ZIP.
3. Manually remove (or use a script to remove) the `CurrentServerDateTime` element line
   from each `.xml` file in the expanded directory.
4. Run `iics package create` to re-assemble the ZIP.
5. Import the reassembled ZIP into the same or a different IICS dev org.
6. Verify:
   - Import completes without errors or warnings.
   - All assets are present and functional in the org.
   - Asset designs, configurations, and metadata are identical to the originals.

Use `testdata/imports/ZZ_TEST_CLI_Unpacked/` as the base for the test. The test package
covers all affected XML asset types and is already committed to the repository.

### R3 - Check `expandNestedZIP` path

Confirm whether nested ZIP files (e.g., `m_Test_Git.DTEMPLATE.zip`, `mt_Test_Git.MTT.zip`)
contain XML files with the same timestamp element. If they do, `expandNestedZIP` in
`cmd/package.go` must also apply `stripServerTimestamp` to its XML entries.

---

## Research Findings

| Item | Finding | Source |
| ---- | ------- | ------ |
| R1 - Informatica KB | **Safe to remove.** No mention of `CurrentServerDateTime` in `docs.informatica.com`, `knowledge.informatica.com`, or any Informatica community or GitHub source. The element is part of the Active Endpoints repository HTTP response envelope (`<aetgt:getResponse>`), not the asset design. It is a sibling of `<types1:Item>` (the actual asset content), not nested inside it. The original Informatica IICS Asset Management CLI dot-file sidecars (`.asset.json`) contain no trace of this field, confirming the official tooling does not use it when re-assembling packages. The field name - "Current**Server**DateTime" - denotes the server clock at response time, which has no meaning to an importer. | Web search across `docs.informatica.com`, `knowledge.informatica.com`, GitHub; structural analysis of `testdata/imports/ZZ_TEST_CLI_Extracted` dot-files; [jbrazda/icai-fault-alert-service](https://github.com/jbrazda/icai-fault-alert-service/blob/master/src/ipd/Explore/Alerting/ProcessObjects/alert-config.PROCESS_OBJECT.xml) |
| R2 - Import test result | **SUCCESSFUL.** Full round-trip on dev org (2026-05-11): export 12 objects (31 files) -> expand (strip applied to 13 XML files) -> create -> import. All 28 imported objects reported SUCCESSFUL. Server returned `checksumValid=true`. No errors or warnings. | Live IICS dev org import job `9DE4KJ6zLdLbc7BcibqWmz` |
| R3 - Nested ZIP XML files | Confirmed: nested ZIPs (DTEMPLATE, MTT) do not contain XML files with this element in observed testdata. `expandNestedZIP` patched defensively. | testdata/imports |

---

## Scope (post-research)

### Files to MODIFY

```text
cmd/package.go    # add stripServerTimestamp(), call it in extractZIPEntries()
                  # and conditionally in expandNestedZIP() based on R3 findings
```

### Files to READ (context only - do NOT modify)

```text
docs/CLAUDE.md
cmd/package.go    # existing expand logic; integration points at lines ~204-213
testdata/imports/ZZ_TEST_CLI_Unpacked/Explore/ZZ_TEST_CLI/Processes/MockupEcho.PROCESS.xml
```

### Files to CREATE

None. No new command, no new flags (unless R2 reveals a need for `--no-strip-timestamp`).

---

## Acceptance Criteria

- [x] R1 - Informatica KB search complete; element confirmed safe to remove (see Research Findings).
- [x] R2 - Round-trip import test: SUCCESSFUL on dev org 2026-05-11 (all 28 objects, checksumValid=true, zero errors).
- [x] R3 - Nested ZIP XML files confirmed; `expandNestedZIP` patched defensively.
- [ ] `iics package expand` produces `.xml` files with no `CurrentServerDateTime` element (any namespace prefix).
- [ ] `iics package create` on the stripped expanded directory produces a ZIP that IICS
  accepts without errors.
- [ ] All other XML content is byte-for-byte identical to the original (verified by diff).
- [ ] `stripServerTimestamp` is a pure function with no side effects; tested in isolation.
- [ ] Repeated `expand` + `create` + `expand` cycles produce identical XML files (stable
  output - no new diffs after the first expand).
- [ ] `go build ./...` succeeds.
- [ ] `go test ./...` passes.
- [ ] `golangci-lint run ./...` reports no new issues.
- [ ] No unrelated code was modified.

---

## Do NOT

- Use `encoding/xml` to parse and re-emit the full XML document.
- Introduce any external process dependency (Java, Ant, xsltproc).
- Add `os.Exit()` calls.
- Modify any file outside `cmd/package.go`.
- Implement before R1 and R2 research tasks are complete and documented.
- Add `Co-Authored-By` trailers to commit messages.
