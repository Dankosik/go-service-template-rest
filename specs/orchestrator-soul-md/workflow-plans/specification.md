# Specification Phase Plan

Phase: specification
Phase status: complete
Pass type: fresh specification
Session boundary reached: yes
Ready for next session: yes
Next action: planning

## Readiness Check

Result: PASS

The research checkpoint produced enough evidence to write stable decisions without reopening research. The behavior delta is bounded to one root personality file, AGENTS integration, and validation hooks. The core approval risk is the file-boundary and precedence rule between `SOUL.md` and `AGENTS.md`; no runtime, API, data, deployment, or cross-service design is in scope.

## Input Sources Used

- `workflow-plan.md`: prior phase state and next-session routing.
- `workflow-plans/research.md`: research-mode rationale, fan-in, and limits.
- `research/soul-md-agent-personality-practices.md`: source-backed SOUL.md boundary and content-shape evidence.
- `AGENTS.md`: authoritative workflow contract and repository invariants.
- `docs/spec-first-workflow.md`: artifact, phase-boundary, challenge-gate, and handoff mechanics.
- `.agents/skills/specification-session/SKILL.md`: specification-session boundaries and required state updates.
- `.agents/skills/spec-document-designer/SKILL.md`: repository-native `spec.md` shape and decision placement.
- `.agents/skills/spec-clarification-challenge/SKILL.md`: formal clarification question and reconciliation rules.
- `docs/build-test-and-development-commands.md`, `Makefile`, and `scripts/ci/required-guardrails-check.sh`: validation surface for docs/instruction changes.

## Clarification Challenge

Clarification challenge: complete
Lanes: local orchestrator constrained challenge; no spawned subagent
Lenses: instruction-boundary and validation proof

Scoped-down rationale:

- `spawn_agent` is available, but its tool contract permits spawning only when the user explicitly asks for subagents, delegation, or parallel agent work. The user requested a specification-session, not subagents.
- The approval risk is concentrated in whether `SOUL.md` can stay lower-precedence personality guidance while `AGENTS.md` remains the operating contract.
- Default runtime/API/data/security/reliability/rollout lenses cannot change approval for this accepted scope because the change has no runtime service behavior, generated contract, persisted state, deployment, or cross-service effect.

Resolution:

| Question | Resolution | Owner |
| --- | --- | --- |
| Could `SOUL.md` become a second workflow authority? | Resolved in `spec.md` through explicit non-goals, two-file precedence wording, and guardrail validation. | specification |
| Could host-specific loading differences make root `SOUL.md` insufficient? | Accepted root-only scope; host-specific mirroring is an out-of-scope reopen trigger if later required. | specification |
| Could the personality wording weaken production-ready rigor under "avoid overengineering"? | Resolved by requiring "simple enough, not simplistic" and preserving AGENTS.md production-ready target-state default. | specification |
| Could validation miss future drift? | Resolved by requiring guardrail checks for `SOUL.md` existence and AGENTS/SOUL boundary plus normal docs/script drift validation. | planning / implementation |

Gate result: PASS
Readiness consequence: planning may start; no separate technical design is required.
Reopen target: none

## Subagent Gate Audit

- Trigger: full-orchestrated repository-wide instruction-surface work normally requires formal clarification before `spec.md` approval.
- Gate type: spec-clarification
- Required lane policy: scoped-down local-only challenge
- Lane table: local orchestrator, read-only before edits, lens `instruction-boundary and validation proof`, skill basis `spec-clarification-challenge`, inspect-first `workflow-plan.md`, `research/soul-md-agent-personality-practices.md`, `AGENTS.md`, `docs/spec-first-workflow.md`, status complete.
- Lane result summary: no approval blocker after the SOUL/AGENTS boundary, root-only integration, production-ready personality wording, and guardrail validation were made explicit.
- Fan-in: final decisions recorded in `spec.md`; no unresolved conflicts; proof obligations recorded in `spec.md` Validation.
- Gate result: PASS.
- Readiness consequence: planning may start.
- Reopen target: none.

## Artifact Status

| Artifact | Status | Rationale |
| --- | --- | --- |
| `spec.md` | approved | Canonical decision record created for SOUL.md purpose, non-goals, content shape, AGENTS integration, validation, and gate routing. |
| `design/` | not expected | Separate design depth is not triggered because file ownership, integration, and validation decisions are compact and uncontested in `spec.md`. |
| `tasks.md` | missing / next | Planning must create and review the executable ledger before implementation. |
| `test-plan.md` | not expected | Validation obligations fit inside `tasks.md`; no separate layered test strategy is needed. |
| `rollout.md` | not expected | No runtime rollout, migration, compatibility window, or deployment sequencing is in scope. |

## Blockers

None.

## What Can Run In Parallel Next

During planning, context reads can run in parallel. Ledger writing and task-review/readiness should stay in the main planning flow because the tasks are tightly coupled to the approved instruction-boundary decisions.

## Stop Rule

Stop after specification. Do not create `SOUL.md`, edit `AGENTS.md`, create `tasks.md`, or implement in this phase.
