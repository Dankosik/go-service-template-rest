# Research

Resolve evidence gaps that can change a task decision. Research supports later decisions; it does not become a source dump or a mandatory phase.

## Read When

- Current repository or external evidence can change scope, feasibility, ownership, dependency choice, contract interpretation, or proof.
- Sources conflict or freshness matters.
- A durable evidence note will be consumed by another phase or session.

## Inputs

- Accepted outcome and the exact decision each question can change.
- Named repository, provider, standards, or official-document sources.
- Current evidence limits and freshness needs.

## Outputs

- Findings with source pointers and dates/versions when relevant.
- Facts, inferences, conflicts, and missing evidence kept distinct.
- The decision implication: decide, constrain, assume, accept risk, require proof, or block.
- A compact `research/*.md` note only when reuse or auditability justifies it.

## Method

For each question, state:

- the decision it can change;
- the most authoritative practical source;
- the minimum evidence needed;
- how absence, conflict, or staleness will be handled;
- when to stop searching.

Use independent read-only lanes only for separable questions where parallel context materially helps. For a dependency or custom mechanism, compare the relevant Go stdlib, established repository pattern, and mature maintained OSS; record rejected options only when they were genuinely viable.

Prefer primary/current sources. A missing hit is not proof of absence unless the searched source is authoritative for absence. Stop when another source is unlikely to change the decision.

## Stop Rule

Finish when every decision-changing question is answered, honestly bounded, or assigned a blocker/reopen owner. Hand the implications to specification or design; do not write their final decisions inside research.
