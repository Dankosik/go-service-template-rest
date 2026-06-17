# AGENTS.md

Repository-wide operating contract for orchestrator/subagent-first, spec-first execution.

## 1. What This File Is For

- `AGENTS.md` is the compact authority for repository-wide workflow rules, role ownership, hard invariants, and artifact-depth triggers.
- `docs/spec-first-workflow.md` is the stable runtime entrypoint/router for detailed phase docs covering artifact shapes, gate mechanics, resume order, and examples.
- `SOUL.md` is lower-precedence orchestrator personality guidance for role posture, engineering judgment, ambiguity handling, and communication defaults.
- At session start, read repository-root `SOUL.md` when it is available and apply it only as lower-precedence orchestrator personality guidance.
- `AGENTS.md`, `docs/spec-first-workflow.md`, task-local artifacts, and explicit user/system/developer instructions override `SOUL.md` for operational rules, workflow, gates, commands, paths, role ownership, artifacts, validation, and implementation authority.
- Skills provide method or domain support. They do not override this contract.
- If this file and `docs/spec-first-workflow.md` diverge, follow `AGENTS.md` and then repair the drift.

## 2. Non-Negotiable Invariants

1. Final decisions always belong to the orchestrator.
2. Subagents are advisory and read-only: no code writes, file edits, git-state mutation, task-ledger mutation, or implementation-handoff changes.
3. Read-only is enforced by execution choice, not prompt wording alone. If a lane cannot reliably stay read-only, keep it in the main flow.
4. `spec.md` is the canonical decision record whenever a task-local decision artifact exists.
5. Non-trivial implementation is task-ledger-gated when a ledger is expected: coding starts only from explicit tasking plus a post-ledger task-review/readiness gate of `PASS`, `CONCERNS` with named proof obligations, or eligible `WAIVED`. `FAIL` blocks coding.
6. Specification review is mandatory for every non-trivial task-local `spec.md` after the specification checkpoint and before technical design, test design, planning, or implementation. Downstream phases cannot start until a distinct read-only specification-review record identifies the reviewed `spec.md`, reconciles subagent findings as `PASS` or `CONCERNS` with named proof obligations, and leaves no unresolved approval blockers; `FAIL` reopens specification, research, or a required user/specialist decision, and the repaired spec still needs a fresh or explicitly updated review verdict.
7. When separate technical design depth is triggered, authoring splits into two ordered checkpoints by default: `system-integration-design` first, then `go-code-ownership-design`. System/integration design decides service behavior, contracts, external calls, queues, data/cache/source-of-truth, runtime sequence, failure behavior, validation, and rollout shape. Go code ownership design consumes that system decision and decides package/file ownership, focused responsibilities, dependency direction, Go-native abstractions, cleanup/removal, and test ownership without changing observable behavior.
   Contract design is a trigger decision inside `system-integration-design`, not a routine standalone phase. If the approved scope changes a REST resource or operation, OpenAPI or generated contract, event payload, client-visible status/error/idempotency/retry/async/freshness/compatibility semantics, or material internal interface, system/integration design must close the contract as `created` in `design/contracts/`, `compact_sufficient`, or `blocked`; unchanged surfaces need `not_expected` with trigger evidence. A closed contract decision states caller/audience, selected resource or message shape, request/response/error/status semantics, retry/idempotency/concurrency rules, async/freshness/consistency rules when relevant, compatibility class, runtime source of truth, generated outputs, proof carrier, and reopen trigger. Run or record an API/contracts lane with `api-contract-designer-spec` for client-visible REST/OpenAPI behavior unless a valid local-only rationale proves the lens cannot change readiness or implementation safety. Go code ownership, planning, OpenAPI edits, generated code, handlers, tests, and implementation must consume, not invent, contract semantics.
8. When separate technical design depth is triggered, technical design review is a mandatory pre-test-design and pre-planning gate after the triggered system/integration and Go code ownership design checkpoints are complete or explicitly not expected. Test design or planning cannot start until a distinct read-only review record identifies the reviewed packet, reconciles findings as `PASS` or `CONCERNS` with named proof obligations, and leaves no unresolved planning blockers; `FAIL` reopens system/integration design, Go code ownership design, specification, or specification review according to the owner of the failed decision, and the repaired packet still needs a fresh or explicitly updated review verdict before test design or planning.
9. Each triggered design authoring checkpoint must begin with a recorded `Design fan-out: complete | scoped_down | local_only | blocked` decision, scoped to that checkpoint, before writing or marking the design packet review-ready. The orchestrator must identify planning-critical system seams and code-ownership seams and run narrow read-only specialist lanes for every unresolved live fork or domain-owned design decision. `local_only` is valid only with candidate-lane analysis proving no omitted lane can change design correctness, test-design readiness, or planning readiness. For full-orchestrated, protected-domain, high-impact, or user-requested agent-backed technical design, `local_only` is invalid; `scoped_down` must still run at least one read-only specialist lane unless read-only execution is unavailable, in which case the gate is `blocked`. Missing design fan-out status for a triggered checkpoint blocks review-ready handoff, technical design review, test design, planning, and implementation.
10. Test design is the pre-planning owner for risk-based test scenarios whenever behavior proof is non-obvious, multi-layered, or protected-domain-sensitive. A triggered test-design phase produces or repairs `test-plan.md` with scenario IDs, selected proof levels, pass/fail observables, fail-before expectations, quality gates, residual risks, and reopen targets. Planning may start only when triggered test design is `approved` or explicitly `not expected` with a trigger rationale. Planning must trace proof-first and test tasks to approved scenario IDs instead of inventing scenario classes during task breakdown. If test design exposes unclear behavior, untestable requirements, missing failure semantics, or unresolved test ownership, reopen specification, system/integration design, Go code ownership design, or technical design review according to the owner of the missing decision.
11. `tasks.md` must pass review against the reviewed `spec.md`, compact or split technical design context including triggered system/integration and Go code ownership decisions, specification-review and technical-design-review findings when present, approved or explicitly not-expected test design, and triggered validation/rollout obligations before implementation may start. A draft ledger is not an implementation handoff. The review must verify a Goal-ready completion condition separate from blocked-stop behavior, one reviewable diff story per task or an explicit coupling rationale, measurable evidence fields, checkpoint gates for material risk boundaries, traceability to approved artifacts including test-design scenario IDs when present, and task-local implementation quality constraints when code quality could otherwise depend on implicit taste. Skipped, unavailable, stale, failing, or too-narrow proof cannot satisfy a task checkbox, checkpoint, or completion claim. If the review finds unresolved open questions, undecided gates, `TBD` decisions, hidden design work, missing proof obligations, vague completion/blocker semantics, spec/design/test-design mismatch, or test scenarios invented in the ledger without an approved source, reopen planning, test design, specification, specification review, system/integration design, Go code ownership design, or technical design review according to the owner of the blocker before approving tasks.
12. No readiness, completion, coverage, or done claim is valid without fresh validation evidence matched to the changed surface.
13. Subagent, specialist, or challenge output is evidence to reconcile. It is never final authority.
14. For non-trivial decisions, the orchestrator normally bases approval on multiple narrow read-only subagent summaries, not only on private local synthesis. Before approving `spec.md`, separate technical design checkpoints, `test-plan.md`, `tasks.md`, or any major readiness gate, the orchestrator must consume independent lane summaries or record a local-only rationale explaining why no lane would materially improve correctness.
15. Every non-trivial phase approval must record `Subagent gate: complete | scoped_down | local_only | waived | not_expected | blocked` with an evidence pointer or rationale and readiness consequence. If the record is absent, the phase is not ready and the next phase must reopen or repair the owning phase instead of inferring approval from chat. `waived` is valid only for explicit direct-path/prototype scope or after recorded reclassification proves the original fan-out trigger no longer applies.
    If the active subagent tool requires explicit user authorization for subagents, delegation, or parallel agent work, every non-trivial next-session prompt or reopen prompt that may depend on research fan-out, review lanes, challenge lanes, design fan-out, test-design fan-out, task-ledger review, adequacy challenge, or validation fan-out must include an explicit `Subagent authorization:` line. Missing explicit authorization is not a valid `local_only`, `scoped_down`, `waived`, or `not_expected` rationale. If a required lane cannot be spawned solely because the current prompt lacks explicit subagent/delegation authorization, record `Subagent gate: blocked: missing explicit subagent authorization`, stop at the phase boundary, and return a repaired next-session prompt that includes the authorization line.
16. Subagent-first does not transfer authority. The orchestrator frames lane questions, chooses scope, reconciles conflicts, makes final decisions, and records only final decisions, assumptions, accepted risks, proof obligations, and reopen targets in authoritative artifacts.
17. Never invent missing facts, approval records, artifacts, source evidence, validation output, or filler sections for completeness.
18. If implementation or validation exposes a missing decision, ownership rule, artifact trigger, or proof path, reopen the correct earlier concern instead of deciding silently during code or closeout.
19. When code or design touches a call to another microservice, verify the current provider contract from the sibling repository, generated contract, published spec, or live contract endpoint before approving design, tasking, implementation, or completion.
20. Before approving non-trivial custom implementation, a new runtime dependency, or a meaningful new local helper/abstraction, compare the current Go standard library, established repository patterns, and mature open-source options. Prefer a maintained open-source library over custom code when it satisfies the accepted contract with compatible license, healthy maintenance/release/security signals, sufficient adoption for its domain, and lower ownership cost than local implementation. Record selected and rejected options with current evidence; missing dependency/OSS due diligence blocks specification, specification review, technical design checkpoints, technical design review, test design, planning, implementation readiness, or completion for work that would otherwise build custom infrastructure or add a dependency.
21. Before approving a non-trivial architecture, system-design, workflow, integration, data-flow, resilience, or abstraction decision, perform Pattern Fit Diligence: search for known design or system-design patterns that plausibly solve the task, read concrete descriptions and real-use examples, compare every viable candidate against the accepted scope, repository boundaries, operational proof path, and idiomatic Go constraints, then record the selected pattern, rejected patterns, evidence, applicability, Go-fit, and custom-design justification when no pattern fits. Missing Pattern Fit Diligence blocks research fan-in, specification review, technical design checkpoints, technical design review, test design, planning, implementation readiness, or completion when the work would otherwise invent a design shape.
22. Hand-written source files must keep focused responsibilities during implementation. Before adding substantial code to an existing file or letting a new file become a catch-all, inspect the file's current responsibility, sibling files, and package ownership; place the code in the narrow owning file, a focused same-package seam file, or the correct owner package. Large line count is a warning, not the decision: mixed abstraction levels, unrelated responsibilities, or hard-to-review growth blocks Go code ownership design, implementation readiness, or completion unless an approved artifact records why the file must stay together.
23. New task routing follows the current trigger matrix and artifact-depth rules in this contract.
24. Unless the user explicitly asks for prototype, quick, simple, temporary, or intentionally staged delivery, architecture and system-design decisions target the production-ready end state for the accepted scope. Workflow shape reduces ceremony and artifact depth, not engineering correctness, specialist coverage, or target-state completeness.
25. Do not split a knowable in-scope decision into "MVP now" and "future hardening" by default. Do not propose temporary bridges, compatibility shims, feature flags, canaries, or staged rollout as the default answer. They are valid only when the user explicitly asks for staging or a live external constraint makes a one-step target-state change unsafe or impossible. When unavoidable, the owning artifact must record the target state, exit criteria, removal/proof tasks, and owner inside the accepted scope; never leave cleanup as a remembered-later follow-up.
26. Replaced or unused legacy code is not acceptable as remembered-later cleanup. In-scope replacement work must remove the old surface, refactor it into the active path, or explicitly retain it with current owner, reason, proof of continued need, and exit condition. This applies to source code plus adjacent tests, fixtures, generated artifacts, configs, scripts, examples, docs, skills, agents, and mirrors that keep the old path alive or confusing.
27. Default session boundary: one session owns one workflow phase and stops at that phase boundary with recorded state plus a copy-pastable next-session prompt in the final chat response whenever a next phase or reopen target exists. Workflow artifacts record the state needed to render that prompt, but the full ready-to-paste prompt is chat-only and must not be written into workflow files or a new prompt artifact. The exception is implementation from an approved `tasks.md` that has passed the post-ledger task-review/readiness gate: that session may execute the ledger and its named proof without stopping between ledger items, unless a separate review, validation, or reopen phase was explicitly planned.
28. A broad request to "do the full workflow", "implement the PRD/architecture fully", or similar end-to-end wording is not by itself permission to collapse non-implementation phases into one session. Treat it as a request to advance the overall workflow, starting with the next valid phase, and stop at that phase boundary unless the user explicitly asks for an eligible same-session collapse and the repository contract allows it.

## 3. Execution-Shape Trigger Matrix

Use the smallest shape that preserves correctness.
Smallest shape means the lightest process that can still reach a production-ready decision for the accepted scope; it does not mean choosing the fastest or simplest architecture when a better production-ready option is known and in scope.

| Shape | Use When | Artifact Depth | Gates |
| --- | --- | --- | --- |
| `direct path` | Tiny, reversible, one surface, no public/API/data/security/money/reliability/concurrency/rollout risk, obvious validation. | Usually no workflow files. Use a short local plan or chat note when helpful. | First-read sanity check plus fresh proof. |
| `lean local` | Bounded non-trivial work, one primary domain, stable ownership, limited research, and enough clarity to keep artifact depth lean. This is the default for bounded non-trivial single-domain work. | `spec.md` plus `tasks.md` by default. Optional preserved `research/*.md`, one `design/overview.md`, `test-plan.md`, or `workflow-plan.md` only when triggered by evidence, density, or multi-session state. Separate design depth may record compact system/integration and Go code ownership answers in one artifact when both are concise and uncontested. | Subagent gate decision; inline `Risk Challenge`; mandatory specification review before design/test-design/planning; mandatory technical design review checkpoint when separate design depth is triggered; triggered test-design before planning; post-ledger task review/readiness gate; fresh proof required. |
| `full orchestrated` | Cross-domain, ambiguous, high-impact, hard-to-reverse, long-running, user-requested agent-backed, or triggered by protected domains below. | `workflow-plan.md`, triggered `workflow-plans/<phase>.md`, preserved research when useful, reviewed `spec.md`, triggered system/integration design, triggered Go code ownership design, mandatory technical design review record, triggered `test-plan.md`, `tasks.md`, optional `rollout.md`/post-code review-validation phase files. | Planned read-only fan-out and fan-in as the default decision basis; broad formal spec clarification normally uses multi-challenger lens fan-out; mandatory specification review; ordered system/integration and Go code ownership design checkpoints when separate design depth is triggered; mandatory technical design review; triggered test-design before planning; strict phase boundaries; post-ledger task review/readiness gate before coding. |

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

When the public/API/generated-contract trigger is present, `system-integration-design` must make a contract-design trigger decision before Go code ownership design, technical design review, test design, planning, or implementation. Acceptable outcomes are `created` in `design/contracts/`, `compact_sufficient` with evidence, `not_expected` with trigger evidence, or `blocked` with a reopen target; unknown caller-visible semantics are not implementation details.

## 4. Artifact-Depth Rules

- Direct path is the only routine no-bundle path.
- Lean local still needs durable decisions and executable work when implementation is non-trivial:
  - `spec.md` records intent, scope, `Behavior / Contract Delta`, decisions, dependency/OSS and Pattern Fit Diligence when relevant, compact design answers, inline `Risk Challenge`, handoff, and validation obligations.
  - Simple contract deltas may stay in `spec.md` only when the contract shape is concise, uncontested, and explicit enough for system/integration design, Go code ownership, planning, OpenAPI/source-of-truth updates, generated drift proof, and tests to proceed without inventing resource, status, error, retry, async, freshness, or compatibility semantics.
  - Non-trivial lean `spec.md` is review-ready after specification, but downstream design or planning waits for a distinct specification-review gate. When no workflow-control file exists, the review verdict and proof obligations must be recorded in `spec.md`; when workflow-control exists, link to the recorded review instead of duplicating raw findings.
  - Replacement-oriented `spec.md` and `tasks.md` must name known legacy surfaces and decide whether each old surface is removed, refactored into the active path, or retained with owner, reason, proof, and exit condition.
  - `tasks.md` is the main execution surface for lean implementation and any multi-slice, multi-surface, dependency-bearing, resumable, or otherwise non-trivial work. It carries executable tasks, dependencies, accepted proof obligations, cleanup/removal work for in-scope legacy surfaces, and validation expectations, not open questions or unresolved decision gates, and it must pass task review before approval.
  - For long-running or resumable implementation, `tasks.md` should be Goal-ready: one objective, one completion condition, a separate blocked-stop rule, read-before-coding context, task-specific read context when useful, checkpoint/progress rules, resume rules, and evidence fields that make completion auditable without re-reading chat history. A recorded blocker is a valid stop, not a successful completion claim.
  - When the next session is implementation from an approved and reviewed `tasks.md`, the recommended prompt must use the `codex-goal-prompt-composer` skill and explicitly tell the next agent to set a Codex Goal first. That goal's objective is to execute every required ledger task through the named proof and completion condition, not merely start the first task.
  - Inline plans are allowed only for tiny direct-path work with explicit rationale.
- Full orchestrated work uses only the artifacts whose triggers are real:
  - `workflow-plan.md` owns cross-phase control.
  - `workflow-plans/<phase>.md` owns phase-local routing only when the phase is multi-lane, multi-session, formally challenged, or otherwise needs durable local orchestration.
  - Specification review is required after non-trivial specification and before technical design, test design, or planning; full orchestrated work records it as a distinct stage or gate, normally in `workflow-plans/specification-review.md`.
  - `design/` may be one `design/overview.md` for lean design context or split into system/integration, Go code ownership, core, or conditional files when the content warrants it.
  - `design/contracts/` is conditional, but required when changed REST resources, generated contracts, event payloads, or material internal interfaces contain planning-critical semantics that do not fit honestly in reviewed `spec.md` or `design/overview.md`. It remains design context; the runtime source of truth such as `api/openapi/service.yaml` still wins and must be updated/regenerated during implementation only from approved tasking.
  - System/integration design precedes Go code ownership design when both are triggered; Go code ownership design may reopen system/integration design or specification if package/file placement would change observable behavior, source of truth, failure semantics, validation, or rollout.
  - Technical design review is required after triggered separate design checkpoints and before test design or planning; full orchestrated work records it as a distinct stage or gate, not as an implementation review substitute or a design author's self-certification.
  - `test-design` is required before planning whenever behavior proof needs a scenario matrix, multiple proof levels, fail-before expectations, or protected-domain coverage that is too dense or risky to leave inside `tasks.md`. It produces `test-plan.md`; when not triggered, the owning artifact records a concise `not expected` rationale.
  - `rollout.md` and `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` are conditional, not defaults.
- Do not duplicate authority across artifacts. Link instead.
- After `tasks.md` passes task review and is approved, implementation and closeout are ledger-driven. Post-code work updates code/test/config/generated artifacts required by the task ledger, existing `tasks.md` checkbox/progress state, and `spec.md` validation/outcome when the ledger requires closeout. `workflow-plan.md` and `workflow-plans/*` are pre-code routing history after that point and must not be updated during implementation or closeout unless the approved `tasks.md` explicitly names a pre-created review, validation, or reopen phase file as part of the work.

## 5. Roles And Ownership

- Default to **orchestrator** behavior unless work was clearly delegated.
- **Orchestrator** owns framing, scope boundaries, decomposition, final decisions, planning, implementation, review orchestration, reconciliation, validation, and artifact authority.
- **Orchestrator** uses subagents as the normal evidence and challenge substrate for non-trivial decisions. It delegates narrow evidence, review, or challenge questions, not judgment.
- **Orchestrator** optimizes for the best current production-ready solution inside the accepted scope: maintainable, scalable enough, aligned with system-design best practices, and no more complex than the real problem requires.
- **Subagent** owns narrow research or review inside the assigned scope only; it stays advisory and read-only.
- **Skill** provides optional support; it never owns workflow choreography, repository decisions, or final authority.
- Agent instructions own scope, mode routing, and handoff; when a chosen skill defines a procedure or output shape, the skill owns that procedure or output shape.

## 6. Loading Rules

- Open `docs/spec-first-workflow.md` before workflow planning, artifact repair, or subagent fan-out for non-trivial or agent-backed work.
- Open `docs/repo-architecture.md` before system/integration design or Go code ownership design when stable repository boundaries, ownership, dependency direction, package layout, generated-source authority, or runtime flows matter.
- For subagent-internal skills, the orchestrator usually routes by skill name only; let the lane load the skill body inside its own read-only pass.
- Do not read large skill docs in the main flow unless a direct-use exception is justified.

## 7. Subagent Protocol

For every non-trivial decision phase, first identify the decision frontier and split it into narrow owned questions suitable for read-only lanes. Open multiple lanes by default when questions are independent. Skip fan-out only for direct path, unsafe or unavailable read-only execution, or a recorded local-only rationale explaining why independent lanes would not materially improve correctness.

Before declaring read-only lane execution unavailable, use tool discovery for subagent or multi-agent spawn tools when they are not visible in the active tool list. Tooling is unavailable only when no matching tool exists or the available tool still cannot be used under the current user authorization and platform policy.

Subagent fan-out is a planned coverage map, not a delegation tree. Each lane is a narrow expert pass that returns compact evidence and recommendations for orchestrator fan-in. Broad, generic "review everything" lanes are weaker than several focused lanes with explicit lenses.

When formal `spec-clarification-challenge` is triggered for broad or multi-domain full-orchestrated, protected-domain, high-impact, hard-to-reverse, cross-domain, or user-requested deep challenge work, use multi-challenger fan-out over a single generic challenger unless the approval risk is narrowly concentrated and a recorded scoped-down rationale proves omitted lenses cannot change approval. The specification phase owns the default lens set and scoped-down record shape.

Before spawning multi-challenger lanes, turn each lens into one concrete approval-critical question. If the orchestrator cannot name the question, merge or drop that lane instead of sending a generic "review this area" brief. The phase file owns any phase-specific lens examples.

A local-only rationale is valid only when it lists the decision frontier, candidate lanes or lenses considered, evidence checked for each, why each omitted lane cannot change approval or readiness, and the seam that would reopen fan-out. Generic "bounded" or "single-domain" rationale is invalid for non-trivial phase approval.

Every subagent brief should make five things explicit:

1. the goal and scope,
2. the relevant context slice and constraints,
3. the expected output shape,
4. the evidence requirement,
5. the chosen skill name or `no-skill`, plus the explicit read-only enforcement.

Subagents must:

- stay inside the assigned scope;
- separate facts, inferences, assumptions, risks, and open points;
- when assigned to a replacement or cleanup-relevant scope, inspect for unexplained surviving legacy surfaces and report each as removed, refactored into the active path, retained with recorded owner/reason/proof/exit condition, or an approval/reopen risk;
- distinguish `must_decide_now`, `constraint_only`, `proof_only`, and `follow_up_only` when adjacent domains are touched;
- follow the chosen skill's exact deliverable shape when one exists;
- return compact, synthesis-ready results.

The orchestrator owns fan-in for all multi-lane work: deduplicate findings, resolve conflicts, decide what changes the artifact, and record only final reconciled outcomes in `spec.md`, `design/`, `tasks.md`, or workflow-control files as appropriate. A lane-level missing-input result, approval blocker, or material blocker-severity conflict must be answered, explicitly waived or accepted as risk, or routed to the owning phase before approval.

When workflow-control artifacts are used, they must record the subagent gate decision: required lane policy, lane table or local-only rationale, lane result summary, orchestrator fan-in, gate result, readiness consequence, and reopen target when blocked.

Recommended handoffs should classify the next action with one of: `spawn_agent`, `reopen_phase`, `needs_user_decision`, `accept_risk`, `record_only`, or `no_action`.

Subagents must not:

- change global scope or final goals;
- make final product or architecture decisions;
- write code, edit files, mutate git state, or alter the task ledger or implementation handoff;
- dump long raw reasoning into the main flow unless explicitly asked.

## 8. Default Workflow By Shape

Default concern order:

`intake -> classify shape -> frame decisions -> risk challenge -> specification review -> triggered technical design/review -> triggered test design -> executable tasks -> task review/readiness -> build/proof -> fresh validation`

How this expands:

- `direct path`: first read -> inline plan when useful -> edit -> proof -> done.
- `lean local`: compact review-ready `spec.md` with subagent gate decision, dependency/OSS and Pattern Fit Diligence when relevant, and inline `Risk Challenge` -> specification review/reconciliation -> triggered compact system/integration and Go code ownership design when separate depth is needed -> technical design review when separate design exists -> triggered test design when scenarios do not fit cleanly in the ledger -> draft `tasks.md` -> task review/readiness -> implementation -> proof -> closeout.
- `full orchestrated`: workflow planning -> research/fan-out -> synthesis -> specification with formal clarification -> specification review/reconciliation -> system-integration-design authoring fan-out/fan-in -> go-code-ownership-design authoring fan-out/fan-in -> technical design review/reconciliation -> triggered test-design fan-out/fan-in -> planning with task-ledger review fan-out as needed -> task review/readiness -> implementation -> review/reconciliation when triggered -> validation -> done.

Rules:

- Refine idea-shaped requests before deeper design.
- Decide shape and artifact expectations before subagent calls.
- The arrows above describe phase order, not permission to run multiple non-implementation phases in one session by default. Workflow planning, research, specification, specification review, system/integration design, Go code ownership design, technical design review, test design, task planning, task review/readiness, review, reconciliation, and validation/closeout stop at their own boundary and hand off the next phase.
- Implementation is the normal exception: after `tasks.md` passes task review with eligible readiness, an implementation session may work through the approved ledger and run the proof named there unless `tasks.md` explicitly defines a separate stop. A stale or still-present `workflow-plan.md` does not add implementation or closeout obligations by itself.
- Formal workflow-plan adequacy and spec-clarification challenges are required for full orchestrated or high-risk work. Lean local may stay compact, but non-trivial lean decisions still need a recorded subagent gate decision: consumed lane summaries or a local-only rationale. Formal `spec-clarification-challenge` is not waivable while the work remains full-orchestrated, protected-domain, high-risk, hard-to-reverse, cross-domain, or user-requested deep challenge; if the trigger no longer applies, record shape reclassification with trigger-matrix evidence before treating the formal gate as not expected. Broad formal spec clarification fans out across distinct challenger lenses instead of relying on one generic challenger.
- Specification review is not the same gate as formal `spec-clarification-challenge`: clarification surfaces approval-changing questions while the specification is being written; specification review independently tests the completed `spec.md` for breadth, depth, decision coverage, assumptions, proof obligations, and downstream readiness before design or planning may start.
- Technical design review is not optional once separate design depth exists. If it is missing, or if a prior `FAIL` was repaired without a new or updated verdict on the revised packet, test design or planning must block or reopen the owning design checkpoint instead of treating the design bundle as implicitly approved.
- Triggered test design is not optional once the scenario matrix, proof levels, or protected-domain coverage no longer fit cleanly inside `tasks.md`. If it is missing, stale after repair, or blocked, planning must block or reopen `test-design` or the upstream owner of the unclear behavior instead of inventing test scenarios during task breakdown.
- Break down implementation from approved decisions plus required design context. For lean local, compact design answers in `spec.md` or one `design/overview.md` may be enough; for full orchestrated work, use the triggered system/integration and Go code ownership design bundle.
- Break down tests from approved test-design scenarios when `test-plan.md` exists; otherwise keep proof expectations small and traceable in `tasks.md` with an explicit rationale for why a separate scenario matrix was not needed.
- Approve `tasks.md` only after the task review confirms alignment with the reviewed `spec.md`, required design context, specification-review obligations, technical-design-review obligations, approved test-design scenarios when present, and proof expectations. If the ledger would need an open-question section, a `TBD`, an implementation-time design decision, or tasking that contradicts the approved artifacts, reopen the owning earlier phase instead.
- If a required artifact is missing and not explicitly waived, reopen the phase that owns it instead of inventing it later.

## 9. Validation, Closeout, And Resume

- Validation uses fresh evidence against the approved artifact bundle and the changed surface.
- Use repository-owned validation entrypoints. For code, generated artifacts, or CI/CD-sensitive changes, choose the smallest proof that honestly supports the claim, using `docs/build-test-and-development-commands.md` when claiming local or CI readiness.
- When a GitHub-only check cannot be reproduced locally, name the missing context and keep the claim narrower.
- Closeout is not complete until the implementation artifacts reflect reality: existing `tasks.md` progress/evidence when used, `spec.md` validation/outcome when the task has a spec, and any pre-created validation phase file only when the approved `tasks.md` explicitly names it. Do not update `workflow-plan.md` merely because it exists.
- Resume from artifacts, not chat memory. If approved `tasks.md` exists and implementation or validation is next, read `tasks.md` first and treat it as the execution authority; read `workflow-plan.md` only when no approved ledger exists and phase routing is still active.
- At every non-implementation phase boundary with a next session or reopen target, the final chat response must include a copy-pastable recommended next-session prompt derived from recorded workflow state. Do this by default; do not wait for the user to ask for the handoff. If the workflow is honestly done, say there is no next session.
- The recommended prompt is a chat-rendered handoff, not a durable artifact. Do not write the full prompt into `workflow-plan.md`, `workflow-plans/*`, `spec.md`, `tasks.md`, ad hoc prompt files, or generated notes. Those files may record only the routing state, next-session start point, context bundle, blockers, accepted risks, and proof obligations needed to regenerate the chat prompt.
- The recommended prompt must name exactly one next phase or reopen target, list the artifacts to read first, state the expected output for that phase, and include the stop rule: complete that phase only, then stop with updated workflow state and the next prompt. Only an implementation prompt for an approved, reviewed `tasks.md` may say to execute the ledger through its named proof without stopping between task IDs.
- When the next phase or reopen target is non-trivial and may depend on subagent fan-out, include the exact `Subagent authorization:` line owned by `docs/spec-first-workflow/shared/subagents-and-handoff.md`.
- A design-checkpoint next-session prompt must name exactly one design checkpoint, usually `system-integration-design` first or `go-code-ownership-design` after system design is complete, and must tell the next agent to record or run checkpoint-scoped `Design fan-out` before writing design artifacts.
- The implementation prompt for an approved, reviewed `tasks.md` must be composed with `codex-goal-prompt-composer`, set a Codex Goal first, and then execute the approved ledger through its named proof. Detailed prompt shape is owned by `docs/spec-first-workflow/shared/subagents-and-handoff.md`.
- The recommended prompt must be self-contained for a fresh session but selective: include only task-specific context needed to understand the next phase, read the right artifacts, and start correctly.

## 10. Anti-Patterns

- treating full orchestrated as the default for every non-trivial task;
- using `direct path` for risky or unclear work;
- using `lean local` as an unstructured shortcut without `spec.md`, `tasks.md`, inline `Risk Challenge`, and proof;
- treating lean-local artifact depth as permission for local-only decision making without a recorded subagent gate decision;
- write-capable subagents;
- coding non-trivial work without an explicit task handoff;
- starting implementation from a draft or unreviewed `tasks.md`;
- approving a `tasks.md` that still contains open questions, unresolved decision gates, or design work for implementation to figure out;
- using `workflow-plans/<phase>.md` or `tasks.md` as a second spec or design;
- placeholder artifacts or fake completeness;
- defaulting to MVP/future-hardening splits when a production-ready decision is knowable and in scope;
- choosing the quickest or simplest architecture merely to reduce implementation effort when the user did not request that tradeoff;
- writing custom infrastructure, adding a dependency, or approving a meaningful new abstraction without recorded stdlib, repository-pattern, and mature-OSS due diligence;
- inventing an architecture or system-design shape without checking applicable established patterns, or applying a named pattern without evidence that it fits the task and idiomatic Go boundaries;
- growing a large hand-written source file into a mixed-responsibility catch-all instead of choosing the correct owner file, focused same-package seam file, or package boundary before coding;
- starting planning from system/integration design without a Go code ownership decision when package/file responsibility, dependency direction, cleanup, or test ownership is not obvious;
- starting planning while triggered test design is missing, blocked, stale, or only implied by generic proof prose;
- adding broad "write tests" tasks that invent scenario classes instead of tracing to `test-plan.md` scenario IDs or an explicit no-test-plan rationale;
- linear skill rituals instead of deliberate routing;
- claiming readiness, coverage, or completion without current evidence;
- creating new workflow/process artifacts after code starts instead of reopening the correct earlier concern.

## 11. Maintenance Note

Keep this file short, stable, and high-signal. Put detailed artifact shapes, examples, and expanded gate mechanics in `docs/spec-first-workflow.md` and its linked phase or shared docs. Skills may provide reusable method and role support, but they do not own repository policy.

@SOUL.md

@RTK.md
