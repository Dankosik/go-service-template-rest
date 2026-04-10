## Context

The repository already has skills for idea refinement, engineering framing, pre-spec challenge, specialist design lanes, and planning. What it does not have is a dedicated orchestrator-facing skill for designing the repository-native `spec.md` itself: choosing the right section depth, translating research into the canonical artifact, and normalizing drafts that are either too thin, too PRD-like, or polluted with raw research.

The user explicitly asked for that missing skill and asked that its notion of "what a good spec looks like" be grounded in external best practices rather than local preference alone. Current research covered GitHub Spec Kit, BMAD Method, Superpowers, and Spec-Driven Workflow, then mapped the reusable patterns back onto this repository's `spec.md` + `plan.md` split.

## Scope / Non-goals

In scope:
- create one new canonical repository skill for orchestrator use while designing `spec.md`
- preserve the repository's current artifact model instead of importing a foreign one
- distill external best practices into a compact runtime skill plus one supporting reference
- add lightweight eval coverage
- update discoverability surfaces and runtime mirrors

Non-goals:
- replacing `idea-refine`, `spec-first-brainstorming`, `pre-spec-challenge`, or `go-design-spec`
- introducing a mandatory PRD, audit, contracts, or proof-artifact pack for every task
- rewriting `AGENTS.md` or `docs/spec-first-workflow.md`
- turning `spec.md` into a full task breakdown or a dumping ground for raw research

## Constraints

- `spec.md` stays the canonical decisions artifact and must retain the repository's default section model unless a specific task clearly benefits from merging sections.
- `plan.md` stays the coder-facing execution artifact for non-trivial work.
- `research/*.md` may hold preserved external evidence, but it must not replace the decisions recorded in `spec.md`.
- The new skill must be lean enough to trigger cleanly and avoid becoming a mini-workflow that duplicates existing skills.
- External patterns must be translated into this workflow, not copied verbatim.
- The skill must escalate back to framing or specialist design when the problem is still immature or contradictory.

## Decisions

1. Create a new canonical skill named `spec-document-designer`.
   - Path: `.agents/skills/spec-document-designer/`
   - Role: orchestrator-facing spec authoring and normalization, not domain-specialist design and not planning

2. The skill will sit between stabilized framing/synthesis and implementation planning.
   Use it when the orchestrator needs to:
   - draft a fresh repository-native `spec.md`
   - repair a bloated or under-specified draft
   - decide which coverage prompts matter for the current change
   - translate preserved research into planning-ready decisions

3. The skill will explicitly not own adjacent responsibilities.
   - If the request is still idea-shaped, it should hand off to `idea-refine`.
   - If the request still needs engineering framing, it should hand off to `spec-first-brainstorming`.
   - If cross-domain contradictions remain, it should hand off to `go-design-spec` or the relevant specialist `*-spec` skills.
   - If the spec is already stable and the next need is execution ordering, it should hand off to `planning-and-task-breakdown`.

4. The skill's artifact bundle will stay small.
   Include:
   - `SKILL.md`
   - `references/spec-patterns.md`
   - `evals/evals.json`
   Exclude:
   - scripts
   - extra design-note docs
   - separate templates that would compete with `AGENTS.md` or `docs/spec-first-workflow.md`

5. The skill will adopt, but translate, four external pattern families.
   - `spec-kit`:
     keep independently testable slices, explicit edge cases, measurable success criteria, assumptions, and strict `spec -> plan -> tasks` separation
   - `BMAD`:
     keep discovery-first assembly, scope ladder, selective NFRs, user journeys, and a polish pass
   - `superpowers`:
     keep explicit design-before-plan posture, lightweight alternatives comparison, and anti-placeholder self-review
   - `spec-driven-workflow`:
     keep clarification sufficiency, latest-standards research when relevant, demoable units, proof artifacts, repository standards, and validation-chain awareness

6. The skill will encode a repo-native mapping layer rather than a foreign template.
   Core translation rules:
   - user journeys, demoable units, and capability slices should influence `Decisions` and `Validation`
   - functional requirements should usually appear as compact decision bullets rather than a giant PRD section
   - NFRs should appear only when materially relevant, usually under `Constraints`
   - raw research must stay in `research/*.md`, with only the decision outcome summarized in `spec.md`
   - task detail belongs in `plan.md`, not in `spec.md`

7. The skill's quality bar will focus on planning readiness.
   A planning-ready `spec.md` must:
   - contain no placeholders, `TBD`, or decorative empty sections
   - keep final decisions separate from raw evidence
   - make open questions and assumptions explicit
   - preserve scope cuts and non-goals
   - surface validation expectations early enough that planning can derive checkpoints without reopening design by default

8. Discoverability will be updated where users already look for local skills.
   - `README.md`

## Open Questions / Assumptions

- Assumption: `spec-document-designer` is the right boundary instead of expanding `go-design-spec`, because the missing capability is spec-shape design and normalization rather than integrated domain decision-making.
- Assumption: one compact reference file is enough; the skill should not need a large template pack if the mapping rules are explicit.
- Open question: none that block implementation.

## Plan Summary / Link

Execution will follow [`plan.md`](plan.md).

Control summary:
1. Write the canonical skill bundle from the decided boundaries and mapping rules.
2. Update discoverability and sync runtime mirrors.
3. Run structural validation on the skill assets and mirrors.

## Validation

Executed:
- read the skill and reference file end-to-end for overlap drift with adjacent skills
- `python3 -m json.tool .agents/skills/spec-document-designer/evals/evals.json >/dev/null`
- `bash ./scripts/dev/sync-skills.sh`
- `bash ./scripts/dev/sync-skills.sh --check`
- `find .claude/skills/spec-document-designer .cursor/skills/spec-document-designer .gemini/skills/spec-document-designer .github/skills/spec-document-designer .opencode/skills/spec-document-designer -maxdepth 3 -type f | sort`
- `git diff --check`

## Outcome

Completed:
- added the canonical orchestrator-facing skill bundle under `.agents/skills/spec-document-designer/`
- added a compact reference that maps external spec patterns onto this repository's `spec.md`/`plan.md` model
- added eval prompts covering fresh spec assembly, draft normalization, and upstream-boundary escalation
- updated `README.md`
- synced the new skill into all runtime mirror directories

Residual risk:
- the skill is structurally validated and research-backed, but it has not yet been exercised in a live feature-spec authoring session against a real user request
