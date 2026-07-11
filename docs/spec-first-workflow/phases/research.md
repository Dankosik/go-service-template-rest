# Research

Resolve evidence gaps that can change a task decision. Research supports later decisions; it does not become a source dump or a mandatory phase.

## Read When

- Current repository or external evidence can change scope, feasibility, ownership, dependency choice, contract interpretation, or proof.
- External-platform behavior, an unfamiliar mechanism, new infrastructure/dependency, or a non-trivial design choice would otherwise be inferred from model memory.
- Sources conflict or freshness matters.
- A durable evidence note will be consumed by another phase or session.

## Inputs

- Accepted outcome and the exact decision each question can change.
- Named repository, provider, standards, official-document, source-code, or credible real-world implementation sources.
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

Use independent read-only lanes only for separable questions where parallel context materially helps.

When decision-changing research depends on conflicting sources, freshness-sensitive external behavior, or an approval-critical claim for a hard-to-reverse choice, obtain an independent read-only semantic challenge of the synthesis before specification or design consumes it. Evidence gathering alone cannot issue that verdict. If authoritative evidence cannot resolve a material conflict, block or return it to the evidence owner instead of choosing by confidence.

For an external platform, unfamiliar mechanism, new infrastructure/dependency, or non-trivial design choice, search current web and repository sources before design. Treat official docs, standards, maintainer source, and provider contracts as authority for current behavior. Use credible engineering articles and real implementations to find proven patterns, integration examples, operational constraints, and failure modes. Do not substitute model memory for current external evidence.

For a dependency or custom mechanism, compare the relevant Go stdlib, established repository pattern, mature maintained OSS, and custom implementation only when each is a real option. Approve external code only from current evidence for maintenance/releases, license, security or vulnerability posture, API stability, transitive cost, domain adoption, and repository/boundary fit. If required evidence is unavailable, block or carry the exact proof gap instead of treating popularity or one article as approval.

Prefer primary/current sources. A missing hit is not proof of absence unless the searched source is authoritative for absence. Stop when another source is unlikely to change the decision.

## Review

When `research only` is the accepted macro-phase boundary for structured or orchestrated work, obtain independent read-only review of the fixed synthesis before returning it. Ordinary supporting research that does not trigger the semantic-challenge rule above is consumed inside specification and uses specification review instead of a duplicate gate; a triggered synthesis challenge remains required before downstream consumption.

The reviewer checks question coverage, source authority and freshness, unresolved conflicts, fact/inference separation, and whether each decision implication follows from the evidence, then returns:

- `PASS`: the sufficient material evidence boundary supports every decision-changing conclusion and has no unowned question or uncovered affected lens;
- `CONCERNS`: a bounded residual evidence risk or downstream proof obligation still needs explicit disposition and fresh review; it does not permit the reviewed synthesis to leave research and may not carry a missing answer owned by research;
- `FAIL`: missing, stale, conflicting, or unavailable evidence makes a required material conclusion unreliable or prevents closure.

The owning root repairs and re-reviews under the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract until convergence. An explicitly requested standalone review of research remains read-only.

## Stop Rule

Finish when every decision-changing question is answered, honestly bounded, or assigned a blocker/reopen owner. When a required `research only` synthesis review applies, it must have returned `PASS`. If required current external evidence is unavailable, name the gap instead of inventing the answer. Hand the implications to specification or design; do not write their final decisions inside research.
