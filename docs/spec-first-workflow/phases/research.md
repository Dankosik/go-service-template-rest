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

- Preserved `research/*.md` notes only when they help later synthesis, auditability, or resume.
- Fan-in implication for specification, or a local-only rationale with reopen seams.

## Stop Rule

Finish research with evidence, limits, conflicts, assumptions, and handoff implications. Do not write review-ready `spec.md`, design artifacts, `tasks.md`, or implementation output in this phase.

## Research

Research is a concern, not always a dedicated phase.

Use local-only research for direct path or when a recorded local-only rationale shows the evidence is trivial, single-source, or not improved by independent lanes.

Dependency/OSS due diligence is a research concern even when it stays compact. Use local research for obvious stdlib or established-repo-pattern choices; use read-only research fan-out when the selected library or custom implementation decision depends on current external health, license/security posture, domain adoption, or integration trade-offs that could materially change approval.

Pattern Fit Diligence is also a research concern when the task has a real design fork. Search for concrete descriptions and examples of relevant patterns, including architecture, integration, consistency, workflow, resilience, data-topology, and Go-friendly implementation patterns. Preserve `research/pattern-fit.md` when the pattern evidence, examples, or candidate comparison would otherwise be lost across sessions; final pattern decisions still belong in `spec.md` or the design bundle.

For non-trivial decisions, first identify distinct evidence questions and normally use read-only fan-out when the questions span more than one domain, artifact family, source-of-truth seam, or risk lens.

Any local-only rationale must list the decision frontier, candidate lanes or lenses considered, evidence checked for each, why each omitted lane cannot change approval or readiness, and the seam that would reopen fan-out. Generic "bounded" or "single-domain" rationale is invalid for non-trivial phase approval.

Preserve `research/*.md` only when it materially helps later synthesis, auditability, or resume. A good research note includes:

- question or scope;
- findings with evidence and limits;
- source notes;
- conflicts, weak evidence, or assumptions;
- handoff implication.

Research notes support decisions but do not own them. Final decisions belong in `spec.md`.
