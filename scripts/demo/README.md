# README demo recording

`record.py` records the demo shown in the main [README](../../README.md) as an
[asciinema](https://asciinema.org) cast, `docs/demo/iics-demo.cast`, and
`make demo` renders it to `docs/demo/iics-demo.gif`.

The demo runs real commands against a live org, so the output contains host
names, emails and the organization name. These are masked before the cast is
written, using rules kept in a local file that is never committed.

## Requirements

- Python 3 (standard library only) on Linux or macOS.
- [agg](https://github.com/asciinema/agg) to render the GIF, for example
  `cargo install --git https://github.com/asciinema/agg`.
- `jq` and `column`, used by one scene.
- A profile with access to the org and agent named in `scenes.json`. Use a
  test org: the editor scene cancels its change, but every command runs for
  real.

## Masking rules

Copy `masks.example.json` to `masks.local.json` (ignored by git) and fill in
your values. Every replacement keeps the length of the text it replaces, so
tables stay aligned; shorter results are padded with spaces and longer ones cut.

| Key | Meaning |
| --- | --- |
| `emails.keep` | Email local parts left as they are, such as service accounts. |
| `emails.names` | Fixed fake names for local parts, ideally the same length. |
| `emails.suffixes` | Endings kept after the fake name, such as `.tst`. |
| `emails.other` | Name for any other local part; `{n}` is a counter. |
| `emails.domains` | Regular expressions and replacements for email domains. |
| `rules` | Regular expressions and replacements for the whole output, applied in order after emails. Group references such as `\\1` work. |
| `forbid` | Text that must not remain (case-insensitive). If any is found, the cast is not written. |

Masking runs on the whole output stream, so text split across output chunks
or cut short by a narrow column is still caught. Write rules that also match
truncated values, as the host name rule in the example does.

## Recording

```bash
make demo
```

This builds `iics`, records with `scenes.json` and renders the GIF. To record
only the cast, or into another file:

```bash
PATH="$PWD:$PATH" python3 scripts/demo/record.py scripts/demo/scenes.json /tmp/demo.cast
asciinema play /tmp/demo.cast
```

`--masks FILE` reads the rules from another file, and `--no-mask` records
without masking, for local checks only.

## Scenes

`scenes.json` sets the terminal size, environment (such as `IICS_PROFILE`) and
the scenes, in order:

- `{"comment": "text"}` types a shell comment.
- `{"cmd": "iics ...", "pause": 2.5}` types and runs a command, then waits.
  `show` types different text than what runs.
- `keys` sends keystrokes to an interactive command:
  `{"k": "\r", "before": 0.6}` waits `before` seconds, then sends `k`.
  Use `\u001b` for Esc and `\u001b[B` for the down arrow.

End interactive scenes with a cancel, never an apply, so recording the demo
does not change the org.
