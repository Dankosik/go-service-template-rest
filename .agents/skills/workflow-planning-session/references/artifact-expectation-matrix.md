# Artifact Expectation Matrix

## Behavior Change Thesis
When loaded for symptom "I need to mark later artifact expectations," this file makes the model record orthogonal `artifact_expectation`, `artifact_state`, `record_validity`, and `waiver_disposition` values instead of compound free-text status.

## When To Load
Load this when artifact status is the active uncertainty. If the problem is how to split content between `workflow-plan.md` and `workflow-plans/workflow-planning.md`, load the control-file authoring reference instead.

## Decision Rubric
- Use `artifact_expectation=expected|conditional|not_expected` only for expectation.
- Use `artifact_state=absent|draft|review_ready|approved|complete|blocked` only for lifecycle.
- Use `record_validity=current|stale|superseded` independently; stale approval is history, not authority.
- A waiver is `artifact_expectation=expected + artifact_state=absent + waiver_disposition=waived` with eligibility, rationale, evidence, and reopen trigger.
- `conditional` and `not_expected` require `artifact_state=absent` and no waiver.
- Review or validation phase workflow files are created during planning only when named multi-session routing uses them, not during workflow planning and not mid-implementation.

## Imitate

Direct-path artifact record:

```markdown
- `workflow-plan.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `workflow-plans/workflow-planning.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `spec.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `design/`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `tasks.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `test-plan.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `rollout.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
```

What to copy: direct-path omissions are `not_expected`, not waivers.

Full-orchestrated artifact record:

```markdown
- `workflow-plan.md`: artifact_expectation=expected, artifact_state=draft, record_validity=current, waiver_disposition=none.
- `workflow-plans/workflow-planning.md`: artifact_expectation=expected only when `ROUTING-PHASE-CONTROL` is satisfied; otherwise artifact_expectation=not_expected. In either case use the matching canonical absent/draft lifecycle and waiver_disposition=none.
- `research/*.md`: artifact_expectation=conditional until the research trigger resolves, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `spec.md`: artifact_expectation=expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `design/`: artifact_expectation=conditional, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `tasks.md`: artifact_expectation=expected, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `test-plan.md`: artifact_expectation=conditional, artifact_state=absent, record_validity=current, waiver_disposition=none.
- `rollout.md`: artifact_expectation=conditional, artifact_state=absent, record_validity=current, waiver_disposition=none.
- Review/validation phase workflow files: count unknown; planning creates only named files needed for multi-session routing before implementation.
```

What to copy: later artifacts are acknowledged without being created or approved.

## Reject

```markdown
Artifact inventory: everything else can be decided later.
```

Failure: loses the handoff contract; the next session cannot tell what is expected, conditional, or waived.

```markdown
Artifacts: all approved or not applicable because workflow planning has enough detail.
```

Failure: invents gate completion and bypasses research, specification, design, and task breakdown.

```markdown
Create `test-plan.md` and `rollout.md` now so the matrix is complete.
```

Failure: starts later artifact-producing work during workflow planning.

## Agent Traps
- Marking `tasks.md` as "not expected" just because the current phase cannot write it.
- Treating `conditional` as permission to create the artifact immediately.
- Recording review/validation phase files as expected for every task "to be safe."
- Mislabeling direct-path `not_expected` artifacts as waivers.
