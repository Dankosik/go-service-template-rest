# Intake

Make the request routing-sufficient: reconstruct enough shared understanding to select `direct`, `structured`, or `orchestrated` work. Intake is a short clarification step, not an interview ritual.

## Read When

- The request is vague, dictated, mixed, or interpretation-sensitive.
- Scope, outcome, authorization, or success cannot yet be stated safely.
- A later phase discovers that it is relying on an untested assumption about user intent.

Skip when the request already supplies a clear outcome, scope, authority, constraints, and success signal.

## Inputs

- The user's request and corrections.
- Existing task artifacts for continuation work.
- Bounded repository inspection for facts about named files, behavior, commands, or prior decisions.

## Outputs

A brief, usually in chat:

- outcome, why it matters, and the authority granted;
- in-scope and out-of-scope work;
- confirmed affected surfaces, current repository owners relevant to routing, and material constraints;
- success/proof signal;
- bounded assumptions and what would invalidate them;
- next useful phase or direct action.

## Method

1. Reconstruct the likely outcome, scope, authority, success signal, and affected surfaces before asking anything.
2. Resolve repository facts, current owners relevant to routing, and current constraints by bounded inspection; leave target ownership and placement to the owning design phase.
3. State safe assumptions with their boundary and reopen condition.
4. Ask a question only under [Decision Ownership](../../../AGENTS.md#decision-ownership).
5. Close intake once the brief makes `direct`, `structured`, or `orchestrated` selectable.

Use `grilling` only when the user explicitly requests an exhaustive stress test or interview. Repetition in the user's input is a priority signal, not a reason to expand the questionnaire.

## Stop Rule

Continue to the selected path when the outcome, scope, authority, constraints, success signal, affected surfaces, and material assumptions are clear enough.

Do not research broadly, design, plan, or implement merely to avoid resolving a genuinely blocking intent decision.
