# Research Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this for research, evidence fan-out, dependency/OSS diligence, or Pattern Fit research.

## Read When

- The active phase is research or evidence fan-out.
- Dependency/OSS due diligence or Pattern Fit evidence could materially change specification readiness.
- A local-only research rationale must be recorded or checked.

## Inputs

- Router and artifact-shape decision from `docs/spec-first-workflow.md` and `shared/artifact-model.md`.
- Current `workflow-plan.md` or `workflow-plans/research.md` when a dedicated research phase exists.
- The concrete evidence questions, candidate lanes, and source targets.

## Outputs

- Evidence questions with answer status, source limits, and specification impact.
- Preserved `research/*.md` notes only when they help later synthesis, auditability, or resume.
- Dependency/OSS and Pattern Fit evidence when they could change approval.
- Conflict, weak-evidence, and assumption register with the owning reopen or proof path.
- Fan-in implication for specification with destination classification, or a local-only rationale with reopen seams.

## Stop Rule

Finish research with evidence, limits, conflicts, assumptions, and handoff implications. Do not write review-ready `spec.md`, design artifacts, `tasks.md`, or implementation output in this phase.

## Research

Research is a concern, not always a dedicated phase.

Start by turning the uncertainty into concrete evidence questions. Each question should name:

- the decision it can change;
- the source target or lane that can answer it;
- the minimum evidence needed for a useful answer;
- the freshness or authority requirement when source age matters;
- the handoff if the answer is missing, conflicting, or only partly proven;
- the expected `spec.md` destination: decision, constraint, assumption, risk, proof obligation, or blocker.

Use local-only research for direct path or when a recorded local-only rationale shows the evidence is trivial, single-source, or not improved by independent lanes.

Dependency/OSS due diligence is a research concern even when it stays compact. Use local research for obvious stdlib or established-repo-pattern choices; use read-only research fan-out when the selected library or custom implementation decision depends on current external health, license/security posture, domain adoption, or integration trade-offs that could materially change approval.

Good Dependency/OSS due diligence answers the contract first, then compares current Go stdlib, established repository patterns, and mature OSS candidates against that contract. Record selected and rejected options with source date or version, maintenance and release signal, license, security or vulnerability signal, API stability, transitive dependency cost, adoption signal appropriate to the domain, integration fit, and why custom code is lower or higher ownership cost. If the answer is custom code, explicitly say why stdlib, repository patterns, and mature OSS do not satisfy the accepted contract. Do not treat popularity, stale snippets, or a single blog post as enough evidence for approval. If current external evidence cannot be checked, record the limit and hand off a proof obligation instead of presenting the decision as ready.

Pattern Fit Diligence is also a research concern when the task has a real design fork. Search for concrete descriptions and examples of relevant patterns, including architecture, integration, consistency, workflow, resilience, data-topology, and Go-friendly implementation patterns. Preserve `research/pattern-fit.md` when the pattern evidence, examples, or candidate comparison would otherwise be lost across sessions; final pattern decisions still belong in `spec.md` or the design bundle.

Good Pattern Fit research names the task forces, compares viable patterns against repository boundaries, source-of-truth ownership, failure behavior, operational proof path, and idiomatic Go fit, then explains why the selected pattern fits now or why the straightforward repo-native design is better. A named pattern is incomplete without rejected-pattern comparison for viable alternatives. Reject pattern candidates explicitly when they add vocabulary, indirection, class-oriented scaffolding, or distributed-systems machinery without solving a current force.

For non-trivial decisions, first identify distinct evidence questions and normally use read-only fan-out when the questions span more than one domain, artifact family, source-of-truth seam, or risk lens.

Any local-only rationale must list the decision frontier, candidate lanes or lenses considered, evidence checked for each, why each omitted lane cannot change approval or readiness, what evidence would change that conclusion, and the seam that would reopen fan-out. Generic "bounded" or "single-domain" rationale is invalid for non-trivial phase approval.

Preserve `research/*.md` only when it materially helps later synthesis, auditability, or resume. Preserve the note when evidence is multi-source, externally time-sensitive, conflict-bearing, too dense for `spec.md`, needed to justify dependency/OSS or Pattern Fit decisions, or likely to be re-read by specification, design, review, planning, or validation. Keep the result compact when a single stable repository read or obvious stdlib answer can be recorded directly in `spec.md`.

A good research note includes:

- question or scope;
- findings with evidence, source date or version when relevant, and limits;
- source notes that identify only the sources that changed or materially constrained the answer;
- conflicts, weak evidence, or assumptions;
- handoff implication for specification with destination classification.

Do not use research notes as source dumps. For each source, keep the shortest useful pointer plus the fact, limit, or contradiction it contributes, and the decision or handoff implication it changes. A source note without that conclusion is not evidence. Stop adding sources when the next source cannot change the decision, and record the source limit when it affects confidence.

When sources or lanes conflict, do not silently average them. Record the conflict, the strongest evidence on each side, the decision that remains blocked or risky, and the owner of the next action: more research, specification decision, specialist lane, accepted risk, or proof obligation. Classify unresolved points as `blocks_spec`, `proof_only`, `accepted_risk`, or `needs_specialist` so specification does not confuse decision blockers with later validation work.

End research with a specification handoff, even when the result is local-only. The handoff should say:

- which findings are ready to become `spec.md` decisions or constraints;
- which findings are assumptions, accepted risks, or proof obligations;
- which decisions remain blocked and the smallest reopen target;
- which unresolved points are `blocks_spec`, `proof_only`, `accepted_risk`, or `needs_specialist`;
- whether Dependency/OSS due diligence and Pattern Fit Diligence are complete, not applicable, scoped down, or blocked;
- which preserved `research/*.md` files the specification phase must read, and why.

For dense fan-in, use compact rows that carry the question, evidence summary, limits or conflicts, specification destination, next owner or action, and preserved note path when one exists.

Research notes support decisions but do not own them. Final decisions belong in `spec.md`.
