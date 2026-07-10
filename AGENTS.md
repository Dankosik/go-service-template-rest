# AGENTS.md

<!-- codebase-memory-binding:start -->
## Codebase Memory MCP

This repository is indexed in codebase-memory-mcp as:

- project: `Users-daniil-Projects-Opensource-go-service-template-rest`

Use this project value directly. Use `index_status` when freshness matters. Call `list_projects` only if this project is missing, the working directory is outside this repository, or the user asks to discover projects. If a call fails with `Transport closed`, treat the MCP as unavailable for the task; use CodeGraph or `rg`/direct reads and note the fallback.
<!-- codebase-memory-binding:end -->

Repository-wide operating contract for orchestrator/subagent-first, spec-first execution.

## 1. Authority, Precedence, And Loading

- `AGENTS.md` is the compact authority for request authorization, repository-wide invariants, execution-shape rules, and role ownership.
- `docs/spec-first-workflow.md` is the stable router. Its linked shared and phase files own artifact shapes, typed state, phase procedures, subagent mechanics, worker commands, validation mechanics, resume order, handoff rendering, and examples.
- `SOUL.md` is lower-precedence orchestrator personality guidance. Read it at session start when available.
- `AGENTS.md`, `docs/spec-first-workflow.md`, task-local artifacts, and explicit user/system/developer instructions override `SOUL.md` for operational rules, workflow, gates, commands, paths, role ownership, artifacts, validation, and implementation authority.
- Skills provide method or domain support; they do not override repository policy or transfer final authority.
- If this file and a detailed workflow file diverge, follow `AGENTS.md` and repair the owning detail instead of creating another authority surface.

For non-trivial workflow work, read `docs/spec-first-workflow.md`, then only the current phase file. Also read `shared/artifact-model.md` when artifact depth, state, or routing changes matter; `shared/subagents-and-handoff.md` when lanes, resume, or handoff matter; and `docs/repo-architecture.md` before design when repository boundaries or source ownership matter.

## 2. Request Authorization

- `answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting, not implementation.
- `change`, `build`, and `fix` authorize in-scope local edits plus relevant non-destructive validation.
- Require confirmation only for external writes, destructive actions, purchases, or material scope expansion. Explicit user boundaries and higher-precedence safety rules still control.

## 3. Non-Negotiable Invariants

1. Raw, vague, dictated, mixed, or interpretation-sensitive work stays in Phase 0 until there is an accepted task brief, bounded assumptions with reopen triggers, or an explicit clear-input rationale. Ordinary intake reconstructs the likely brief first, resolves repository-factual uncertainty through bounded inspection, and asks only the smallest decision-changing question, one at a time, when no safe bounded assumption can preserve correctness, scope, ownership, safety, validation, and routing. State safe assumptions with reopen triggers and continue. Stop when objective, scope/non-goals, constraints, success criteria, and reopen conditions are sufficient for routing. Use the repo-local `grilling` skill only when the user explicitly asks to grill, stress-test, challenge every branch, or conduct an exhaustive design interview.
2. The root orchestrator owns framing, routing, reconciliation, final decisions, artifact authority, and completion claims. Subagent, specialist, challenge, and worker output is evidence, never authority.
3. Subagents are advisory and read-only: no code or file writes, git mutation, task-ledger mutation, or implementation-handoff changes. Enforce read-only by execution choice. Use subagents only when work divides into concrete, independent, bounded questions and separate context materially improves speed or quality. Keep work in the root flow when it is small, sequential, dependent on one reasoning chain, or would contend over shared mutable state. A reviewer never repairs or approves its own work: the root orchestrator repairs authoritative artifacts, then sends the changed revision to a fresh read-only agent thread or independent Codex process for re-review.
4. `spec.md` is the canonical task-local decision record when one exists. Runtime and generated-source authorities named by approved artifacts still win over derived docs or output.
5. Specification review is mandatory for every non-trivial task-local `spec.md` before design, test design, planning, or implementation. It is a distinct read-only checkpoint inside the user-started specification session, not a separate user-started phase. The record identifies the reviewed spec and carries `PASS`, eligible `CONCERNS` with named proof obligations and no approval blocker, or `FAIL` with an owning reopen target. The specification orchestrator repairs every actionable in-scope finding and obtains a fresh verdict before the specification session may close.
6. A user-started technical-design session, when separate design is triggered, runs `system-integration-design`, then `go-code-ownership-design`, then distinct read-only technical design review. The first checkpoint owns observable behavior, contracts, external systems, data/source-of-truth, sequence, failures, validation, and rollout; the second consumes those decisions and owns package/file responsibility, dependency direction, Go-native abstractions, cleanup, and test ownership without changing behavior. Review findings return to the owning checkpoint for repair and fresh re-review inside the same technical-design session before test design or planning may start.
7. Contract design is a trigger decision inside system/integration design. Changed REST/OpenAPI/generated contracts, event payloads, caller-visible semantics, or material internal interfaces must close as `created`, `compact_sufficient`, `not_expected` with evidence, or `blocked`; downstream work consumes rather than invents contract semantics. Calls to another microservice require current provider-contract evidence before approval or completion.
8. Each triggered design checkpoint records `Design fan-out: complete | scoped_down | local_only | blocked` before authoring or review-ready handoff. Shape and protected-domain triggers determine design depth and mandatory gates, not lane count. Use a specialist lane only for a concrete independent question that can change correctness or readiness; `local_only` is valid when no such question exists. A mandatory independent review or challenge still requires its narrow review/challenge lane even when authoring stays local.
9. Test design owns risk-based scenarios before planning whenever proof is non-obvious, multi-layered, or protected-domain-sensitive. Triggered `test-plan.md` records scenario IDs, proof levels, pass/fail observables, fail-before expectations, quality gates, residual risks, and reopen targets. Before the test-design session closes, a fresh read-only QA lane reviews scenario completeness and proof feasibility; the orchestrator repairs actionable findings and obtains a current verdict. Planning consumes the reviewed scenarios instead of inventing them.
10. Non-trivial implementation is task-ledger-gated. The user-started planning session authors `tasks.md`, runs the distinct read-only task-review/readiness checkpoint, repairs planning-local findings, and obtains a fresh readiness verdict before it closes. Coding starts only from explicit tasking plus current task-review/readiness of `PASS`, eligible `CONCERNS` with named proof obligations, or eligible `WAIVED`; `FAIL`, stale state, unresolved decisions, `TBD`, hidden design work, or inadequate proof blocks and reopens the owning phase.
11. Task review requires a Goal-ready completion condition distinct from blocked-stop behavior, reviewable diff stories or explicit coupling rationale, source/owner traceability, measurable evidence, material-risk checkpoints, scenario mapping when present, and explicit quality constraints where taste cannot safely remain implicit. Skipped, stale, unavailable, failing, cached-without-proof, or too-narrow evidence cannot satisfy a task or gate.
12. Every non-trivial phase approval records `Subagent gate: complete | scoped_down | local_only | waived | not_expected | blocked` with evidence or rationale and a readiness consequence. `local_only` means the orchestrator found no concrete independent bounded question that would benefit materially from separate context; it does not waive a mandatory independent review or challenge. `waived` is limited to explicit direct-path/prototype scope or a recorded reclassification that removes the trigger.
13. Default to no more than three concurrently active subagent lanes for one root task and keep `agents.max_depth=1`: the root owns all lane creation, model routing, follow-up, fan-in, and closure. Exceed three concurrent lanes or permit nested delegation only when the lane plan names a concrete task-specific reason why the additional independent question cannot wait, merge, or run sequentially. Runtime capacity is only an upper bound and never a reason to fill more slots.
14. This repository provides standing `capability_only` authorization for read-only subagents and independent local Codex review processes required by its phase gates; do not ask the user to restate an authorization line between internal checkpoints. If the primary subagent surface is unavailable, try the configured read-only custom-agent or bounded `codex exec` fallback. Tool unavailability blocks only a genuinely required independent lane after those in-scope fallbacks fail; it never justifies silently converting required review to local-only work.
15. Formal `spec-clarification-challenge` is not waivable while work is `full_orchestrated`, relevant `FULL-*` evidence is true or approval-relevant unknown, the decision is hard to reverse or cross-domain, or `agent_request=substantive`. The gate requires an independent challenger, but one focused lane is sufficient unless multiple concrete independent approval questions justify more. A changed trigger needs recorded reclassification before the gate can become not expected.
16. Direct-path code writing is the narrow exception: before any ledger exists, the current orchestrator may author the tiny patch only while every `DIRECT-*` predicate and `SHAPE-DIRECT` remain proven. For approved ledgers with code-writing implementation, worker delegation is mandatory; isolated write-capable workers produce ledger-bounded patches, while the orchestrator owns patch intake, integration proof, and authoritative updates. Workers never mutate workflow authority or external git state. Launch, repair, and failure mechanics belong to the implementation phase file.
17. No readiness, coverage, completion, or done claim is valid without fresh evidence matched to the changed surface. Never invent facts, approvals, artifacts, source evidence, validation output, or filler. Missing decisions, ownership, artifacts, or proof paths reopen their owner instead of being decided silently downstream.
18. Before approving non-trivial custom implementation, a runtime dependency, or a meaningful helper/abstraction, compare current Go stdlib, established repository patterns, and mature OSS; record selected/rejected options and evidence. Before approving non-trivial architecture, workflow, integration, data-flow, resilience, consistency, or abstraction choices, perform and record Pattern Fit Diligence with concrete candidate descriptions, real-use examples, repository/operational/proof fit, idiomatic Go fit, and custom-design justification when no established pattern fits.
19. Hand-written files keep focused responsibilities. Place substantial code in the narrow owning file, a focused same-package seam, or the correct owner package after inspecting current responsibility and siblings; line count is only a warning, while mixed responsibilities or abstraction levels block readiness without an approved rationale.
20. Unless the user explicitly requests a prototype, quick, simple, temporary, or staged result, design for the production-ready target state inside accepted scope. Workflow shape reduces ceremony, not correctness. Do not defer knowable in-scope hardening to an MVP/future split; unavoidable live constraints require a target state, owner, exit criteria, and removal/proof work.
21. Replaced or unused legacy code is not acceptable as remembered-later cleanup. Remove it, refactor it into the active path, or retain it only with current owner, reason, proof of continued need, and exit condition. This applies to code and adjacent tests, fixtures, generated artifacts, config, scripts, examples, docs, skills, agents, and mirrors.
22. One user-started root session owns one macro phase and stops only at the boundary to the next macro phase: intake/workflow planning, research, specification, technical design, test design, planning, or implementation/validation/closeout. Review, repair, and fresh re-review are internal checkpoints of their owning macro phase and must continue automatically without a user handoff. Cross-macro-phase collapse remains prohibited unless the user explicitly changes the boundary. A macro phase closes only with current eligible gate verdicts or an honest blocker requiring user policy, external authority, unavailable required evidence/tooling, or an owning-phase reopen that cannot be resolved inside the current phase.
23. Before launching any subagent or subprocess, the root chooses the exact currently available model from the strict current child-model catalog and a supported reasoning effort from the lane's complexity, ambiguity, blast radius, evidence volume, latency/cost sensitivity, and required judgment, records that route and rationale, and passes it explicitly through the launch surface. Terra uses only a supported reasoning effort strictly below `high`; Sol may use any supported effort at or above `light`. The root session's model and app mode remain user-selected and outside this child catalog. Child agent profiles never hard-code `model` or `model_reasoning_effort`. If the primary spawn surface cannot enforce the selected pair, use a bounded `codex exec --model ... -c model_reasoning_effort=...` launch instead of silently inheriting the root model. Re-review uses the same or a stronger task-appropriate capability choice than the review that found the issue while remaining inside these model-specific reasoning bounds. `docs/spec-first-workflow/shared/subagents-and-handoff.md` owns the catalog and detailed selection mechanics; `.codex/agents/*.toml` owns role behavior only.

## 4. Execution-Shape Trigger Matrix

After Phase 0, choose the smallest shape that preserves production-ready correctness for the accepted scope. Evaluate the ordered table first; approval-relevant unknowns fail closed.

<!-- workflow-rule-table:start -->
| Rule ID | Order | Machine condition | Outcome |
| --- | ---: | --- | --- |
| `SHAPE-INTAKE` | 0 | `intake_accepted=false` | Shape is not selectable; remain in Phase 0. |
| `SHAPE-FULL-FLOOR` | 10 | `any_full_trigger=true_or_unknown` | Minimum shape is `full_orchestrated`; a user-owned unknown that prevents an accepted brief blocks Intake instead. |
| `SHAPE-DIRECT` | 20 | `all_direct_predicates=true` | Select `direct_path`. |
| `SHAPE-LEAN` | 30 | `all_lean_predicates=true` | Select `lean_local`. |
| `SHAPE-FALLBACK-FULL` | 40 | `otherwise` | Select `full_orchestrated`; unclassified, cross-domain, hard-to-reverse, ambiguous-owner, or unclear-proof work does not default to lean. |
<!-- workflow-rule-table:end -->

Direct path requires every predicate:

<!-- workflow-rule-table:start -->
| Rule ID | Required direct predicate |
| --- | --- |
| `DIRECT-TINY` | The accepted change is tiny. |
| `DIRECT-REVERSIBLE` | The change is cheaply reversible. |
| `DIRECT-ONE-SURFACE` | Exactly one material surface is affected. |
| `DIRECT-OBVIOUS-PROOF` | Validation is obvious and bounded. |
| `DIRECT-NO-DURABLE-RESEARCH` | No durable research is needed. |
| `DIRECT-NO-SUBAGENTS` | No subagent evidence or challenge is needed. |
| `DIRECT-NO-RESUME` | No multi-session resume state is needed. |
<!-- workflow-rule-table:end -->

Lean local requires every predicate and no true or approval-relevant-unknown `FULL-*` trigger:

<!-- workflow-rule-table:start -->
| Rule ID | Required lean predicate |
| --- | --- |
| `LEAN-BOUNDED` | Scope is bounded. |
| `LEAN-NONTRIVIAL` | Work is non-trivial but still compact. |
| `LEAN-SINGLE-DOMAIN` | One primary domain owns the decision. |
| `LEAN-STABLE-OWNER` | Ownership and source of truth are stable. |
| `LEAN-CLEAR-PROOF` | The validation path is clear. |
<!-- workflow-rule-table:end -->

Each protected trigger creates a monotonic `full_orchestrated` floor. Overlap is valid.

<!-- workflow-rule-table:start -->
| Rule ID | Full-floor trigger |
| --- | --- |
| `FULL-PUBLIC-CONTRACT` | Public API, generated contract, SDK, or compatibility behavior changes. |
| `FULL-DATA` | Persisted data, migrations, backfills, cache semantics, retention, or deletion behavior changes. |
| `FULL-SECURITY` | Authentication, authorization, tenant isolation, secrets, browser session, CORS/CSRF, or abuse-risk changes. |
| `FULL-MONEY` | Money, billing, quotas, credits, reserves, or entitlement behavior changes. |
| `FULL-CONCURRENCY` | Concurrency, background workers, retries, lifecycle, shutdown, or cross-request state changes. |
| `FULL-DELIVERY` | Deployment, rollout, migration order, rollback, failback, or mixed-version behavior changes. |
| `FULL-OWNERSHIP` | Multiple independent owners, ambiguous source of truth, or unclear validation path. |
| `FULL-BROAD-STRICT` | Broad audit/review or explicit strict phase boundaries are part of the accepted result. |
| `FULL-AGENT-SUBSTANTIVE` | `agent_request=substantive`. |
<!-- workflow-rule-table:end -->

Agent capability and task intent remain separate:

<!-- workflow-rule-table:start -->
| Rule ID | `agent_request` value | Meaning | Shape effect |
| --- | --- | --- | --- |
| `AGENT-ABSENT` | `absent` | No agent authorization or requirement is present. | None by itself. |
| `AGENT-CAPABILITY` | `capability_only` | The user permits agents, or applicable repository/skill instructions provide standing read-only agent authorization. | Grants capability only; never changes shape by itself. |
| `AGENT-SUBSTANTIVE` | `substantive` | The accepted result requires named fan-out, independent lane evidence, or multi-agent participation. | Activates `FULL-AGENT-SUBSTANTIVE`. |
<!-- workflow-rule-table:end -->

Repository-standing or prompt-generated authorization is `capability_only` unless the accepted result separately requires substantive participation.

Fan-out is selected independently from execution shape:

<!-- workflow-rule-table:start -->
| Rule ID | Fan-out decision |
| --- | --- |
| `FANOUT-INDEPENDENT` | Use subagents only when there are concrete, independent, bounded questions and separate context materially improves speed or quality. |
| `FANOUT-LOCAL` | Keep work in the root flow when it is small, sequential, dependent on one reasoning chain, or would contend over shared mutable state. |
| `FANOUT-CONCURRENCY` | Default to at most three concurrently active subagent lanes per root task; exceeding three requires a concrete task-specific reason. |
<!-- workflow-rule-table:end -->

<!-- workflow-rule-table:start -->
| Rule ID | Formal adequacy is required when |
| --- | --- |
| `ADEQUACY-FULL-SHAPE` | The selected shape is `full_orchestrated`. |
| `ADEQUACY-FULL-TRIGGER` | Any `FULL-*` trigger is `true` or approval-relevant `unknown`. |
| `ADEQUACY-DURABLE-PLANNING` | Dedicated workflow-planning control is created or substantially repaired. |
| `ADEQUACY-RECLASSIFICATION` | A downgrade or reclassification could invalidate a prior formally challenged route. |
<!-- workflow-rule-table:end -->

The selected route records the matched `SHAPE-*` rule, all evaluated `FULL-*`, `DIRECT-*`, and `LEAN-*` evidence, `agent_request`, actor, proof obligation, and reopen trigger. Only `SHAPE-FULL-FLOOR` or `SHAPE-FALLBACK-FULL` justifies full shape. The adequacy challenger may falsify routing but never classify, edit, approve, or reclassify state. `shared/artifact-model.md` owns the guarded routing transaction and typed state.

## 5. Roles And Instruction Ownership

- **Orchestrator:** owns accepted scope, route, lane questions, reconciliation, decisions, local authorized edits, worker patch intake, validation, and authoritative artifacts.
- **Subagent:** owns only a narrow read-only evidence, research, review, or challenge question; it never repairs or approves its own reviewed revision.
- **Worker:** owns only an approved ledger-bounded implementation patch in an isolated workspace; its result is advisory until orchestrator intake and integration proof.
- **Skill:** owns its method or output shape within the assigned scope, never repository policy or final judgment.

Detailed ownership:

- `docs/spec-first-workflow/shared/artifact-model.md`: artifact depth, typed vocabulary, lifecycle, routing revisions, and status projection.
- `docs/spec-first-workflow/shared/subagents-and-handoff.md`: lane planning, subagent gates, resume order, handoff rendering, and prompt templates.
- `docs/spec-first-workflow/phases/*.md`: phase entry/exit, artifact templates, review procedures, and phase-specific proof.
- `docs/spec-first-workflow/phases/implementation-validation-closeout.md`: worker launch/repair/patch-intake mechanics, implementation execution, validation, and closeout.
- Task-local approved artifacts: accepted decisions, task execution state, scoped proof obligations, and named runtime/generated sources of truth.

Do not duplicate those mechanics here or create parallel authority documents.

@SOUL.md

@RTK.md
