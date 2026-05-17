# Workflow Simplification Specification

Status: approved for implementation planning

## Context

`go-service-template-rest` is an AI-native Go REST service template with a rigorous orchestrator-first, spec-first workflow. The current workflow intentionally protects decision ownership, context preservation, safe delegation, executable handoff, and validation proof. The user's concern is not that these stages are worthless. The concern is that real template usage shows the current process creates too much phase and artifact overhead for modern LLM agents.

The proposal is grounded in:
- local workflow docs and contracts: `AGENTS.md`, `RTK.md`, `docs/spec-first-workflow.md`, `docs/subagent-contract.md`, `docs/subagent-brief-template.md`, `docs/build-test-and-development-commands.md`
- current workflow skill surfaces under `.agents/skills/`
- the completed `specs/railway-auto-migrations/` bundle as the concrete local example
- external workflow practices preserved in `research/external-agent-workflow-practices.md`
- local pain map preserved in `research/current-workflow-pain-map.md`

## Current Workflow Pain Points

1. The current workflow has the right quality concerns but too many default-looking control surfaces.
   - `workflow-plan.md`, `workflow-plans/<phase>.md`, `spec.md`, split `design/`, `tasks.md`, and conditional research/test/rollout files are all valid in full orchestrated work.
   - For bounded local work, this becomes more state to maintain than the task needs.

2. Phase-local workflow plans often duplicate master workflow state.
   - The phase file is useful when it owns lanes, fan-in, challenge status, or multi-session routing.
   - It is mostly overhead when it only repeats current phase, blockers, next action, and stop rule.

3. The design bundle carries valuable questions in an expensive shape.
   - Affected surfaces, runtime sequence, and ownership/source-of-truth are critical.
   - Four separate design files are excessive for many bounded template changes.

4. Challenge gates are valuable but over-prescribed for local low-risk work.
   - Independent challenge should remain mandatory for high-risk or full orchestrated work.
   - Local bounded work should be allowed to use a recorded checklist-style self-check instead of a separate challenge lane.

5. The direct and lightweight/lean paths exist, but they are not prominent enough.
   - The detailed workflow already allows direct-path collapse.
   - In practice, agents see the full artifact layout first and tend to create more process than needed.

6. More files create more stale-state risk.
   - Multiple status carriers can diverge on current phase, artifact status, blockers, and proof.
   - Modern agents benefit from fewer, stronger state surfaces.

## External Practice Synthesis

External systems do not solve this by removing discipline:

- BMAD preserves analysis, planning, solutioning, implementation, readiness checks, stories, and review, but also offers quick flow for small work.
- Superpowers preserves brainstorm, plan, implementation, TDD, review, and finish, but relies on composable skills and executable plans rather than a large fixed artifact bundle for every task.
- GSD preserves discuss, plan, execute, verify, and ship, but keeps state in persistent `.planning` artifacts, uses quick flags, fresh contexts, wave execution, and state validation/sync.
- GitHub Spec Kit preserves constitution, specification, implementation plan, tasks, and implementation, with optional clarify/analyze/checklist commands.
- Kiro Quick Plan preserves requirements, design, and tasks but generates them in one pass without phase approval gates for well-understood features.
- OpenSpec's progressive-rigor and delta-spec model fits brownfield services because changes can describe added, modified, or removed behavior instead of re-documenting the whole system.
- Task Master preserves PRD-to-task decomposition, dependency ordering, complexity expansion, and test strategy, with an optional more structured RPG template for complex work.
- Warp treats plans as persistent editable objects with version history, selective execution, progress monitoring, export, and cross-conversation reuse.

The shared pattern is: keep the quality concerns, make the heavier mechanisms conditional, and put the agent on an executable path as soon as the real risk is bounded.

## Recommended Simplified Workflow

Adopt a trigger-based workflow model:

```text
intake -> classify shape -> frame decisions -> risk challenge -> executable tasks -> build/proof -> fresh validation
```

The named concerns remain:
- framing/specification
- research when evidence is missing or external/current facts matter
- design when ownership, sequence, data, API, rollout, or dependency shape matters
- planning when implementation is more than a tiny local edit
- review/challenge when ambiguity, risk, or independent judgment materially improves quality
- validation before completion claims

The change is that these concerns no longer imply separate phase files or separate sessions by default.

## Target Modes

This proposal keeps the current execution-shape concept but changes the default posture:

| Mode | Role |
| --- | --- |
| `direct path` | Tiny, reversible fixes that can be handled in one local pass without durable workflow artifacts. |
| `lean local` | New default for bounded non-trivial single-domain work. It is the renamed and strengthened successor to today's `lightweight local`; existing docs may mention `lightweight local` as a compatibility alias during migration. |
| `full orchestrated` | Current full workflow, preserved for named risk triggers, long-running work, multi-agent coordination, or user-requested strict phase boundaries. |

The guiding rule is: use the smallest mode that preserves correctness. `full orchestrated` requires a named trigger; it is not the default for every non-trivial task.

## Approval Record

Approved direction: adopt the trigger-based simplification model.

Resolved approval choices:

1. Keep `workflow-plan.md` as the optional state/control file for compatibility. Do not introduce `workflow.md` in the first implementation.
2. Make `lean local` the default for bounded non-trivial single-domain work; keep `lightweight local` as a compatibility alias in old artifacts and transitional docs.
3. Allow lean local work to approve `spec.md` with a recorded inline `Risk Challenge` instead of a formal `spec-clarification-challenge`, but only when no escalation trigger is present.
4. Allow lean design answers either inside `spec.md` or in one `design/overview.md`, depending on how much durable technical context the task needs.
5. Keep `tasks.md` required for lean implementation and for any multi-slice, multi-surface, dependency-bearing, resumable, or otherwise non-trivial implementation. Tiny direct-path work may use an inline plan with an explicit waiver.
6. Use delta-style `Behavior / Contract Delta` in lean specs by default.
7. Treat proof-first or test-first implementation as the default for behavior changes and bug fixes, with explicit waiver only when the changed surface does not support a useful failing test.
8. Update docs and workflow-related skills in one coordinated implementation pass so agent instructions, workflow docs, subagent docs, and skills do not drift.

Implementation constraints:
- Do not rewrite historical task bundles.
- Preserve the full orchestrated path for high-risk, cross-domain, public-contract, data, security, money, reliability, rollout-sensitive, long-running, or user-requested agent-backed work.
- Preserve non-negotiable invariants: orchestrator authority, read-only advisory subagents, canonical `spec.md`, executable task handoff when needed, and fresh validation evidence before completion claims.

## Keep, Merge, Remove, Or Make Conditional

| Current Piece | Recommendation | Replacement Or Rule |
| --- | --- | --- |
| Orchestrator owns final decisions | Keep | Non-negotiable invariant in `AGENTS.md`. |
| Subagents are read-only/advisory | Keep | Triggered only when independent evidence or challenge is needed. |
| `spec.md` canonical decisions | Keep | For task-local bundles, `spec.md` remains the decision record. |
| Direct path / lean local / full orchestrated | Keep and promote | Move trigger matrix near the top of `AGENTS.md` and `docs/spec-first-workflow.md`; keep `lightweight local` as a compatibility alias. |
| `workflow-plan.md` | Keep but narrow | Single live state/control file for full orchestrated or real multi-session work; optional for lean local and usually absent for direct path. |
| `workflow-plans/<phase>.md` | Make conditional | Create only for multi-lane, multi-session, or formal challenge phases. Do not create for lean local tracks. |
| Research phase | Keep as concern | Preserve `research/*.md` only when evidence must survive resume or audit. Otherwise cite sources in `spec.md`. |
| Spec clarification challenge | Make trigger-based | Required for full orchestrated, high-risk, hard-to-reverse, public contract, data/security/money/reliability work. Inline `Risk Challenge` replaces it for lean local. |
| Workflow-plan adequacy challenge | Make trigger-based | Required when workflow routing itself is complex or agent-backed; local self-check for lean local. |
| Split `design/` bundle | Merge by default | For lean local, answer affected surfaces, sequence, and ownership in `spec.md` or one `design/overview.md`. Split files only when each dimension has real content. |
| `tasks.md` | Keep and promote | Main lean execution artifact. Required for lean implementation; tiny direct path can use inline plan. |
| `test-plan.md` | Conditional | Only when proof obligations are too large for `tasks.md`. |
| `rollout.md` | Conditional | Only for migrations, compatibility windows, deployment order, or rollback/failback. |
| Review/validation phase files | Conditional | Only for named multi-session review or validation checkpoints. |
| Fresh validation evidence | Keep | Non-negotiable completion rule. |

## Trigger Matrix

| Shape | Use When | Required Artifacts After Simplification | Gates | Stop Rule |
| --- | --- | --- | --- | --- |
| Direct path | Tiny, reversible, one surface, no public/API/data/security/money/reliability risk, obvious validation. | Usually none; optional brief note in chat or `spec.md` only if resume value exists. | Local first-read check and fresh validation evidence. | Finish in one pass; do not create workflow files for ceremony. |
| Lean local | Bounded non-trivial change, one primary domain, stable ownership, limited external research, implementation can be safely reasoned locally. | `spec.md` plus `tasks.md`; optional `research.md`, one `design/overview.md`, or `workflow-plan.md` only when triggered by evidence, density, or multi-session state. | Inline `Risk Challenge`; optional preserved research; no mandatory subagent. | May collapse research/spec/design/planning locally if `Risk Challenge` is `PASS` or `CONCERNS`; `FULL_REQUIRED` escalates before coding. |
| Full orchestrated | Cross-domain, ambiguous, high-impact, hard-to-reverse, user-requested agent-backed, public contract, schema/data migration, security, money, reliability, concurrency, rollout-sensitive, or long-running work. | `workflow-plan.md`, triggered `workflow-plans/<phase>.md`, `research/*.md` when useful, approved `spec.md`, design bundle, `tasks.md`, optional `test-plan.md`/`rollout.md`. | Formal challenge/review lanes as triggered; implementation readiness `PASS`, `CONCERNS`, or eligible `WAIVED`. | Phase boundaries remain strict; no implementation without executable handoff. |

Escalation triggers from direct or lean local to full orchestrated:
- public API or generated contract changes
- persisted data, migrations, backfills, cache semantics, or deletion behavior
- auth, tenant isolation, secrets, browser session, or abuse-risk changes
- money, billing, quotas, or user-visible entitlement logic
- concurrency, background workers, retries, lifecycle, or shutdown semantics
- deployment, migration, compatibility, rollback, or cross-version behavior
- multiple independent owners or ambiguous source-of-truth
- unclear validation path
- user explicitly requests subagents, broad audit, or strict spec-first phase boundaries

## Artifact Model Before And After

### Current Default Mental Model

```text
workflow-plan.md
workflow-plans/<phase>.md
research/*.md
spec.md
design/overview.md
design/component-map.md
design/sequence.md
design/ownership-map.md
tasks.md
optional test-plan.md
optional rollout.md
optional review/validation phase files
```

### Proposed Default Mental Model

```text
Direct path:
  chat/local plan + fresh proof

Lean local:
  spec.md
  tasks.md               # default for lean implementation
  research/*.md          # only when evidence must persist
  workflow-plan.md       # only when multi-session state is needed

Full orchestrated:
  workflow-plan.md
  workflow-plans/<phase>.md only for triggered complex phases
  research/*.md when useful
  spec.md
  design/overview.md or split design/ only when triggered by content
  tasks.md
  optional test-plan.md / rollout.md / review-validation phase files
```

## Lean `spec.md` Shape

Lean specs should stay compact but explicit. Use this shape as the default starting point for bounded non-trivial Go service changes:

```markdown
# <Feature / Change>

Mode: lean local
Status: planned | implementing | verified

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

## Compact Design
Affected surfaces:
- `api/openapi/service.yaml`
- `internal/...`

Ownership / source of truth:
- ...

Sequence / failure behavior:
- ...

## Risk Challenge
1. What irreversible or externally visible decision could be wrong?
   Answer: ...
2. What hidden invariant or owner could this break?
   Answer: ...
3. What fresh proof will make the completion claim trustworthy?
   Answer: ...
Gate: PASS | CONCERNS | FULL_REQUIRED

## Task Handoff
Use `tasks.md`.

## Outcome
Pending until fresh validation evidence exists.
```

This is a lighter form of the existing canonical `spec.md`, not a new artifact type.

## Lean `tasks.md` Shape

Lean `tasks.md` is the main execution surface:

```markdown
# Tasks

Readiness: PASS | CONCERNS
Consumes: `spec.md`
Proof: `go test ./...` plus targeted checks below.

- [ ] T001 Add failing test for <behavior>
  Files: `internal/...`
  Proof: test fails for expected reason before implementation.

- [ ] T002 Implement minimal behavior
  Files: `internal/...`
  Proof: targeted test passes.

- [ ] T003 Update contract/generated surfaces if needed
  Files: `api/openapi/service.yaml`, generated files
  Proof: generation/check command succeeds.

- [ ] T004 Run validation and record outcome
  Proof: `go test ./...`, `rtk make check`, or the smallest relevant command.
```

For behavior changes and bug fixes, test-first or proof-first execution is the default. If a failing test is not useful for a docs/config/mechanical change, the waiver belongs in `tasks.md` as a proof obligation, not in chat.

## Quality-Preservation Mechanisms

1. Replace "file count" with "answered questions."
   - Every non-direct change must answer: What changes? What stays out of scope? What owns the behavior? What is the execution path? What proves it?

2. Keep `tasks.md` strong.
   - The final implementation handoff should still use bounded tasks with dependencies and proof expectations when work is non-trivial.
   - For lean local, `tasks.md` is more important than phase-control files.

3. Make design depth content-triggered.
   - If affected surfaces, sequence, and ownership fit in a concise section, keep them together.
   - If any section becomes dense or contested, split into `design/`.

4. Make challenge gates risk-triggered.
   - Full orchestrated work still uses challenge lanes.
   - Lean local work records an inline `Risk Challenge` result and accepted risks.

5. Promote validation evidence.
   - Completion still requires fresh evidence matched to changed surfaces.
   - No simplification should weaken `docs/build-test-and-development-commands.md` as the source for local proof commands.

6. Reduce stale-state surfaces.
   - Fewer phase files means fewer places for current phase, blockers, and next action to drift.

7. Preserve explicit reopen rules.
   - If implementation discovers missing ownership, sequence, contract, or validation decisions, reopen the right earlier concern instead of inventing context during coding.

8. Keep always-loaded context short.
   - A project-context or Go context card may be useful later, but it should be a compact routing aid, not another large mandatory document.

## Risks And Mitigations

| Risk | How Simplified Workflow Prevents It |
| --- | --- |
| Agents abuse direct path for risky work | Put escalation triggers near the top of `AGENTS.md`; require explicit rationale for skipping artifacts. |
| Hidden ownership/design decisions leak into coding | Require the lean `Compact Design` section or split design when the answers are not obvious. |
| Loss of context across sessions | Use `workflow-plan.md` only when multi-session state is real; otherwise keep `spec.md` concise and authoritative. |
| Lower independent challenge quality | Keep formal challenge gates for high-risk/full orchestrated work; record inline `Risk Challenge` only for bounded work. |
| Validation gets watered down | Keep fresh validation evidence non-negotiable and map proof to changed surfaces. |
| Backward incompatibility with existing bundles | Treat old bundles as valid; update docs/skills to accept both old full artifact shape and new simplified default. |
| Lean local becomes a vague shortcut | Require `spec.md`, `tasks.md`, and inline `Risk Challenge` for bounded non-trivial implementation. |
| Test-first rule becomes performative | Require proof-first by default for behavior changes, but allow explicit waiver for docs/config/mechanical changes where a failing test is not useful. |

## Exact Repo Surfaces To Change If Approved

1. `AGENTS.md`
   - Promote the direct / lean local / full orchestrated trigger matrix and artifact-depth rules near the top.
   - Reframe full orchestrated flow as risk-triggered, not the default for every non-trivial task.
   - Keep invariants: orchestrator authority, read-only subagents, canonical `spec.md`, task-ledger-gated implementation when needed, fresh validation evidence.
   - Define `lightweight local` as a compatibility alias for `lean local` during the transition.

2. `docs/spec-first-workflow.md`
   - Rewrite artifact model around direct path, lean local, and full orchestrated triggers.
   - Make `workflow-plans/<phase>.md` conditional.
   - Allow merged lean design answers in `spec.md` or one design artifact.
   - Add lean `spec.md` and `tasks.md` shapes, including `Behavior / Contract Delta` and inline `Risk Challenge`.
   - Move exception rules out of the late appendix and into primary routing.

3. `docs/subagent-contract.md`
   - Clarify that subagents are triggered by unresolved owning questions, not by default phase ceremony.
   - Preserve read-only advisory boundary.

4. `docs/subagent-brief-template.md`
   - Add a shorter challenge/review brief variant for triggered lanes.
   - Keep evidence and handoff classifications.

5. `.agents/skills/`
   - Update workflow-planning, research, specification, technical-design, planning, and validation skills to prefer the trigger matrix.
   - Update challenge skills to support inline `Risk Challenge` and local checklist waiver text for lean local work.
   - Update design/planning skills to accept merged lean design context and lean `tasks.md` as the primary execution handoff.

6. `docs/build-test-and-development-commands.md`
   - Likely no semantic change, but implementation should link validation rules back to this doc.

7. Example bundles or templates
   - Do not rewrite historical bundles.
   - Add or adjust future examples only if needed to show the simplified path.

8. Future command/skill UX
   - Consider a later `workflow-next` / `go-next` helper after the docs and skills are simplified.
   - Do not add this as part of the first rewrite unless it can be implemented without adding a new state layer.

## Resolved Approval Questions

The approval questions are resolved by the approval record above. Reopen specification only if implementation discovers that one of these choices cannot be applied without weakening the preserved invariants or expanding the target artifact model.

## Validation

For this research/specification phase:
- External claims have source links in `research/external-agent-workflow-practices.md`.
- Local claims have repo file references in `research/current-workflow-pain-map.md`.
- `workflow-plan.md` records the current phase, proposal status, next action, and artifact status.
- `spec.md` is the canonical decision record and does not duplicate raw research.
- No implementation files outside `specs/workflow-simplification/` were changed.

Implementation validation:
- Claim: workflow simplification docs, subagent docs, workflow/session skills, helper/challenge/design/planning skills, and runtime skill mirrors are complete for the approved docs/skills-only scope.
- Scope: `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/subagent-contract.md`, `docs/subagent-brief-template.md`, canonical `.agents/skills/` updates, and synced runtime skill mirrors. Runtime Go code, generated API, migrations, and deployment surfaces were not touched, so broader `rtk make check` is not part of this docs/skills-only closeout claim.
- Consistency sweeps on 2026-05-17:
  - `rtk rg -n "lightweight local|lightweight-local" AGENTS.md docs .agents/skills/*/SKILL.md .agents/skills/*/references/*.md` returned only compatibility-alias references.
  - `rtk rg -n "direct/local" AGENTS.md docs .agents/skills/*/SKILL.md .agents/skills/*/references/*.md` returned no matches.
  - `rtk rg -n "full orchestrated.*default|default.*full orchestrated|full artifact bundle|full bundle" AGENTS.md docs .agents/skills/*/SKILL.md .agents/skills/*/references/*.md` returned only anti-pattern or "do not force full bundle" wording.
  - `rtk rg -n 'required core design|required design artifacts|spec\.md \+ design' AGENTS.md docs .agents/skills/*/SKILL.md .agents/skills/*/references/*.md` returned no matches.
- Verification commands on 2026-05-17:
  - `rtk make agents-check` passed (`agents check complete`).
  - `rtk make skills-check` passed (`skills check complete (non-destructive)`).
  - `rtk git diff --check` passed with no output.
- Conclusion: verified for the docs/skills-only implementation scope.

## Outcome

Implemented the approved trigger-based workflow simplification:
- `AGENTS.md` now makes direct path / lean local / full orchestrated routing and escalation triggers prominent while preserving orchestrator authority, read-only subagents, canonical `spec.md`, task-ledger-gated implementation when needed, and fresh validation evidence.
- `docs/spec-first-workflow.md` now treats artifact depth as trigger-based, includes lean `spec.md` and `tasks.md` shapes, and keeps historical full bundles valid.
- Subagent docs now route lanes by unresolved owned questions and include a short challenge/review brief variant without relaxing read-only or evidence rules.
- Workflow/session, helper, challenge, spec/design, and planning skills now support the trigger matrix, inline lean `Risk Challenge`, merged lean design context, and formal gates for triggered high-risk/full-orchestrated work.
- Skill mirrors were refreshed with `rtk make skills-sync` and verified by `rtk make skills-check`.
