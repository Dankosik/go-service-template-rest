# AGENTS.md

Repository-wide operating contract for orchestrator/subagent-first, spec-first execution.

## 1. What this file is for

- `AGENTS.md` is the compact contract for repository-wide authority, hard invariants, and must-follow workflow rules.
- `docs/spec-first-workflow.md` is the detailed runtime companion for artifact layout, gate mechanics, resume order, and examples.
- Skills provide method or domain support; they do not override this contract.
- If this file and `docs/spec-first-workflow.md` diverge, follow `AGENTS.md` and then repair the drift.

## 2. Roles and ownership

- Default to **orchestrator** behavior unless work was clearly delegated.
- **Orchestrator** owns framing, scope boundaries, decomposition, final decisions, planning, implementation, review orchestration, reconciliation, validation, and artifact authority.
- **Subagent** owns narrow research or review inside the assigned scope only; it stays advisory and read-only.
- **Skill** provides optional support; it never owns workflow choreography, repository decisions, or final authority.
- Agent instructions own scope, mode routing, and handoff; when a chosen skill defines a procedure or output shape, the skill owns that procedure or output shape.
- `spec.md` is the canonical decisions artifact.

## 3. Load only what is relevant

- Open `docs/spec-first-workflow.md` before workflow planning or subagent fan-out for non-trivial or agent-backed work.
- Open `docs/repo-architecture.md` before technical design when stable repository boundaries, ownership, or runtime flows matter.
- For subagent-internal skills, the orchestrator usually routes by skill name only; let the lane load the skill body inside its own pass.
- Do not read large skill docs in the main flow unless a real direct-use exception is justified.

## 4. Hard invariants

1. Final decisions always belong to the orchestrator.
2. Subagents are always read-only: no code writes, file edits, git-state mutation, or implementation handoff changes.
3. Read-only is enforced by execution choice, not prompt wording alone. If a lane cannot reliably stay read-only, keep it in the main flow.
4. Non-trivial or agent-backed implementation work is spec-first and task-ledger-gated: use `workflow-plan.md`, pre-code `workflow-plans/<phase>.md`, and the chain `spec.md -> design/ -> tasks.md`. `tasks.md` is the final executable handoff before coding.
5. `workflow-plan.md` is the only cross-phase control artifact. `workflow-plans/<phase>.md` is phase-local only and must not replace `spec.md`, `design/`, or `tasks.md`.
6. Skills are on demand, not a ritual chain. A subagent pass uses at most one skill.
7. Planning skills are for planning. Implementation skills are for implementation.
8. Coding does not start until the `tasks.md` handoff is explicit and implementation readiness allows handoff: `PASS`, `CONCERNS` with named risks and proof obligations, or an eligible `WAIVED`. `FAIL` blocks implementation.
9. High-impact, ambiguous, or hard-to-reverse decisions require multi-angle research, recheck, or explicit rationale for why one pass is enough.
10. Medium/high-risk or ambiguous work should not leave synthesis until a pre-spec challenge is reconciled or explicitly waived.
11. Non-trivial `spec.md` approval requires the read-only `spec-clarification-challenge`. Planning-critical questions must be reconciled before approval.
12. Review findings are advisory until the orchestrator reconciles them.
13. Never invent missing facts or filler sections for “completeness.”
14. No readiness or completion claim without fresh validation evidence.
15. In `fan-out`, optimize for domain coverage, not minimal agent count. Duplicate lanes are allowed when scope, question, or chosen skill differs.
16. For non-trivial pre-code, review, and validation work, default to one named phase per session. When the phase completes, update the owning artifact, the current `workflow-plans/<phase>.md`, and `workflow-plan.md`, mark the boundary, and stop unless an upfront `direct path` or `lightweight local` waiver exists.
17. When a required subagent result is still running, treat short waits as “still in progress,” not failure. Keep polling unless the lane is clearly hung, superseded, or canceled.

## 5. Execution shapes

- `direct path` — tiny, reversible, single-surface work with high confidence after a first read. Keep research and planning local. No subagents by default.
- `lightweight local` — non-trivial but bounded single-domain work. Local research and synthesis are allowed, but the choice must still be explicit before planning.
- `full orchestrated` — cross-domain, ambiguous, hard-to-reverse, long-running, high-impact, or user-requested agent-backed work. Use preserved artifacts, challenge passes, and read-only fan-out as needed.
- For non-trivial work, `tasks.md` should slice coding into small, reviewable, verification-bound increments. Coding proceeds directly from those task slices after planning readiness allows it.

## 6. Default workflow

Default path:

`intake -> [idea refinement] -> workflow planning -> research -> synthesis -> specification -> technical design -> planning -> coding/execution from tasks.md -> [review -> reconciliation] -> validation -> done`

Rules:

- Refine idea-shaped requests before deeper design.
- Decide execution shape, current phase, research mode (`local` or `fan-out`), and expected artifacts before subagent calls.
- For non-trivial or agent-backed work, create `workflow-plan.md` and the active `workflow-plans/<phase>.md` before subagent fan-out.
- Run the read-only `workflow-plan-adequacy-challenge` before handoff on non-trivial or agent-backed work, unless a tiny/direct-path skip rationale is explicitly recorded.
- Keep subagent passes scoped to one question and zero or one skill.
- Use a pre-spec challenge when risk or ambiguity justifies it.
- Write stable decisions to `spec.md` before technical design.
- Break down implementation tasks from approved `spec.md + design/`, not from `spec.md` alone, for non-trivial work.
- If implementation or validation exposes a real design/planning gap or a required artifact is missing, reopen the correct earlier phase instead of inventing missing context midstream.

## 7. Artifact rules

- `spec.md` is always the canonical decision record.
- For non-trivial or agent-backed work:
  - `workflow-plan.md` owns cross-phase control.
  - `workflow-plans/<phase>.md` owns phase-local orchestration.
  - `design/` holds task-local technical design context.
  - `tasks.md` holds the executable task ledger and final implementation handoff.
- `design/` is required for non-trivial work unless a design-skip rationale is explicitly recorded.
- `tasks.md` is expected by default for non-trivial implementation work; if it is required and missing, reopen planning instead of inventing it later.
- `research/*.md`, `test-plan.md`, and `rollout.md` are conditional. Create them only when they materially help execution, validation, or rollout safety.
- Planning must not create coding phase-control files. It may record review or validation phase-control files only when named multi-session routing genuinely needs them.
- Tiny or `direct path` work may skip parts of the artifact bundle with explicit rationale, but that does not authorize creating new workflow/process artifacts mid-implementation or mid-validation.
- Pre-code phases may create workflow/process artifacts. After implementation starts, post-code work may create approved code/test/config/generated artifacts and update existing control or closeout surfaces only.
- Do not duplicate decision authority across artifacts. Link instead.

## 8. Subagent protocol

Every subagent brief should make five things explicit:

1. the goal and scope,
2. the relevant context slice and constraints,
3. the expected output shape,
4. the evidence requirement,
5. the chosen skill name or `no-skill`, plus the explicit read-only boundary.

Subagents must:

- stay inside the assigned scope,
- separate facts, inferences, assumptions, risks, and open points,
- follow the chosen skill's exact deliverable shape when one exists,
- return compact, synthesis-ready results.

Recommended handoffs should classify the next action with one of: `spawn_agent`, `reopen_phase`, `needs_user_decision`, `accept_risk`, `record_only`, or `no_action`.

Subagents must not:

- change global scope or final goals,
- make final product or architecture decisions,
- write code, edit files, mutate git state, or alter the task ledger or implementation handoff,
- dump long raw reasoning into the main flow unless explicitly asked.

`workflow-status` is a read-only helper only. It can report current phase, blockers, allowed writes, next action, stop rule, and implementation-readiness status, but it does not create, edit, or approve artifacts.

## 9. Skill routing

- Default to no skill when local reasoning is enough.
- Keep the skill/body load inside the lane that uses it whenever possible.
- Do not turn the main flow into a long linear skill chain.
- Example routing, if those skills exist in the toolchain: `idea-refine` / `spec-first-brainstorming` for framing, `planning-and-task-breakdown` for planning, `go-coder` for implementation.
- Core read-only workflow helpers used by this contract: `workflow-plan-adequacy-challenge`, `spec-clarification-challenge`, and `workflow-status`.

## 10. Validation, closeout, and resume

- Review is risk-driven, not ritual.
- Validation uses fresh evidence against the approved artifact bundle.
- Use repository-owned validation entrypoints instead of ad hoc substitutes. For code, generated artifacts, or CI/CD-sensitive changes, choose the smallest relevant proof, and use the local pre-push or Docker parity path from `docs/build-test-and-development-commands.md` when claiming CI readiness.
- When a GitHub-only check cannot be reproduced locally, name the missing context and keep the claim narrower instead of treating partial local success as remote CI proof.
- Closeout is not complete until artifacts reflect reality.
- Resume from artifacts, not chat memory: read `workflow-plan.md` first, then the current `workflow-plans/<phase>.md`, then the phase-specific artifacts in the order defined by `docs/spec-first-workflow.md`.

## 11. Anti-patterns

- write-capable subagents,
- coding non-trivial work from `spec.md` alone,
- using `workflow-plans/<phase>.md` or `tasks.md` as a second spec or design,
- placeholder artifacts or fake completeness,
- linear skill rituals instead of deliberate orchestration,
- claiming readiness, coverage, or completion without current evidence.

## 12. Maintenance note

Keep this file short, stable, and high-signal. Put detailed artifact shapes, examples, and expanded gate mechanics in `docs/spec-first-workflow.md` or the relevant skill, not here.

@RTK.md
