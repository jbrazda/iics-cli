# Interactive prompts

Commands that run interactively (wizards such as `user create --interactive`,
`user edit`,
`group create -i`, `environment create`, and pickers shown when `--id` or
`--name` is omitted) render terminal forms when stdin and stderr are both a
terminal.

## Keys

| Prompt       | Keys                                                                 |
| ------------ | -------------------------------------------------------------------- |
| Text input   | Type, `Enter` to accept (an empty value keeps the shown default)     |
| Single list  | Arrows or `j` / `k` to move, `/` to filter, `Enter` to choose, `Esc` to cancel |
| Checklist    | `Space` or `x` to toggle, `/` to filter, `Ctrl+A` to select all, `Enter` to confirm |
| Yes / No     | Left / right or `y` / `n`, `Enter` to confirm                        |
| Any prompt   | `Ctrl+C` cancels the command                                         |

Checklists open with the current values already checked (for example a user's
current groups and roles), so confirming without changes keeps them.

The role privilege editor (`role edit`) has its own grid keys; see
[role](role.md#interactive-privilege-editor).

## Accessible mode

Set `IICS_ACCESSIBLE=1` to render prompts as plain, line-by-line questions that
work with screen readers:

```bash
IICS_ACCESSIBLE=1 iics user create --interactive
```

```powershell
$env:IICS_ACCESSIBLE = "1"; iics user create --interactive
```

## Piped input

When stdin or stderr is not a terminal (for example answers piped in by a
script, or stderr redirected to a file), prompts fall back to the line-based
format:

| Prompt      | Answer format                                                   |
| ----------- | --------------------------------------------------------------- |
| Text input  | One line; empty keeps the default shown in brackets             |
| Single list | Option number; `0`, `q` or empty cancels                        |
| Yes / No    | `y`, `yes`, `n`, `no`; empty uses the default                   |

Wizards with checklists (`user create --interactive`, `user edit`, `group
create`/`update -i`, `environment create -i`, `role edit`) require a terminal
and report an error instead of prompting; use `--from-file` or the
non-interactive commands (for example `user update-roles`) in scripts.

## Confirmations

Destructive commands (`delete`, `package expand --clean`,
`environment configs set`) ask a plain `[y/N]` question even on a terminal,
so an answer can always be piped (`echo y | iics ...`). Only `y` or `Y`
proceeds. Use `--yes` / `-y` to skip the question.
