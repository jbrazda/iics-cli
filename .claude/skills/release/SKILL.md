---
name: release
description: Release a new iics-cli version from dev - checks, CHANGELOG, a release PR to main, fast-forward (keeps every commit) or squash merge, tag, GoReleaser, and CI monitoring. Use when asked to release, cut, tag or publish a version, or to merge dev into main for a release.
---

# Release

The release goes from `dev` to `main` through a pull request, which records the
release and runs CI on it. How it lands on `main`:

- **Fast-forward (default):** push `dev` to `main`. Every commit is kept,
  `main` and `dev` stay identical, and GitHub marks the PR as merged because
  its commits are now on `main`. The `main` ruleset requires linear history
  and the CI checks, so the push needs no bypass as long as `dev` has no
  merge commits and CI passed on its head commit.
- **Squash (when the user asks for one commit):** `gh pr merge --squash`, then
  reset `dev` to `main`. Merging `main` back into `dev` instead adds a merge
  commit, and the next fast-forward then needs the admin bypass.

Stop and report on any failure. Ask before every step that pushes.

## 1. Branch guard and sync

```bash
git branch --show-current            # must be dev
git fetch origin
git status --short                   # must be clean
git merge-base --is-ancestor origin/main dev   # fast-forward possible
```

If `dev` is behind `origin/dev`, stop. If `main` is not an ancestor of `dev`
(a squash release or a commit made on `main`), a fast-forward is impossible:
report it and offer squash, or `git rebase origin/main` on `dev` first (that
rewrites `dev`, so it needs `git push --force-with-lease origin dev`; ask).

Check for merge commits the linear-history rule would reject:

```bash
git log --merges --oneline origin/main..dev   # must print nothing
```

If it prints any, report them: the fast-forward then only goes through with
the user's admin bypass. Ask whether to push anyway or squash instead.

## 2. Checks

```bash
gofmt -s -l .                        # must print nothing
go vet ./...
golangci-lint run ./...
go test -race ./...
```

Check that the Go version in `go.mod` matches `go-version` in
`.github/workflows/*.yml`, and that `make completions` leaves no changes.

## 3. Pending CRs and bugs

List the files in `docs/ChangeRequests/pending/` and `docs/issues/pending/`
with the commit that implemented each (`git log --oneline -- <file>`). Ask the
user which ones are confirmed. Move only those with `git mv` to `completed/`
and commit: `chore: move <ids> to completed`.

## 4. Version

Show the latest tag (`git tag --sort=-version:refname | head -1`) and the
commits since it (`git log --oneline <tag>..dev`). Suggest a version: minor
for features or breaking changes, patch for fixes only. Ask the user to
confirm; it must match `v\d+\.\d+\.\d+`.

## 5. CHANGELOG

In `CHANGELOG.md`, rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`
(today) and add an empty `## [Unreleased]` above it. Edit in place; do not
rewrite the file. Check that breaking changes to commands, flags or output
fields are called out. Lint the file (`markdown-lint` skill), then commit
`chore: update CHANGELOG for vX.Y.Z release` (no `Co-Authored-By` trailer)
and push `dev`.

## 6. Release PR

```bash
gh pr create --base main --head dev --title "Release vX.Y.Z" --body-file <notes>
```

The body: one line on what the release is, the merge method, and the
CHANGELOG section for the version. End it with the PR attribution line. Wait
for the PR checks (`gh pr checks <n> --watch`); stop if any fail.

## 7. Land on main

Fast-forward:

```bash
git push origin dev:main
gh pr view <n> --json state          # MERGED
```

Squash, only when asked:

```bash
gh pr merge <n> --squash --subject "Release vX.Y.Z (#<n>)"
git fetch origin
git diff --quiet dev origin/main     # must succeed: the squash holds all of dev
git reset --hard origin/main         # on dev
git push --force-with-lease origin dev
```

The force push rewrites `dev` (only `main` is protected by the ruleset), so
ask before it. If `git diff --quiet` fails, `dev` has commits the squash does
not hold: stop and report them.

## 8. Tag

Tag the commit now on `main` and push the tag; it starts the GoReleaser
workflow.

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z" origin/main
git push origin vX.Y.Z
```

## 9. Monitor

```bash
gh run list --workflow Release --limit 3
gh run watch <run-id> --exit-status
gh release view vX.Y.Z
```

Report the result of the Release job and the release URL with its assets.
