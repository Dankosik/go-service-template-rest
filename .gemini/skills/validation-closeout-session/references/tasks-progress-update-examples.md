# Tasks Progress Update Examples

## Behavior Change Thesis
When loaded for updating an existing task ledger during closeout, this file makes the model mark only freshly proved task items instead of bulk-checking the ledger, adding new tasks, or creating `tasks.md` after implementation.

## When To Load
Load this only when a task-local `tasks.md` already exists and closeout must align checkbox or progress state with proof observed in this session.

## Decision Rubric
- Update checkbox state only for existing items whose proof obligation was actually run and observed in this session.
- Leave an item unchecked when required proof failed, was skipped, stale, missing, or too narrow; record a reopen note only if the existing ledger format supports notes.
- Do not add, split, reorder, or rewrite ledger items during validation.
- Do not create `tasks.md` during closeout. If a non-trivial workflow expected it and it is missing, reopen planning.
- Do not infer every task is done from a broad command unless the ledger's proof obligations are actually covered by that command.

## Imitate

```markdown
- [x] T001 Phase 1: Implement export job API handler
  - Closeout proof: `go test ./internal/httpapi/export -count=1` passed in this session.
- [x] T002 Phase 1: Keep generated OpenAPI output current
  - Closeout proof: `make openapi-check` passed in this session.
- [ ] T003 Phase 1: Validate migration compatibility
  - Closeout status: blocked. `make migrate-check` failed; reopen implementation at T003.
```

Copy the item-by-item proof mapping: each checkbox follows a fresh command result.

```markdown
Ledger update: skipped.
Reason: this direct-path task explicitly waived `tasks.md` in the approved workflow; do not invent a task ledger during closeout.
```

Copy the explicit skip when the approved workflow did not use a ledger.

## Reject

```markdown
- [x] T001-T006 All tasks done because implementation is complete.
```

Fails because implementation presence is not item-level closeout proof.

```markdown
- [x] T004 Migration validation
- [ ] T007 Fix the migration validation failure during closeout
```

Fails because closeout cannot add repair tasks or check failed validation.

```markdown
Created a new `tasks.md` so the validation closeout has somewhere to record progress.
```

Fails because validation is artifact-consuming and cannot create the ledger.

## Agent Traps
- Checking every task after one broad command passes, even when some items expected different proof.
- Trusting a review-agent summary as checkbox evidence.
- Treating the ledger as a place to plan the fix for failed proof.
