# Specification

Define meaning before mechanism: what must be true before deciding how to implement it. Preserve the
user/operator-visible outcome, material constraints, available evidence, and
completion bar while keeping the spec as small as the decision surface allows.

## Read When

- Behavior, scope, invariants, compatibility, authority, or proof expectations
  must survive into design or planning.
- Implementation would otherwise infer product meaning from chat.
- An existing `spec.md` needs repair after evidence or review.

## Inputs

- Accepted brief and relevant research.
- Current runtime/generated contracts and source-of-truth evidence.
- Existing spec and review findings for repair work.

## Outputs

For structured/orchestrated work, a compact `spec.md`; direct work may keep the
same decisions inline. Use only applicable sections:

```markdown
# <User/operator-visible outcome>
status: draft | ready | blocked

## Scope and non-goals
## Behavior and contract delta
## Invariants and edge cases
## Decisions, constraints, and authorities
## Success criteria and proof expectations
## Risks, assumptions, and reopen conditions
## Blockers / decisions needed  <!-- only when blocked -->
```

`Blockers / decisions needed` carries only an item the user or a named external
owner owns under [Decision
Ownership](../../../AGENTS.md#decision-ownership).

Record changed, removed, and deliberately unchanged behavior. For a replaced
surface, either make removal/refactor part of the target state or state the
current compatibility owner, evidence for retention, and exit condition.

## Method

Specification owns observable scope, behavior, policy, authority, and source-of-truth semantics. Technical design owns runtime mechanism, sequence, and package/file placement unless a mechanism is itself an accepted external constraint. Close this boundary before detailing rules so downstream work cannot silently turn a mechanism choice into product meaning.

Derive the affected behavior surface from the accepted brief, relevant
research, current runtime/generated contracts, and repository or consumer
surfaces the accepted outcome can affect; do not infer coverage only from what
an existing spec already mentions. Start with the behavior delta and important
unchanged invariants. Consider an additional lens only when current evidence
shows it can change that meaning, constrain it, or alter required proof. For
each triggered lens, record the decision, unchanged constraint, proof
consequence, or named reopen owner. Persist this inventory only when it changes
an action, verdict, handoff, or resume decision.

For each material rule, state only the parts that can change interpretation:
actor or context, trigger or input, preconditions, rule or invariant, state and
side effects, observable outcome, applicable rejection/failure/absence/replay/
recovery behavior, deliberately unchanged behavior, and the nearest feasible
falsifier or proof expectation. Keep each rule unambiguous and observable at the
correct ownership level; define decision-changing terms, units, and bounds. Use
stable rule IDs only when multiple owners or downstream artifacts need them.

Use plain prose by default. When a material rule remains ambiguous, use the
smallest matching representation:

- explicit precondition, trigger, and response for conditional behavior;
- decisive context/event/outcome examples for interpretation-sensitive rules;
- a compact decision table for interacting conditions, precedence, and
  default/no-match behavior;
- a compact state model for lifecycle, invalid or repeated events, and terminal
  behavior;
- a quality scenario naming source, stimulus, environment, affected surface,
  response, and response measure for a material non-functional outcome.

Keep the representation to the ambiguity it resolves. Leave exhaustive
scenarios to test design, and omit unaffected concerns and unset targets.

Ground each decision-changing factual claim in current evidence and each
normative choice in an explicit accepted decision. Label inferences and
assumptions, and expose missing or conflicting evidence. Missing support
narrows the claim or blocks/reopens its owner; it is not evidence of absence.
Keep raw evidence in research.

## Decision Bar

The spec is ready when:

- material behavior, task-specific success criteria, and important unchanged
  invariants are explicit and falsifiable;
- every material lens triggered by the affected behavior surface has the
  decision, unchanged constraint, or proof consequence needed by the accepted
  scope; examples include public contract, data, security, money, reliability,
  performance, observability/operability, concurrency/lifecycle, delivery, and
  cross-service behavior;
- runtime/generated sources of truth and decision-changing evidence are named
  where they matter;
- non-goals do not hide required target-state work;
- assumptions have boundaries and reopen conditions;
- no unresolved scope, behavior, policy, authority, source-of-truth, or
  risk-acceptance alternative owned by Specification remains;
- design receives the accepted behavior, constraints, and decision drivers
  needed to choose any live mechanism or placement alternative;
- design and planning can proceed without inventing product meaning;
- no material `TBD` remains.

Only uncertainty about proving an already accepted rule may carry as a
downstream proof obligation. Missing scope, behavior, invariant, ownership,
compatibility, authority, or risk-acceptance decisions remain current-phase
defects.

Unless the accepted outcome is explicitly a prototype or staged result, close
every materially triggered production behavior needed by the accepted scope,
including applicable failure, compatibility, security, lifecycle, and
operability semantics. Do not invent numeric targets, policies, consumers, or
scope. Do not add exhaustive sections for unaffected concerns.

## Review

Apply focused root self-review. Run independent [Specification
Review](specification-review.md) only when the shared review trigger applies.

Use the shared [Review
Independence](../shared/subagents-and-handoff.md#review-independence) contract for
disposition, repair, and convergence.

## Stop Rule

Continue to design or planning when the Decision Bar is met and any triggered
review has returned `PASS` or dispositioned `CONCERNS`. Reopen intake, research,
or a specialist decision when the missing answer belongs there; reopen a user
decision only under
[Decision Ownership](../../../AGENTS.md#decision-ownership). Do not start
implementation from a blocked spec.
