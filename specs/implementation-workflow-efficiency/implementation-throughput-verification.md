# Whole-ledger implementation: instruction verification

Historical record of the earlier 2026-09-06 policy. Its absolute static-check
ban is superseded by the later user-authorized
[repair-feedback revision](repair-feedback-verification.md). This record is
evidence, not current execution authority.

Date: 2026-09-06. Repository baseline:
`4911f204e5fcc930ea84658742e464486c78d1e0`.

The user's latest clarification is the authority: retain Definition, Technical
Design, and Planning with their reviews, remove the separate Test Design phase
entirely, choose and write tests during Implementation, and implement
the full ledger with useful parallel tasks and subtasks; execute no tests,
builds, lint, diagnostics, smoke/live checks, or reviews during that work.
Validate the assembled result only after every code task is implemented.
The first local revision retained diagnostic exceptions; this final revision
removes them. This file records change evidence, not an alternative workflow.

## Canonical instruction changes

- Test Design, its review route, phase owner, and Test Plan V1 matrix interface
  are removed. Planning readiness requires accepted behavior and final outcomes,
  not prior cases, fixtures, assertions, or exact commands. Testing methods are
  optional in-task expertise and no longer require a pre-approved test plan.
- AGENTS, Implementation, Evidence Contract, and Validation Routing now agree
  on the whole-ledger boundary. Finishing a task, a wave, or all currently
  runnable work is insufficient to start final validation.
- Implementation results allow only Implemented or Blocked and have no proof
  or review field. Accepted belongs only to the Completion result. The existing
  interface and native Lead identifiers remain; no additional role is needed.
- A task packet's Final validation section carries required claims and checks.
  The old Accept when / Focused check / Integrated check shape is removed from
  the canonical template. Verification-only work belongs to Completion.
- Ready tasks consume available Implemented code or agreed contracts. Missing
  live infrastructure cannot hold independent coding. Overlapping writers and
  actual unavailable implementation inputs still constrain dispatch.
- Worker, Go coding/testing/debugging/verification methods, source references,
  native adapters, README, and command/feature guides no longer prescribe
  checks during ledger implementation. Source navigation and generation remain
  code production; generation does not trigger drift-check execution.
- Direct Work, delegated verification, a partial final delivery, and a
  diagnostic-recovery label cannot bypass the ledger's execution boundary.
- Final review covers the complete changed outcome, including protected
  task-local invariants and cross-task interactions. A multi-task candidate is
  an expected review input, not a handoff error requiring unit review cycles.
- Resume replaces obsolete task timing and fields in place. Prior results may
  supply actual evidence for final reuse; they do not retain an old execution
  mode. Required external-effect and release evidence remains required.
- Six completed process-experiment reports were removed from the working tree;
  their surviving decisions are canonical and their historical evidence is in
  Git. The source scan found no remaining instruction link to those records.

## Static interpretation comparison

These are source-level before/after interpretations by the current agent in
one session, not independent review or executed agent trajectories. Session
instructions identify the model family as GPT-6; exact effective runtime model
and effort were not independently measured. No model, effort, or runtime
configuration was changed. The user report and local sources dated 2026-09-06
supply the pressure; no vendor capability or speed claim is inferred.

| Same task input | Previous instruction path | Final instruction path |
| --- | --- | --- |
| T1 code is ready; T2 consumes it. | Unit proof/acceptance, or diagnostic exception in the first local revision. | T1 Implemented -> local integration -> T2 immediately, without check execution. |
| T2 and T3 have disjoint writers, and T2 has two independent subsets. | Parallelism with unit acceptance gates. | Parallel tasks and subsets, then immediate handoffs without gates. |
| A worker suggests a quick compile or gopls diagnostic. | Optional focused proof/diagnostic could run. | No execution during ledger implementation. |
| A delegated verification task or Direct Work wrapper is proposed for T1. | Could reach a single-unit verification boundary. | Cannot bypass the unfinished ledger. |
| T1 is ready but T2 is blocked on a missing code input. | Unit acceptance could proceed independently. | Preserve the gap; continue other coding; do not validate a completed subset. |
| All 12 code tasks and their test code are assembled. | Unit checks followed by global checks. | One consolidated final validation stage, with claim-matched commands and delta repairs. |
| Code for an authorization or monetary invariant is ready. | Protected-unit reviewer required. | Immediate implementation handoff; protected risk is covered in final review. |
| Task behavior is closed, but no test plan, case list, or PostgreSQL exists. | A separate Test Design result and its review could hold Planning. | Go directly through Planning to Implementation; the executor writes tests in place and final validation supplies runtime evidence. |
| Runtime has Catalog's agreed contract but joint integration is unavailable. | Explicit preparation admission and blocked unit acceptance. | Implement the consumer; preserve joint integration as final evidence. |
| A later deployment action is gated. | Per-unit/target acceptance could hold progress. | Finish code, validate locally, then perform only authorized release actions; Completion still needs their required proof. |
| Old packet has Accepted receipts and per-task checks. | Old timing can survive in a pinned packet. | Normalize to Implemented and Final validation while preserving required claims. |
| A task explicitly stops after Technical Design. | Reviewed macro-phase stop. | Same stop; pre-Implementation phase structure remains. |

## Final proof

- `make template-owned-purity-check`: PASS after phase removal.
- `make docs-contract-check`: PASS.
- Relative file links after phase removal: 252 checked, zero missing.
  Retired phase/interface owners and their routes: absent. The macro route is
  Definition -> Technical Design -> Planning -> Implementation, with final-only
  check execution retained.
- Specification, System Design, Go Ownership Design, and Codex project/runtime
  configuration remain byte-identical to baseline. Definition, Technical Design,
  and Planning retain review; the retired test phase has no review trigger.
- `git diff --check`: PASS.

No product test suite, PostgreSQL/Vespa environment, production target, or
consumer checkout was exercised or changed. No commit, push, deployment, or
fleet synchronization was performed. Running consumers require adoption of the
new instructions and normalization of their active packets; local edits do not
change an already-running session's loaded instructions automatically.

This proves instruction consistency and structural parity, not that a model
will obey every future trajectory or that delivery now takes 8–10 hours. Those
claims require comparable real ledger runs, actual timing, and final defect
assessment. Existing validation commands remain for final or standalone work;
the old per-task execution process is not a selectable mode.

## Follow-up audit: four findings repaired

The subsequent read-only audit found four surviving sources of the old policy.
The user explicitly authorized fixing all four, including machine-local Codex
instructions. The repair changes four repository files plus
`/Users/daniil/.codex/AGENTS.md`; the machine-local file is outside Git/template
propagation. Model names, model pins, and native tool capabilities are unchanged.

| Finding | Repair | Boundary retained |
| --- | --- | --- |
| OpenCode injects per-unit proof/review into the Task tool description. | Its catalog now describes check-free implementation and whole-ledger final validation. | Existing role names and non-Task tool descriptions are preserved. |
| Global gopls diagnostics and scans trigger after edits. | Global selectors choose relevant final checks; all ledger validation waits for complete assembled code. | Call-reference navigation, final security checks, and pre-release evidence remain. |
| Delegation/model selection needs a named check or prior proof plan. | Shared and Codex briefs use accepted behavior, writable scope, and expected outcome. Executors choose tests while coding. | Closed product/architecture decisions, disjoint writers, native capacity, and model policy remain. |
| Ownership design requires proof levels before implementation. | The skill closes production source ownership and records only known existing proof locations. New tests and proving layers belong to Implementation. | Production package/file/dependency ownership remains required. |

Proof of the repair:

- Executed the OpenCode tool-definition hook with Task and non-Task inputs:
  updated wording, all six roles, original description retention, and unchanged
  non-Task output PASS. This verifies the injected text, not an agent trajectory.
- All five changed files match the after-edit hashes in the local repair
  snapshot; the retired clauses are absent and 21 relative links resolve.
- `git diff --check`: PASS.
- `make template-owned-purity-check`: PASS after all four repairs.

Local before/after snapshots and hashes:
`/var/folders/9r/ft1t72w13r765bpf61v9mly00000gn/T/workflow-four-fixes-__d1_o4_/`.
The static comparison above now covers the formerly missed delegation and
machine-instruction paths. No claim of universal future model compliance or
measured speed is made.
