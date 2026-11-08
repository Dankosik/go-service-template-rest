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

Record only requester meaning needed by the next phase. Reserve `Open questions`
for non-blocking questions that need input from the requester or an external
source. Omit the section when none remain.

- Distinguish requirements from preferences in the section they affect. For
  each material delegated choice, state what the next agent is responsible
  for deciding, the outcome that guides the choice, and the limits that
  apply. The agent makes and acts on that decision within the authority
  already granted. Routine technical choices follow
  [Decision Ownership](../../../AGENTS.md#decision-ownership).
- Label a material assumption where it affects meaning and state what would
  prompt reconsideration. Silence alone cannot delegate a requester-owned
  decision or grant authority to act.
- Check that the next agent can choose and justify a course of action within
  those limits. Return a decision to the requester only when new evidence
  changes the agreed outcome, reveals a material preference that cannot be
  inferred, or calls for additional authority. Create the file only when
  every required section is concrete enough for Specification and no
  unresolved question can change its meaning.

A material change updates the file and reopens Intake plus every downstream
decision it invalidates.
