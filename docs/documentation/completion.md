# completion

Generate shell completion scripts for `iics`.

Completion scripts enable tab-completion of commands, subcommands, and flags in your shell.
They are generated dynamically from the installed binary, so they automatically reflect
the commands and flags of the version you have installed.

## Synopsis

```bash
iics completion <shell>
```

## Subcommands

| Subcommand   | Description                            |
| ------------ | -------------------------------------- |
| `bash`       | Generate bash completion script        |
| `zsh`        | Generate zsh completion script         |
| `fish`       | Generate fish completion script        |
| `powershell` | Generate PowerShell completion script  |

---

## bash

### One-time load (current session only)

```bash
source <(iics completion bash)
```

### Permanent installation - Linux

```bash
iics completion bash > /etc/bash_completion.d/iics
```

Then start a new shell or run `source /etc/bash_completion.d/iics`.

### Permanent installation - macOS

Requires `bash-completion@2` (the macOS system bash does not support dynamic completion):

```bash
brew install bash-completion@2
iics completion bash > $(brew --prefix)/etc/bash_completion.d/iics
```

Add to `~/.bash_profile` if not already present:

```bash
[[ -r "$(brew --prefix)/etc/profile.d/bash_completion.sh" ]] && \
  source "$(brew --prefix)/etc/profile.d/bash_completion.sh"
```

---

## zsh

### One-time load (current session only)

```bash
source <(iics completion zsh)
```

### Permanent installation

If shell completion is not already enabled, add this to `~/.zshrc`:

```bash
autoload -U compinit; compinit
```

Then install the completion file onto your `fpath`:

```bash
iics completion zsh > "${fpath[1]}/_iics"
```

Start a new shell or reload:

```bash
source ~/.zshrc
```

### macOS with oh-my-zsh

```bash
iics completion zsh > ~/.oh-my-zsh/completions/_iics
```

---

## fish

### One-time load (current session only)

```bash
iics completion fish | source
```

### Permanent installation

```bash
iics completion fish > ~/.config/fish/completions/iics.fish
```

Completions are loaded automatically by fish on the next shell start.

---

## powershell

### One-time load (current session only)

```powershell
iics completion powershell | Out-String | Invoke-Expression
```

### Permanent installation

Append to your PowerShell profile (creates the file if it does not exist):

```powershell
iics completion powershell >> $PROFILE
```

---

## Pre-generated scripts

The `completions/` directory in the repository contains pre-generated scripts for all
supported shells. These are regenerated automatically when running `make completions` or
`make all`, and must be kept in sync with the command set.

| File                    | Shell       |
| ----------------------- | ----------- |
| `completions/iics.bash` | bash        |
| `completions/iics.zsh`  | zsh         |
| `completions/iics.fish` | fish        |
| `completions/iics.ps1`  | PowerShell  |

To regenerate after adding or changing commands:

```bash
make completions
```

## See also

- [profile](profile.md) - manage connection profiles
- [login](login.md) - authenticate and cache a session
