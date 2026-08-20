# Intake

Distill raw or interpretation-sensitive input into one accepted task contract
that preserves the user's intended result and makes routing mechanical. Use only
the clarification needed to close that contract.

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

An Intake brief, usually in chat, that lets the workflow router and first owner recover the user's intent without reinterpreting the raw request. State the accepted user-visible or operator-visible outcome and, when it disambiguates that outcome, its business meaning; granted authority and any external effect outside that grant; meaningful scope and non-goals; the user- or operator-observable state, response, or effect that would make the request complete, distinct from the test, command, or proof strategy that may establish it; routing-changing repository facts; and bounded assumptions with their boundaries and reopen conditions. Omit inherited defaults and empty fields. End with the smallest valid path and first owner.

## Method

1. Distill one coherent contract rather than paraphrasing: apply explicit corrections, collapse repetition without losing priority, preserve operative values and exclusions, and separate the requested result from suggested means. Resolve clear supersession; give every remaining conflict one disposition through items 2–4 before accepting the brief.
2. Inspect only enough current repository state to resolve every still-unknown [Choose A Path](../../spec-first-workflow.md#choose-a-path) discriminator that can change the path or first owner: the referenced surface and current owner or artifact, affected breadth and ownership count, reversibility, protected-domain or cross-service triggers, validation boundedness, and dirty or continuation state when relevant. Record each established fact with its routing consequence. Route a question that requires decision-changing evidence to Research.
3. Under [Decision Ownership](../../../AGENTS.md#decision-ownership), keep an inference as a bounded assumption only when being wrong would leave the first owner's output usable without reopening an Intake-owned intent or authority decision; state its boundary and objective reopen condition.
4. Ask a question only under [Decision Ownership](../../../AGENTS.md#decision-ownership).
5. Finish with the smallest valid path. For `structured` or `orchestrated`, name the routing fact that makes the next-smaller path insufficient. Name the first owner and the exact accepted input or open question it receives; otherwise return the Stop Rule's single blocking question.

Use `grilling` only when the user explicitly requests an exhaustive stress test or interview. Repetition in the user's input is a priority signal, not a reason to expand the questionnaire.

## Stop Rule

End Intake only when all three checks pass:

- Every material statement about outcome, business meaning, scope or non-goals, authority, and observable success is user-supplied, established by current authority, or labeled as a bounded assumption with its boundary and objective reopen condition.
- Every routing-changing repository fact has a recorded routing consequence, and the smallest path and named first owner follow from the accepted contract and those facts.
- The first owner can begin from the brief without choosing an Intake-owned result, business meaning, requested scope boundary, authorization, or observable success signal.

Give every unresolved item exactly one disposition: a bounded assumption, a downstream question with its owner and routing consequence, or the single blocking question permitted by [Decision Ownership](../../../AGENTS.md#decision-ownership). Carry evidence, behavior, mechanism, placement, proof-strategy, or execution questions downstream only when the accepted outcome, business meaning, scope, and authority already make their answer a downstream decision. Keep any unresolved user-owned outcome, business meaning or policy value, requested scope boundary, or authorization decision that changes the first owner's permitted work in Intake as the blocking question.
