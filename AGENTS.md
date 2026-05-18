# AGENTS.md

Repository-wide operating contract for orchestrator/subagent-first, trigger-based spec-first execution.

## 1. What This File Is For

- `AGENTS.md` is the compact authority for repository-wide workflow rules, role ownership, hard invariants, and artifact-depth triggers.
- `docs/spec-first-workflow.md` is the detailed runtime companion for artifact shapes, gate mechanics, resume order, and examples.
- Skills provide method or domain support. They do not override this contract.
- If this file and `docs/spec-first-workflow.md` diverge, follow `AGENTS.md` and then repair the drift.

## 2. Non-Negotiable Invariants

1. Final decisions always belong to the orchestrator.
2. Subagents are advisory and read-only: no code writes, file edits, git-state mutation, task-ledger mutation, or implementation-handoff changes.
3. Read-only is enforced by execution choice, not prompt wording alone. If a lane cannot reliably stay read-only, keep it in the main flow.
4. `spec.md` is the canonical decision record whenever a task-local decision artifact exists.
5. Non-trivial implementation is task-ledger-gated when a ledger is expected: coding starts only from explicit tasking plus a post-ledger task-review/readiness gate of `PASS`, `CONCERNS` with named proof obligations, or eligible `WAIVED`. `FAIL` blocks coding.
6. When separate technical design depth is triggered, technical design review is a mandatory pre-planning gate. Planning cannot start until a distinct read-only review record identifies the reviewed packet, reconciles findings as `PASS` or `CONCERNS` with named proof obligations, and leaves no unresolved planning blockers; `FAIL` reopens technical design or specification, and the repaired packet still needs a fresh or explicitly updated review verdict before planning.
7. `tasks.md` must pass review against the approved `spec.md`, compact or split technical design context, technical-design-review findings when present, and triggered validation/rollout obligations before implementation may start. A draft ledger is not an implementation handoff. If the review finds unresolved open questions, undecided gates, `TBD` decisions, hidden design work, missing proof obligations, or spec/design mismatch, reopen planning, specification, technical design, or technical design review according to the owner of the blocker before approving tasks.
8. No readiness, completion, coverage, or done claim is valid without fresh validation evidence matched to the changed surface.
9. Subagent, specialist, or challenge output is evidence to reconcile. It is never final authority.
10. Never invent missing facts, approval records, artifacts, source evidence, validation output, or filler sections for completeness.
11. If implementation or validation exposes a missing decision, ownership rule, artifact trigger, or proof path, reopen the correct earlier concern instead of deciding silently during code or closeout.
12. New task routing follows the current trigger matrix and artifact-depth rules in this contract.
13. Unless the user explicitly asks for prototype, quick, simple, temporary, or intentionally staged delivery, architecture and system-design decisions target the production-ready end state for the accepted scope. Workflow shape reduces ceremony and artifact depth, not engineering correctness or target-state completeness.
14. Do not split a knowable in-scope decision into "MVP now" and "future hardening" by default. Do not propose temporary bridges, compatibility shims, feature flags, canaries, or staged rollout as the default answer. They are valid only when the user explicitly asks for staging or a live external constraint makes a one-step target-state change unsafe or impossible. When unavoidable, the owning artifact must record the target state, exit criteria, removal/proof tasks, and owner inside the accepted scope; never leave cleanup as a remembered-later follow-up.
15. Default session boundary: one session owns one workflow phase and stops at that phase boundary with recorded state plus a copy-pastable next-session prompt whenever a next phase or reopen target exists. The exception is implementation from an approved `tasks.md` that has passed the post-ledger task-review/readiness gate: that session may execute the ledger and its named proof without stopping between ledger items, unless a separate review, validation, or reopen phase was explicitly planned.
16. A broad request to "do the full workflow", "implement the PRD/architecture fully", or similar end-to-end wording is not by itself permission to collapse non-implementation phases into one session. Treat it as a request to advance the overall workflow, starting with the next valid phase, and stop at that phase boundary unless the user explicitly asks for an eligible same-session collapse and the repository contract allows it.

## 3. Execution-Shape Trigger Matrix

Use the smallest shape that preserves correctness.
Smallest shape means the lightest process that can still reach a production-ready decision for the accepted scope; it does not mean choosing the fastest or simplest architecture when a better production-ready option is known and in scope.

| Shape | Use When | Artifact Depth | Gates |
| --- | --- | --- | --- |
| `direct path` | Tiny, reversible, one surface, no public/API/data/security/money/reliability/concurrency/rollout risk, obvious validation. | Usually no workflow files. Use a short local plan or chat note when helpful. | First-read sanity check plus fresh proof. |
| `lean local` | Bounded non-trivial work, one primary domain, stable ownership, limited research, local reasoning can safely close the decision frontier. This is the default for bounded non-trivial single-domain work. | `spec.md` plus `tasks.md` by default. Optional preserved `research/*.md`, one `design/overview.md`, or `workflow-plan.md` only when triggered by evidence, density, or multi-session state. | Inline `Risk Challenge`; mandatory technical design review checkpoint when separate design depth is triggered; post-ledger task review/readiness gate; no mandatory subagent; fresh proof required. |
| `full orchestrated` | Cross-domain, ambiguous, high-impact, hard-to-reverse, long-running, user-requested agent-backed, or triggered by protected domains below. | `workflow-plan.md`, triggered `workflow-plans/<phase>.md`, preserved research when useful, approved `spec.md`, triggered design bundle, mandatory technical design review record, `tasks.md`, optional `test-plan.md`/`rollout.md`/post-code review-validation phase files. | Formal challenge/review lanes as triggered; broad formal spec clarification normally uses multi-challenger lens fan-out; mandatory technical design review when design depth is triggered; strict phase boundaries; post-ledger task review/readiness gate before coding. |

New workflow decisions should use `lean local` for bounded non-trivial single-domain work.

Escalate direct or lean work to `full orchestrated` when any trigger is present:

- public API, generated contract, SDK, or compatibility behavior changes;
- persisted data, migrations, backfills, cache semantics, retention, or deletion behavior;
- auth, authorization, tenant isolation, secrets, browser session, CORS/CSRF, or abuse-risk changes;
- money, billing, quotas, credits, reserves, or entitlement behavior;
- concurrency, background workers, retries, lifecycle, shutdown, or cross-request state;
- deployment, rollout, migration order, rollback, failback, or mixed-version behavior;
- multiple independent owners, ambiguous source-of-truth, or unclear validation path;
- broad audit/review, user-requested subagents, or explicit strict phase boundaries.

## 4. Artifact-Depth Rules

- Direct path is the only routine no-bundle path.
- Lean local still needs durable decisions and executable work when implementation is non-trivial:
  - `spec.md` records intent, scope, `Behavior / Contract Delta`, decisions, compact design answers, inline `Risk Challenge`, handoff, and validation obligations.
  - `tasks.md` is the main execution surface for lean implementation and any multi-slice, multi-surface, dependency-bearing, resumable, or otherwise non-trivial work. It carries executable tasks, dependencies, accepted proof obligations, and validation expectations, not open questions or unresolved decision gates, and it must pass task review before approval.
  - For long-running or resumable implementation, `tasks.md` should be Goal-ready: one objective, one stopping condition, read-first context, checkpoint/progress rules, and evidence fields that make completion auditable without re-reading chat history.
  - When the next session is implementation from an approved and reviewed `tasks.md`, the recommended prompt must be a `/goal` starter prompt whose objective is to execute all ledger tasks through the named proof and stopping condition, not merely start the first task.
  - Inline plans are allowed only for tiny direct-path work with explicit rationale.
- Full orchestrated work uses only the artifacts whose triggers are real:
  - `workflow-plan.md` owns cross-phase control.
  - `workflow-plans/<phase>.md` owns phase-local routing only when the phase is multi-lane, multi-session, formally challenged, or otherwise needs durable local orchestration.
  - `design/` may be one `design/overview.md` for lean design context or split into core/conditional files when the content warrants it.
  - Technical design review is required after triggered separate design and before planning; full orchestrated work records it as a distinct stage or gate, not as an implementation review substitute or a design author's self-certification.
  - `test-plan.md`, `rollout.md`, and `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` are conditional, not defaults.
- Do not duplicate authority across artifacts. Link instead.
- After `tasks.md` passes task review and is approved, implementation and closeout are ledger-driven. Post-code work updates code/test/config/generated artifacts required by the task ledger, existing `tasks.md` checkbox/progress state, and `spec.md` validation/outcome when the ledger requires closeout. `workflow-plan.md` and `workflow-plans/*` are pre-code routing history after that point and must not be updated during implementation or closeout unless the approved `tasks.md` explicitly names a pre-created review, validation, or reopen phase file as part of the work.

## 5. Roles And Ownership

- Default to **orchestrator** behavior unless work was clearly delegated.
- **Orchestrator** owns framing, scope boundaries, decomposition, final decisions, planning, implementation, review orchestration, reconciliation, validation, and artifact authority.
- **Orchestrator** uses subagents to sharpen or challenge the current design frontier, not to outsource system design. It delegates narrow questions, not judgment.
- **Orchestrator** optimizes for the best current production-ready solution inside the accepted scope: maintainable, scalable enough, aligned with system-design best practices, and no more complex than the real problem requires.
- **Subagent** owns narrow research or review inside the assigned scope only; it stays advisory and read-only.
- **Skill** provides optional support; it never owns workflow choreography, repository decisions, or final authority.
- Agent instructions own scope, mode routing, and handoff; when a chosen skill defines a procedure or output shape, the skill owns that procedure or output shape.

## 6. Loading Rules

- Open `docs/spec-first-workflow.md` before workflow planning, artifact repair, or subagent fan-out for non-trivial or agent-backed work.
- Open `docs/repo-architecture.md` before technical design when stable repository boundaries, ownership, dependency direction, or runtime flows matter.
- For subagent-internal skills, the orchestrator usually routes by skill name only; let the lane load the skill body inside its own read-only pass.
- Do not read large skill docs in the main flow unless a direct-use exception is justified.

## 7. Subagent Protocol

Open a subagent lane only when an unresolved owning question needs independent evidence, challenge, or specialist review. Do not use subagents as default ceremony, except that full-orchestrated triggered technical design always gets a distinct read-only technical-design-review gate before planning; the gate may be local only when a scoped-down rationale is recorded.

When formal `spec-clarification-challenge` is triggered for broad or multi-domain full-orchestrated, protected-domain, high-impact, hard-to-reverse, cross-domain, or user-requested deep challenge work, prefer multi-challenger fan-out over a single generic challenger. The default broad clarification shape is five read-only challenger lanes with distinct lenses:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Use more lanes when additional independent approval-risk domains are real, including when one default lens bundles domains that are independently approval-critical for the task. Use fewer lanes only with a recorded scoped-down rationale; a single challenger lane is appropriate only for a narrow formal gate whose approval risk is concentrated in one question.

Before spawning multi-challenger lanes, turn each lens into one concrete approval-critical question. If the orchestrator cannot name the question, merge or drop that lane instead of sending a generic "review this area" brief. Include sibling lens names in each brief so the lane can avoid duplicating adjacent coverage.

Every subagent brief should make five things explicit:

1. the goal and scope,
2. the relevant context slice and constraints,
3. the expected output shape,
4. the evidence requirement,
5. the chosen skill name or `no-skill`, plus the explicit read-only boundary.

Subagents must:

- stay inside the assigned scope;
- separate facts, inferences, assumptions, risks, and open points;
- distinguish `must_decide_now`, `constraint_only`, `proof_only`, and `follow_up_only` when adjacent domains are touched;
- follow the chosen skill's exact deliverable shape when one exists;
- return compact, synthesis-ready results.

The orchestrator owns fan-in for all multi-lane work: deduplicate findings, resolve conflicts, decide what changes the artifact, and record only final reconciled outcomes in `spec.md`, `design/`, `tasks.md`, or workflow-control files as appropriate. A lane-level missing-input result, approval blocker, or material blocker-severity conflict must be answered, explicitly waived or accepted as risk, or routed to the owning phase before approval.

Recommended handoffs should classify the next action with one of: `spawn_agent`, `reopen_phase`, `needs_user_decision`, `accept_risk`, `record_only`, or `no_action`.

Subagents must not:

- change global scope or final goals;
- make final product or architecture decisions;
- write code, edit files, mutate git state, or alter the task ledger or implementation handoff;
- dump long raw reasoning into the main flow unless explicitly asked.

## 8. Default Workflow By Shape

Default concern order:

`intake -> classify shape -> frame decisions -> risk challenge -> executable tasks -> task review/readiness -> build/proof -> fresh validation`

How this expands:

- `direct path`: first read -> inline plan when useful -> edit -> proof -> done.
- `lean local`: compact `spec.md` with inline `Risk Challenge` -> draft `tasks.md` -> task review/readiness -> implementation -> proof -> closeout.
- `full orchestrated`: workflow planning -> research/fan-out as triggered -> synthesis -> specification with formal clarification when triggered -> technical design -> technical design review/reconciliation -> planning -> task review/readiness -> implementation -> review/reconciliation when triggered -> validation -> done.

Rules:

- Refine idea-shaped requests before deeper design.
- Decide shape and artifact expectations before subagent calls.
- The arrows above describe phase order, not permission to run multiple non-implementation phases in one session by default. Workflow planning, research, specification, technical design, technical design review, task planning, task review/readiness, review, reconciliation, and validation/closeout stop at their own boundary and hand off the next phase.
- Implementation is the normal exception: after `tasks.md` passes task review with eligible readiness, an implementation session may work through the approved ledger and run the proof named there unless `tasks.md` explicitly defines a separate stop. A stale or still-present `workflow-plan.md` does not add implementation or closeout obligations by itself.
- Formal workflow-plan adequacy and spec-clarification challenges are trigger-based: required for full orchestrated or high-risk work, optional/local for lean local, and usually skipped for direct path with rationale. Broad formal spec clarification normally fans out across distinct challenger lenses instead of relying on one generic challenger.
- Technical design review is not optional once separate design depth exists. If it is missing, or if a prior `FAIL` was repaired without a new or updated verdict on the revised packet, planning must block or reopen the design phase instead of treating the design bundle as implicitly approved.
- Break down implementation from approved decisions plus required design context. For lean local, compact design answers in `spec.md` or one `design/overview.md` may be enough; for full orchestrated work, use the triggered design bundle.
- Approve `tasks.md` only after the task review confirms alignment with `spec.md`, required design context, technical-design-review obligations, and proof expectations. If the ledger would need an open-question section, a `TBD`, an implementation-time design decision, or tasking that contradicts the approved artifacts, reopen the owning earlier phase instead.
- If a required artifact is missing and not explicitly waived, reopen the phase that owns it instead of inventing it later.

## 9. Validation, Closeout, And Resume

- Validation uses fresh evidence against the approved artifact bundle and the changed surface.
- Use repository-owned validation entrypoints. For code, generated artifacts, or CI/CD-sensitive changes, choose the smallest proof that honestly supports the claim, using `docs/build-test-and-development-commands.md` when claiming local or CI readiness.
- When a GitHub-only check cannot be reproduced locally, name the missing context and keep the claim narrower.
- Closeout is not complete until the implementation artifacts reflect reality: existing `tasks.md` progress/evidence when used, `spec.md` validation/outcome when the task has a spec, and any pre-created validation phase file only when the approved `tasks.md` explicitly names it. Do not update `workflow-plan.md` merely because it exists.
- Resume from artifacts, not chat memory. If approved `tasks.md` exists and implementation or validation is next, read `tasks.md` first and treat it as the execution authority; read `workflow-plan.md` only when no approved ledger exists and phase routing is still active.
- At every non-implementation phase boundary with a next session or reopen target, the final chat response must include a copy-pastable recommended next-session prompt derived from recorded workflow state. Do this by default; do not wait for the user to ask for the handoff. If the workflow is honestly done, say there is no next session.
- The recommended prompt must name exactly one next phase or reopen target, list the artifacts to read first, state the expected output for that phase, and include the stop rule: complete that phase only, then stop with updated workflow state and the next prompt. Only an implementation prompt for an approved, reviewed `tasks.md` may say to execute the ledger through its named proof without stopping between task IDs.
- The implementation prompt for an approved, reviewed `tasks.md` must start with a short `/goal Complete ... without stopping until ...` line, then provide a separate `Implementation brief` with task-local context, artifact read order, constraints, proof obligations, progress-update rule, and blocked-stop rule. Keep execution rules out of the goal line itself.
- The recommended prompt must be self-contained for a fresh session but selective: include only task-specific context needed to understand the next phase, read the right artifacts, and start correctly; do not dump unrelated history, generic repo rules, or full artifact text.

## 10. Anti-Patterns

- treating full orchestrated as the default for every non-trivial task;
- using `direct path` for risky or unclear work;
- using `lean local` as an unstructured shortcut without `spec.md`, `tasks.md`, inline `Risk Challenge`, and proof;
- write-capable subagents;
- coding non-trivial work without an explicit task handoff;
- starting implementation from a draft or unreviewed `tasks.md`;
- approving a `tasks.md` that still contains open questions, unresolved decision gates, or design work for implementation to figure out;
- using `workflow-plans/<phase>.md` or `tasks.md` as a second spec or design;
- placeholder artifacts or fake completeness;
- defaulting to MVP/future-hardening splits when a production-ready decision is knowable and in scope;
- choosing the quickest or simplest architecture merely to reduce implementation effort when the user did not request that tradeoff;
- linear skill rituals instead of deliberate routing;
- claiming readiness, coverage, or completion without current evidence;
- creating new workflow/process artifacts after code starts instead of reopening the correct earlier concern.

## 11. Maintenance Note

Keep this file short, stable, and high-signal. Put detailed artifact shapes, examples, and expanded gate mechanics in `docs/spec-first-workflow.md` or the relevant skill, not here.

@RTK.md
