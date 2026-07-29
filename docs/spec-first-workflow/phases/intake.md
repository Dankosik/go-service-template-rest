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

A brief, usually in chat, under the [Task Contract](../../../AGENTS.md#task-contract), containing only the accepted outcome and authority plus any scope boundary, constraint, success/proof signal, surface/owner fact, or bounded assumption that can change routing. Include the business meaning only when it changes interpretation, and finish with the Stop Rule's result.

## Method

1. Reconstruct the likely outcome, scope, authority, and success signal before asking anything.
2. Inspect only repository facts and surface/owner signals that can change path selection; leave decision-changing evidence and mechanism or placement to their owning phases.
3. State safe assumptions with their boundary and reopen condition.
4. Ask a question only under [Decision Ownership](../../../AGENTS.md#decision-ownership).
5. Finish with the Stop Rule's named path and first action, or its single blocking question.

Use `grilling` only when the user explicitly requests an exhaustive stress test or interview. Repetition in the user's input is a priority signal, not a reason to expand the questionnaire.

## Stop Rule

End Intake with a named path and first action when that action can proceed without guessing user intent or exceeding granted authority. Otherwise, return the one user-owned blocking question permitted by [Decision Ownership](../../../AGENTS.md#decision-ownership).

Carry open evidence, mechanism, placement, proof-strategy, or execution questions to their owning phase; Intake needs only enough information to route them.
