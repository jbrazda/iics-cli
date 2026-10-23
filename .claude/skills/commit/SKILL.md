---
name: commit
description: Commit work in the iics-cli repo with the project's checks - branch guard (dev only), gofmt, go vet, golangci-lint, tests, docs and completions, CR and bug file moves, then a conventional commit without a Co-Authored-By trailer. Use when asked to commit, save, or check in changes in this repo, including "commit and push".
---

# Commit

Follow these steps in order. Stop and report on any failure.

## 1. Branch guard

Run `git branch --show-current`. Continue only on `dev`; otherwise stop and ask.

## 2. Checks

```bash
gofmt -s -w .
go vet ./...
golangci-lint run ./...
go test ./...
```

Fix what they report, then run them again. If `go.mod` changed, check that its
Go version matches `go-version` in `.github/workflows/*.yml`.

## 3. Completeness

Check the diff (`git status --short`, `git diff --stat`) against these rules:

- A command or flag was added or changed: `docs/documentation/<resource>.md`
  and the `README.md` commands table are updated, and `make completions` was
  run. Commit the `completions/` changes with the code.
- A user-visible fix or feature: `CHANGELOG.md` has an entry under
  `## [Unreleased]`.
- A CR was implemented: its file moves from `docs/ChangeRequests/new/` to
  `pending/` in the same commit. A bug was fixed: same for `docs/issues/`.
  Move files to `completed/` only when the user confirms.
- Markdown files were changed: they lint clean (`markdown-lint` skill).

Report anything missing and fix it, or ask if it needs a decision.

## 4. Stage and commit

- Stage the files of this change by path (`git add <paths>`). Do not use
  `git add -A`; leave unrelated or unknown files unstaged and mention them.
- Write a conventional commit message (`feat:`, `fix:`, `docs:`, `chore:`,
  `refactor:`, `test:`; scope optional, CR number in the subject when there is
  one, for example `feat(agent): ... (CR-0049)`). The body says why.
- No `Co-Authored-By` trailer (project rule; it overrides the default
  attribution).
- Commit with a heredoc (`git commit -F - <<'EOF'`), then show
  `git log --oneline -1` and `git status --short`.

## 5. Push

Push only when the user asked for it: `git push origin dev`.
