# Implementation / Validation / Closeout

On entering this macro phase, and only then, the root establishes the single Codex Goal required by `AGENTS.md`, starts the native Codex App Worker task that produces the requested change, reviews and integrates the returned diff, and proves the accepted outcome.

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

### Worker Assignment And Acceptance

Every authorized implementation change is produced by one native Codex App task for the repository project in a managed-worktree environment backed by a dedicated Codex-managed Git worktree ([official Worktrees](https://developers.openai.com/codex/environments/git-worktrees)). Direct work assigns one accepted outcome to one Worker. Ledger work assigns exactly one ready task to one Worker. Only one write Worker runs at a time, and it owns that outcome or task until root acceptance or a genuine upstream blocker.

The root selects the smallest starting state that contains the accepted implementation input: omit the optional starting state when the project default already contains it; select an existing branch when that branch owns the accepted state; select the working tree only when required accepted changes are uncommitted. It then dispatches as soon as the Goal and brief are ready.

For every new App Worker task, the root explicitly selects and passes both the model and reasoning effort through the App's supported launch controls; never inherit an App default. Record the task identity, selected model and effort, and a short basis in transient execution context or existing ledger evidence. When the selection relies on eval evidence, the basis names the exact eval artifact and compared model/effort configuration.

Use `gpt-5.6-terra` with `medium` as the normal implementation baseline. Use `gpt-5.6-luna` with `low` only for bounded low-risk mechanical work when a named representative workflow eval artifact shows retained task success and evidence completeness. Use `gpt-5.6-sol` when material difficulty, ambiguity, reversibility, or consequence of error requires frontier capability. Security, concurrency, data, migration, and cross-service scope are risk signals, not automatic Sol triggers.

Choose effort independently of the model. Use `low` for latency-sensitive simple work only when representative evals preserve quality, and `medium` for normal implementation. Use `high`, `xhigh`, or `max` only when a named representative workflow eval artifact for the relevant task class shows a meaningful quality gain; reserve `max` for the hardest quality-first work. Same-task corrections keep their selected model and effort unless failure evidence justifies an explicit escalation. A composer selection changed after dispatch configures a future turn; it is not evidence of the active turn's effective model or effort. When that distinction matters, use only native metadata or events for the active turn.

After dispatch, follow the App's [event stream](https://developers.openai.com/codex/app-server#events): `turn/started` confirms the active turn, `item/*` carries progress and authoritative completed items, `turn/completed` supplies the terminal turn status, and `thread/status/changed` reports runtime status transitions. The root does not actively poll or narrate unchanged state. It resumes result intake, acceptance, and Git handoff/integration when the native completion or status signal arrives.

An implementation-owned gap returns to the same App task and managed worktree with concrete findings; the root never authors the repair. The root re-inspects the correction and its affected proof. Tightly scoped Worker checks and criteria mapping are task-local feedback, not acceptance: the root independently judges behavior, quality, test adequacy, scope, and completeness from the report, evidence, and diff.

After accepting and integrating a ledger task, the root records its evidence before starting a fresh App task in a fresh managed worktree for the next ready task. Never replace the App task for a same-task correction or reuse it for the next task.

### Worker Brief And Result Intake

Before editing, the Worker verifies that its physical current directory and Git top level are the assigned managed worktree. It treats that worktree as the only writable repository checkout. A source-checkout absolute path maps to the same repository-relative path in the managed worktree; an unmappable required path blocks instead of authorizing a write outside the worktree.

Keep the brief outcome-first and limited to:

```text
Goal / context: <accepted direct outcome or one ready task; minimal authorities and paths>
Constraints: <task-specific editable, forbidden, permission, and role boundaries>
Evidence: <current facts, required sources or skills, and proving commands>
Success: <observable acceptance criteria>
Output: <changed/deleted files, exact proof results, and unmet criteria>
Stop: <genuine blocker or authority condition>
```

The constraints tell the Worker that it is not root and cannot create, continue, or complete a Goal; delegate; self-accept; update task or workflow status; start another task; or claim repository completion. Do not copy the workflow, App lifecycle, generic strictness language, or unrelated repository context into the brief.

The root records the returned App task, thread, and managed-worktree identity, assigned outcome or task, corrections, report, diff, proof, acceptance, and integration state in transient execution context or existing ledger evidence, never a second ledger. The Worker return is evidence, not authority. The root still reads the report, diff, and proof and decides acceptance; blocked reports, unmet criteria, failed operations, or ambiguous task identity cannot be accepted by prose self-identification.

Rerun relevant proof in the integration workspace when Worker evidence does not establish the integrated state. Do not rerun an unchanged command without a changed risk surface. Before an intentional stop, record the blocker and next executable task; on resume, follow the shared [Resume Order](../shared/artifact-model.md#resume-order) instead of reconstructing progress from chat.

## Review

Always inspect the final diff for:

- correctness against accepted behavior and invariants;
- error, cancellation, retry, concurrency, transaction, and resource-lifetime behavior where relevant;
- contract/generated-source drift;
- security, privacy, money, data, and rollout risk in scope;
- ownership, unnecessary abstraction/dependencies, and stale replacement surfaces;
- tests that prove behavior rather than implementation detail.

The root reviews every App Worker result and its proof before acceptance. Apply every matching review skill locally and evaluate all materially affected risk and specialist lenses in one coherent root inspection. A review skill supplies a method, not a subagent lane. Never launch a built-in subagent, reviewer, specialist, or re-review lane anywhere inside implementation/validation/closeout.

Every ledger task receives root acceptance review before the next task starts. After every task is accepted, generated outputs and task evidence are current, and terminal validation is complete, the root reviews the final integrated diff against the accepted outcome and every affected lens.

Return every implementation-owned finding to the App task that owns the affected direct outcome or ledger task. Collect compatible findings as one bounded correction, continue that task in its managed worktree, rerun affected proof, and have the root re-inspect the correction and every transitively affected lens. Reopen upstream decisions rather than broadening implementation. A passing command is insufficient when its implemented fixtures or assertions do not exercise the binding proof obligation; a matching selector name alone is not evidence.

Treat edits to tests, fixtures, golden files, skip or exclusion settings, lint/build configuration, and proof commands as proof-surface changes. They require an accepted task or behavior reason; a green result obtained by weakening or removing an oracle or bypassing a triggered gate is invalid.

Validation, in-scope Worker repair, root re-inspection, revalidation, and closeout run automatically in the same root session. An implementation-owned failure never produces a next-session prompt.

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

Finish when the direct outcome or every ledger task has passed root acceptance, the root has reviewed the final integrated diff and affected lenses, the accepted completion condition is met, and relevant proof passes. Return implementation-owned gaps to their owning App task. Stop and reopen planning, test design, technical design, specification, research, or user/external authority only when that owner must change a decision or supply unavailable evidence.
