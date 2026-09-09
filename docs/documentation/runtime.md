# runtime

Manage IICS runtime environments. Alias: `rt`.

Runtime environments are groups of one or more Secure Agents that execute data integration tasks.

## Synopsis

```bash
iics runtime <subcommand> [flags]
iics rt <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                      |
| ---------- | -------------------------------- |
| `list`     | List runtime environments        |
| `get`      | Get a single runtime environment |
| `create`   | Create a runtime environment     |
| `update`   | Update a runtime environment     |
| `configs`  | Manage Secure Agent group service properties |

---

## runtime list

### Flags

| Flag       | Type         | Default | Description                                              |
| ---------- | ------------ | ------- | ------------------------------------------------------- |
| `--limit`  | int          | 200     | Max results                                             |
| `--skip`   | int          | 0       | Results to skip                                         |
| `--filter` | string array |         | Client-side filter, e.g. `isShared==true`. Repeatable; AND-ed |

`--filter` supports `==` and `!=` on the raw API field names. Note that boolean
fields absent from the response (a `false` value) do not match; filter on the
value that is present.

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column        | Description                          |
| ------------- | ----------------------------------- |
| `id`          | Runtime environment ID              |
| `name`        | Runtime environment name            |
| `federatedId` | Federated ID                        |
| `isShared`    | Whether the environment is shared   |
| `agents`      | Number of Secure Agents in the group |
| `updateTime`  | Last modification timestamp         |

### Examples

```bash
iics runtime list

iics rt list --output json

iics runtime list --filter isShared==true
```

```powershell
iics runtime list

iics rt list --output json

iics runtime list --filter isShared==true
```

---

## runtime get

### Flags

| Flag     | Type   | Description             |
| -------- | ------ | ----------------------- |
| `--id`   | string | Runtime environment ID  |
| `--name` | string | Runtime environment name |

Exactly one of `--id` / `--name` is required; they are mutually exclusive.

All [global flags](../../README.md#global-flags) apply.

### Output

Table mode prints:

1. `Runtime Environment: <name>` followed by a vertical `PROPERTY` / `VALUE`
   listing (`id`, `orgId`, `orgUUID`, `federatedId`, `isShared`, `createdBy`,
   `updatedBy`, create/update timestamps).
2. A `Serverless Config:` section when the environment has one.
3. `Agents (N):` and a table of the member agents with columns `NAME`, `HOST`,
   `PLATFORM`, `VERSION`, `ACTIVE`, `READY`, `UPGRADE`, `FEDERATED ID`,
   `GROUP ID`.

`--output json` / `yaml` / `csv` render the environment record directly.

### Examples

```bash
iics runtime get --id <runtime-id>

iics runtime get --name "My Group"

iics rt get --id <runtime-id> --output json
```

```powershell
iics runtime get --id <runtime-id>

iics runtime get --name "My Group"

iics rt get --id <runtime-id> --output json
```

---

## runtime create

Create a runtime environment (Secure Agent group). Provide a JSON definition
file, or run interactively.

The minimal file is just `{"name": "..."}`; the `@type` discriminators the v2 API
requires (`"runtimeEnvironment"` on the body, `"agent"` on each `agents[]`
element) are added automatically. An explicit `@type` in the file is honored.

### Flags

| Flag                | Type   | Description                                              |
| ------------------- | ------ | ------------------------------------------------------- |
| `--from-file`       | string | JSON file with the runtime environment definition       |
| `--interactive`, `-i` | bool | Prompt for the fields interactively                     |

The interactive wizard runs when `-i` is given, or when `--from-file` is omitted
and stdin is a terminal. With both `--from-file` and `-i`, the file pre-fills the
prompts. Without a file and without a terminal, `--from-file` is required.

### Interactive prompts

1. `Environment Name` (required).
2. `Is shared` (y/N).
3. Agent management menu - `Add agents` lists the currently unassigned Secure
   Agents to pick from; `Remove agents` removes from the working selection;
   `Done` creates the environment with the selected agents.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics runtime create --from-file my-runtime.json

# interactive
iics runtime create
iics runtime create -i --from-file seed.json
```

---

## runtime update

The `@type` discriminator is added automatically, as for `create`.

### Flags

| Flag          | Type   | Required | Description                      |
| ------------- | ------ | -------- | -------------------------------- |
| `--id`        | string | yes      | Runtime environment ID           |
| `--from-file` | string | yes      | JSON file with updated fields    |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics runtime update --id <runtime-id> --from-file updated-runtime.json
```

---

## runtime configs

Manage Secure Agent group service property overrides. Uses
`GET` / `PUT /api/v2/runtimeEnvironment/<id>/configs`. The Secure Agent group ID
is the runtime environment ID.

### runtime configs get

Show the service property overrides for a group.

| Flag        | Type   | Description |
| ----------- | ------ | ----------- |
| `--id`      | string | Secure Agent group (runtime environment) ID |
| `--name`    | string | Secure Agent group (runtime environment) name |
| `--service` | string | Show only this service's properties |

Exactly one of `--id` / `--name` is required.

In table mode, each service is printed as a section of property/value rows. An
empty result prints `No service property overrides.`. `--output json` / `yaml`
render the raw document.

```bash
iics runtime configs get --id <groupId>
iics runtime configs get --name "My Group" --service Data_Integration_Server --output json
```

### runtime configs set

Replace the service property overrides for a group from a JSON file shaped as
`{"<Service_Name>":[{ ...settings... }]}`.

| Flag           | Type   | Description |
| -------------- | ------ | ----------- |
| `--id`         | string | Secure Agent group (runtime environment) ID |
| `--name`       | string | Secure Agent group (runtime environment) name |
| `--from-file`  | string | JSON file with the overrides (required) |
| `--yes` / `-y` | bool   | Skip the confirmation prompt |

Exactly one of `--id` / `--name` is required.

```bash
iics runtime configs set --id <groupId> --from-file props.json --yes
```

## See also

- [agent](agent.md) - manage Secure Agents within runtime environments
