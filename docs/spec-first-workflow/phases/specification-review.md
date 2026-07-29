# Specification Review

Review the fixed candidate through one kernel:
reconstruct -> falsify -> disposition -> verdict.

Apply the shared [Review
Independence](../shared/subagents-and-handoff.md#review-independence) contract.
That shared contract owns review triggering, the generic finding envelope,
verdict semantics, convergence, read-only boundaries, and return routing. This
file owns only Specification-specific affected-surface reconstruction,
falsifiers, evidence anchors, and completion coverage.

## Read When

- The shared Review Independence contract selects or requires independent review
  of a fixed specification.
- The user explicitly requests a standalone independent specification review.

## Inputs

- The current fixed `spec.md` candidate or diff.
- Accepted brief, relevant evidence, named sources of truth, and repository or
  consumer surfaces the accepted outcome can affect.
- Prior findings for a follow-up review.

## Outputs

Use the shared [Review Finding
Envelope](../shared/subagents-and-handoff.md#review-finding-envelope). For
Specification, anchor each surviving finding to the fixed candidate and the
contradicting evidence or omitted affected surface.

A surviving finding is Specification-blocking when it exposes a missing or
contradictory Specification-owned decision that would force design or planning
to invent product meaning.

## Review Method

Reconstruct the affected behavior surface independently from the accepted
brief, relevant research, current runtime/generated contracts, and repository
or consumer surfaces the accepted outcome can affect.
Do not treat omission from the spec as evidence that a lens is not triggered.
Falsify it with the smallest decision-changing questions below, then issue one
evidence-bounded verdict on the current candidate.

Try to falsify the spec with the smallest decision-changing questions:

- Which material affected surface reconstructed outside the candidate is
  omitted, contradicted, or unsupported enough that design or planning must
  invent product meaning?
- Which decision-changing claim, authority, invariant, compatibility or failure
  expectation, or proof obligation fails against a named current source or
  lacks a feasible falsifier?
- Does a non-goal, risk acceptance, or proof obligation hide a decision the spec
  must make now?
- Could two reasonable downstream implementations satisfy the candidate yet
  differ in any material actor, context, trigger, precondition, contract
  outcome, invariant or precedence rule, state transition or side effect,
  source-of-truth, absence, currentness, or finality semantics, rejection,
  duplication, replay, failure, or recovery behavior, compatibility or
  deliberately unchanged behavior, or measurable success condition?

Report only findings that can change readiness or required proof. Do not block
on writing style, optional detail, or a lens independently shown not to be
triggered.

## Stop Rule

Stop when every independently reconstructed material affected surface has been
tested, every surviving readiness or required proof issue is anchored, and one
evidence-bounded verdict covers the fixed candidate. Return routing follows the
shared contract. `PASS` is available only when no such divergence or other
Specification-owned decision remains. Use `CONCERNS` only for a bounded proof or
risk obligation over already accepted behavior; unresolved observable behavior,
policy, authority, compatibility, source-of-truth semantics, or success meaning
is `FAIL` and reopens Specification.
