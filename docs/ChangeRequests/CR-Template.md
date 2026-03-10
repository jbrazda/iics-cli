# CR: [Short Title]

## Scope

- Files to change: [provide a list of files to be modified]
- Files to READ ONLY (don't modify): [list of files]
- Forbidden directories:  [ i.e. tests/, docs/]

## Problem

[Describe The problem]

## Desired Change

[Instructions]

## Acceptance Criteria

- [ ] Criteria 1
- [ ] Existing passing tests still pass
- [ ] No new dependencies added

## Do NOT

- Refactor unrelated code
- Change function signatures
- Modify test files

```
This pattern works because:
- It pre-scopes which files Claude reads
- It points to existing utilities (avoiding unnecessary code duplication searches)
- It defines explicit acceptance criteria Claude can self-verify against
- The "Do NOT" section prevents scope creep
```
