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

An Intake brief, usually in chat, that lets the workflow router and first owner recover the user's intent without reinterpreting the raw request. Under the [Task Contract](../../../AGENTS.md#task-contract), state the accepted user-visible or operator-visible outcome and, when it disambiguates that outcome, its business meaning; granted authority and any external effect outside that grant; meaningful scope and non-goals; the observable success signal; routing-changing repository facts; and bounded assumptions with their boundaries and reopen conditions. Omit inherited defaults and empty fields. End with the smallest valid path and first owner.

## Method

1. Distill rather than paraphrase: preserve explicit values, corrections, priorities, exclusions, and the requested result; separate that result from suggested means; label inferred meaning as a bounded assumption or the permitted blocking question.
2. Inspect only enough current repository state to identify the referenced surface, existing owner or artifact, reversibility or protected-domain trigger, and continuation state when any can change the path or first owner; record the routing consequence. Route a question that requires decision-changing evidence to Research.
3. Under [Decision Ownership](../../../AGENTS.md#decision-ownership), keep an inference as a bounded assumption only when being wrong would leave the first owner's output usable without reopening an Intake-owned intent or authority decision; state its boundary and objective reopen condition.
4. Ask a question only under [Decision Ownership](../../../AGENTS.md#decision-ownership).
5. Finish with the Stop Rule's smallest valid path and first owner, or its single blocking question.

Use `grilling` only when the user explicitly requests an exhaustive stress test or interview. Repetition in the user's input is a priority signal, not a reason to expand the questionnaire.

## Stop Rule

End Intake with the smallest valid path and named first owner when that owner can begin from the brief without rereading the raw request to decide the intended user-visible or operator-visible result, meaningful scope/non-goals, granted authority, or observable success signal. Carry each open evidence, mechanism, placement, proof-strategy, or execution question to its owning phase; none may conceal an unresolved user-owned intent or authority decision. Otherwise, return the single blocking question permitted by [Decision Ownership](../../../AGENTS.md#decision-ownership).
