# AGENTS.md


Repository-wide contract for producing reliable Go-service changes with the least workflow needed.

## Authority And Loading

- Explicit user, system, and developer instructions win.
- This file owns request authorization and repository-wide invariants.
- [docs/spec-first-workflow.md](docs/spec-first-workflow.md) is the workflow router. Read only the current phase file and any shared file needed for the decision at hand.
- Task-local artifacts own accepted task decisions. Runtime and generated-source authorities named by those artifacts still win over derived prose.
- `SOUL.md` is lower-precedence engineering and communication guidance. Skills provide methods; neither overrides this contract or task-local decisions.

## Authorization

- `answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting, not implementation.
- `change`, `build`, and `fix` authorize in-scope local edits and relevant non-destructive validation.
- A Codex Goal is an execution control for the implementation/validation/closeout macro phase only. Do not create or continue one during intake, research, specification, technical design, test design, planning, or their review and repair loops, even when those phases edit repository workflow artifacts. Direct work enters implementation immediately.
- On entering implementation/validation/closeout, create or continue exactly one root-thread Codex Goal immediately before the first implementation edit, regardless of task size, and complete it only after root diff inspection, any risk-triggered independent review, and fresh validation evidence. A small direct change does not need an independent reviewer merely because a Goal exists. Reuse a matching active Goal; do not create separate Goals for subagents, workers, tasks, or internal checkpoints. If an unrelated Goal is active, ask the user to pause or clear it. A workflow opt-out does not waive this execution control.
- Ask before external writes, destructive actions, purchases, or material scope expansion. Do not ask before ordinary repository reads, in-scope edits, or tests.
- Respect an explicit boundary such as `read-only`, `docs-only`, `research only`, or a named phase.

## Explicit Workflow Opt-Out

- Honor a workflow opt-out when the implementation request is clear enough to act on and the user explicitly says the workflow may be skipped or bypassed and asks to proceed to implementation.
- A valid opt-out overrides the normal path and phase routing for that request. Proceed directly to implementation; do not first require or run workflow-start checks, phase/readiness gates, workflow artifacts, or workflow-only delegation and review. Do not create a record merely to document the opt-out.
- The opt-out waives process, not scope, safety, permission, or proof. Preserve explicit boundaries, ask before external or destructive actions, inspect the affected code before editing, do not invent a materially unresolved behavior or ownership decision, and run fresh validation proportionate to the change.
- If implementation exposes a genuinely blocking decision, ask only for that decision. After it is resolved, continue directly unless the user withdraws the opt-out or requests workflow artifacts.

## Working Contract

1. Reconstruct the intended outcome before acting. Inspect repository facts instead of asking the user for facts the repository can answer. Ask only for a decision that would materially change scope, behavior, ownership, safety, or proof; otherwise state a bounded assumption and continue.
2. Describe the outcome, constraints, success criteria, and stop conditions. Do not prescribe steps the model can choose reliably, repeat rules across files, or create artifacts solely to prove that process happened.
3. Choose the smallest path that preserves correctness. Direct work may proceed without workflow artifacts; otherwise use [the workflow router](docs/spec-first-workflow.md), which owns path selection, phase order, review gates, and movement rules. Respect a user-named phase boundary.
4. Public contracts, persisted data, security, money, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof, but not automatically full-depth work or a durable artifact in every phase. When the accepted outcome spans multiple deployables, repositories, or managed dependencies, completion covers the full affected deployment graph; apply [System Release Closure](docs/spec-first-workflow/phases/system-integration-design.md#system-release-closure), and narrow the claim or report the external blocker when any required surface is outside current authority or proof.
5. Evidence before invention. Prefer current Go stdlib and established repository patterns. Before structured/orchestrated work designs against an external platform, unfamiliar mechanism, new infrastructure/dependency, or non-trivial architecture choice, research current official docs/source and credible real implementations or engineering writeups. Treat official sources as contract authority, use real-world sources for proven patterns and operational pitfalls, and do not rely on model memory for current external behavior. Add custom machinery only when viable researched options do not fit, and record why.
6. Keep ownership explicit. Put substantial code in the narrow owning package/file, preserve generated-source discipline, and remove replaced code and adjacent stale artifacts unless current compatibility evidence justifies retention.
7. Skills define the method; subagents provide separate context and independence. Apply matching skills locally by default. Delegate only one concrete bounded question when separate evidence, context, or review independence can change the result; a domain label, matching keyword, or desire for more confidence does not create a lane. For ledger implementation, one worker owns one task until the root accepts its diff and proof or returns concrete gaps; after acceptance, a fresh worker owns the next task. The root does not repair the assigned task, advance the ledger early, or create a separate reviewer lane for task acceptance. For applicable structured or orchestrated non-implementation candidates, run the router's autonomous pre-review challenge before the separate required reviewer. [Subagents And Handoff](docs/spec-first-workflow/shared/subagents-and-handoff.md) owns both contracts; the root verifies delegated output and owns synthesis, integration, task acceptance, and completion claims.
8. Do not claim ready, complete, fixed, or covered without fresh evidence matched to the claim. Report unavailable or narrower proof honestly and name the next useful check.

## Instruction Ownership

- Keep global rules here.
- Keep path selection, phase order, review routing, and movement rules in `docs/spec-first-workflow.md`.
- Keep phase-specific method in `docs/spec-first-workflow/phases/`.
- Keep artifact persistence and status rules in `docs/spec-first-workflow/shared/artifact-model.md`.
- Keep delegation, review independence, resume, and handoff rules in `docs/spec-first-workflow/shared/subagents-and-handoff.md`.
- Keep task-specific decisions in task-local artifacts.
- When two surfaces repeat a rule, retain the narrowest canonical owner and replace the other copy with a link.

@SOUL.md

@RTK.md
