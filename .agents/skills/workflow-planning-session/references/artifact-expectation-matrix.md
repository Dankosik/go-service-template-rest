# Artifact Expectation Matrix

## When To Load
Load this when a workflow-planning session needs examples for marking later artifacts as `approved`, `draft`, `missing`, `expected`, `not expected`, or explicitly waived.

Use it to calibrate artifact expectations only. Do not create `spec.md`, `design/`, `plan.md`, `tasks.md`, `test-plan.md`, `rollout.md`, or post-code phase files from this reference.

## Direct / Lightweight / Full Examples

### Direct path example

```markdown
Execution shape: direct path
Artifacts:
- `workflow-plan.md`: not expected; inline skip rationale is enough.
- `workflow-plans/workflow-planning.md`: not expected.
- `research/*.md`: not expected.
- `spec.md`: waived for tiny direct-path work; rationale recorded inline.
- `design/`: waived; no ownership, data, contract, runtime-sequence, or rollout ambiguity.
- `plan.md`: waived; inline 1-step plan is enough.
- `tasks.md`: waived; no ledger needed.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.
Challenge: adequacy challenge skipped with direct-path rationale.
```

### Lightweight local example

```markdown
Execution shape: lightweight local
Artifacts:
- `workflow-plan.md`: expected now because this is a dedicated workflow-planning session.
- `workflow-plans/workflow-planning.md`: expected now.
- `research/*.md`: not expected unless local research produces reusable evidence.
- `spec.md`: expected later; status `missing`.
- `design/`: conditional; may be waived later only with explicit design-skip rationale.
- `plan.md`: conditional; expected if the work stays non-trivial after specification.
- `tasks.md`: conditional; expected by default if `plan.md` is expected.
- `test-plan.md`: not expected unless validation obligations become too layered for `plan.md`.
- `rollout.md`: not expected unless migration, compatibility, deploy, or failback sequencing is triggered.
Challenge: run unless an explicit lightweight-local waiver records why full artifact-chain challenge is unnecessary.
```

### Full orchestrated example

```markdown
Execution shape: full orchestrated
Artifacts:
- `workflow-plan.md`: expected now; status `draft` until adequacy findings are reconciled.
- `workflow-plans/workflow-planning.md`: expected now; status `draft` until handoff fields agree with master.
- `research/*.md`: expected later for reusable multi-lane evidence.
- `spec.md`: expected later; status `missing`.
- `design/`: expected later; core artifacts required after approved `spec.md`.
- `plan.md`: expected later after approved `spec.md + design/`.
- `tasks.md`: expected later by default with `plan.md`.
- `test-plan.md`: conditional; mark `trigger unknown` rather than creating it now.
- `rollout.md`: conditional; mark `trigger unknown` rather than creating it now.
- `workflow-plans/implementation-phase-N.md`: count unknown; planning must create any used files before implementation starts.
Challenge: adequacy challenge required before workflow-planning handoff.
```

## Good / Bad Lane Rows

Good lane rows for artifact expectation discovery:

| Lane | Role | Owned Question | Skill | Artifact Consequence |
| --- | --- | --- | --- | --- |
| L1 | `qa-agent` | Are validation obligations likely to need a later `test-plan.md`, or can they fit in `plan.md`? | `go-qa-tester-spec` | Mark `test-plan.md` conditional until research confirms. |
| L2 | `delivery-agent` | Is rollout choreography likely enough to plan a later `rollout.md` trigger? | `go-devops-spec` | Mark `rollout.md` conditional; do not create it during workflow planning. |

Bad lane rows:

| Lane | Role | Owned Question | Skill | Why It Fails |
| --- | --- | --- | --- | --- |
| Lx | `qa-agent` | Own later test-plan output so the matrix looks complete. | `go-qa-tester-spec` | Starts a later artifact-producing phase during workflow planning. |
| Ly | `delivery-agent` | Mark every implementation/review/validation phase file expected just in case. | `go-devops-spec` | Conditional phase files belong in planning only when the approved phase structure uses them. |

## Handoff Examples

Good matrix handoff:

```markdown
Artifact status summary:
- `workflow-plan.md`: draft, current session owns repair.
- `workflow-plans/workflow-planning.md`: draft, active phase file.
- `spec.md`: missing, expected in specification.
- `design/`: missing, expected if work remains non-trivial after approved `spec.md`.
- `plan.md`: missing, expected after approved `spec.md + design/`.
- `tasks.md`: missing, expected by default with `plan.md`.
- `test-plan.md`: conditional, trigger unknown.
- `rollout.md`: conditional, trigger unknown.
Next session starts with: research.
Stop rule: do not create any missing later artifact in this session.
```

Bad matrix handoff:

```markdown
All artifacts: approved or not applicable.
Reason: workflow planning has enough detail for later phases.
```

Why it is bad: it invents artifact status and bypasses the repo's decision/design/planning gates.

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11, but the service returned a 402 credit-limit error. The links below were gathered through fallback web search and are only calibration sources; the repo-local `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

- arc42 building block view: https://docs.arc42.org/section-5/
- arc42 runtime view: https://docs.arc42.org/section-6/
- arc42 architecture decisions: https://docs.arc42.org/section-9/
- arc42 quality requirements: https://docs.arc42.org/section-10/
- arc42 risks and technical debt: https://docs.arc42.org/section-11/
- Michael Nygard, "Documenting Architecture Decisions": https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions
