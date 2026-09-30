# Interactive prompts

Commands that run interactively (wizards such as `user create --interactive`,
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
| Checklist   | Comma-separated numbers (`1,3,5`); empty keeps defaults (marked `*`); `0` selects none |
| Yes / No    | `y`, `yes`, `n`, `no`; empty uses the default                   |
| Time zone   | Search text, then the number of a match; empty keeps the current value; `0` clears |

Commands that require a terminal (for example `role edit`) report an error
instead of prompting.
