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

Each section records only requester meaning needed by the next phase. Place a
labeled assumption in the section whose meaning it affects. Omit `Open
questions` when none survive. Create the file only when every required section
is concrete enough for Specification and no unresolved question can change its
meaning. A material change updates the file and reopens Intake plus every
downstream decision it invalidates.
