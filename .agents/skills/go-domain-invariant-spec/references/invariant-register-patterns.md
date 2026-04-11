# Invariant Register Patterns

## When To Load
Load this when a domain spec needs an invariant register, owner assignment, enforcement-point choices, or a review pass for hidden or descriptive-only business rules.

Use repository product/spec artifacts first. In this repo, good behavior anchors include `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/repo-architecture.md`, and specs such as `specs/agents-skills-source-of-truth/spec.md`. Use the external links only to calibrate invariant-register shape.

## Register Shape
Use a compact table when the rule set is small:

| Field | Purpose |
| --- | --- |
| `id` | Stable short ID used for traceability, such as `INV-SKILL-CANONICAL-001`. |
| `statement` | One falsifiable business rule. |
| `type` | `local_hard_invariant` or `cross_service_process_invariant`. |
| `owner` | Domain owner that has authority to keep the rule true. |
| `source_of_truth` | Artifact, entity, aggregate, service, or policy source. |
| `enforcement_point` | Where the domain decision is guarded, stated without picking low-level implementation too early. |
| `observable_pass_fail` | How a reviewer or test can tell whether the rule held. |
| `violation_outcome` | Reject, deny, compensate, forward-recover, manual intervention, or accepted risk. |
| `downstream_handoff` | API, data, reliability, security, or test implications that follow after the domain decision. |

## Example Invariant Statements
- `SkillCanonicalSource`: `.agents/skills/<skill>` is the only canonical authoring surface for repository skills; runtime mirrors are derived surfaces and must not become competing sources of truth.
- `SpecDecisionAuthority`: for non-trivial task work, final approved decisions live in `spec.md`; `workflow-plan.md`, `design/`, `plan.md`, and `tasks.md` must not override that decision record.
- `SubagentReadOnly`: subagents may research and review but must not edit repository files, mutate git state, or change the implementation plan.
- `GeneratedContractAuthority`: generated OpenAPI bindings are derived from `api/openapi/service.yaml`; generated code cannot become the authoritative contract source.
- `ImplementationReadinessGate`: implementation may start only after the planning exit gate is `PASS`, accepted `CONCERNS`, or eligible `WAIVED`; `FAIL` routes to the named earlier phase.

Good invariant statement:

```text
INV-SKILL-CANONICAL-001
Statement: A repository skill change is authoritative only when it is made under `.agents/skills`; mirrored runtime copies are derived from that canonical source.
Type: local_hard_invariant
Owner: repository skill source-of-truth policy
Source of truth: specs/agents-skills-source-of-truth/spec.md
Enforcement point: skill sync/check workflow and reviewer discipline
Observable pass/fail: no active tooling or docs treat top-level `skills/` as canonical
Violation outcome: reject the change or repair the mirror/source-of-truth drift before validation closes
Downstream handoff: tooling docs and sync validation must point at `.agents/skills`
```

Bad invariant statement:

```text
Keep skills organized and up to date.
```

Why it fails: it has no owner, no boundary, no pass/fail signal, and no defined violation outcome.

## Good And Bad State Transition Specs
Good transition spec for a source-of-truth change:

```text
State: canonical_skill_change
Trigger: user asks to modify a repo-local skill
Preconditions:
- target path is under `.agents/skills`
- frontmatter identity requirements are known
Allowed transitions:
- canonical_skill_change -> mirrors_refreshed after sync updates derived runtime copies
- canonical_skill_change -> validation_blocked when sync/check output shows drift
Forbidden transitions:
- canonical_skill_change -> done while docs or tooling still point at a removed canonical path
Violation outcome: block closeout and repair the source-of-truth mismatch
```

Bad transition spec:

```text
Update skill files and sync if necessary.
```

Why it fails: "if necessary" hides who decides necessity and what state is legal next.

## Edge-Case Prompts
- What business rule would become false if two sources both claim to be canonical?
- Can this rule be tested without reading implementation code?
- Is the owner a business/domain owner, or just the first component that happens to notice the violation?
- Does the invariant need synchronous enforcement, eventual correction, or explicit accepted risk?
- What happens during mixed-version rollout or partial mirror refresh?
- If a duplicate command repeats the same logical operation, does the invariant still hold?
- If a read projection is stale, is it allowed to drive this decision?

## Downstream Handoff Notes
- API handoff: expose only the externally observable behavior that follows from the invariant; do not invent status codes until the rejection or acceptance semantics are stable.
- Data handoff: identify source-of-truth ownership and DB-enforceable constraints after deciding the invariant boundary.
- Reliability handoff: identify retry, timeout, replay, and reconciliation obligations for invariants that cross async or dependency boundaries.
- Security handoff: tenant, identity, and authorization rules that affect correctness should stay in the invariant register, then hand off fail-closed expectations.
- QA handoff: every critical invariant needs at least one positive, one negative, and one edge-case proof.

## Exa Source Links
- [Domain-Driven Design Reference](https://www.domainlanguage.com/wp-content/uploads/2016/05/DDD_Reference_2015-03.pdf) for bounded context, aggregate, and invariant ownership vocabulary.
- [ddd-crew Aggregate Design Canvas](https://github.com/ddd-crew/aggregate-design-canvas) for documenting aggregate responsibilities, invariants, and corrective policies.
- [Cosmic Python: Aggregates and Consistency Boundaries](http://www.cosmicpython.com/book/chapter_07_aggregate.html) for invariants as conditions that must hold after operations.
