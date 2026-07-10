# Specification

Define what must be true before deciding how to implement it. Keep the spec as small as the decision surface allows.

## Read When

- Behavior, scope, invariants, compatibility, ownership policy, or proof expectations must survive into design or planning.
- Implementation would otherwise infer product meaning from chat.
- An existing `spec.md` needs repair after evidence or review.

Direct work may skip a durable spec when the accepted outcome and proof are obvious.

## Inputs

- Accepted brief and relevant research.
- Current runtime/generated contracts and source-of-truth evidence.
- Existing spec and review findings for repair work.

## Outputs

A compact `spec.md` or inline decision record containing only applicable sections:

```markdown
# <Outcome>
status: draft | ready | blocked

## Scope and non-goals
## Behavior and contract delta
## Invariants and edge cases
## Decisions and constraints
## Risks, assumptions, and proof expectations
```

Record changed, removed, and deliberately unchanged behavior. For a replaced surface, either include removal/refactor work or state the current compatibility owner, evidence for retention, and exit condition.

## Decision Bar

The spec is ready when:

- observable behavior and important unchanged invariants are explicit;
- public contract, data, security, money, concurrency/lifecycle, delivery, or cross-service effects in scope have relevant decisions and proof expectations;
- runtime/generated sources of truth are named where they matter;
- non-goals do not hide required target-state work;
- assumptions have boundaries and reopen conditions;
- design and planning can proceed without choosing product meaning;
- no material `TBD` or unresolved alternative remains.

Choose production-ready behavior for the accepted scope unless the user asked for a prototype or staged result. Do not add exhaustive sections for unaffected concerns.

## Review

Perform a focused self-review for ordinary bounded work. Use the independent [Specification Review](specification-review.md) when required by the user or when the spec is high-impact, hard to reverse, ambiguous, cross-owner, or difficult for the author to falsify.

After actionable findings, repair the spec and recheck the changed decision surface. Preserve accepted concerns only when they are bounded risks or downstream proof obligations, not missing spec decisions.

When specification owns the review, review, repair, and fresh re-review run as internal checkpoints in the same root session; they do not produce a next-session prompt.

## Stop Rule

Continue to design or planning when the decision bar is met and the required review has no blocker. Reopen intake, research, or a user/specialist decision when the missing answer belongs there. Do not start implementation from a blocked spec.
