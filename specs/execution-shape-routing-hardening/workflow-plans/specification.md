# Specification Phase

## Session Objective

Convert the completed `B01-F01`-`B01-F11` research packet into one compact, review-ready `spec.md`, reconcile the required formal clarification gate, and stop before specification review or implementation work.

## Readiness Check

- Pass type: fresh specification pass.
- Input readiness: `ready`; accepted scope, non-goals, evidence, candidate models, Pattern Fit Diligence, proof obligations, and reopen conditions are present.
- Current execution shape: `full orchestrated`.
- Dependency/OSS diligence: `not expected`; no new runtime dependency or workflow engine is selected.
- Pattern Fit Diligence: `complete`; preserved in `research/pattern-fit.md` and reconciled in `research/synthesis.md`.
- Specification status: `review_ready`; formal clarification is complete and reconciled.
- Targeted research reopened: `no`.
- Local blockers: `none`.

## Input Sources

- `workflow-plan.md`: accepted objective, all eleven findings, artifact expectations, and current route.
- `workflow-plans/research.md`: completed research gate and proof-only limits.
- `research/synthesis.md`: complete finding-to-decision/proof handoff and cross-reopen audit.
- `research/shape-and-execution-authority.md`: direct actor, classifier ownership, and authorization evidence.
- `research/status-reclassification-and-resume.md`: typed state, transition, resume, phase-file, and status evidence.
- `research/adequacy-enforcement-and-mirrors.md`: adequacy, semantic enforcement, CI, and mirror evidence.
- `research/pattern-fit.md`: selected/rejected workflow and state-model patterns.

## Clarification Challenge Plan

- Clarification challenge: `complete`.
- Gate type: formal multi-lane `spec-clarification-challenge`.
- Required lane policy: default broad lens set; five distinct read-only lanes, run in two parallel waves because the active tool permits three child lanes at once, followed by targeted rechecks of repaired seams.
- Subagent gate: `complete`.
- Scoped-down rationale: not applicable; every default lens is retained.

| Lane | Lens | Approval-critical question | Skill | Mode | Status |
| --- | --- | --- | --- | --- | --- |
| `C1` | scope and spec coherence | Does the candidate spec close every `B01-F01`-`B01-F11` decision without scope loss, hidden alternatives, design/task leakage, or a closure that reopens another finding? | `spec-clarification-challenge` | read-only | `complete; recheck clear` |
| `C2` | domain invariants and edge cases | Are shape precedence, direct execution, authorization intent, legal reclassification, stale-state, research-skip, phase-file, and artifactless-status edge semantics complete and mutually consistent? | `spec-clarification-challenge` | read-only | `complete; recheck clear` |
| `C3` | architecture ownership and dependency boundaries | Does each policy, procedure, consumer, enforcement, and mirror responsibility have one canonical owner without duplicating authority or introducing an unjustified design/dependency checkpoint? | `spec-clarification-challenge` | read-only | `complete; recheck clear` |
| `C4` | API, data, compatibility, and source-of-truth consequences | Does the no-runtime-API/data scope remain honest while historical-bundle, legacy-alias, artifactless, and canonical/mirror compatibility behavior is fully decided? | `spec-clarification-challenge` | read-only | `complete; recheck clear` |
| `C5` | security, reliability, delivery, and validation proof | Do fail-closed routing, advisory authority, deterministic enforcement, CI invocation, eval claims, mirror states, and fresh proof obligations make the repaired contract regression-protected? | `spec-clarification-challenge` | read-only | `complete; recheck clear` |

Sibling lenses are `C1`-`C5`. Each lane must return only review-readiness-changing questions, exact evidence anchors, classification, and the smallest next action; lanes do not edit files or approve the spec.

## Clarification Disposition

| Lens | Strongest initial concern | Final disposition | Owner / evidence pointer |
| --- | --- | --- | --- |
| `C1` | Freshness composition, artifactless revisioning, and collapse semantics were under-specified. | Repaired from existing evidence; targeted recheck `CLEAR`. | `spec.md` D5-D7 and Formal Clarification Gate |
| `C2` | Direct eligibility, stale dependent state, post-ledger reopen precedence, and envelope lifetime needed exact edge rules. | Repaired from existing evidence; targeted recheck `CLEAR`. | `spec.md` D3, D5, D6, D9 |
| `C3` | Shape, adequacy, status, and checker ownership risked duplicate authority. | Repaired from existing evidence; targeted recheck `CLEAR`. | `spec.md` D2, D8-D10 and Authority And Consumer Matrix |
| `C4` | Historical mapping, envelope provenance, and mirror requiredness needed closed source-of-truth semantics. | Repaired from existing evidence; targeted recheck `CLEAR`. | `spec.md` D5, D9, D11 |
| `C5` | Checker authority, mirror aggregation, and Make/CI/CD carrier coverage needed non-circular proof boundaries. | Repaired from existing evidence; targeted recheck `CLEAR`. | `spec.md` D10-D11 and Validation |

Resolution: all approval-changing questions were answered from preserved research and current repository evidence. No lane conflict, accepted risk, targeted research, specialist reopen, or user decision remains. Implementation proof remains downstream and is not treated as clarification evidence.

## Subagent Gate Audit

- Trigger: full-orchestrated, broad multi-owner workflow-contract repair with a formal clarification requirement.
- Required lane policy: five default read-only clarification lenses using exactly `spec-clarification-challenge`; all were retained.
- Execution: `C1`-`C5` completed; repaired seams received targeted rechecks and every recheck returned `CLEAR`.
- Orchestrator fan-in: initial questions were deduplicated, answered from preserved research/current repository evidence, and recorded only as final decisions, constraints, proof obligations, or reopen conditions in `spec.md`.
- Missing-input or severity conflicts: none survive.
- Accepted risks: none.
- Readiness consequence: specification may proceed to a distinct read-only specification review; design, test design, planning, and implementation may not start from this gate alone.
- Reopen target: specification if review finds an approval blocker; research only if review exposes missing evidence that cannot be resolved from the current packet.

## Order And Parallelism

1. Write one candidate `spec.md` from the complete research synthesis.
2. Run `C1`-`C3` in parallel, then `C4`-`C5` as soon as capacity is available.
3. Reconcile every surviving question from existing evidence; reopen only when a decision cannot be made honestly.
4. If reconciliation materially changes a core decision, rerun only the affected lens once.
5. Mark `spec.md` review-ready and update master/phase routing only when no clarification blocker remains.

## Completion Marker

Complete: `spec.md` contains final orchestrator-owned decisions for every finding, selected/rejected models, legacy-surface dispositions, Pattern Fit outcome, proof obligations, and reopen conditions; all five clarification lenses and targeted rechecks are reconciled; master and phase-local state agree; and the next route is specification review.

## Next Action

Start exactly one distinct read-only specification-review phase for the review-ready `spec.md`. Create `workflow-plans/specification-review.md` in that phase because durable multi-lane review routing is expected for this task.

## Phase State

- Phase status: `complete`.
- Completion marker: review-ready `spec.md`, reconciled clarification gate, and synchronized master routing.
- Session boundary reached: `yes`.
- Ready for next session: `yes`.
- Next session starts with: `specification-review`.
- Blockers: `none`.
- Parallelizable work in this phase: `none`; the boundary has been reached.

## Stop Rule

Stop at the specification boundary. Do not start specification review, create `design/`, `test-plan.md`, or `tasks.md`, edit workflow-contract or implementation surfaces, or run implementation/validation work.
