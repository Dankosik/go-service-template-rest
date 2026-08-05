# Research

Research owns discovering, verifying, falsifying, comparing, and synthesizing the evidence that can change a task decision. It closes decision-changing questions rather than accumulating sources.

## Read When

- Current repository or external evidence can change scope, feasibility, ownership, dependency choice, contract interpretation, or proof.
- External-platform behavior, an unfamiliar mechanism, new infrastructure/dependency, or a non-trivial design choice would otherwise be inferred from model memory.
- Sources conflict or freshness matters.
- A durable evidence note will be consumed by another phase or session.

## Inputs

- Accepted outcome, current-state boundary, and the exact decision, owner, or criterion each question can change.
- Named repository, runtime, workload, telemetry, user/operator, provider, standards, official-document, source-code, or credible real-world implementation sources when they can change the decision.
- Known hypotheses or live alternatives, current evidence limits, freshness needs, and any concrete input a downstream phase already requires.

## Outputs

- A compact open-item map. For each item, retain the affected decision and owner, its Method classification, downstream disposition, and any carried constraint, assumption, risk, proof obligation, blocker/reopen condition, or objective refresh trigger.
- For each evidence question, retain one cross-source synthesis organized by decision-changing claim rather than by source. It separates established fact, inference, conflict, authoritative absence, assumption, and unknown; treats an unavailable or unsearched decision-relevant surface as unknown rather than absence; disposes the leading hypothesis and material counter-evidence; carries for each claim a locator a later reader can reopen without repeating the search, plus that claim's scope, date, and evidence limits; explains how authority, applicability, agreement, conflict, and counter-evidence make the conclusion follow or remain unresolved; states which downstream decision the evidence supports, constrains, eliminates, or leaves open; and records why further relevant source inspection or, when applicable, a safe discriminating probe is unlikely to change that disposition.
- For each decision-changing quantity, retain its provenance label.
- When a solution choice is live, a compact candidate map: the neutral frame; each candidate's decision slot and relationship; materially distinct families at the live decision level and representative implementations only where relevant; scanned rungs; local-fit evidence or rejection reason for excluded viable candidates; any decision-flip condition; and the bounded stop rationale.
- A compact `research/*.md` note only when the evidence must be reused, audited,
  or refreshed.

If persisted, retain only sanitized evidence or a safe pointer, never secrets or restricted data.

## Method

Scale depth to decision impact, reversibility, uncertainty, and evidence volatility, not source or lane count. Before searching, classify each open item and route it to its smallest owner: research an evidence question; route a target, policy, or risk-tolerance choice to its owner under [Decision Ownership](../../../AGENTS.md#decision-ownership); route a mechanism choice to design; or carry a later proof obligation to test design. Do not substitute missing proof for an unset acceptance target. Label each decision-changing quantity as a measured baseline, external limit or quota, forecast, accepted target, or assumption. Before selecting a branch, derive the evidence lenses from the named decision, its authority path, and every current surface that can independently change the answer. For each plausible lens, record `researched`, `established by current authoritative evidence`, or `not triggered: <current evidence showing why it cannot change the decision>`.

A secondary source is a lead to the owner of a claim, never the carried
authority: follow every decision-changing claim back to the source that owns it
before recording it.

### Question Closure

Before substantive search, write each decision-changing question in this form; revise it only when evidence changes the decision boundary:

- the decision, owner, driver, or observable criterion it can change;
- the leading hypothesis or live alternatives and what evidence would falsify them, when present;
- the smallest evidence that could falsify the leading implication;
- the authoritative source boundary and most authoritative practical source within it;
- the semantic terms, identifiers or aliases, applicable versions, and failure modes that could expose differently named or contradictory evidence;
- the minimum coverage needed across every authority, local-applicability, observation, and counter-evidence surface that can independently change the answer;
- how absence, conflict, or staleness will be handled;
- when to stop searching.

A missing hit is not proof of absence unless the searched source is authoritative for absence. For a material absence claim, record the authoritative surfaces and search boundary.

Actively test the leading claim against material counter-evidence and the strongest viable alternative, including reuse, status quo, deletion, or process change when one can satisfy the accepted outcome. Do not invent weights or aggregate scores. When an uncertain driver could reverse the implication, record the range or threshold that would flip it, its owner, and the smallest evidence that would resolve it.

For questions other than solution discovery, apply the [Stop Rule](#stop-rule) to the affected decision.

### Branch Selection

Choose the smallest triggered branch and load only its rule:

- [Current-state or semantic baseline](research-branches.md#current-state-or-semantic-baseline) when repository, generated, runtime, persisted or deployed, consumer, or operator-observed state can change accepted behavior or a later decision, especially when those surfaces may diverge.
- [Current external contract](research-branches.md#current-external-contract) when external-platform, Go, toolchain, runtime, or dependency behavior would otherwise be inferred from memory.
- [Solution discovery evidence](research-branches.md#solution-discovery-evidence) when a mechanism or implementation choice remains live.
- [Empirical claim or probe](research-branches.md#empirical-claim-or-probe) when the decision uses empirical/runtime evidence or authoritative sources cannot resolve a decision-changing empirical claim.
- [Conflict or freshness](research-branches.md#conflict-or-freshness) when material sources conflict, freshness can flip the decision, or an approval-critical hard-to-reverse claim needs semantic challenge.

After applying the first branch, load another only when its distinct evidence pressure can independently change the same or another named decision; one question may therefore require more than one branch.

After Question Closure and branch selection expose the current question and
lens map, apply the shared [Delegation
Decision](../shared/subagents-and-handoff.md#delegation-decision). The root
retains cross-source synthesis and every downstream disposition.

Apply [Downstream input closure](research-branches.md#downstream-input-closure) only when research identifies a concrete required external input or a conclusion carries a later proof obligation.

## Review

When `research only` is the accepted macro-phase boundary, apply root self-review and obtain independent read-only review only when the shared review trigger applies. Ordinary supporting research is consumed inside specification; a risk-triggered synthesis challenge remains separate.

The reviewer checks the Outputs against the Method: affected-lens and question coverage; open-item and quantitative provenance; source authority, applicability, and freshness; falsification; current-state semantic baseline and probe limits when triggered; candidate decision levels, relationships, local fit, sensitivity, and saturation when a solution choice is live; unresolved conflicts; fact/inference separation; and evidence-backed downstream dispositions, then returns:

- `PASS`: the sufficient material evidence boundary supports every decision-changing conclusion and has no unowned question or uncovered affected lens;
- `CONCERNS`: a bounded residual evidence risk or downstream proof obligation may move after its owner, observable, and refresh/reopen condition are recorded; it may not carry a missing answer owned by research;
- `FAIL`: missing, stale, conflicting, or unavailable evidence makes a required material conclusion unreliable or prevents closure.

The owning root applies the shared [Review Independence](../shared/review-independence.md) trigger and the loaded Subagents And Review owner for finding disposition, convergence, and standalone-review boundaries.

## Stop Rule

Finish when every required Output is present; every triggered evidence surface has been inspected or its unavailability recorded with the decision effect; each leading implication has been tested against material counter-evidence and dispositioned; the cross-source synthesis gives the named downstream owner an explicit decision implication; and its recorded stop rationale shows that further source inspection or, when applicable, a safe discriminating probe is unlikely to change, narrow, or reopen the affected decision. For solution discovery, also satisfy the branch's [candidate-space saturation rule](research-branches.md#solution-discovery-evidence). The next owner must be able to act without repeating the search or re-synthesizing the sources. A bounded gap required by the next phase blocks and reopens its smallest evidence or decision owner; an optional later proof may carry only with its owner and recheck condition. Any triggered `research only` review must have returned `PASS` or dispositioned `CONCERNS`. Hand the implications to the named downstream owner; do not write the final specification or design decision inside research.
