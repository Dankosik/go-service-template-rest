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

- Accepted brief and relevant research, or the user's own request when intake was
  skipped.
- Current runtime/generated contracts and source-of-truth evidence.
- Accepted decisions already recorded by a canonical document in the affected
  area.
- Existing spec and review findings for repair work.

## Outputs

For structured/orchestrated work, synthesize a compact behavioral contract in
`spec.md`; direct work may keep the same rules inline. A summary of inputs is
traceability evidence, not the contract. Use only applicable sections:

```markdown
# <User/operator-visible outcome>
status: draft | ready | blocked
Problem: <problem this outcome removes, in user or operator terms>  <!-- when the spec outlives its chat -->

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

Specification owns every material choice that can change user- or
operator-visible meaning: scope, behavior, policy, contract semantics,
authority, source-of-truth semantics, compatibility, and proof expectations.
This ownership includes cross-service behavior and outcomes visible only
through rejection, failure, replay, recovery, finality, or operator action.
Apply `go-api-contract` here when client-visible REST semantics are affected;
close the audience, request and response meaning, errors and status, validation,
limits, retry and idempotency behavior, concurrency or async semantics,
freshness, and compatibility before technical design.

Technical design owns concrete contract/schema representation, runtime
mechanism and sequence, and package/file placement only after those semantics
are fixed, unless a mechanism is an accepted external constraint. It may choose
only behaviorally equivalent realizations. If a representation or mechanism
choice would select among materially different observable outcomes, reopen
Specification.

Derive the affected behavior surface from the accepted brief, relevant
research, current runtime/generated contracts, and repository or consumer
surfaces the accepted outcome can affect; do not infer coverage only from what
an existing spec already mentions. Start with the behavior delta and important
unchanged invariants. Sweep the actors a surface inventory alone can miss, such
as an operator or the owner of a migrating consumer, and give each affected actor
a behavioral rule or an explicit unchanged or non-goal disposition. Consider an
additional lens only when current evidence shows it can change that meaning,
constrain it, or alter required proof. For
each triggered lens, record the decision, unchanged constraint, proof
consequence, or named reopen owner. Persist this inventory only when it changes
an action, verdict, handoff, or resume decision.

Close each accepted intent and decision-changing evidence implication as a
behavioral rule or an explicit unchanged, non-goal, or blocked disposition. For
each candidate rule, ask whether two reasonable implementations could satisfy
its wording yet differ in a user- or operator-visible result. Resolve every such
divergence from current authority or evidence, a Specification-owned decision,
or a bounded assumption with an objective reopen condition.

A material rule satisfies the **material-rule contract** only when it fixes
every interpretation-changing part:

- actor and context, trigger and input, and preconditions;
- normative rule, invariant, precedence, states, and allowed, invalid,
  repeated, and terminal transitions;
- observable outcome and contract delta across applicable normal, boundary,
  rejection, absence, duplication, replay, failure, recovery, and compatibility
  behavior;
- side effects required, forbidden, or already possible at each observable
  outcome, plus deliberately unchanged behavior;
- decision-changing terms, units, bounds, and default or no-match behavior;
- the nearest feasible falsifier or proof expectation.

Omit a part only when current evidence shows it cannot change interpretation.
Use stable rule IDs only when multiple owners or downstream artifacts need them.
Take any decision-changing term already owned by [Domain
Vocabulary](../../repo-architecture.md#domain-vocabulary) with its recorded
meaning instead of redefining it.

When source-of-truth semantics are triggered, state which fact is authoritative,
which observations are derived, what absence, currentness, and finality mean,
and which authority wins a conflict. Name a concrete runtime or storage
mechanism only when it is an accepted external constraint.

Use plain prose by default. When a material rule remains ambiguous, use the
smallest matching representation:

- explicit precondition, trigger, and response for conditional behavior;
- decisive context/event/outcome examples for interpretation-sensitive rules;
- a compact decision table for interacting conditions, precedence, and
  default/no-match behavior;
- a compact state model for lifecycle, invalid or repeated events, and terminal
  behavior;
- a quality scenario naming source, stimulus, environment, affected surface,
  response, and response measure for a material non-functional outcome;
- a literal fragment such as a schema or type shape when it fixes the decision
  more precisely than prose can, kept to the decision-carrying parts rather than
  a working example.

Keep the representation to the ambiguity it resolves. Leave exhaustive
scenarios to test design, and omit unaffected concerns and unset targets. Name a
canonical authority when a decision rests on it, but carry no implementation
path, symbol, or code beyond that fragment; placement and code remain design and
implementation decisions.

Ground each decision-changing factual claim in current evidence. Resolve each
normative choice from the accepted outcome, a named authority, or the applicable
owner under [Decision Ownership](../../../AGENTS.md#decision-ownership), and
record the chosen rule and rationale. Mark any claim the repository cannot
re-derive, such as a value stated only in the request or a shape taken only from
a prototype. When a canonical document already records an accepted decision on
the same question, its recorded reopen condition is the test for contradicting
it. A bounded assumption is valid only when it keeps the accepted outcome honest
and useful; name its safe boundary and
objective reopen condition. Missing user- or external-owner policy blocks that
owner. Missing support for a Specification-owned choice requires a narrower
claim or a recorded Specification decision, not deferral to design. Keep raw
evidence in research.

## Decision Bar

The spec is ready only when the affected behavior surface has been reconstructed
from every applicable source named in Method, each resulting materially affected
case has one falsifiable behavioral answer, and every live downstream
realization permitted by the spec is behaviorally equivalent:

- each accepted intent and decision-changing evidence implication is closed as
  a traceable, falsifiable behavioral rule grounded in accepted intent, current
  evidence or a named authority, or an explicit Specification-owned decision
  with rationale, and passes the material-rule contract and two-implementation
  divergence test; otherwise it has an explicit unchanged or non-goal
  disposition, or a blocker owned by the user or a named external owner;
- triggered source-of-truth semantics satisfy the authority, absence,
  currentness, finality, and conflict contract in Method rather than merely
  naming a source;
- changed, removed, compatible, and deliberately unchanged behavior is explicit,
  and non-goals do not hide required target-state work;
- each success criterion names its observable scope and pass/fail condition;
  numeric targets appear only when supported by accepted authority or evidence;
- each proof expectation names its evidence boundary without selecting an
  implementation mechanism;
- each assumption names the affected rule, safe boundary, objective invalidating
  evidence, reopen owner, and reopen condition;
- every materially triggered lens has the decision, unchanged constraint, or
  proof consequence required by the accepted scope;
- no unresolved scope, behavior, policy, authority, source-of-truth,
  compatibility, or risk-acceptance alternative owned by Specification remains;
- design receives closed behavioral semantics and may choose only concrete
  representations, mechanisms, sequences, and placements that preserve them
  without choosing product or system meaning;
- no material `TBD` remains.

Only uncertainty about proving an already accepted rule may carry as a
downstream proof obligation. Missing scope, behavior, invariant, ownership,
compatibility, authority, source-of-truth, success meaning, or risk-acceptance
decisions remain current-phase defects.

Unless the accepted outcome is explicitly a prototype or staged result, close
every materially triggered production behavior required by the accepted scope.
Use only triggered rules and supported values; unaffected concerns create no
section or decision.

## Review

Apply focused root self-review. Run independent [Specification
Review](specification-review.md) only when the shared review trigger applies.

Use the shared [Review Independence](../shared/review-independence.md) trigger and
the loaded Subagents And Review contract for
disposition, repair, and convergence.

## Stop Rule

Continue to design or planning when the Decision Bar is met and any triggered
review has returned `PASS` or dispositioned `CONCERNS`. Reopen intake, research,
or a specialist decision when the missing answer belongs there; reopen a user
decision only under
[Decision Ownership](../../../AGENTS.md#decision-ownership). Do not start
implementation from a blocked spec.
