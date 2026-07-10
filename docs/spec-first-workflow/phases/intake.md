# Intake

Turn raw input into enough shared understanding to act or route. Intake is a short clarification step, not an interview ritual.

## Read When

- The request is vague, dictated, mixed, or interpretation-sensitive.
- Scope, outcome, authorization, or success cannot yet be stated safely.
- A later phase discovers that it is relying on an untested assumption about user intent.

Skip when the request already supplies a clear outcome, scope, constraints, and success signal.

## Inputs

- The user's request and corrections.
- Existing task artifacts for continuation work.
- Bounded repository inspection for facts about named files, behavior, commands, or prior decisions.

## Outputs

A brief, usually in chat:

- outcome and why it matters;
- in-scope and out-of-scope work;
- confirmed affected surfaces and material constraints;
- success/proof signal;
- bounded assumptions and what would invalidate them;
- next useful phase or direct action.

## Method

1. Reconstruct the likely brief before asking anything.
2. Resolve repository facts by inspection.
3. Ask one question only when its answer would materially change behavior, scope, ownership, safety, authorization, or proof and no safe bounded assumption exists.
4. State safe assumptions with their reopen condition and continue.
5. Stop once the brief is sufficient to choose `direct`, `structured`, or `orchestrated` work.

Use `grilling` only when the user explicitly requests an exhaustive stress test or interview. Repetition in the user's input is a priority signal, not a reason to expand the questionnaire.

## Stop Rule

Continue to the next needed phase when the outcome, scope, constraints, success signal, and material assumptions are clear enough. Stop and ask the smallest blocking question when guessing would change correctness or authority.

Do not research broadly, design, plan, or implement merely to avoid resolving a genuinely blocking intent decision.
