# Spec-First Workflow

Detailed runtime companion to [AGENTS.md](../AGENTS.md) for subagent-first spec-first work.

## 1. Authority And Purpose

`AGENTS.md` is authoritative for roles, invariants, execution-shape triggers, and hard boundaries. When the two documents diverge, follow `AGENTS.md` first and repair this companion. This document explains how to apply those rules in task-local artifacts without forcing the full bundle onto every bounded change.

The workflow keeps the same quality concerns:

- clear framing and scope cuts;
- final decision ownership by the orchestrator;
- read-only advisory subagents as the normal evidence surface for non-trivial decisions;
- canonical `spec.md` when a task-local decision artifact exists;
- executable task handoff when implementation is non-trivial;
- fresh validation evidence before completion claims.

The simplification is about artifact depth, not decision quality or specialist coverage. Use the smallest workflow shape that preserves correctness.

Subagent-first means non-trivial decisions are normally synthesized from narrow read-only lane summaries. Artifact depth may stay lean, but the decision basis should not be single-threaded when independent evidence, challenge, or specialist review can improve correctness.

Default decision quality:

- Unless the user explicitly asks for prototype, quick, simple, temporary, or intentionally staged delivery, choose the production-ready architecture and system-design answer for the accepted scope.
- Default to a target-state plan: decide the final architecture now, and make the implementation ledger include the work needed to reach that state without remembered-later cleanup.
- Before approving non-trivial custom implementation, a new runtime dependency, or a meaningful new helper/abstraction, compare the current Go standard library, established repository patterns, and mature open-source options. Prefer a maintained open-source library over custom code when it fits the accepted contract and has compatible license, healthy maintenance/release/security signals, sufficient adoption for its domain, and lower ownership cost than local implementation.
- Dependency/OSS due diligence must use current evidence when freshness matters. Useful signals include recent releases or commits, issue and maintainer activity, documented compatibility, license, security advisories or `govulncheck` relevance, transitive dependency cost, API stability, stars or other domain-appropriate adoption signals, and how naturally the library fits repository ownership boundaries.
- Record the selected option, rejected options, evidence date or source, and reason for custom code when no suitable dependency is chosen. Missing due diligence blocks approval or readiness for work that would otherwise build custom infrastructure or add a dependency.
- Before approving a non-trivial architecture, system-design, workflow, integration, data-flow, resilience, or abstraction decision, run Pattern Fit Diligence. Search for known design or system-design patterns that plausibly solve the task, read concrete pattern descriptions and real-use examples, compare candidates against the accepted scope, repository boundaries, operational proof path, and idiomatic Go constraints, then record the selected pattern, rejected patterns, evidence, applicability, and Go-fit. Prefer a proven pattern when it fits; justify custom design only when the known candidates fail a concrete task force.
- Pattern Fit Diligence is not cargo-culting. Do not force Gang-of-Four, enterprise, cloud, or distributed-systems vocabulary onto a simple Go change. The comparison should explain why a pattern fits this task now, or why the straightforward repo-native design is better.
- Code-level pattern fit is a coding and review concern below architecture/system-design. When local implementation is becoming verbose, duplicated, branch-heavy, or helper-heavy, prefer current Go stdlib and established repo idioms first, then small Go-idiomatic code patterns that reduce code and clarify ownership. Do not import class-oriented design-pattern scaffolding into Go unless a concrete local force justifies it and simpler explicit code was rejected.
- Maintainable implementation shape is part of decision quality. Planning and coding must preserve focused file and package responsibilities instead of growing hand-written source files into catch-all modules. Line count is a smoke signal, not the only rule: mixed abstraction levels, unrelated concerns, and hard-to-review growth require a focused same-package seam file, a correct owner package, or a recorded rationale for keeping the code together.
- When a change replaces an old path, replaced or unused legacy code must be removed, refactored into the active path, or explicitly retained with current owner, reason, proof of continued need, and exit condition. This applies to source code and adjacent tests, fixtures, generated output, config, scripts, examples, docs, skills, agents, and mirrors.
- Do not create an MVP-now/future-hardening split when the production-ready decision is knowable and in scope.
- Temporary bridges, compatibility shims, feature flags, canaries, or staged rollout are not default recommendations. Use them only when the user requests staging or a live external constraint makes a one-step target-state change unsafe or impossible.
- When staging is unavoidable, record the target state, exit criteria, removal/proof tasks, and owner in the owning artifact as part of the accepted scope. Do not leave the cleanup as a follow-up or future-hardening note.
- Scope cuts are allowed only as clear non-goals, constraints, or accepted risks. They are not a license to defer required architecture, ownership, contract, reliability, security, or validation decisions.
- When design or code calls another microservice, verify the provider's current contract from its sibling repository, generated contract, published spec, or live contract endpoint before approval or completion, and record the source used in the owning artifact or proof.

## 2. Execution Shapes

| Shape | Use When | Artifact Depth | Gate |
| --- | --- | --- | --- |
| `direct path` | Tiny, reversible, one surface, obvious validation, no protected-domain trigger. | Usually none; a short inline plan or chat note is enough. | Local first-read sanity check and fresh proof. |
| `lean local` | Bounded non-trivial single-domain work, stable ownership, limited research, and enough clarity to keep artifact depth lean. | `spec.md` plus `tasks.md` by default; optional preserved research, one `design/overview.md`, or `workflow-plan.md` only when triggered. | Subagent gate decision; inline `Risk Challenge`; mandatory specification review before design or planning; mandatory technical design review checkpoint when separate design depth is triggered; post-ledger task review/readiness gate. |
| `full orchestrated` | Cross-domain, ambiguous, hard-to-reverse, high-impact, long-running, user-requested agent-backed, or protected-domain work. | `workflow-plan.md`, triggered `workflow-plans/<phase>.md`, preserved research when useful, reviewed `spec.md`, triggered design bundle, mandatory technical design review record, `tasks.md`, optional companion artifacts. | Planned read-only fan-out and fan-in as the default decision basis, mandatory specification review, mandatory technical design review when design depth is triggered, post-ledger task review/readiness gate, and strict phase boundaries. |

Use `lean local` for bounded non-trivial single-domain work. This changes the amount of workflow ceremony, not the expected production readiness, expert coverage, or evidence quality of the chosen solution.

### Escalation Triggers

Escalate from direct or lean local to full orchestrated when the task touches:

- public API, generated contracts, SDK behavior, or compatibility promises;
- persisted data, migrations, backfills, cache semantics, retention, or deletion behavior;
- auth, authorization, tenant isolation, secrets, browser session, CORS/CSRF, or abuse risk;
- money, billing, quotas, credits, reserves, or entitlements;
- concurrency, background workers, retry policy, lifecycle, shutdown, or shared state;
- deployment, rollout, rollback, failback, mixed-version behavior, or migration order;
- multiple independent owners or ambiguous source-of-truth;
- unclear validation path;
- broad audits, user-requested subagents, or explicit strict phase boundaries.

If a trigger appears after work starts, record the reopen target and move to the fuller path instead of stretching the current shortcut.

### Default Session Boundary Policy

Phase arrows describe order, not a default license to collapse multiple phases into one chat session.

Default rule: one session owns one workflow phase, then stops. When the phase has a next phase or reopen target, update the relevant workflow state and end the final chat response with a copy-pastable `Recommended next-session prompt` derived from the recorded artifacts.

A broad user request such as "do the full workflow", "implement the PRD and architecture fully", or "create all necessary documents" advances the overall workflow, but it does not override the one-phase session boundary. Start with the next valid phase, finish that phase honestly, then stop with the next-session prompt.

This boundary rule applies to:

- workflow planning;
- research and fan-in;
- specification and clarification-gate reconciliation;
- specification review and reconciliation;
- technical design;
- technical design review and reconciliation;
- task planning, task-ledger review, and implementation-readiness handoff;
- post-code review or reconciliation phases;
- validation and closeout;
- targeted reopen phases.

The normal exception is implementation from an approved `tasks.md` that has passed the post-ledger task-review/readiness gate. Once implementation readiness is `PASS`, eligible `CONCERNS`, or eligible `WAIVED`, the implementation session may execute the approved ledger items and the proof named by the ledger without stopping between task IDs. After that point, workflow-control files are pre-code routing history unless the approved ledger explicitly names a separate review, validation, or reopen phase file as part of the work.

Direct path work has no durable phase boundary by default, so it may still complete inline with fresh proof.

## 3. Artifact Model By Shape

### Direct Path

Direct path may skip durable workflow artifacts. It still needs:

- a bounded local understanding of the requested change;
- no protected-domain trigger;
- an explicit proof command or manual proof before claiming completion.

Do not create `workflow-plan.md`, `workflow-plans/`, `spec.md`, or `tasks.md` just for ceremony.

### Lean Local

Lean local is the default for bounded non-trivial single-domain work.

Expected artifacts:

- `spec.md`: compact decision record, review-ready after specification and downstream-ready only after specification review passes.
- `tasks.md`: executable task ledger and implementation handoff after task review passes.

Conditional artifacts:

- `research/*.md`: when evidence must survive resume, audit, or later synthesis.
- `design/overview.md`: when compact design answers are too dense for `spec.md` but do not need split design files.
- `workflow-plan.md`: when multi-session state or reopen routing needs a durable control file.

Non-trivial lean-local work must run and record a specification review gate after `spec.md` is written and before compact tasking, separate `design/overview.md`, or planning starts. The review should use at least one read-only specification-review lane unless the task has an explicit direct/prototype waiver; when independent product, domain, API/data, architecture, security, reliability, delivery, or validation questions exist, split them into narrow lanes. If no workflow-control artifact exists, record the verdict and named obligations in `spec.md`; if workflow-control exists, record or link the review there.

If lean local uses a separate `design/overview.md`, run and record a technical design review checkpoint before writing or approving `tasks.md`. The checkpoint should consume a read-only lane summary when an independent design review question exists. A local read-only orchestrator review is allowed only with a recorded local-only rationale, must be distinct from the design-writing pass, and must record `PASS`, `CONCERNS`, or `FAIL`.

Not expected by default:

- `workflow-plans/<phase>.md`;
- split `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`;
- `test-plan.md`;
- `rollout.md`;
- review or validation phase files.

Lean local must not become an unstructured shortcut or a local-only decision path. It requires a recorded subagent gate decision, inline `Risk Challenge`, executable tasks, and fresh proof.

### Full Orchestrated

Full orchestrated keeps the existing full workflow, but all heavier artifacts are trigger-scoped.

Separate technical design is the exception to optional review routing: once `design/overview.md` or split `design/` is triggered, technical design review is mandatory before planning. The review record may live in `workflow-plan.md`, the active phase file, or `workflow-plans/technical-design-review.md` when the review needs durable routing, lanes, blockers, or a session boundary.

Specification review is not optional for non-trivial work. It runs after the specification checkpoint and before technical design, compact lean tasking, or planning. The review record may live in `workflow-plan.md`, the active phase file, `workflow-plans/specification-review.md` when the review needs durable routing, or the lean-local `spec.md` when no workflow-control artifact is used.

Typical layout:

```text
specs/<feature-id>/
  workflow-plan.md
  workflow-plans/
    workflow-planning.md        # only for a dedicated routing phase
    research.md                 # only for a dedicated research phase
    specification.md            # only when formal specification routing is needed
    specification-review.md     # required when non-trivial spec review routing needs durable state
    technical-design.md         # only when dedicated design routing is needed
    technical-design-review.md  # required when separate technical design is triggered and review routing needs durable state
    planning.md                 # only when dedicated planning routing is needed
    review-phase-N.md           # only when planning names a multi-session review checkpoint
    validation-phase-N.md       # only when planning names a multi-session validation checkpoint
  research/
    <topic>.md                  # only when evidence needs to persist
  spec.md
  design/
    overview.md                 # entrypoint when design is triggered
    component-map.md            # split only when useful
    sequence.md                 # split only when useful
    ownership-map.md            # split only when useful
    data-model.md               # conditional
    dependency-graph.md         # conditional
    contracts/                  # conditional design context, not runtime authority
  tasks.md
  test-plan.md                  # conditional
  rollout.md                    # conditional
```

Do not point agents at a specific task-local `specs/...` bundle as required precedent unless that directory exists in the current checkout.

## 4. Lean `spec.md`

Lean specs should answer the planning-critical questions without becoming a design bundle.

Recommended shape:

```markdown
# <Feature / Change>

Mode: lean local
Status: draft | review_ready | approved | implementing | verified
Subagent gate: complete | scoped_down | local_only | waived | not_expected | blocked

## Intent
What changes and why.

## Scope / Non-goals
In:
- ...

Out:
- ...

## Behavior / Contract Delta
ADDED:
- ...

MODIFIED:
- ...

REMOVED:
- ...

## Decisions
- D1: ...
- D2: ...

## Dependency / OSS Due Diligence
Applies: yes | no
Selected approach:
- <stdlib | existing repo pattern | OSS dependency | custom implementation>
Evidence:
- <current source/date, adoption, maintenance, license, security, fit>
Rejected options:
- <option> because <reason>
Custom-code justification:
- <required when selected approach is custom implementation>

## Pattern Fit Diligence
Applies: yes | no
Selected pattern:
- <named pattern or "straightforward repo-native design">
Evidence:
- <source/date or repository precedent, concise pattern description, and real-use example>
Applicability:
- <why the pattern's forces match this task, or why no known pattern fits>
Go fit:
- <why the shape preserves idiomatic Go: explicit control flow, small interfaces, context/cancellation, package ownership, and simple composition>
Rejected patterns:
- <pattern> because <scope/reliability/operability/Go-fit mismatch>

## Compact Design
Affected surfaces:
- `internal/...`

Legacy surfaces:
- Does this change replace an existing path? If yes, list known old identifiers, routes, configs, commands, generated outputs, fixtures, docs, or mirrors to remove, refactor, retain, or prove not applicable. If no, record `No known replacement surface`.

Ownership / source of truth:
- ...

Sequence / failure behavior:
- ...

## Subagent Gate Decision
Gate type: <research fan-out | spec-clarification | local-only rationale | not expected>
Required lane policy: <default lens set | expanded lane set | scoped-down lane set | local-only rationale | not expected>
Consumed lane summaries or rationale:
- <lane/fan-in evidence pointer, or local-only rationale with candidate lanes considered>
Fan-in result:
- <orchestrator-owned resolution>
Readiness consequence: <next phase allowed yes/no, with proof obligations when allowed with concerns>
Reopen target: <none | research | specification | technical-design | planning>

## Risk Challenge
1. What irreversible or externally visible decision could be wrong?
   Answer: ...
2. What hidden invariant or owner could this break?
   Answer: ...
3. Are we writing custom code for a problem already solved by current stdlib, an established repo pattern, or mature OSS?
   Answer: ...
4. What fresh proof will make the completion claim trustworthy?
   Answer: ...
Gate: PASS | CONCERNS | FULL_REQUIRED

## Task Handoff
Use `tasks.md` only after specification review is `PASS` or `CONCERNS` with named proof obligations.

## Validation
Forward-looking proof obligations.

## Outcome
Pending until fresh validation evidence exists.
```

Rules:

- `Behavior / Contract Delta` describes added, modified, and removed behavior instead of restating the whole system.
- Replacement specs must name known legacy surfaces, expected remove/refactor/retain semantics, and the proof that will show each old surface is gone, active through the new path, intentionally retained, or out of scope.
- If the change does not replace an existing path, record `No known replacement surface` so planning does not invent cleanup work.
- `Dependency / OSS Due Diligence` is required when the change would add a dependency, create custom infrastructure, introduce a meaningful helper/abstraction, or solve a problem with plausible standard-library, repository-pattern, or OSS alternatives. If not applicable, record `Applies: no` only when the reason is obvious from scope.
- `Pattern Fit Diligence` is required when the change needs a non-trivial architecture, workflow, integration, resilience, consistency, data-flow, abstraction, or system-design choice. Keep it compact in lean specs when the choice is simple; preserve `research/pattern-fit.md` when multiple candidates, external sources, or examples need to survive; use `design/pattern-fit.md` only when the comparison is planning-critical and too large for `spec.md` or `design/overview.md`.
- `Compact Design` answers affected surfaces, ownership/source-of-truth, and sequence/failure behavior. If those answers become dense or contested, split into design artifacts or escalate.
- `Subagent Gate Decision` is required for non-trivial lean specs. If workflow-control already records the same audit, link to it instead of duplicating raw lane output. A non-trivial lean `spec.md` without this section or link remains draft.
- `Risk Challenge` is the lean replacement for a formal challenge lane only when no escalation trigger is present.
- `FULL_REQUIRED` blocks lean coding and routes to full orchestrated work.
- `Outcome` stays pending until fresh evidence exists.

## 5. Lean `tasks.md`

Lean `tasks.md` is the main execution surface.

Recommended shape:

```markdown
# Tasks

## Goal Contract

Goal objective: Complete <feature/change> by executing this ledger from `T001` through final validation.
Stopping condition: all required tasks are checked, required proof passes or records a concrete blocker, and ledger-owned closeout evidence is current.
Read first: `spec.md`, plus `design/overview.md` when that artifact exists. Do not read `workflow-plan.md` for implementation unless no approved ledger exists yet.
Do not change: <non-goals and preserved constraints from `spec.md`>
Progress log: after each checkpoint, update the checkbox/evidence lines; if blocked, stop and record `Blocked:` with the missing input or failing proof.

## Implementation Handoff

Task ledger review: PASS | CONCERNS | FAIL | WAIVED
Implementation readiness: PASS | CONCERNS | FAIL | WAIVED
Consumes: reviewed `spec.md`, specification-review result, compact design or `design/`, technical-design-review result when present
Design fan-out status: <complete | scoped_down | local_only | blocked | not expected with rationale>
Subagent gates consumed: <gate status and artifact/evidence pointer, or not expected with rationale>
Ledger-review fan-out: <complete | scoped_down | local_only | not_expected | blocked>
Ledger-review fan-out rationale: <required when local_only, scoped_down, or not_expected>
Proof: <smallest sufficient proof command or manual proof>
Reopen target: <none | planning | specification | specification-review | technical-design | technical-design-review>

## Tasks

Legacy cleanup audit:
| Surface | Status | Evidence | Retention owner/reason/exit |
| --- | --- | --- | --- |
| <old identifier/path/config/doc> | removed/refactored/retained/not_applicable | <command/read> | <only if retained> |

- [ ] T001 Add failing proof for <behavior>
  Files: `internal/...`
  Proof: targeted proof fails for the expected reason before implementation.
  Evidence: Pending.

- [ ] T002 Implement scoped production behavior
  Files: `internal/...`
  Proof: targeted proof passes.
  Evidence: Pending.

- [ ] T003 Run validation and record outcome
  Proof: `go test ./...`, `rtk make check`, or the smallest relevant command.
  Evidence: Pending.
```

Rules:

- Use markdown checkboxes and stable task IDs.
- Name one objective and one stopping condition so the ledger can drive a long-running `/goal` without extra chat context.
- Treat non-trivial `tasks.md` as Goal-ready by default in this repository. That means the ledger should contain the Goal Contract fields a later handoff needs; it does not mean a Goal prompt is rendered before the ledger passes review/readiness.
- Keep the Goal contract derivative: it may summarize approved scope, constraints, proof, and stop rules, but must not introduce new decisions or weaken implementation readiness.
- Write the objective and stopping condition so a later implementation handoff can explicitly ask the next session to set a Codex Goal covering all executable ledger tasks.
- Point implementation at the files, docs, plans, or logs it must read first.
- Include checkpoint/progress rules when the ledger spans multiple tasks, sessions, or proof loops.
- Name dependencies when task order matters.
- Include `Subagent gates consumed` and ledger-review fan-out status for non-trivial ledgers; missing gate state keeps `tasks.md` draft.
- Name exact files when known, or narrow artifact/package surfaces when exact file choice is not knowable yet.
- Include proof expectations and an evidence slot per task or checkpoint.
- For replacement work, include executable cleanup audit/removal tasks for every known in-scope old surface: code, tests, fixtures, generated artifacts, configs, docs, scripts, examples, skills, agents, and mirrors. A retained legacy surface must carry owner, reason, proof, and exit condition instead of becoming an implicit follow-up.
- For replacement work, include a `Legacy cleanup audit` table. Each known old surface must have one status: `removed`, `refactored`, `retained`, or `not_applicable`; retained rows must include owner, reason, proof, and exit condition.
- Do not include unresolved open questions, `TBD` decisions, or pending decision gates in `tasks.md`. A ready ledger may carry accepted risks and proof obligations, but any implementation-blocking question must reopen specification, technical design, or technical design review.
- Treat a newly written `tasks.md` as a draft until the task-ledger review has compared it against reviewed `spec.md`, specification-review obligations, required design context, technical-design-review obligations, and triggered validation or rollout obligations.
- For behavior changes and bug fixes, proof-first or test-first is the default.
- For docs, config, or mechanical changes where a failing test is not useful, record the waiver as a proof obligation in `tasks.md`, not only in chat.

## 6. Workflow Control Artifacts

### `workflow-plan.md`

Use `workflow-plan.md` when cross-phase or multi-session state is real. It owns:

- execution shape and rationale;
- current phase and phase status;
- session boundary state;
- next-session routing;
- next-session context bundle;
- artifact status and trigger rationale;
- blockers, accepted assumptions, accepted risks, and reopen targets;
- active gate status such as clarification, adequacy, task-ledger review, or implementation readiness.
- subagent gate audit when the phase is non-trivial, formally challenged, review-bound, or agent-backed.

It does not own final decisions, technical design, executable tasks, raw research, or validation transcripts.

Once `tasks.md` is approved, `workflow-plan.md` no longer owns implementation progress, task completion, or closeout state. It may remain useful historical context, but agents must not update it during implementation or closeout. Pre-created review or validation phase files may be updated only when the approved ledger explicitly names them as required artifacts.

### `workflow-plans/<phase>.md`

Create a phase-local file only when the phase needs durable local orchestration: multi-lane routing, fan-in, formal challenge status, a multi-session stop rule, or named review/validation checkpoints.

It owns:

- local lanes or order/parallelism;
- phase-local completion marker;
- local stop rule;
- next action;
- local blockers;
- gate or handoff status for that phase.
- subagent lane plan, fan-in status, and local-only or scoped-down rationale when relevant.

It must not replace `spec.md`, `design/`, or `tasks.md`.

### Status Vocabulary

Use status words proportionally: `approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, or `conditional`.

- `waived` requires eligible direct-path, lean, or explicitly user-requested prototype rationale and scope.
- `not expected` requires trigger-based rationale when the artifact would otherwise be plausible.
- `conditional` means a later phase must decide the trigger because current evidence is insufficient; do not use it to postpone a knowable production-readiness decision.

## 7. Research

Research is a concern, not always a dedicated phase.

Use local-only research for direct path or when a recorded local-only rationale shows the evidence is trivial, single-source, or not improved by independent lanes.

Dependency/OSS due diligence is a research concern even when it stays compact. Use local research for obvious stdlib or established-repo-pattern choices; use read-only research fan-out when the selected library or custom implementation decision depends on current external health, license/security posture, domain adoption, or integration trade-offs that could materially change approval.

Pattern Fit Diligence is also a research concern when the task has a real design fork. Search for concrete descriptions and examples of relevant patterns, including architecture, integration, consistency, workflow, resilience, data-topology, and Go-friendly implementation patterns. Preserve `research/pattern-fit.md` when the pattern evidence, examples, or candidate comparison would otherwise be lost across sessions; final pattern decisions still belong in `spec.md` or the design bundle.

For non-trivial decisions, first identify distinct evidence questions and normally use read-only fan-out when the questions span more than one domain, artifact family, source-of-truth seam, or risk lens.

Any local-only rationale must list the decision frontier, candidate lanes or lenses considered, evidence checked for each, why each omitted lane cannot change approval or readiness, and the seam that would reopen fan-out. Generic "bounded" or "single-domain" rationale is invalid for non-trivial phase approval.

Preserve `research/*.md` only when it materially helps later synthesis, auditability, or resume. A good research note includes:

- question or scope;
- findings with evidence and limits;
- source notes;
- conflicts, weak evidence, or assumptions;
- handoff implication.

Research notes support decisions but do not own them. Final decisions belong in `spec.md`.

## 8. Specification, Clarification, And Review Gates

`spec.md` is always the decision authority for task-local decisions.

For direct path, no `spec.md` is usually needed.

For lean local:

- write a compact review-ready `spec.md`;
- consume multiple narrow subagent summaries or record a local-only rationale;
- run the inline `Risk Challenge`;
- proceed to specification review only when the orchestrator has reconciled lane outputs or the local-only rationale, and the gate is `PASS` or `CONCERNS` with named proof obligations;
- proceed to compact tasking or technical design only after specification review is `PASS` or `CONCERNS` with named proof obligations;
- escalate when the gate is `FULL_REQUIRED`.

For full orchestrated or protected-domain work:

- run formal `spec-clarification-challenge` before `spec.md` is marked review-ready;
- for broad or multi-domain full-orchestrated, protected-domain, high-impact, hard-to-reverse, cross-domain, or user-requested deep challenge work, use multi-challenger lens fan-out rather than one generic challenger by default;
- use read-only challenger output as questions for orchestrator reconciliation, not as authority;
- store final reconciled outcomes in `spec.md`;
- run a separate specification review after the completed `spec.md` exists and before technical design or planning;
- record gate status in `workflow-plan.md` and the active phase file when those files are used.

Formal `spec-clarification-challenge` is not waivable while the work remains full-orchestrated, protected-domain, high-impact, hard-to-reverse, cross-domain, or user-requested deep challenge. If the trigger no longer applies, first record shape reclassification with trigger-matrix evidence, then record the required subagent gate decision or local-only rationale for the new shape. Otherwise, missing formal clarification blocks `spec.md` from becoming review-ready.

Formal clarification asks only review-readiness-changing questions. Ordinary downstream design detail should be recorded as a constraint, proof obligation, follow-up, or `defer_to_design`, not as a reason to inflate `spec.md`. Do not classify architecture, ownership, contract, reliability, security, rollout, or validation choices this way when they are required to choose a production-ready solution for the accepted scope.

Default broad clarification lenses:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Each lens is a separate read-only lane, usually `challenger-agent` with `spec-clarification-challenge`. Lanes may run in parallel when their questions are independent. Add extra lanes for real independent review-readiness-risk domains, including when one default lens bundles domains that are independently review-readiness-critical for the task. Use fewer lanes only with a recorded scoped-down rationale; a single lane is appropriate only for a narrow formal gate whose review-readiness risk is concentrated in one question.

Before spawning, convert every lens into a concrete review-readiness-critical question and lens-specific inspect-first list. Do not send five challengers the same generic "challenge this spec" prompt. If two lenses produce the same question, merge them or split the real underlying owner question before fan-out.

Do not collapse broad formal clarification into one generic challenger merely because one agent could inspect all domains. Use the default lens set as separate read-only lanes. Fewer lanes require `Scoped-down rationale:` listing every default lens, the review-readiness-critical question considered for that lens, retained lane or lanes, and evidence-backed reason each omitted lens cannot change `spec.md` review-readiness. If any omitted lens has an unresolved review-readiness-critical question, that lens must run.

`Risk Challenge=CONCERNS` in lean local does not by itself trigger formal multi-challenger clarification. It requires named proof obligations and a check for unresolved scope, ownership, validation, or escalation gaps. Route to formal clarification only when those gaps cannot be honestly closed inline or another escalation trigger appears.

Multi-lane workflow-control records should use:

```text
Clarification challenge: complete | blocked | not_expected
Lanes: <agent + skill summary>
Lenses: <lens list>
Scoped-down rationale: <why fewer than the broad default, when applicable>
Resolution: <orchestrator-owned fan-in result>
```

### Specification Review

Specification review is the mandatory post-spec gate for non-trivial work. It is not the same thing as `spec-clarification-challenge`: clarification finds approval-changing questions while candidate decisions are being finalized; specification review inspects the completed `spec.md` for breadth, depth, decision coverage, assumptions, proof obligations, and downstream readiness.

Run specification review after the specification session records `spec.md` as review-ready and before any of these start:

- compact lean tasking;
- separate technical design;
- planning;
- implementation.

Specification review must be read-only and falsification-oriented:

- inspect the completed `spec.md`, workflow-control state, preserved research, formal clarification fan-in, and any linked source-of-truth artifacts;
- check scope/non-goals, behavior/contract delta, product or operator expectations, domain invariants, edge cases, public/API/data/source-of-truth effects, dependency/OSS diligence, Pattern Fit Diligence, legacy-surface handling, security/reliability/delivery implications, validation proof obligations, and downstream handoff clarity;
- verify that every material decision is explicit enough for design or planning without rediscovering product meaning;
- distinguish missing spec decisions from design-owned mechanism choices and planning-owned task ordering;
- report only findings that can change approval, require a named proof obligation, or prevent the next phase from starting honestly.

For non-trivial work, use at least one distinct read-only specification-review lane. Use multiple lanes by default when independent review lenses could change approval, including product/scope coherence, domain invariants, API/data/source-of-truth, architecture ownership, security/reliability/delivery, validation/QA, dependency/OSS, Pattern Fit, and legacy cleanup. A scoped-down review must list candidate lenses considered and why omitted lenses cannot change review readiness. Local-only specification review is valid only for explicit direct-path/prototype waiver or when read-only lane execution is unavailable and the workflow records the consequence as `scoped_down` or blocked.

Specification review must include a compact lens coverage table. Each considered lens is marked `covered`, `not_applicable`, `concern`, or `fail`, with an evidence pointer and short reason. `PASS` is not valid until every readiness-critical lens has a recorded status; omitted lenses require the scoped-down rationale.

Each surviving finding must use this minimum shape:

```text
Finding: <short title>
Spec anchor: <spec section/path>
Evidence: <artifact/source pointer>
Impact: <downstream readiness consequence>
Classification: <classification below>
Required disposition: <repair | user decision | accepted risk | proof obligation | record only>
```

Specification review gate status:

- `PASS`: technical design, compact lean tasking, or planning may start from the reviewed spec.
- `CONCERNS`: the next phase may start only with named accepted risks and proof obligations carried into design, planning, `tasks.md`, `test-plan.md`, or `rollout.md`.
- `FAIL`: downstream phases must not start; reopen specification, research, targeted specialist review, or user decision. Repair alone is not enough to continue; the revised spec needs a follow-up review verdict of `PASS` or `CONCERNS`.

Classify findings by strongest downstream-readiness impact:

- `blocks_spec_approval`: the spec cannot become downstream-ready until the issue is resolved.
- `reopens_specification`: `spec.md` must change before review can pass.
- `reopens_research`: missing evidence prevents an honest spec decision.
- `requires_user_decision`: the missing decision is external product, business, policy, or legal judgment.
- `accepted_risk_candidate`: the orchestrator may proceed only by naming the accepted risk and boundary.
- `proof_obligation`: the spec is coherent, but later artifacts must carry a named proof.
- `record_only`: useful context that does not affect downstream entry.

Record the review result in the active workflow-control surface: `workflow-plan.md`, `workflow-plans/specification-review.md` when a dedicated review phase needs durable routing, or the lean-local `spec.md` when no workflow-control artifact exists. The record must name the reviewed `spec.md`, reviewer or lanes, scope, lens coverage table, findings in the required shape, orchestrator resolution, final gate status, accepted risks, proof obligations, readiness consequence, and reopen target. Review subagents do not edit `spec.md`; if findings require content changes, route to specification repair and run a follow-up review after the repair.

Non-trivial workflow-control records should also include a `Subagent Gate Audit` when a phase depends on fan-out, formal clarification, specification review, technical design review, task-ledger review, or an explicit local-only decision:

```text
Subagent Gate Audit:
- Trigger: <why lanes are required, scoped down, waived, or not expected>
- Gate type: <research fan-out | spec-clarification | specification-review | workflow-adequacy | technical-design-authoring fan-out | technical-design-review | task-ledger-review | review/validation fan-out>
- Required lane policy: <default lens set | expanded lane set | scoped-down lane set | local-only rationale>
- Lane table: <lane id, agent, mode, lens/domain, owned question, skill/no-skill, inspect-first target, order/parallelism, read-only enforcement, status>
- Lane result summary: <strongest finding, classification, recommended handoff, evidence pointer>
- Fan-in: <orchestrator resolution, action, owner/artifact updated, unresolved conflicts, accepted risks, proof obligations>
- Gate result: <complete | blocked | waived | not_expected | PASS | CONCERNS | FAIL>
- Readiness consequence: <next phase allowed yes/no, with proof obligations when allowed with concerns>
- Reopen target: <required when blocked or failed>
```

`Subagent Gate Audit` statuses of `complete`, `PASS`, or `CONCERNS` are invalid when a required lane is missing or pending, a lane blocker or material severity conflict is unresolved, or scoped-down/local-only rationale does not explain why omitted lanes cannot change the decision. Missing audit for a non-trivial phase approval keeps the owning phase draft or blocked; later phases must reopen or repair that phase instead of inferring readiness.

`Lens` is metadata for coverage, not a replacement for `spec-clarification-challenge` classifications. Lane outputs use `blocks_spec_approval`, `blocks_specific_domain`, and `non_blocking_but_record`; shared handoffs use the classifications from `docs/subagent-contract.md`.

The orchestrator owns fan-in:

- deduplicate overlapping questions and findings;
- compare conflicting assumptions across lanes;
- classify each surviving point by strongest justified impact: approval blocker, domain reopen, record-only constraint, proof obligation, accepted risk, or no-action item;
- preserve a short fan-in table or equivalent status in the workflow-control file: lens, strongest finding, classification, action, and owner;
- treat lane-level missing input, unresolved blockers, and material blocker-severity conflicts as blocking the relevant approval area until answered, explicitly waived or accepted as risk, or routed to the owning phase;
- update `spec.md` only with final reconciled outcomes, not raw lane transcripts;
- reopen research, design, planning, or a specialist lane when a finding exposes a missing owner decision.

## 9. Design Depth

Design is content-triggered.

Lean local may keep design answers in `spec.md` `Compact Design` when affected surfaces, ownership, and sequence/failure behavior are concise and uncontested.

Use one `design/overview.md` when lean design context needs more room but still fits one artifact.

Split into design artifacts when the task needs durable, planning-critical context by dimension:

- `design/component-map.md`: affected packages, modules, generated surfaces, adapters, responsibility changes, stable surfaces, and intentional non-touches.
- `design/sequence.md`: runtime order, sync/async boundaries, side effects, failure points, retry/recovery behavior, and parallel versus sequential behavior.
- `design/ownership-map.md`: source-of-truth ownership, allowed dependency direction, generated-code authority, adapter responsibility, and explicit non-owners.

Conditional design artifacts:

- `design/data-model.md`: persisted state, schema, cache contract, projections, replay behavior, retention, or migration shape.
- `design/dependency-graph.md`: module/package dependency shape, generated-code dependency flow, coupling risk, or source-of-truth ambiguity.
- `design/pattern-fit.md`: selected and rejected design or system-design patterns, source descriptions, real-use examples, task applicability, Go-fit, repository-fit, and custom-design justification when the comparison is too dense for `design/overview.md`.
- `design/contracts/`: API/event/generated/material internal interface design context. Runtime authorities like `api/openapi/service.yaml` still win.
- `test-plan.md`: validation obligations are too large or layered for `tasks.md`.
- `rollout.md`: migration sequencing, compatibility window, deploy order, rollback, or failback matters.

If a design trigger is real but the required decision is missing, reopen specification or technical design instead of burying it in `tasks.md`.

### Technical Design Authoring Fan-Out

Separate technical design is not eligible for a private integrated design pass by default. Before writing or marking `design/` review-ready, identify the planning-critical design frontier and record the design-specialist fan-out decision in `workflow-plans/technical-design.md`, `workflow-plan.md`, or the lean-local artifact that owns the design checkpoint.

Use read-only specialist lanes for every unresolved live fork or domain-owned design decision that could change ownership, interfaces, persistence, async or sync semantics, failure behavior, observability, rollout, validation, package boundaries, dependency choice, or Pattern Fit outcome. Typical lane families are architecture/integration, API or contracts, data/source-of-truth, security, reliability/lifecycle, observability, delivery/rollout, performance, QA/proof, dependency/OSS, and Pattern Fit. Each lane owns one concrete question and returns advisory evidence for orchestrator fan-in.

Record the authoring gate in this shape:

```text
Design fan-out: complete | scoped_down | local_only | blocked
Candidate seams: <planning-critical seams considered>
Lane table: <lane id, lens/domain, owned question, skill/no-skill, inspect-first target, read-only enforcement, status>
Collapsed seams: <duplicate or consequence-only seams folded into the integrated design pass>
Escalation seams: <seams that require another lane, specification reopen, research, or user decision>
Fan-in outcome: <orchestrator reconciliation that changes or confirms the design bundle>
Review-ready consequence: <ready for technical design review | blocked | reopen specification/research>
```

`local_only` is valid only when the record lists candidate lanes or lenses considered, the evidence checked for each, why each omitted lane cannot change design correctness or planning readiness, and the seam that would reopen fan-out. Generic "single domain", "bounded", or "obvious" wording is not enough. Missing `Design fan-out` status, skipped candidate-lane analysis, unresolved lane blockers, or material severity conflicts block review-ready handoff and technical design review.

For full-orchestrated, protected-domain, high-impact, or user-requested agent-backed technical design, `local_only` is not an eligible authoring result. A scoped-down gate must still run at least one read-only specialist lane unless read-only execution is unavailable; unavailable read-only execution records `Design fan-out: blocked` and routes to the smallest unblock path.

## 10. Technical Design Review

Technical design review is mandatory whenever separate design depth is triggered. It is the special pre-planning gate that tests whether the design bundle is coherent enough for executable planning.

This gate is not required for direct path work or for lean-local work whose design stays inside `spec.md` `Compact Design`; the inline `Risk Challenge` covers that smaller path. It is required when lean local creates a separate `design/overview.md`, and it is required for full-orchestrated triggered design.

Run technical design review after the design bundle and any triggered conditional artifacts are written, but before `tasks.md` or the task-ledger review/readiness handoff is approved.

If technical design review returns `FAIL`, the next action is a reopen of technical design or specification. After the repair, planning still waits for a follow-up technical design review verdict on the revised packet. The follow-up may be targeted to the failed findings and changed artifacts when the repair is narrow, but it must still check that adjacent design assumptions remain valid and record a new or explicitly updated gate status.

The review packet must be explicit enough that the reviewer does not rediscover phase state from scratch:

- specification-review-approved `spec.md`;
- design entrypoint and triggered design artifacts, with status and trigger rationale;
- triggered `test-plan.md`, `rollout.md`, or explicit not-expected rationale when those surfaces matter;
- workflow-control paths that define the current phase, blockers, and expected review result;
- known assumptions, accepted trade-offs, non-goals, and reopen conditions.

The review must be read-only and risk-driven:

- inspect specification-review-approved `spec.md`, the design bundle, triggered conditional artifacts, `docs/repo-architecture.md` when boundaries matter, and relevant specialist outputs;
- check source-of-truth ownership, dependency direction, runtime sequence, failure behavior, conditional artifact triggers, validation/rollout handoff, dependency/OSS due diligence, Pattern Fit Diligence, and accidental complexity;
- separate design defects from implementation preferences;
- identify any live fork where two plausible design options would materially change ownership, interfaces, data shape, async or sync semantics, operability, rollout, or validation, and verify the design has selected one with a rejection reason for the other;
- challenge the design from the first safe implementation slice: ask whether planning can create executable tasks without adding architecture, ownership, contract, sequencing, rollout, or validation policy;
- choose the strongest justified gate status, avoiding both over-blocking on proof-only concerns and under-blocking on missing ownership, contract, sequencing, rollout, or validation decisions;
- explain why the status is not stronger or weaker, especially for `CONCERNS` versus `FAIL`;
- when recommending `FAIL`, name the smallest reopen target, the decision or artifact that must change, and the concrete condition that a follow-up review should verify;
- return findings as advisory evidence for orchestrator reconciliation.

Technical design review is not a second design pass. If a finding requires a new decision, rewrite of the design bundle, or changed approval boundary, route it back to technical design or specification instead of solving it inside review.

For lean local with one `design/overview.md`, a local read-only orchestrator review is acceptable only when a recorded local-only rationale explains why no independent review lane would materially improve correctness. The checkpoint and result must be recorded before `tasks.md`.

For full orchestrated triggered design, use at least one distinct read-only review lane. Add specialist lanes when independent API, data, security, reliability, observability, delivery, performance, or QA design risks are real. A design-integrator lane is the default fit when the hard part is coherence across specialist concerns. A local-only review requires an explicit scoped-down rationale and cannot be used when independent design questions remain.

When the design packet records a `Design fan-out rationale`, review that rationale first. Do not require retroactive specialist lanes solely because multiple domains are mentioned; require them only when the rationale misses an unresolved live fork, domain-owned decision, or planning-critical proof gap.

Review gate status:

- `PASS`: planning may start from the reviewed design context.
- `CONCERNS`: planning may start only with named accepted design risks and proof obligations.
- `FAIL`: planning must not start; reopen technical design or specification. Repair alone is not enough to enter planning; the revised packet needs a follow-up review verdict of `PASS` or `CONCERNS`.

Gate decision discipline:

- Use `PASS` only when the reviewer has tried to falsify the design against source-of-truth ownership, sequence/failure behavior, validation, rollout, dependency/OSS due diligence, Pattern Fit Diligence, and artifact-trigger expectations and found no planning blocker.
- Use `CONCERNS` only when the design is coherent enough for planning and the remaining risk can be carried as a named accepted risk or proof obligation without asking implementation to choose a missing design decision.
- Use `FAIL` when planning would have to choose between live design options, repair ownership or dependency direction, define missing contract/data/rollout/failure semantics, or resolve a spec/design contradiction.
- Use `record_only` or no finding for cleaner-code preferences unless the issue changes planning safety or production-readiness proof.

Classify findings by strongest planning impact:

- `blocks_planning`: planning would invent or hide an important decision if it started now.
- `reopens_design`: the design bundle must change before review can pass.
- `reopens_spec`: the approved problem frame, invariant, scope, or contract must change.
- `accepted_risk_candidate`: the risk may be accepted only if the orchestrator names the reason and boundary.
- `proof_obligation`: planning may proceed only if the obligation is carried into `tasks.md`, `test-plan.md`, or `rollout.md`.
- `record_only`: useful context that does not affect planning entry.

Record the review result in the active workflow-control surface: `workflow-plan.md`, `workflow-plans/technical-design-review.md` when a dedicated review phase needs durable routing, or the lean-local artifact that owns the review checkpoint. The record must name the reviewed packet, reviewer or lane, scope, findings, orchestrator resolution, final gate status, and planning-input obligations. Follow-up review after `FAIL` must also name the prior failed review, the revised artifacts or decisions, which blockers were closed, any remaining accepted risks or proof obligations, and the new gate status. `CONCERNS` is valid only when every accepted risk and proof obligation is named for planning. Post-code discovery of a missing required technical design review reopens the earlier phase instead of creating a new review artifact after implementation starts.

## 11. Planning, Task Review, And Implementation Readiness

Planning turns reviewed decisions and required design context into `tasks.md`.

Direct path may use an inline plan.

Lean local and full orchestrated work use `tasks.md` for non-trivial implementation.

Planning must not invent missing design context. If exact tasking requires a missing decision, reopen the earlier concern.

`tasks.md` is a draft until the task-ledger review/readiness gate checks it against the approved artifact chain. This gate must run after the ledger is written or materially repaired and before implementation starts.

Task-ledger review must verify:

- specification review is `PASS` or `CONCERNS` with named accepted risks and proof obligations; a missing, `FAIL`, stale-after-repair, or unresolved specification-review gate blocks handoff;
- every in-scope behavior, non-goal, constraint, and accepted decision from reviewed `spec.md` is represented in executable tasking, preserved constraints, or explicit non-task rationale;
- every accepted specification-review `CONCERNS` proof obligation is represented in executable tasking, design constraints, `test-plan.md`, `rollout.md`, or explicit non-task rationale;
- every approved dependency/OSS due-diligence decision is represented in executable dependency, integration, license/security, generation, or proof tasks where relevant; if due diligence is missing for custom infrastructure or a new dependency, reopen specification or technical design instead of letting implementation decide;
- every approved Pattern Fit decision is represented in executable tasking, design-preserving constraints, validation, or explicit non-task rationale; if pattern comparison is missing for an invented design shape, reopen research, specification, or technical design instead of asking implementation to choose a pattern;
- when separate technical design depth was triggered, design fan-out is `complete`, valid `scoped_down`, or eligible `local_only`; a missing, `blocked`, or ineligible `local_only` authoring gate reopens technical design before planning can approve `tasks.md`;
- file and package placement is narrow enough that implementation will not have to choose where a substantial code block belongs; when work touches a large or mixed-responsibility hand-written file, the ledger names the owning file, focused new seam file, package boundary, or approved rationale for keeping the code together;
- known in-scope legacy surfaces are represented as removal/refactor work, retained-surface rationale with owner/reason/proof/exit condition, or explicit not-applicable proof; missing cleanup coverage is a planning blocker, not implementation discretion;
- replacement ledgers include a per-surface cleanup audit table; generic prose is not enough when known old surfaces exist;
- required compact design, `design/overview.md`, or split `design/` ownership, sequence, dependency, failure, and conditional-artifact rules are reflected in task order and proof expectations;
- technical-design-review `CONCERNS` are carried as named accepted risks and proof obligations, and any `FAIL`, unresolved `blocks_planning`, `reopens_design`, or `reopens_spec` finding blocks handoff;
- triggered `test-plan.md`, `rollout.md`, review phase, or validation phase obligations are either represented in the ledger or explicitly marked not expected with rationale before code starts;
- the ledger contains no open-question section, unresolved decision gate, `TBD`, hidden design work, or instruction for implementation to decide architecture, ownership, contract, sequencing, rollout, or validation policy.
- subagent gates consumed by planning are listed, no lane blocker or material severity conflict remains unresolved, and subagent-derived proof obligations are mapped into `tasks.md`, `test-plan.md`, or `rollout.md`.

Before marking `tasks.md` approved for non-trivial work, use a read-only task-ledger review fan-out by default. Typical lanes are coverage and traceability, dependency ordering, proof and QA, and any triggered API, data, security, reliability, delivery, performance, observability, or rollout lens. Each lane reviews the draft ledger against approved artifacts only; no lane edits `tasks.md` or makes final readiness decisions. A local-only or scoped-down ledger review must explicitly evaluate the default lanes and explain why each omitted lane cannot change readiness. Missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`. Without recorded task-ledger review fan-out status or `Ledger-review fan-out rationale:`, implementation readiness remains `FAIL` or blocked.

If the review finds a blocker, use the smallest owning reopen target:

- `planning` for missing task coverage, wrong ordering, vague proof, missing evidence fields, or workflow-control handoff gaps that do not change approved decisions or design;
- `specification review` when a required review verdict is missing, stale after repair, or has unresolved blocking findings;
- `technical design review` when a required review verdict is missing, stale after repair, or has unresolved blocking findings;
- `technical design` when the ledger needs ownership, sequence, dependency, rollout, validation, or conditional-artifact context the design does not provide;
- `specification` when the missing or contradictory point changes accepted scope, behavior, invariant, public contract, non-goal, or approval boundary.

Task-ledger review and implementation readiness use the same status vocabulary:

- `PASS`: coding may start; no hidden architecture, ownership, contract, sequencing, rollout, or validation decision is needed for the next slice.
- `CONCERNS`: coding may start only with named accepted risks and explicit proof obligations; these concerns must be closed as decisions, not open questions.
- `FAIL`: coding must not start; route to the named earlier phase.
- `WAIVED`: allowed only for tiny direct-path or explicitly user-requested prototype scope with explicit rationale.

Readiness belongs in the planning handoff when planning artifacts exist. `workflow-plan.md` and `workflow-plans/planning.md` record the gate status when those artifacts are used; `tasks.md` may carry a short reference. Implementation may start only after task-ledger review produces `PASS`, eligible `CONCERNS`, or eligible `WAIVED`.

Planning consumes the specification review result for all non-trivial work. Missing review, blocking review, or repaired spec after `FAIL` without a follow-up verdict is a planning-entry failure and a task-review blocker, not a detail to infer inside `tasks.md`. When the review result is `CONCERNS`, planning must copy the accepted spec risks and proof obligations into the task-ledger review/readiness handoff and the relevant ledger or companion artifacts.

Planning also consumes the technical design review result whenever separate design depth was triggered. Missing review, blocking review, or repaired design after `FAIL` without a follow-up verdict is a planning-entry failure and a task-review blocker, not a detail to infer inside `tasks.md`. When the review result is `CONCERNS`, planning must copy the accepted design risks and proof obligations into the task-ledger review/readiness handoff and the relevant ledger or companion artifacts.

## 12. Coding, Review, Reconciliation, And Validation

Coding consumes the approved task handoff. It may create or edit code, tests, migrations, configs, generation inputs, and generated output required by the task ledger.

Before adding substantial code to an existing hand-written source file, inspect its current responsibility, sibling files in the package, and package owner. If the new code is a distinct concern, abstraction level, mapping, validation, lifecycle, adapter, or test-helper policy, place it in a focused same-package seam file or the correct owner package instead of enlarging a catch-all file. If that split would change approved architecture, public contract, dependency direction, generated-source ownership, or another protected decision, stop and reopen the owning phase.

Coding may use the selected dependency or custom approach recorded in approved artifacts. If implementation discovers that the chosen approach needs a new runtime dependency, custom infrastructure, or a material helper/abstraction not covered by dependency/OSS due diligence, stop and reopen specification, technical design, or planning according to where the decision belongs. Do not add the dependency or build the custom substitute silently inside coding.

Coding may implement the selected design or system-design pattern recorded in approved artifacts. If implementation discovers that the chosen shape needs a different pattern, a previously rejected pattern, or a custom design not covered by Pattern Fit Diligence, stop and reopen research, specification, technical design, or planning according to where the decision belongs. Do not introduce a new pattern or pattern-like abstraction during coding just because it seems cleaner locally.

Coding may use local code-level patterns only to simplify approved behavior. Good candidates include table-driven tests for several meaningful cases, guard clauses, same-package policy seams, first-class function strategy, narrow consumer-owned interfaces, map-driven dispatch, middleware or decorator only at an existing composition seam, and functional options only when optional construction has real combinatorial pressure. If the pattern adds files, interfaces, callbacks, option bags, or indirection without reducing duplication, branch complexity, ownership ambiguity, or test burden, inline the code or use the stdlib/repo idiom instead.

Cleanup made necessary by the approved task is part of implementation scope. Coding removes stale old-path code and adjacent artifacts, refactors old code into the active path when that is the approved target state, or stops at the smallest reopen target when retention/removal would change public contract, data behavior, security, reliability, rollout, generated contracts, or another protected domain.

If implementation discovers an old surface not named by the approved spec or ledger, classify it before editing: in-scope and safe to remove/refactor, intentionally retained by an existing approved artifact, or requiring reopen because removal or retention changes contract, data, security, reliability, rollout, generated-source, or another protected-domain behavior.

Implementation sessions may continue across the approved `tasks.md` items and the ledger's named proof checks. They must not use implementation momentum to create or approve missing specification, design, planning, review, or validation-phase artifacts.

Post-code work is ledger-driven. It may update only:

- existing `tasks.md` checkbox/progress state;
- `spec.md` `Validation` and `Outcome`.
- existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when the approved `tasks.md` explicitly names that pre-created phase file as part of the post-code checkpoint.

Do not update `workflow-plan.md` or phase-control files merely because they exist. After `tasks.md` is approved, those files are not the implementation source of truth.

Do not create new workflow/process artifacts after implementation starts. Reopen the earlier phase that owns the missing artifact instead.

Review is read-only and risk-driven. Review findings are advisory until the orchestrator reconciles them. Review should flag unexplained surviving replaced or unused code, tests, fixtures, configs, docs, generated artifacts, skills, agents, or mirrors as merge-risk findings unless an approved artifact records why the surface remains with owner, reason, proof, and exit condition.

Review should also flag custom implementations, newly added dependencies, or meaningful helper abstractions that lack approved stdlib, repository-pattern, and OSS due diligence. Severity depends on ownership risk, security/license exposure, transitive dependency cost, and whether a mature maintained library or standard-library path appears to satisfy the same contract.

Review should also flag architecture, workflow, integration, resilience, data-flow, or abstraction shapes that lack approved Pattern Fit Diligence when they appear invented, cargo-culted, or inconsistent with the selected pattern. Severity depends on whether the missing comparison could change ownership, interfaces, failure behavior, validation, rollout, or idiomatic Go implementation shape.

Review should also flag verbose local code that missed an obvious Go-native simplification or small code-level pattern, and pattern-shaped code that adds indirection without reducing duplication, branch complexity, ownership ambiguity, or test burden.

Review should also flag hand-written source files that grew into mixed-responsibility, multi-abstraction-level, or hard-to-review catch-all modules when the approved artifacts did not justify that placement. Severity depends on whether the file now hides ownership, couples unrelated concerns, blocks focused tests, or makes future changes likely to land in the wrong owner.

Severity for unexplained surviving replaced paths is risk-based: `high` when the old path can still execute, import, generate, or validate; `medium` for test, fixture, doc, config, skill, agent, or mirror drift; `low` only when the surface is clearly unreachable, non-authoritative, and unlikely to mislead future work.

Validation uses fresh evidence. A closeout claim is valid only when the commands or manual proof actually cover that claim, including targeted negative searches or reads for retired identifiers and references where text proof is reliable, retained-surface proof when old artifacts remain, generated or mirror drift proof when owning sources changed, and whitespace/drift checks for changed docs or tooling.

Negative proof must name the retired identifiers, paths, commands, config keys, generated files, fixtures, docs, skills, agents, or mirrors searched. A generic search such as `rg legacy` is not sufficient unless the retired surface is literally named `legacy`.

If implementation or validation discovers legacy cleanup that cannot be completed inside the approved scope, record the blocker in the allowed ledger or closeout surface and reopen the smallest owning phase: planning for missing tasking/proof, technical design for new ownership/tooling semantics, or specification for changed scope, public contract, protected-domain behavior, or retention policy.

## 13. Subagents

Subagents are the normal read-only evidence surface for non-trivial decision work, not phase ceremony. Lanes must be narrow, question-owned, evidence-oriented, and reconciled by the orchestrator.

Read-only must be enforced by the actual execution choice. If a lane cannot reliably stay read-only, keep that question in the main orchestrator flow instead of delegating it.

Use `docs/subagent-contract.md` and `docs/subagent-brief-template.md` for reusable brief shape.

If the active tool surface exposes subagent spawning only after an explicit user request for subagents, delegation, or parallel agent work, the repository workflow must carry that request in next-session prompts instead of requiring the user to remember it manually. Before declaring spawning unavailable, use tool discovery for subagent or multi-agent spawn tools when none are visible. If a required lane cannot run solely because the current prompt lacks explicit authorization, record `Subagent gate: blocked: missing explicit subagent authorization`; do not downgrade the gate to `local_only`, `scoped_down`, `waived`, or `not_expected`.

Use this exact line in any non-trivial next-session or reopen prompt whose next phase may depend on research fan-out, specification review lanes, clarification challenge lanes, design fan-out, technical design review, task-ledger review, workflow-plan adequacy challenge, review fan-out, or validation fan-out:

```text
Subagent authorization: I explicitly request and authorize read-only subagents, delegation, and parallel agent work for every repository workflow gate that requires or benefits from fan-out in this session. Spawn the required read-only lanes without asking again; the orchestrator retains final authority and reconciles results.
```

Every lane needs:

- goal and exact question;
- scope and constraints;
- lens or specialist domain when multiple challenge lanes share the same artifact;
- expected output;
- evidence requirement;
- skill name or `no-skill`;
- read-only enforcement.

A lane uses at most one skill. If the selected skill defines a stricter deliverable, follow it. Otherwise use the shared envelope from `docs/subagent-contract.md`. Multi-lane challenge improves coverage only when lanes have distinct lenses and an explicit fan-in path.

Lane planning should be a coverage map:

- choose the independent questions that can change the current decision or gate;
- assign each question to the narrowest suitable expert lane;
- include sibling lens names when lanes share the same artifact bundle;
- merge duplicate lanes and record why omitted lenses cannot change the decision;
- preserve only compact lane summaries and reconciled outcomes in authoritative artifacts.

## 14. Resume Order

Resume from artifacts, not chat memory.

If approved `tasks.md` exists and implementation, review, validation, or closeout is next:

1. read `tasks.md`;
2. read the artifacts named by `tasks.md`, usually reviewed `spec.md`, specification-review result, and any required design, test-plan, or rollout context;
3. read `workflow-plans/<phase>.md` only when `tasks.md` explicitly names a pre-created review or validation phase file.

If there is no approved `tasks.md` and `workflow-plan.md` exists:

1. read `workflow-plan.md`;
2. read the current `workflow-plans/<phase>.md` if the task uses one;
3. read the files named in the `Next session context bundle`;
4. then read phase artifacts as needed.

If there is no approved `tasks.md` and no `workflow-plan.md` because the task is direct or lean:

1. read `spec.md` when it exists;
2. read the specification-review record when non-trivial work is moving beyond specification;
3. read `tasks.md` when implementation or validation is next;
4. read optional `research/*.md` or `design/overview.md` only when named or needed.

Treat missing expected artifacts as incomplete unless an explicit waiver covers that exact artifact.

## 14. Final Chat Handoff

When any non-implementation workflow phase reaches a boundary and a next session or reopen target exists, the final chat response must include a copy-pastable recommended next-session prompt derived from recorded artifacts. This is default behavior; the user does not need to ask the agent to stop or produce the handoff prompt.

Assume the next session is context-blind: it can read repository files, but it cannot see the current chat. The prompt should carry a short task-specific context capsule that explains the current state, why the named next step is next, and what the next session must not lose. It should not become a transcript, broad project summary, or second copy of the artifacts.

Select context by relevance:

- include the current workflow state, accepted objective, and the reason this exact next phase or reopen target is next;
- include exact artifact paths, task IDs, phase names, commands, blocker names, accepted decisions, accepted assumptions, accepted risks, and proof obligations that matter to the next phase;
- include one-line reasons for non-obvious files in the context bundle so the next session knows why to read them;
- omit generic repository rules already covered by `AGENTS.md`, long artifact excerpts, resolved debate history, unrelated prior-session details, and context the next phase can cheaply rediscover from the listed files;
- when uncertain, include a bounded assumption or reopen target instead of padding the prompt with unrelated context.

The recommended prompt should be operational, not just descriptive. Include:

- the exact next phase or reopen target;
- the artifact read order, task-local paths, and short reasons for any non-obvious context files;
- the immediate objective and expected output for that one phase;
- important blockers, accepted assumptions, accepted risks, and proof obligations from recorded state;
- `Subagent authorization:` with explicit permission for read-only subagents, delegation, and parallel agent work whenever the next phase is non-trivial or may run subagent/readiness gates;
- a stop rule telling the next session to complete only that phase, update workflow state, and produce the following next-session prompt if another phase remains.

When the next phase is `technical-design`, the prompt must tell the next session to first record or run `Design fan-out` and only then write or repair integrated design artifacts.

For implementation from approved `tasks.md` that has passed task-ledger review/readiness, compose the prompt with `.agents/skills/codex-goal-prompt-composer/SKILL.md`. The prompt must explicitly tell the next session to set a Codex Goal first, then execute all required tasks in the approved ledger from start to finish. It must not rely on a slash command being parsed from the handoff. It may tell the next session to execute the approved ledger and run its named proof without stopping between task IDs. It must still prohibit creating or approving missing pre-code workflow artifacts during implementation.

Implementation goal handoff rules:

- use `codex-goal-prompt-composer` whenever the recommended next-session prompt sets a Codex Goal;
- apply that skill's Goal Line Quality Gate before returning the prompt;
- start the fenced prompt with `First, set a Codex Goal for this session:` followed by a short durable goal objective;
- the next paragraph must say `After the goal is set, execute every required task in <tasks.md path> from start to finish`;
- derive `<approved objective>` and `<verifiable stopping condition>` from the `tasks.md` Goal Contract and implementation-readiness handoff;
- scope the goal to all executable tasks in the approved ledger, from the recorded first task or checkpoint through final validation, not just the first task ID;
- keep the Codex Goal objective as a durable objective only; do not pack artifact lists, constraints, risks, commands, or detailed execution rules into it;
- put all execution details under `Implementation brief` so the durable goal stays stable while the working instructions remain readable;
- include working directory, artifact read order, task-local paths, accepted constraints, accepted risks, proof obligations, and named validation commands or manual proof;
- include `Subagent/readiness gates: <status, evidence artifact, proof obligations, reopen target if blocked>` whenever the next phase depends on a subagent/review gate or local-only rationale;
- if readiness is `CONCERNS` or `WAIVED`, keep the Codex Goal objective focused on the approved objective and put the concern, waiver rationale, and required proof obligations in the implementation brief;
- tell the next session to update only ledger-owned progress/evidence and closeout surfaces permitted by `tasks.md`;
- if the `tasks.md` Goal Contract is missing or too vague to form a verifiable Codex Goal, do not invent a broad objective; reopen planning to repair the Goal Contract or mark the implementation prompt as blocked;
- include a blocked-stop rule: if an implementation-blocking decision, missing artifact, or failing proof cannot be resolved inside the approved ledger, stop, record the blocker in the allowed ledger/closeout surface, and return the exact reopen target instead of inventing new workflow artifacts.

Recommended implementation prompt shape:

```text
First, set a Codex Goal for this session:
Complete <approved objective> by executing every required task in `<task-local>/tasks.md` without stopping until <verifiable stopping condition>.

After the goal is set, execute every required task in `<task-local>/tasks.md` from start to finish. Start at <T001 or recorded checkpoint>, continue through the ledger's final validation/proof, and do not redefine success around a smaller slice.

Implementation brief:

Work in `<absolute repo path>`.
Subagent authorization: I explicitly request and authorize read-only subagents, delegation, and parallel agent work for every repository workflow gate that requires or benefits from fan-out in this session. Spawn the required read-only lanes without asking again; the orchestrator retains final authority and reconciles results.
Read first:
- `<task-local>/tasks.md` because it is the approved implementation ledger and source of truth.
- `<task-local>/spec.md` because it is the canonical decision record.
- <additional task-local artifacts named by `tasks.md`, each with a one-line reason>.

Current state:
- Next phase: implementation.
- Task-ledger review: <PASS | CONCERNS | WAIVED>.
- Implementation readiness: <PASS | CONCERNS | WAIVED>.
- Subagent/readiness gates: <status, evidence artifact, proof obligations, reopen target if blocked>.
- First executable task/checkpoint: <T001 or named checkpoint>.
- Accepted concerns or waiver: <none | named concern/waiver plus proof obligation>.

Execution rules:
- Execute all required tasks in `tasks.md` in dependency order through the ledger's named proof; do not stop between task IDs unless blocked.
- Preserve the accepted constraints, non-goals, risks, and proof obligations recorded in the listed artifacts.
- Do not create or approve missing pre-code workflow artifacts during implementation.
- Update existing `tasks.md` progress/evidence and any ledger-owned closeout surfaces exactly as allowed by `tasks.md`.
- If blocked by a missing decision, missing artifact, or unresolved failing proof outside the approved ledger, stop with the blocker, evidence, and exact reopen target.
```

Use no prompt when the workflow is honestly done.

The prompt is chat-only. It is not a workflow artifact and must not become a second source of truth.

Before returning the prompt, apply the start test: a new session with no chat history should know the single next phase, why it is next, what to read first, what constraints and proof obligations matter, and where to stop. Remove any sentence that does not help that session start or avoid a real mistake.

## 15. Anti-Patterns

Avoid:

- treating full orchestrated as the default for every non-trivial task;
- using direct path for risky, ambiguous, public, data, security, money, reliability, concurrency, or rollout work;
- using lean local without `spec.md`, `tasks.md`, inline `Risk Challenge`, and proof;
- using lean local as a local-only decision path without a recorded subagent gate decision;
- making `workflow-plans/<phase>.md` a second master plan, spec, design bundle, or task ledger;
- planning non-trivial implementation from `spec.md` alone when design context is triggered;
- starting implementation from an unreviewed `tasks.md` or treating a draft ledger as approval;
- approving non-trivial specs while formal challenge is required and unresolved;
- splitting work into MVP plus future hardening when the production-ready decision is knowable and in scope;
- picking the fastest or simplest architecture by default instead of the best production-ready choice for the accepted scope;
- inventing a custom architecture or system-design shape without Pattern Fit Diligence, or applying a named pattern without task-specific evidence and Go/repository fit;
- importing class-oriented design-pattern scaffolding into Go or adding pattern-shaped helpers when direct stdlib/repo-native code is shorter and clearer;
- growing large hand-written source files as an implementation shortcut instead of placing new code in the focused owner file, same-package seam file, or correct package boundary;
- creating `test-plan.md`, `rollout.md`, split design files, or review/validation phase files for completeness;
- creating new process artifacts after coding starts;
- using subagents for broad ceremony rather than narrow unresolved questions;
- claiming done without fresh, scope-matched evidence.
