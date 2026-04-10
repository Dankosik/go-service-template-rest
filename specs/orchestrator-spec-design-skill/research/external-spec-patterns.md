# External Spec Pattern Research

Retrieved: 2026-04-10

## Goal

Study popular, current specification/design-document workflows and extract the parts that are worth adapting into a repository-local orchestrator skill for designing `spec.md`.

## Sources Reviewed

### 1. GitHub Spec Kit

- Repo: <https://github.com/github/spec-kit>
- Popularity snapshot: 86,811 stars, 7,458 forks, updated 2026-04-10
- Files reviewed:
  - <https://github.com/github/spec-kit/blob/main/README.md>
  - <https://github.com/github/spec-kit/blob/main/templates/spec-template.md>
  - <https://github.com/github/spec-kit/blob/main/templates/plan-template.md>
  - <https://github.com/github/spec-kit/blob/main/templates/tasks-template.md>
  - <https://github.com/github/spec-kit/blob/main/spec-driven.md>

Key patterns:
- Treat `spec -> plan -> tasks` as an explicit artifact chain.
- Make user stories independently testable slices rather than a flat feature bucket.
- Separate product intent from implementation details in the spec.
- Require explicit edge cases, functional requirements, entities, success criteria, and assumptions.
- Make the task plan trace back to story boundaries and delivery order.

What is useful for us:
- strong spec-to-plan handoff discipline
- independently testable slices as a coverage lens
- measurable success criteria and explicit assumptions

What does not fit directly:
- its feature-spec template behaves more like a PRD than this repository's lean `spec.md`
- its richer artifact pack (`research.md`, `data-model.md`, `contracts/`, `quickstart.md`) would become a competing source of truth if copied wholesale

### 2. BMAD Method

- Repo: <https://github.com/bmad-code-org/BMAD-METHOD>
- Popularity snapshot: 44,202 stars, 5,245 forks, updated 2026-04-10
- Files reviewed:
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/README.md>
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/src/bmm-skills/2-plan-workflows/bmad-create-prd/workflow.md>
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/src/bmm-skills/2-plan-workflows/bmad-create-prd/steps-c/step-03-success.md>
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/src/bmm-skills/2-plan-workflows/bmad-create-prd/steps-c/step-04-journeys.md>
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/src/bmm-skills/2-plan-workflows/bmad-create-prd/steps-c/step-08-scoping.md>
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/src/bmm-skills/2-plan-workflows/bmad-create-prd/steps-c/step-09-functional.md>
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/src/bmm-skills/2-plan-workflows/bmad-create-prd/steps-c/step-10-nonfunctional.md>
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/src/bmm-skills/2-plan-workflows/bmad-create-prd/steps-c/step-11-polish.md>
  - <https://github.com/bmad-code-org/BMAD-METHOD/blob/main/src/bmm-skills/3-solutioning/bmad-create-architecture/steps/step-07-validation.md>

Key patterns:
- Build specs through guided discovery rather than one-pass templating.
- Split success into user, business, technical, and measurable outcomes.
- Use narrative user journeys to expose hidden capabilities and edge semantics.
- Keep MVP, growth, and vision explicitly separated to resist scope blur.
- Add only relevant non-functional requirement categories instead of filling every possible section.
- Run a polish/reconciliation pass that removes duplication and rescues soft ideas that rigid templates often lose.
- Validate architecture/spec readiness before handoff.

What is useful for us:
- discovery-first assembly
- selective NFR coverage
- scope ladder (`MVP / later / not now`)
- polish pass for contradictions, duplication, and lost soft constraints

What does not fit directly:
- the step-file workflow is too rigid for this repository's orchestrator-first process
- BMAD's PRD and architecture docs are fuller documents than our `spec.md` should usually be

### 3. Superpowers

- Repo: <https://github.com/obra/superpowers>
- Popularity snapshot: 145,094 stars, 12,430 forks, updated 2026-04-10
- Files reviewed:
  - <https://github.com/obra/superpowers/blob/main/README.md>
  - <https://github.com/obra/superpowers/blob/main/skills/brainstorming/SKILL.md>
  - <https://github.com/obra/superpowers/blob/main/skills/writing-plans/SKILL.md>
  - <https://github.com/obra/superpowers/blob/main/skills/brainstorming/spec-document-reviewer-prompt.md>
  - <https://github.com/obra/superpowers/blob/main/docs/superpowers/specs/2026-01-22-document-review-system-design.md>
  - <https://github.com/obra/superpowers/blob/main/docs/superpowers/specs/2026-03-23-codex-app-compatibility-design.md>

Key patterns:
- Never skip design/spec, even when the task looks simple.
- Explore context first, then compare 2-3 approaches, then write the design.
- Require user approval and a self-review pass before planning.
- Use human-readable design docs with motivation, empirical findings, concrete changes, scope summary, and future considerations.
- Run a spec reviewer that blocks only real planning hazards: placeholders, contradictions, ambiguity, scope spread, YAGNI drift.

What is useful for us:
- anti-placeholder review discipline
- lightweight but explicit design approval before planning
- requirement to compare alternatives when the design is still open
- human-readable spec prose instead of template sludge

What does not fit directly:
- its workflow is more interactive and user-loop-heavy than this repository usually wants in the main flow
- its implementation plans are much more prescriptive than our `plan.md` needs for every task

### 4. Spec-Driven Workflow

- Repo: <https://github.com/liatrio-labs/spec-driven-workflow>
- Popularity snapshot: 77 stars, 6 forks, updated 2026-04-09
- Files reviewed:
  - <https://github.com/liatrio-labs/spec-driven-workflow/blob/main/README.md>
  - <https://github.com/liatrio-labs/spec-driven-workflow/blob/main/prompts/SDD-1-generate-spec.md>
  - <https://github.com/liatrio-labs/spec-driven-workflow/blob/main/docs/specs/01-spec-ai-techniques-deep-dive-page/01-spec-ai-techniques-deep-dive-page.md>

Key patterns:
- Explicitly check whether clarification is sufficient before pretending a spec is ready.
- When relevant technologies materially shape design, research current external standards before finalizing the spec.
- Structure specs around goals, user stories, demoable units, proof artifacts, non-goals, repository standards, technical considerations, security considerations, success metrics, and open questions.
- Treat proof artifacts as part of the planning/validation chain, not just an afterthought.

What is useful for us:
- clarity gate before spec finalization
- latest-standards research when design depends on current external guidance
- demoable units and proof artifacts as validation lenses
- repository standards as a first-class section or coverage concern

What does not fit directly:
- its spec documents are fuller product specs than our canonical `spec.md`
- it assumes a numbered docs tree and audit/proofs artifacts that would duplicate our existing `workflow-plan.md`, `plan.md`, and validation flow if copied directly

## Cross-Source Synthesis

Across the sources, the stable best practices are:

1. A spec must encode user-visible behavior and acceptance semantics, not just technical intent.
2. The spec must be explicit about scope boundaries and what is intentionally not being done.
3. The spec should make planning possible without forcing the planner to rediscover core decisions.
4. The spec should be reviewable for placeholders, contradictions, and ambiguous sections before planning starts.
5. Validation expectations belong in the spec early, even if detailed test execution lives elsewhere.
6. A good spec keeps assumptions and open questions visible instead of inventing answers.
7. Multi-artifact workflows work best when each artifact has one job; problems start when the same decision is duplicated across PRD/spec/plan/tasks.

## Adaptation For This Repository

This repository should not import any of the external templates verbatim.

Instead, the new skill should translate the patterns into the existing artifact model:

- `spec.md` remains the canonical decisions artifact.
- `workflow-plan.md` remains the routing artifact for non-trivial work.
- `plan.md` remains the coder-facing execution artifact.
- `research/*.md` remains the place for preserved evidence and external pattern notes.

### External Pattern -> Repo Mapping

| External pattern | Keep? | Where it belongs here |
|---|---|---|
| independently testable user stories / slices | yes | summarized in `Decisions` and `Validation`; detailed execution stays in `plan.md` |
| edge cases | yes | `Constraints`, `Decisions`, or `Open Questions / Assumptions`, depending on whether they are settled |
| functional requirements | yes | translate into decision bullets or capability summaries, not a giant PRD chapter by default |
| key entities / domain objects | yes when material | `Context` or `Decisions` |
| measurable success criteria | yes | `Validation` and sometimes `Outcome` |
| MVP / later / vision scope ladder | yes | `Scope / Non-goals` and `Decisions` |
| selective NFRs | yes | `Constraints` |
| proof artifacts / demoable units | yes | `Validation` plus a compact summary in `Decisions` |
| repository standards | yes | `Constraints` or `Context` |
| detailed task breakdown | no in `spec.md` | `plan.md` only |
| raw research narratives | no in `Decisions` | `research/*.md` only |
| step-file choreography | no | convert into concise working rules inside the skill |

## Resulting Design Implication

The new skill should teach the orchestrator to do three things well:

1. Choose the right `spec.md` depth for the task without abandoning the repository's default section set.
2. Pull in the useful coverage prompts from external frameworks without copying their artifact sprawl.
3. Normalize bloated, under-specified, or research-dumped drafts into a planning-ready `spec.md`.
