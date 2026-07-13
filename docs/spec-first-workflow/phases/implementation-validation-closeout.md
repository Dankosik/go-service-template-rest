# Implementation / Validation / Closeout

On entering this macro phase, and only then, establish the single root Codex Goal required by `AGENTS.md` immediately before the first implementation edit. Then make the requested change, review the resulting diff, and prove the accepted outcome. Implementation may be local or delegated; the root remains accountable.

## Read When

- The request authorizes change/build/fix and required decisions are ready.
- Direct work has a clear inline outcome and proof, or structured/orchestrated work has a ready independently reviewed ledger.
- Existing implementation needs repair, review, validation, or closeout.

## Inputs

- Accepted direct outcome or current reviewed `tasks.md`.
- Required spec, design, test, and rollout decisions named by the work.
- Current repository state, including pre-existing user changes.
- Repository-owned generation and validation commands.

## Outputs

- In-scope code, tests, config, migrations, generated output, and docs.
- Updated task progress when a ledger exists.
- Review findings and repairs proportionate to risk.
- Fresh validation evidence and an evidence-clamped final claim.

## Implement

1. Inspect the owning code, callers, siblings, tests, and generated/manual boundary before editing.
2. For a defect, fix the narrowest owning surface whose contract the reproducer proves is violated; do not patch only the reported entry point when sibling callers share that contract.
3. Preserve accepted behavior, ownership, failure, cleanup, rollout, and proof decisions.
4. Prefer stdlib and existing repository patterns. Do not add a dependency, interface, helper layer, or architectural pattern without a present need and an owner.
5. Remove replaced paths and adjacent stale artifacts unless current compatibility evidence requires retention.
6. Keep changes reviewable and avoid unrelated cleanup.
7. If implementation exposes a missing product, contract, source-of-truth, ownership, test-strategy, or rollout decision, stop and reopen that owner instead of inventing it.

The root may implement directly. Delegate a patch only when the bundle is concrete and independent or isolation materially reduces risk/contended writes. A worker brief names the outcome, task IDs, workspace/boundary, required context, forbidden decisions, proof, and blocker output. Worker success is advisory until the root inspects the diff and reruns relevant proof in the integration workspace.

When `tasks.md` exists, update each task or checkpoint checkbox and evidence immediately after its proof. Before an intentional stop, record the blocker and next executable task; on resume, follow the shared [Resume Order](../shared/artifact-model.md#resume-order) instead of reconstructing progress from chat.

## Review

Always inspect the final diff for:

- correctness against accepted behavior and invariants;
- error, cancellation, retry, concurrency, transaction, and resource-lifetime behavior where relevant;
- contract/generated-source drift;
- security, privacy, money, data, and rollout risk in scope;
- ownership, unnecessary abstraction/dependencies, and stale replacement surfaces;
- tests that prove behavior rather than implementation detail.

After relevant validation, structured or orchestrated work requires independent read-only review of the exact candidate final diff and proof evidence before closeout. Direct work uses independent review when the user requires it or the change is high-impact, hard to reverse, broad, ambiguous, or difficult to falsify locally.

The reviewer returns one verdict under the shared convergence contract:

- `PASS`: the current validated diff has no known in-scope defect, unapproved decision, unowned question, proof gap, or uncovered affected lens;
- `CONCERNS`: a bounded risk or downstream proof obligation still needs explicit owner disposition and fresh review; it does not permit closeout;
- `FAIL`: an implementation defect, unapproved decision, missing proof, or invalid evidence prevents closeout.

Repair in-scope findings, then revalidate and re-review until convergence. Any mutation after review invalidates affected review and proof dispositions. Reopen upstream decisions rather than broadening implementation. A passing command is insufficient when its implemented fixtures or assertions do not exercise the binding proof obligation; a matching selector name alone is not evidence.

Treat edits to tests, fixtures, golden files, skip or exclusion settings, lint/build configuration, and proof commands as proof-surface changes. They require an accepted task or behavior reason; a green result obtained by weakening or removing an oracle or bypassing a triggered gate is invalid.

Validation, post-code review, in-scope repair, revalidation, fresh re-review, and closeout run automatically in the same root session. An implementation-owned failure never produces a next-session prompt.

## Validate

Run the smallest fresh evidence set that covers the claim:

- targeted tests for changed behavior;
- build, type, lint, race, integration, or repository gates relevant to affected packages;
- contract, migration, generation, or mirror drift checks when their source changes;
- a smoke/manual check when automated proof is unavailable or insufficient;
- targeted negative searches for identifiers and references that should be gone.

Worker output, cached results, unrelated green checks, skipped commands, and too-narrow tests do not prove the claim. When a required check cannot run, record the command, reason, narrower evidence, and unverified remainder.

Reconcile both directions: every accepted obligation and every ledger task on the current completion path maps to its implementation or an already accepted evidence-backed no-implementation disposition, and to adequate proof; every material change maps back to accepted scope. Keep this reconciliation inline unless an existing ledger owns it. Preserve unrelated pre-existing changes.

## Close Out

Mark a task complete only after its proof passes. The final response states:

- what changed;
- the most important design/behavior consequence;
- validation actually run and result;
- remaining risk, unavailable proof, or blocker;
- the exact reopen owner when unfinished.

Use `complete`, `fixed`, `ready`, or equivalent only when fresh evidence supports the full claim. A blocker is a valid outcome, not successful completion.

## Stop Rule

Finish when the accepted completion condition is met, the required review has returned `PASS`, and relevant proof passes. Continue in-scope repair when the failure is implementation-owned. Stop and reopen planning, test design, technical design, specification, research, or user/external authority only when that owner must change a decision or supply unavailable evidence.
