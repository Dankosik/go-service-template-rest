# 65 Coder Detailed Plan Index: Tooling Boilerplate Adoption

## Execution Context
This feature package contains four approved specs and four execution-grade coder plans.
Each plan is independent but has dependency ordering constraints.

## Execution Mode
- Recommended global mode: `batch` by spec package.
- Recommended order:
1. Spec 01 (`go tool` + `tool` directives)
2. Spec 02 (`mockgen`)
3. Spec 03 (`stringer`)
4. Spec 04 (`sqlc`)

Rationale:
- Spec 01 establishes shared tooling baseline used by Specs 02-04.

## Task Graph
- GLOBAL-T01 -> GLOBAL-T02 -> GLOBAL-T03 -> GLOBAL-T04
- GLOBAL-T01 maps to detailed plan 01.
- GLOBAL-T02 maps to detailed plan 02.
- GLOBAL-T03 maps to detailed plan 03.
- GLOBAL-T04 maps to detailed plan 04.

## Task Cards

### Task ID
GLOBAL-T01

Objective:
- Execute detailed plan for Spec 01.

Spec Traceability:
- `01-go-tool-and-tool-directives-spec.md`

Change Surface:
- Toolchain/build/docs baseline.

Task Sequence:
1. Execute all tasks in `65-coder-detailed-plan-01-go-tool-and-tool-directives.md`.

Verification Commands:
- See plan 01 task cards.

Expected Evidence:
- Tooling baseline complete and validated.

Review Checklist:
- Plan 01 checkpoints passed.
- Required evidence captured.

Ambiguity Triggers:
- Any unresolved blocker in plan 01.

Change Reconciliation:
- Recorded in plan 01.

Progress Status:
- `todo`

### Task ID
GLOBAL-T02

Objective:
- Execute detailed plan for Spec 02.

Spec Traceability:
- `02-mockgen-spec.md`

Change Surface:
- Test generation + selected seam migrations.

Task Sequence:
1. Execute all tasks in `65-coder-detailed-plan-02-mockgen.md`.

Verification Commands:
- See plan 02 task cards.

Expected Evidence:
- Mock generation baseline + first seam adoption.

Review Checklist:
- Plan 02 checkpoints passed.

Ambiguity Triggers:
- Any unresolved blocker in plan 02.

Change Reconciliation:
- Recorded in plan 02.

Progress Status:
- `todo`

### Task ID
GLOBAL-T03

Objective:
- Execute detailed plan for Spec 03.

Spec Traceability:
- `03-stringer-spec.md`

Change Surface:
- Enum generation and safe replacement of handwritten `String()` for eligible internal enums.

Task Sequence:
1. Execute all tasks in `65-coder-detailed-plan-03-stringer.md`.

Verification Commands:
- See plan 03 task cards.

Expected Evidence:
- Stringer workflow and guardrails operational.

Review Checklist:
- Plan 03 checkpoints passed.

Ambiguity Triggers:
- Any unresolved blocker in plan 03.

Change Reconciliation:
- Recorded in plan 03.

Progress Status:
- `todo`

### Task ID
GLOBAL-T04

Objective:
- Execute detailed plan for Spec 04.

Spec Traceability:
- `04-sqlc-spec.md`

Change Surface:
- SQL generation workflow + first vertical slice + checks.

Task Sequence:
1. Execute all tasks in `65-coder-detailed-plan-04-sqlc.md`.

Verification Commands:
- See plan 04 task cards.

Expected Evidence:
- Sqlc baseline and first production-grade path validated.

Review Checklist:
- Plan 04 checkpoints passed.

Ambiguity Triggers:
- Any unresolved blocker in plan 04.

Change Reconciliation:
- Recorded in plan 04.

Progress Status:
- `todo`

## Checkpoint Plan
- CP-GLOBAL-1: after GLOBAL-T01
  - Go/no-go for generation-heavy specs (02/03/04) based on tooling baseline readiness.
- CP-GLOBAL-2: after GLOBAL-T03
  - Go/no-go for sqlc rollout based on test and generation guard maturity.
- CP-GLOBAL-3: after GLOBAL-T04
  - Final closure review across all four specs.

## Clarification Contract
Required fields:
- `request_id`
- `blocked_task_id`
- `ambiguity_type` (`contract`, `invariant`, `security`, `reliability`, `test`, `other`)
- `conflicting_sources`
- `decision_impact`
- `proposed_options`
- `owner`
- `resume_condition`

Resolution policy:
- Blocked global task cannot continue until corresponding detailed-plan blocker is resolved.

## Coverage Matrix
- Spec 01 obligations -> `65-coder-detailed-plan-01-go-tool-and-tool-directives.md`
- Spec 02 obligations -> `65-coder-detailed-plan-02-mockgen.md`
- Spec 03 obligations -> `65-coder-detailed-plan-03-stringer.md`
- Spec 04 obligations -> `65-coder-detailed-plan-04-sqlc.md`

## Execution Notes
- Detailed execution stays per-spec.
- This index file is orchestration-only and does not replace per-spec task cards.
