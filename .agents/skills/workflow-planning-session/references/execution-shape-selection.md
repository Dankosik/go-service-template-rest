# Execution Shape Selection

## Behavior Change Thesis
When loaded for symptom "I am unsure whether this needs direct path, lightweight local, or full orchestrated routing," this file makes the model choose the smallest defensible execution shape with an explicit escalation trigger instead of forcing ceremony for tiny work or under-routing cross-domain work.

## When To Load
Load this only when execution shape is the active uncertainty. If the shape is already chosen and the hard part is lane design, artifact status, file authoring, or the adequacy boundary, load that narrower reference instead.

## Decision Rubric
- Choose `direct path` only for tiny, reversible, single-surface work with high confidence after a first read and no need for preserved research, subagents, or multi-session handoff.
- Choose `lightweight local` for bounded, non-trivial, mostly single-domain work where local research is enough unless it reveals a cross-domain seam.
- Choose `full orchestrated` when the work crosses material API, data, security, reliability, observability, delivery, or domain-invariant boundaries; when the decision is hard to reverse; or when user-requested agent-backed work needs durable fan-out evidence.
- Escalate from `lightweight local` to `full orchestrated` if local reading exposes persisted-state, public contract, tenant isolation, auth/trust-boundary, concurrency/lifecycle, rollout, or proof-obligation uncertainty.
- Do not let "the user named this skill" override the tiny/direct-path exception; a workflow-planning session can return "do not create workflow-control files" when the repository contract says ceremony would add noise.

## Imitate

Direct-path calibration:

```markdown
Execution shape: direct path
Why: one reversible edit in one skill reference; no runtime behavior, repository architecture, public contract, persisted data, or multi-session resume risk.
Workflow artifacts: not expected; inline skip rationale is enough.
Adequacy challenge: skipped with tiny/direct-path rationale.
Stop rule: proceed only with the requested tiny local edit; no research or planning artifacts.
```

What to copy: the explanation names why the full workflow contract is not buying safety.

Lightweight-local calibration:

```markdown
Execution shape: lightweight local
Why: bounded single-package behavior change; no visible persisted-state, tenant, API contract, or rollout seam after the initial read.
Research mode: local in the next session.
Escalation trigger: reopen workflow planning if local research finds API, data, security, reliability, or rollout ambiguity.
Stop rule: after workflow-control handoff; do not start the local research read.
```

What to copy: the escalation trigger is part of the route, not an afterthought.

Full-orchestrated calibration:

```markdown
Execution shape: full orchestrated
Why: tenant-scoped async exports touch REST contract, job state, tenant isolation, signed URL security, admin authorization, reliability, observability, QA, and rollout.
Research mode: fan-out in the next session.
Adequacy challenge: required before handoff.
Stop rule: stop after workflow-control artifacts and challenge reconciliation; next session starts with research fan-out.
```

What to copy: the route names the affected seams without doing their research.

## Reject

```markdown
Execution shape: full orchestrated
Why: the repository prefers orchestrators and workflow artifacts.
```

Failure: generic process preference is not a task-specific risk argument.

```markdown
Execution shape: lightweight local
Why: probably only code.
Escalation trigger: none.
```

Failure: "only code" hides the exact seams that would force fan-out.

## Agent Traps
- Treating `workflow-planning-session` as a command to create workflow files even when the task is clearly tiny.
- Choosing `full orchestrated` because many skills exist, not because the task crosses material decision seams.
- Choosing `lightweight local` without recording what would make that choice wrong.
- Starting the next research step while "just checking" whether the chosen shape is right.
