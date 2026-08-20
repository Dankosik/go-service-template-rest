# Codex Read-Only Lanes

## Read When

Read immediately before spawning or waiting on a Codex research, challenge, or
review subagent. Apply the shared [Read-Only
Carrier](../shared/read-only-carrier.md) and installed callable schemas.

## Dispatch And Identity

Use a fresh built-in project subagent for one lane brief. Pass no inherited
turns unless irreproducible user context requires the smallest bounded recent
set. Select the matching project agent type; an implementation review uses
`task-acceptance-agent`, with `critical-reviewer-agent` reserved for a justified
highest-consequence boundary.

For implementation review, begin the brief with:

```text
Execution role: ACCEPTANCE_REVIEWER
Skill: $acceptance-reviewer
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree
```

Retain the native subagent identity returned by dispatch. The lane exists only
after that identity is returned; wait and follow-up controls address only that
identity. A missing identity or wait capability is a carrier failure, not
progress, and does not permit a substitute review.

Carry the selected model and effort through installed structured fields when
the schema exposes them. A fixed-candidate review is fresh and one-shot; a new
candidate or unit receives a new lane.
