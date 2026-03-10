# Key Practices to Minimize Token Use

## 1. Use CLAUDE.md as Project Memory (not a data dump)

Claude loads `CLAUDE.md` automatically at session start. Keep it under 5k tokens. Populate it with: project summary & active features, tech stack, code style & naming conventions, known bugs and next TODOs. If it becomes impossible to keep all relevant info under 5k tokens, split less critical sections into separate files under the `docs/` directory.

## 2. One task per session

One session per logical task works best - one bug fix, one feature, one refactor. Don't try to fix three bugs and add two features in one conversation. Between sessions, leave breadcrumbs via a `CLAUDE.md` file with key architecture decisions, file locations, and conventions so Claude spends fewer tokens rediscovering your project structure.

## 3. Use subagents for research**

Delegate research with "use subagents to investigate X." They explore in a separate context, keeping your main conversation clean for implementation. When Claude researches a codebase it reads lots of files, all of which consume your context. Subagents run in separate context windows and report back summaries. 

## 4. Use `/compact` and plan mode**

Use `/compact` when you notice Claude losing track, and `/clear` when switching to completely different work. Press Shift+Tab twice in the terminal to enter plan mode before expensive operations - planning first prevents costly rework, so Claude outlines the approach before writing code and you catch issues early.

## 5. Explicit file boundaries in your CR

Use your `CLAUDE.md` (or CR file) to explicitly specify which files Claude can read and which directories are forbidden. This prevents unnecessary context consumption from irrelevant code.

## 6. Include self-verification criteria

Include tests, screenshots, or expected outputs so Claude can check itself. This is the single highest-leverage thing you can do - Claude performs dramatically better when it can verify its own work, like running tests, comparing screenshots, and validating outputs.

---

## Summary Workflow

```text
CR.md (scoped change request)
  ↓
/plan mode - Claude outlines approach, you review
  ↓
Implementation (single session, one task)
  ↓
Self-verification via tests in acceptance criteria
  ↓
/compact or /clear before next task
```
