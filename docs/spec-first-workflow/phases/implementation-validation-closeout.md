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

The root may implement direct work locally. When an accepted `tasks.md` exists, use the shared [implementation worker loop](../shared/subagents-and-handoff.md#implementation-worker-loop): select the next ready task in dependency order and assign exactly that one task to one worker. The worker owns the task until the root accepts it or a genuine upstream blocker reopens another owner. It does not mark the task complete, start another task, or launch a reviewer.

A worker brief names the outcome, one task ID, workspace/boundary, required context, forbidden decisions, acceptance criteria, proof, and blocker output. The worker returns its exact diff, acceptance-criteria mapping, commands/results, and blockers. The root inspects the integrated task diff and proof. If any accepted criterion, scope boundary, or required proof is missing, return the same task to the same worker with concrete bounded gaps; do not repair it in the root or start the next task. If the task is accepted, update its checkbox and evidence immediately, then launch a fresh worker for the next ready task; reuse the prior worker only for corrections to its own task. Worker success is advisory until this root acceptance completes.

Rerun relevant proof in the integration workspace when worker evidence does not already establish the integrated state. Do not rerun an unchanged command without a changed risk surface. Before an intentional stop, record the blocker and next executable task; on resume, follow the shared [Resume Order](../shared/artifact-model.md#resume-order) instead of reconstructing progress from chat.

## Review

Always inspect the final diff for:

- correctness against accepted behavior and invariants;
- error, cancellation, retry, concurrency, transaction, and resource-lifetime behavior where relevant;
- contract/generated-source drift;
- security, privacy, money, data, and rollout risk in scope;
- ownership, unnecessary abstraction/dependencies, and stale replacement surfaces;
- tests that prove behavior rather than implementation detail.

Every ledger task receives root acceptance review before the next task starts. That inspection is part of orchestration, not an independent reviewer lane: do not launch a reviewer after each task, worker return, edit, or correction. After every task is accepted, generated outputs and task evidence are current, and terminal validation is complete, inspect the final integrated diff. Independent whole-diff review runs only when the user explicitly requests it or a concrete trigger makes the integrated change high-impact, hard to reverse, broad, ambiguous, protected-domain-sensitive, cross-task-sensitive, or difficult for the root to falsify locally. It supplements rather than replaces per-task root acceptance. Small direct work follows the same explicit-or-risk trigger rule. Never launch independent review solely because implementation occurred, a review skill matches, a Goal is active, or more confidence would be desirable.

When independent review is triggered, the reviewer returns one verdict under the shared convergence contract:

- `PASS`: the current validated diff has no known in-scope defect, unapproved decision, unowned question, proof gap, or uncovered affected lens;
- `CONCERNS`: a bounded risk or downstream proof obligation still needs explicit owner disposition and fresh review; it does not permit closeout;
- `FAIL`: an implementation defect, unapproved decision, missing proof, or invalid evidence prevents closeout.

Return an implementation-owned finding to the worker that owns the affected task. Collect compatible findings for that task, repair them as one coherent batch, rerun affected proof, and obtain focused re-review from the same gate reviewer when independent review was triggered. Reuse unaffected lens dispositions unless the repair changes their evidence; use full affected-surface re-review only when shared assumptions or domain boundaries change. Do not review after each finding or edit. Reopen upstream decisions rather than broadening implementation. A passing command is insufficient when its implemented fixtures or assertions do not exercise the binding proof obligation; a matching selector name alone is not evidence.

Treat edits to tests, fixtures, golden files, skip or exclusion settings, lint/build configuration, and proof commands as proof-surface changes. They require an accepted task or behavior reason; a green result obtained by weakening or removing an oracle or bypassing a triggered gate is invalid.

Validation, in-scope repair, revalidation, any required post-code review or focused re-review, and closeout run automatically in the same root session. An implementation-owned failure never produces a next-session prompt.

## Validate

Run focused proof while implementing, then one terminal fresh evidence set for the frozen candidate. Do not rerun an unchanged command unless a new patch, finding, or required final bundle changes what it proves. The terminal set covers the claim with:

- targeted tests for changed behavior;
- build, type, lint, race, integration, or repository gates relevant to affected packages;
- contract, migration, generation, or mirror drift checks when their source changes;
- integrated target-environment proof across the affected deployment graph when the accepted outcome is system-wide; provider deployment status or one component's readiness alone is insufficient;
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

Finish when every ledger task has passed root acceptance, the accepted completion condition is met, any triggered independent review has returned `PASS`, and relevant proof passes. Return implementation-owned gaps to their task worker. Stop and reopen planning, test design, technical design, specification, research, or user/external authority only when that owner must change a decision or supply unavailable evidence.
