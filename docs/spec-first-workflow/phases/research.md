# Research

Resolve only evidence gaps that can change a task decision. Research closes decision-changing questions rather than accumulating sources.

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

- A compact open-item map. For each item, retain the affected decision and owner, its Method classification, downstream disposition, and any carried constraint, assumption, risk, proof obligation, blocker/reopen condition, or objective refresh trigger. For an evidence question, also retain fact/inference/conflict/missing-evidence status; any hypothesis disposition; a claim-level source with scope/revision/date; evidence limits; and the decision implication. For a decision-changing quantity, retain its provenance label.
- When a solution choice is live, a compact candidate map: the neutral frame; each candidate's decision slot and relationship; materially distinct families at the live decision level and representative implementations only where relevant; scanned rungs; local-fit evidence or rejection reason for excluded viable candidates; any decision-flip condition; and the bounded stop rationale.
- A compact `research/*.md` note only when reuse or auditability justifies it.

## Method

Scale depth to decision impact, reversibility, uncertainty, and evidence volatility, not source or lane count. Before searching, classify each open item and route it to its smallest owner: research an evidence question; route a target, policy, or risk-tolerance choice to its decision owner; route a mechanism choice to design; or carry a later proof obligation to test design. Do not substitute missing proof for an unset acceptance target. Label each decision-changing quantity as a measured baseline, external limit or quota, forecast, accepted target, or assumption. Map only materially affected evidence lenses, marking each researched, established by current authoritative evidence, or not triggered with a concrete reason.

For each question, state:

- the decision, owner, driver, or observable criterion it can change;
- the leading hypothesis or live alternatives and what evidence would falsify them, when present;
- the smallest evidence that could falsify the leading implication;
- the authoritative source boundary and most authoritative practical source within it;
- the minimum evidence needed;
- how absence, conflict, or staleness will be handled;
- when to stop searching.

Choose the smallest triggered research branch: current-state or semantic baseline; current external contract; solution discovery and comparison; representative empirical probe; or conflicting or freshness-sensitive evidence. Add another branch only for an independent evidence pressure that can change the named decision.

Use independent read-only lanes only for separable questions where parallel context materially helps.

When current state can change accepted behavior or a later decision, establish only the smallest decision-relevant baseline, including as applicable the canonical contract, observed runtime behavior, material drift, callers or operators, persisted or deployed state, external consumers, mixed-version constraints, and the first unsupported edge. Trace an affected value or contract from origin through persistence, transport, consumer, and derived output only as far as the decision requires; preserve grain or cardinality, identifiers, units, absence or error states, version authority, and lossy transformations. Matching names, historical intent, or receiver capability do not prove equivalent semantics or runtime use. Inspect ADRs, issues, incidents, or runbooks only when replacing unexplained brownfield behavior and only for a still-current constraint; history alone is not a current contract.

Actively test the leading claim against material counter-evidence and the strongest viable alternative, including reuse, status quo, deletion, or process change when one can satisfy the accepted outcome. Do not invent weights or aggregate scores. When an uncertain driver could reverse the implication, record the range or threshold that would flip it, its owner, and the smallest evidence that would resolve it.

When the way to realize a behavior remains unresolved, do not anchor discovery on a user-proposed pattern, product, or implementation. Derive vendor-neutral search terms from the required behavior, invariants, workload, failure and recovery model, quality scenarios, constraints, and unacceptable trade-offs. Use pattern catalogs and reference architectures to expand vocabulary, not as proof of local production fit. Name the responsibility or decision slot each candidate occupies and classify it as a substitute, prerequisite, complement, or defense-in-depth mechanism. Compare or rank only substitutes for the same slot and live decision level: discover mechanism families when the mechanism is unresolved, or compare implementations within an accepted mechanism without reopening it. Record separately any product, library, managed service, transport, or topology roles used to realize the candidate.

When decision-changing research depends on conflicting sources, freshness-sensitive external behavior, or an approval-critical claim for a hard-to-reverse choice, consider an independent read-only semantic challenge before specification or design consumes it. If authoritative evidence cannot resolve a material conflict, block or return it to the evidence owner instead of choosing by confidence.

For an external platform, unfamiliar mechanism, new infrastructure/dependency, or non-trivial design choice, search current web and repository sources before design. Treat official docs, standards, maintainer source, and provider contracts as authority for current behavior. Use credible engineering articles and real implementations to find proven patterns, integration examples, operational constraints, and failure modes. Do not substitute model memory for current external evidence.

When a decision depends on Go, toolchain, standard-library, module, runtime, or
dependency behavior, establish the current repository Go version and consult
current authoritative sources for only that decision. Do not expand this into
a full Go review.

For a live solution choice, scan only the relevant rungs: existing repository or organization reuse, Go stdlib, native database/broker/platform/provider capability, already approved and operated infrastructure, managed service, mature maintained OSS, and custom implementation. Do not force every rung or a fixed candidate count. Compare surviving substitutes within a slot against the same decision drivers and context-matched operational evidence. Tutorials, examples, and reference architectures may demonstrate an integration shape, but do not alone establish production fit. For every surviving candidate, collect only applicable evidence that can change local approval: accountable owner, support, and on-call; availability, entitlement, region, quota, and SLA; compatibility, boundary fit, and provisioning; security guidance, advisories, unsafe defaults, and data custody; pricing unit and material cost at the accepted workload; lifecycle, deprecation, and upgrade policy; adoption and migration feasibility; and portability, exit, and failback limitations. For external code also require current evidence for maintenance/releases, license, security or vulnerability posture, API stability, transitive cost, domain adoption, and repository/boundary fit. If required evidence is unavailable, block or carry the exact proof gap.

For a material absence claim, record the authoritative surfaces and search boundary. For empirical or runtime evidence, record the version or configuration when relevant, method or command, environment, workload or sample, timestamp, result, and known limits. When a decision depends on a performance, capacity, reliability, cost, quality, or version-specific empirical claim, also establish a comparable current baseline and representative workload or data envelope, including only distributions that can change the result; do not infer causality from a single snapshot or profile. If authoritative sources cannot resolve such a claim and an authorized safe representative surface is available, run the smallest reversible or read-only probe that can discriminate the live implications; otherwise carry the exact blocker or later proof obligation. For freshness-sensitive evidence, record `valid as of` plus an objective refresh trigger or earliest downstream checkpoint. If persisted, retain only sanitized evidence or a safe pointer, never secrets or restricted data.

When research already identifies a concrete external input required by a downstream phase, apply the router's [implementation-input closure](../../spec-first-workflow.md#implementation-input-closure): record its owner, authoritative source, required shape, availability, and earliest required checkpoint without inventing later design inputs. When an implication requires proof, name the current observable or proving surface and setup availability/derivability, or the missing-proof owner; test design still owns scenario and proof-level selection.

Prefer primary/current sources. A missing hit is not proof of absence unless the searched source is authoritative for absence. For solution discovery, stop only when further searches by problem, decision force, failure mode, and known alias are unlikely to reveal a materially different viable candidate at the live decision level; source depth alone does not establish candidate-space saturation. Otherwise stop when another source is unlikely to change the decision.

## Review

Use focused self-review by default. Trigger independent read-only review of a fixed synthesis only when the user requests it or its conclusion is high-impact, hard to reverse, cross-owner, or weakly falsifiable. Ordinary supporting research is consumed inside specification without a duplicate gate.

The reviewer checks the Outputs against the Method: affected-lens and question coverage; open-item and quantitative provenance; source authority, applicability, and freshness; falsification; current-state semantic baseline and probe limits when triggered; candidate decision levels, relationships, local fit, sensitivity, and saturation when a solution choice is live; unresolved conflicts; fact/inference separation; and evidence-backed downstream dispositions, then returns:

- `PASS`: the sufficient material evidence boundary supports every decision-changing conclusion and has no unowned question or uncovered affected lens;
- `CONCERNS`: a bounded residual evidence risk or downstream proof obligation still needs explicit disposition and fresh review; it does not permit the reviewed synthesis to leave research and may not carry a missing answer owned by research;
- `FAIL`: missing, stale, conflicting, or unavailable evidence makes a required material conclusion unreliable or prevents closure.

The owning root repairs material findings and performs another proportionate check when the reviewed surface changes. An explicitly requested standalone review of research remains read-only.

## Stop Rule

Finish when every decision-changing question is supported, honestly bounded, or blocked with a reopen owner, and the next owner can distinguish established, uncertain, and freshness-sensitive material without repeating the same search or inventing missing evidence. A bounded gap required by the next phase blocks and reopens its smallest evidence or decision owner; an optional later proof may carry only with its owner and recheck condition. Resolve any triggered review before handoff; do not write the final specification or design decision inside research.
