# Template Readiness Review And Improvement Planning Workflow Plan

## Frame

- Goal: produce a subagent-backed architecture and maintainability review of whether this repository is ready as a production-business-code template.
- Central lens: can a team clone the template and immediately understand where new business functionality belongs, which style and boundaries to follow, and how to avoid fighting the folder structure or duplicating utilities.
- Scope: repository structure, durable docs, bootstrap, app/domain/infra/config/API/data/test surfaces, generated-code boundaries, extension paths, helper/source-of-truth clarity, and documentation examples.
- Non-goals: implementation, broad rewrite design, generic Go style review, changing generated code, or overfitting recommendations to the sample `ping` surface.
- Constraints: subagents are read-only and advisory; one skill at most per subagent pass; final synthesis belongs to the orchestrator; concrete findings need path references and fresh repository evidence.
- Follow-up goal: turn the completed advisory review into a preimplementation decision/design/planning packet so a later session can implement without rediscovering context.
- Follow-up non-goals: no code edits in this session, no fake business domain, no placeholder auth, no generated-code hand edits, and no migration/sqlc sample rename without an explicit maintainer decision.

## Routing

- Review execution shape: full orchestrated.
- Review rationale: the user explicitly requested subagents, and the review crossed architecture, API transport, data, maintainability, and QA/test boundaries.
- Follow-up planning execution shape: lightweight local, with a pre-code phase-collapse waiver.
- Follow-up waiver rationale: this session is artifact-only, implementation is explicitly deferred to a later session, and the required cross-domain research/review plus synthesis challenge already completed in `workflow-plans/review.md`; running another subagent wave would add more ceremony than decision quality.
- Current phase: done.
- Current phase plan: `workflow-plans/validation-phase-1.md`.
- Phase status: complete.
- Session boundary reached: yes.
- Ready for next session: no; Phase 1 implementation and validation are complete.
- Next session starts with: none unless the maintainer requests a dedicated post-code review or a deferred follow-up.

## Artifact Status

- `workflow-plan.md`: active, updated for completed review plus completed follow-up planning.
- `workflow-plans/review.md`: complete.
- `workflow-plans/planning.md`: complete.
- `workflow-plans/implementation-phase-1.md`: complete.
- `workflow-plans/validation-phase-1.md`: complete.
- `spec.md`: approved under the lightweight local phase-collapse waiver; outcome recorded.
- `design/`: approved under the lightweight local phase-collapse waiver; core artifacts created.
- `plan.md`: approved, with implementation-readiness status `CONCERNS`.
- `tasks.md`: complete for Phase 1.
- `test-plan.md`: not expected; validation obligations are small enough for `plan.md` and `tasks.md`.
- `rollout.md`: not expected; no runtime rollout is being planned.
- `research/*.md`: not expected; the review evidence was preserved through subagent notifications and the final review report, and decisions are now captured in `spec.md`.

## Review Lanes

Planned review lanes, all read-only. Architecture, maintainability, API transport, data, and QA run in initial parallel fan-out; synthesis-challenge runs after initial synthesis:

| Lane | Role | One skill | Owned question |
| --- | --- | --- | --- |
| workflow-adequacy | `challenger-agent` | `workflow-plan-adequacy-challenge` | Are these workflow-control artifacts sufficient for this agent-backed review before fan-out? |
| architecture | `architecture-agent` | `go-design-review` | Do package boundaries, ownership rules, and extension seams make new business use cases easy to place? |
| maintainability | `quality-agent` | `go-language-simplifier-review` | Is code style easy to imitate, and are helper/source-of-truth patterns clear rather than duplicated or ambiguous? |
| api-transport | `api-agent` | `go-chi-review` | Are OpenAPI/generated/chi/http boundaries clear for adding endpoints without breaking generated-code rules? |
| data | `data-agent` | `go-db-cache-review` | Are Postgres, SQLC, migrations, and repository placement clear for adding persisted business behavior? |
| qa | `qa-agent` | `go-qa-review` | Is the test layout obvious for new business logic, HTTP contracts, adapters, and integration coverage? |
| synthesis-challenge | `challenger-agent` | `pre-spec-challenge` | After initial synthesis, what template-readiness risks or missing seams could undermine the recommendations? |

## Gate Status

- Workflow plan adequacy challenge: complete; no blocking findings, two recordable wording findings reconciled in this artifact.
- Fan-out status: complete; architecture, maintainability, API transport, data, QA, and synthesis-challenge lanes returned.
- Fan-in and synthesis status: complete; challenger findings reconciled by merging overlapping first-feature findings, softening the auth recommendation, and downgrading helper cleanup to a lower-priority opportunity.
- Validation evidence: `go test -count=1 ./...` passed for non-integration packages; static inspection used `rg --files`, `find`, `go list ./...`, and focused file reads.
- Spec clarification gate: waived by the lightweight local phase-collapse rationale above; the completed review already included a synthesis-challenge lane and no new product/business-policy answer is being invented here.
- Workflow plan adequacy challenge for planning handoff: waived by the same lightweight local rationale; next-session routing and task ledger are explicit in the created artifacts.
- Implementation readiness: `CONCERNS`; implementation proceeded only within the selected non-breaking guidance/guardrail scope and proof obligations in `plan.md`.
- Implementation phase 1: complete; all ledger tasks T001-T011 are done.
- Validation phase 1: complete; `make guardrails-check`, `go test -count=1 ./cmd/service/internal/bootstrap ./internal/infra/http`, and `go test -count=1 ./...` passed. OpenAPI and SQLC checks were not run because no matching source-of-truth or generated surfaces changed.

## Blockers And Assumptions

- Blockers: none.
- Assumption: the follow-up planning pass was artifact-only; implementation was deferred to this implementation session.
- Assumption: the completed review evidence is sufficient to plan a bounded non-breaking hardening pass without another subagent wave.
- Accepted concern: this plan does not add a runnable fake business domain; it documents the first production-shaped path instead.
- Accepted concern: this plan does not rename `ping_history` schema/generated sample surfaces; that remains a maintainer decision if stronger fixture isolation is desired.
- Accepted concern: this plan documents the protected-operation seam but does not design or implement auth policy.

## Completion Marker

The review phase was complete when:

- workflow adequacy findings are reconciled or explicitly recorded as non-blocking,
- initial review lanes have returned or a required lane is explicitly marked unavailable with a limitation,
- challenger pressure-test findings are reconciled,
- the orchestrator has inspected enough repository evidence to cite concrete paths,
- the final report separates issues, opportunities, preserved strengths, suggested placement guidance, and open maintainer questions.

Completion status: complete. The final review report is returned in chat; no code changes were made beyond these required workflow-control artifacts.

The follow-up planning phase is complete when:

- `spec.md`, core `design/`, `plan.md`, and `tasks.md` exist,
- implementation readiness is explicit,
- next-session implementation and validation phase-control files exist,
- implementation remains deferred.

Follow-up planning status: complete.

The Phase 1 implementation and validation closeout are complete when:

- T001-T012 are checked in `tasks.md`,
- the implementation and validation phase-control files record completion,
- required validation evidence is fresh and passing,
- no out-of-scope generated-code, auth, fake-domain, Redis/Mongo adapter, or SQLC fixture rename was introduced.

Phase 1 implementation and validation status: complete.
