# Intent

Use one task-local `specs/<task>/intent.md` to preserve requester meaning after
the Intake interview and before Specification. It is the synthesized intent,
not a transcript, behavior specification, design, proof plan, or task list.

```markdown
# Intent: <short desired outcome>

## Problem

## Desired outcome

## Affected actors and systems

## Scope and non-goals

## Constraints

## Success signal

## Open questions
- <non-blocking question>
```

Record only requester meaning needed by the next phase. Omit `Open questions`
when none survive.

- Distinguish requirements from preferences in the section they affect. When
  the requester delegates a material choice, state what the agent may decide
  and within what limits. Routine technical choices follow
  [Decision Ownership](../../../AGENTS.md#decision-ownership).
- Label a material assumption where it affects meaning and state what would
  prompt reconsideration. Silence alone cannot delegate a requester-owned
  decision or grant authority to act.
- Before writing the file, check that the next agent can tell what they must
  deliver, what they may decide, and when a decision must return to the
  requester. Create it only when every required section is concrete enough
  for Specification and no unresolved question can change its meaning.

A material change updates the file and reopens Intake plus every downstream
decision it invalidates.
