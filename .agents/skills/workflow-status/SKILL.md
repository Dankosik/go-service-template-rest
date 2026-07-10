---
name: workflow-status
description: "Read-only workflow status and next-action helper. Report from one identifiable durable task path or an explicit orchestrator-attested current-session direct_state_envelope; otherwise fail closed with unsupported: no durable task state."
---

# Workflow Status

## Eligibility And Outcome

Use for one identified task when the user asks where the workflow is, what blocks it, what may be written next, or whether implementation may start. Accept only:

- one explicit task-local path;
- a current working directory already inside one task-local path;
- exactly one task-local artifact path in the prompt; or
- an orchestrator-attested current-session direct_state_envelope with current provenance and SHAPE-DIRECT evidence.

Do not infer the task from recency, broad scans, chat memory, or file absence. If no eligible source exists, return exactly:

`unsupported: no durable task state`

The outcome is a compact read-only projection of current canonical state and the first safe next action. It is not a phase, approval, waiver, reclassification, or repair.

## Canonical Owners

- [Workflow router](../../../docs/spec-first-workflow.md) owns phase routing and the artifact-first resume path.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns typed state, routing identity, validity, transitions, legacy mapping, direct envelopes, and artifact expectations.
- The active [phase file](../../../docs/spec-first-workflow/phases/) owns allowed writes, success, stop, and reopen semantics.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns resume order and final handoff semantics.

Read the smallest canonical owner needed for the question. Do not reproduce or locally extend its state matrices.

## Side Effects

None. Do not edit files, create artifacts, run mutating generation, change git state, approve gates, attest envelopes, execute transitions, or repair conflicting routing. The report may describe what another session is allowed to write.

## Unique Decision Rule

1. Prefer current approved `tasks.md` for implementation/review/validation status. Treat older workflow control as historical unless the ledger names a pre-created phase file.
2. Without an approved ledger, use current `workflow-plan.md` plus the active phase file only when routing identities agree.
3. Read phase artifacts named by those owners only as needed. Use a direct envelope only when no durable state exists; never combine it with durable state or reuse it in a later session.
4. Consume canonical execution shape, artifact expectation/lifecycle/validity, procedural gate state, review verdict, session boundary, handoff readiness, adequacy, waiver disposition, and legacy mapping. Never invent a value or translate an unmapped legacy token.
5. Stale, superseded, missing, conflicting, or unproven state can describe history but cannot authorize readiness.
6. Implementation may start only through current SHAPE-DIRECT authority in the same session or a current approved ledger whose required artifacts and task-review/readiness gate authorize implementation. Eligible CONCERNS or prototype-scoped WAIVED status must carry the canonical rationale, risks, proof obligations, and route. Otherwise answer no; use unknown only for a genuinely contradictory identified task.

## Report Shape

Keep the report compact:

```text
Workflow Status
- Source/provenance: <task path and canonical source | current-session direct envelope | unsupported>
- Execution shape: <canonical value, matched SHAPE-* rule, decisive evidence>
- Routing identity / validity: <scope, revision, current|stale|superseded|conflict>
- Adequacy: <required by ADEQUACY-* yes/no; result, evidence, validity>
- Phase/session/handoff: <canonical values>
- Artifacts: <expectation, lifecycle, validity, waiver disposition>
- Gates/verdicts: <procedural state, verdict, validity>
- Missing gate or blocker: <first meaningful blocker or none>
- Allowed writes for current phase: <from the owning phase; this skill writes nothing>
- Next action: <recorded route or first safe action>
- Stop rule: <from the owning phase>
- Implementation may start: <yes direct in this session | yes | yes in recorded next session | yes with recorded concerns | yes under recorded waiver | no | unknown, with one reason>
```

## Stop And Reopen

Stop after the report. When artifacts conflict or proof is missing, name the conflict/gap and its canonical reopen owner; do not repair it.

When no eligible source exists, add only the request for one task-local path or a current orchestrator-attested direct envelope, then stop.
