# CR-0048: Esc and q cancel prompts; single-key confirmations

## CR Type

- [x] Enhancement (interactive prompts, menu)

## Problem

- Prompts that read a line from the terminal (`[y/N]` confirmations, the
  menu's "Press Enter to return" pause, `login` profile creation,
  `role get` privilege prompt) echo arrow keys as `^[[A` / `^[[B`.
- In a command started from the menu, pickers and text prompts can only be
  left with `Ctrl+C`; `Esc` and `q` do nothing.

## Pattern

The common terminal UI convention (fzf, lazygit, k9s, gh):

| Key      | List, select, yes/no | Text input   | List filter open or applied |
| -------- | -------------------- | ------------ | --------------------------- |
| `Esc`    | Cancel the prompt    | Cancel       | Close or clear the filter   |
| `q`      | Cancel the prompt    | Typed letter | Typed letter                |
| `Ctrl+C` | Cancel               | Cancel       | Cancel                      |

Single-key questions read keys in raw mode: nothing is echoed, escape
sequences (arrows, function keys) are skipped, `Esc` / `q` mean no or back.

## Desired Change

- `internal/tui/keys.go`:
  - `ReadKey` reads one key in raw mode and skips escape sequences, also when
    they arrive in the same read as a key.
  - `KeyConfirm` asks a `[y/N]` / `[Y/n]` question answered with one key.
  - `RunForm` runs a huh form with a Bubble Tea message filter that turns
    `Esc` and `q` into a cancel, unless the focused field has an enabled
    binding for `Esc` (filter open or applied); `q` passes through in text
    inputs.
- `Prompter` (text, select, confirm, multi-select), the review select, the
  ACL editor principal picker and the role privilege editor use `RunForm`
  (`Esc`, `q` in lists, `Ctrl+C`).
- Wizards (user, profile, group, runtime environment, config) and the
  `role create` form use `RunWizard`: `Esc` or `Ctrl+C` cancels; `q` is
  always typed because these forms contain text inputs.
- Both run the huh form inside a small Bubble Tea model that applies the
  cancel keys and renders a hint line under the form's key help:
  `esc, q (in lists) or ctrl+c to cancel` or `esc or ctrl+c to cancel`.
  Accessible mode keeps huh's line prompts.
- On a terminal, `confirmAction`, `user delete`, `login` (create missing
  profile), `role get` (show privileges) and the menu pause use single keys.
  Without a terminal the line format is kept so answers can be piped.
- Menu pause: `Enter` or `Esc` returns to the menu, `q` or `Ctrl+C` quits.

## Verification

- Unit tests: `parseKey` (arrows, CSI with parameters, SS3, alt+key, UTF-8),
  `cancelKeysFilter` for select (including filter mode), input and confirm.
- Pseudo-terminal checks: arrow keys then `n` or `y` on a delete confirmation;
  `Esc` on the `folder list --pick-project` filter prompt; `q` on its project
  list; `profile add` with a scratch config: `q` typed into the text field,
  `Esc` cancels, config unchanged; hint lines shown.
- Manual check in a real terminal still recommended for the menu pause.
