# CR-0047: Connection management, connector metadata, project and folder PATCH

## CR Type

- [x] New resource (`connector`)
- [x] New subcommand (`connection test`, `search`, `objects`, `metadata`)
- [x] Enhancement (`connection get`, `connection update`, `project update`, `folder update`, menu)

## Problem

- `connection update` sends PUT; the V2 API updates a connection with
  `POST /api/v2/connection/<id>`. JSON bodies lack the V2 `@type`
  discriminator (same cause as BUG-0015).
- The `Connection` struct misses most documented fields, and there is no way
  to test a connection, list its source/target objects, read runtime
  attribute metadata, or search connections by runtime environment.
- The V2 `connector` resource (available connectors and their attribute
  metadata) is not covered.
- `project update` sends PUT; the V3 API uses PATCH. Folder update already
  uses PATCH, but name-based URIs are not percent-encoded and an empty
  success body prints an empty name and ID.
- There is no interactive way to rename a project or folder or change its
  description.

References:

- [Connections (V2)](https://developer.informatica.com/tpn/v2.0/reference/connectivity)
- [Connectors (V2)](https://developer.informatica.com/tpn/v2.0/reference/connector)
- [Updating a project](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/projects-and-folders/updating-a-project.html)
- [Updating a folder](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/projects-and-folders/updating-a-folder.html)

## JSON vs XML

The TPN pages only show `application/xml`. Live calls against a DEV org on
2026-10-19 with `Accept: application/json` returned
`application/json;charset=UTF-8` for every endpoint below, so the CLI uses
JSON throughout. Findings that differ from the TPN docs:

- `GET /connector` returns extra fields: `orgId`, `description`,
  `updateTime`, `pluginId`, `guid`, `sdkType`, `displayName`. `type` and
  `publisher` are often absent.
- `GET /connector/metadata?connectorName=<name>` takes the connector
  **name** (for example `SqlServer`, `Amazon S3 v2`), matches by prefix and
  returns an **array** of `connectorMetadata`. `attributes[].list` is an
  array of strings, not a string. CCI connectors return a single
  `agentGroupId` attribute plus `cciMeta`, a JSON-encoded string that holds
  the real attribute definitions.
- `GET /connection/test/<id>` returns `{"@type":"success","description":"..."}`.
- `GET /connection/{source|target}/<id>` returns `connListItem` objects;
  `searchPattern` is a case-insensitive substring match.
- `GET /connection/{source|target}/<id>/metadata` returns
  `runtimeAttribute` objects (`maxLength` int, `mandatory` and
  `sessionVarAllowed` bool, `possibleValues` string array, extra
  `uniqueName`). Connectors without runtime attributes return
  `403 APP_13405 The connection has no object.`
- `GET /connection` ignores the `type` and `name` query parameters; the
  filters are applied client-side.

## Project and folder PATCH findings

Verified against the DEV org on 2026-10-19:

- PATCH by name and by ID returns `204 No Content`; the CLI reports the
  identifier it used.
- `PATCH /public/core/v3/projects/<project ID>/folders/<folder ID>` returns
  `404 No static resource`, although it is documented. `/folders/<folder ID>`
  works for folders in any project, so `folder update --project-id` is
  deprecated and ignored.
- Object IDs from the objects API equal project and folder IDs.

## Desired Change

### connection

- `get --id | --name`.
- `update --partial` sends `Update-Mode: PARTIAL`; update uses POST.
- create and update set `"@type":"connection"`.
- `test --id | --name`.
- `search --type <uiType> --runtime-id | --agent-id`.
- `objects --id | --name [--target] [--search-pattern]`.
- `metadata --id | --name [--target]`.
- `list --type` and `--name` filter client-side.

### connector (new)

- `list`.
- `metadata --name <connectorName> [--exact]`: one match shows its
  attributes (CCI attributes parsed from `cciMeta`); several matches show a
  summary.

### project and folder

- `project update` uses PATCH by `--id` or by current `--project-name`.
- `folder update` percent-encodes names, validates flag combinations and
  updates by ID through `/folders/<id>`.
- Both accept `--interactive/-i`: pick a project (and folder) from the
  objects API with a name filter, edit name and description, confirm.
- Menu group "Projects and folders" with "Edit project" and "Edit folder".

### project and folder list (added after review)

- `project list [--name REGEX] [-i]` and
  `folder list [--name REGEX] [--project-name | --project-id | --pick-project] [-i]`.
- Columns: path, description, created, created by, updated, updated by, ID.
- The regex matches the full path (`project/folder`), case-insensitive.
- Created time and created by are only available from the internal FRS API
  (`/frs/api/v1/Projects`, `/Folders`, `/Projects('<id>')/Folders`, header
  `IDS-SESSION-ID`), which the IICS web UI uses and Informatica does not
  document. FRS returns user IDs; they are resolved through the users API.
  When FRS fails, the commands fall back to the objects API without the
  created columns and print a note. The developer approved this dependency.
- Menu: "List projects", "List folders", "List folders in a project".

## Verification

- `go test ./...` covers each new client method.
- Live smoke test against the DEV org for each command: connector list and
  metadata, connection get/test/objects/metadata/search/list filters,
  connection create and partial update, project and folder PATCH by name
  and ID. Throwaway objects were deleted afterwards.
- Not run live: the interactive pickers (`-i`, menu entries) need a manual
  check in a terminal.
