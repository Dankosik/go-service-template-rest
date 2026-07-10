---
name: validation-closeout-session
description: "Run the internal validation/closeout checkpoint of an active implementation session after required review/re-repair is current; return proof, done, or the exact repair/reopen target."
---

# Validation Closeout Session

## Eligibility And Outcome

Use inside an active implementation session after implementation and required post-code review/reconciliation are current, when an approved `tasks.md` or eligible direct-session envelope defines the completion claim and proof. Skip missing-ledger planning work, review-only requests, and attempts to create new pre-code artifacts after coding.

The outcome is an evidence-clamped closeout: done only when every required claim is freshly proven, otherwise partially verified or blocked with the exact failed proof and owner.

## Canonical Owners

- [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) owns ledger-first execution, proof, allowed closeout writes, blocker records, and done semantics.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns validity, direct-session envelope limits, and typed state.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns lane gates and actionable reopen prompt rendering.
- `go-verification-before-completion` supplies the fresh-evidence method.

Load command, task-progress, workflow-plan-completion, or proof references only when the active ledger or failure requires them. Do not broad-load references to reconstruct a generic closeout checklist.

## Allowed Side Effects

This session may run non-destructive validation and update only the existing closeout surfaces authorized by the approved ledger: `tasks.md` progress/evidence, ledger-owned `spec.md` Validation/Outcome, and pre-created review or validation phase files explicitly named by `tasks.md`.

It must not implement fixes itself, create or approve missing pre-code artifacts, change behavior/design/task scope, update `workflow-plan.md` merely because it exists, create new review/validation phase files, mutate external systems, or alter git state beyond read-only inspection. In-scope failures return to the implementation root for worker repair and fresh review/validation in the same session.

## Unique Method

1. Bind each requested completion claim to its task ID, changed files, proof obligation, freshness requirement, and expected pass/fail observable.
2. Use `go-verification-before-completion` to select the smallest fresh command set that covers the changed surface and repository-owned gates.
3. Include targeted negative proof for retired identifiers, generated/mirror drift proof for changed owners, and retained-surface proof when the ledger requires them.
4. Map every in-scope changed file to a ledger task, checkpoint, or allowed closeout surface before checking completion.
5. Record evidence only after it passes in the integration workspace. Skipped, stale, cached, unavailable, failing, or too-narrow evidence cannot satisfy a task.
6. Clamp the final claim to the evidence; a blocker is a valid stop, never a successful closeout.

## Success, Blocked Stop, And Reopen

Success requires all required ledger tasks and checkpoints checked with current evidence, repository and generated/mirror gates passing for the changed surface, allowed spec outcome updates current, and no unmatched in-scope changed file.

Stop blocked or partially verified when any required command fails or is unavailable, proof is narrower than the claim, a task remains unchecked, a changed file lacks ledger ownership, or a new decision/artifact is required. Record task/checkpoint, exact command or read, output, affected surface, narrower proven claim, unchecked work, and reopen target.

Return an approved in-scope defect to the active implementation root for automatic worker repair, patch intake, fresh re-review, and revalidation. Reopen planning for missing tasking/proof, test design for missing scenarios, technical design for ownership/mechanism, specification for scope/contract/protected behavior, or workflow planning for invalid routing. Do not repair it inside this read-only checkpoint.

This checkpoint never renders a prompt for implementation-local repair. Only the implementation root renders an upstream macro-phase reopen prompt when necessary. If the workflow is honestly done, do not invent another prompt.
