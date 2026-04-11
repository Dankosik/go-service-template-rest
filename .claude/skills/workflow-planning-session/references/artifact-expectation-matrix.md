# Artifact Expectation Matrix

## Behavior Change Thesis
When loaded for symptom "I need to mark later artifact expectations," this file makes the model record trigger-aware `expected`, `missing`, `draft`, `approved`, `not expected`, `conditional`, or `waived` statuses instead of inventing completeness, creating later artifacts early, or marking everything "not applicable."

## When To Load
Load this when artifact status is the active uncertainty. If the problem is how to split content between `workflow-plan.md` and `workflow-plans/workflow-planning.md`, load the control-file authoring reference instead.

## Decision Rubric
- `approved` means the artifact already exists and has passed its gate; do not infer approval from intent.
- `draft` means the artifact exists but handoff requirements or challenge findings remain open.
- `missing, expected later` means the artifact is required by the chosen shape or likely phase sequence but does not belong to this workflow-planning session.
- `conditional, trigger unknown` means research or later planning must decide; do not create it "just in case."
- `not expected` means the repository contract does not call for that artifact for this task.
- `waived` requires an eligible tiny/direct-path or explicit local waiver rationale; do not use it as a synonym for missing.
- Post-code phase workflow files are created during planning only when the approved phase structure uses them, not during workflow planning and not mid-implementation.

## Imitate

Direct-path artifact record:

```markdown
- `workflow-plan.md`: not expected; inline direct-path skip rationale is enough.
- `workflow-plans/workflow-planning.md`: not expected.
- `spec.md`: waived for tiny direct-path work; rationale recorded inline.
- `design/`: waived; no ownership, data, contract, runtime-sequence, or rollout ambiguity.
- `plan.md`: waived; inline one-step plan is enough.
- `tasks.md`: waived; no ledger needed.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.
```

What to copy: waived items include why the waiver is eligible.

Full-orchestrated artifact record:

```markdown
- `workflow-plan.md`: draft, current workflow-planning session owns repair.
- `workflow-plans/workflow-planning.md`: draft, active phase file.
- `research/*.md`: missing, expected later for reusable fan-out evidence.
- `spec.md`: missing, expected after research and synthesis.
- `design/`: missing, expected after approved `spec.md`.
- `plan.md`: missing, expected after approved `spec.md + design/`.
- `tasks.md`: missing, expected by default with `plan.md`.
- `test-plan.md`: conditional, trigger unknown.
- `rollout.md`: conditional, trigger unknown.
- Post-code phase workflow files: count unknown; planning must create any used files before implementation.
```

What to copy: later artifacts are acknowledged without being created or approved.

## Reject

```markdown
Artifact status: everything else can be decided later.
```

Failure: loses the handoff contract; the next session cannot tell what is expected, conditional, or waived.

```markdown
Artifacts: all approved or not applicable because workflow planning has enough detail.
```

Failure: invents gate completion and bypasses research, specification, design, and implementation planning.

```markdown
Create `test-plan.md` and `rollout.md` now so the matrix is complete.
```

Failure: starts later artifact-producing work during workflow planning.

## Agent Traps
- Marking `tasks.md` as "not expected" just because the current phase cannot write it.
- Treating `conditional` as permission to create the artifact immediately.
- Recording post-code phase files as expected for every task "to be safe."
- Forgetting that direct-path waivers need a reason, not just a label.
