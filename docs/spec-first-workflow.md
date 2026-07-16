# Spec-First Workflow

Stable router for repository work. `AGENTS.md` owns authorization and global invariants; this file owns execution-path selection, phase order, review routing, and movement between phases.

## Choose A Path

Use `direct` only when all of these are true:

- the request is clear enough to act on;
- the change is small, reversible, and has one obvious owner;
- no unresolved contract, data, security, money, concurrency, delivery, or cross-service decision exists;
- validation is obvious and bounded;
- durable resume state and independent evidence are unnecessary.

Direct work that satisfies these conditions enters the [implementation phase](spec-first-workflow/phases/implementation-validation-closeout.md#worker-assignment-and-acceptance) with one accepted outcome. That phase owns Worker execution, root review, and bounded validation.

Use `structured` for the normal non-trivial case. Keep a reviewed `spec.md` and reviewed `tasks.md`; create design and test artifacts only when their decisions must survive.

Use `orchestrated` when coordination itself is a real problem: broad or multi-owner scope, hard-to-reverse decisions, conflicting evidence, explicit multi-agent work, or likely multi-session execution. Orchestrated work may still omit research, design, test-plan, or rollout artifacts when their questions are not present.

Re-evaluate the path only when evidence changes risk, ownership, reversibility, or proof. A path is not a quality tier.

### Required Spine

Structured and orchestrated work evaluates the phase router in order:

1. establish the accepted outcome at intake;
2. resolve decision-changing evidence, or state why research is unnecessary;
3. complete specification and independent specification review;
4. complete system and Go-ownership design when implementation would otherwise choose mechanism or placement, then independently review the design;
5. complete test design when proof is non-obvious, then obtain independent QA review;
6. complete `tasks.md` and independent task review/readiness;
7. enter [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md#worker-assignment-and-acceptance) with one direct outcome or the next ready planned ledger wave; that phase owns Worker assignment, root acceptance, integrated review, adaptation to execution drift, and validation.

Scoping down research, design, or test design needs one concrete reason in the current artifact or handoff, not a new phase-control file. Specification, planning, and their review gates remain required.

For review and handoff, the owning macro phases are specification (including any supporting intake and research), technical design (system/integration plus Go ownership), test design, planning, and implementation/validation/closeout. A user-named `research only` boundary makes research the owning macro phase and requires independent synthesis review; other supporting-step boundaries stop under their own stop rule without creating an extra review receipt.

Before each required review of an applicable structured or orchestrated non-implementation owning macro phase, run one autonomous read-only challenge probe before the separate reviewer. The applicable boundaries are Specification, combined Technical Design, Test Design, Planning, and an explicit `research only` boundary; run it once after the whole candidate meets its authoring bar, with Technical Design waiting for both system/integration and Go ownership. Direct work, supporting steps, and Implementation/Validation/Closeout do not run this probe. A closed candidate may return `DONE` immediately. [Autonomous Pre-Review Challenge](spec-first-workflow/shared/autonomous-pre-review-challenge.md) owns the protocol.

## Phase Router

| Need | Read | Outcome |
| --- | --- | --- |
| Clarify raw or interpretation-sensitive input. | [Intake](spec-first-workflow/phases/intake.md) | A routing-sufficient brief or one blocking question. |
| Resolve evidence that can change a decision. | [Research](spec-first-workflow/phases/research.md) | Supported findings, limits, conflicts, and decision implications. |
| Define behavior, invariants, scope, and proof expectations. | [Specification](spec-first-workflow/phases/specification.md) | A compact decision record, reviewed as required by path and risk. |
| Choose runtime/system behavior. | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) | Contracts, sources of truth, sequence, failure, data, and rollout decisions. |
| Choose Go ownership and placement. | [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md) | Package/file owners, dependency direction, cleanup, and test ownership. |
| Make non-obvious proof explicit. | [Test Design](spec-first-workflow/phases/test-design.md) | Risk scenarios, proof levels, observables, and commands. |
| Turn decisions into executable work. | [Planning](spec-first-workflow/phases/planning.md) | A small, dependency-ordered ledger with completion proof. |
| Change, review, validate, and close out. | [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) | Working changes and evidence-clamped completion. |

### Review Routing

Required review is an internal method of the artifact-owning phase:

| Review need | Read | Outcome |
| --- | --- | --- |
| Falsify a standalone research synthesis. | [Research](spec-first-workflow/phases/research.md#review) | Evidence findings and verdict returned to research. |
| Falsify a completed spec. | [Specification Review](spec-first-workflow/phases/specification-review.md) | Findings and `PASS`, `CONCERNS`, or `FAIL` returned to specification. |
| Test technical design and ownership readiness. | [Technical Design Review](spec-first-workflow/phases/technical-design-review.md) | Findings and verdict returned to technical design. |
| Falsify non-obvious scenarios and proof feasibility. | [Test Design](spec-first-workflow/phases/test-design.md#review) | Independent QA findings and verdict returned to test design. |
| Test whether a ledger is executable. | [Task Review / Readiness](spec-first-workflow/phases/task-review-readiness.md) | Findings and verdict returned to planning. |
For every required non-implementation review, phase movement waits for the [shared `PASS`-only convergence rule](spec-first-workflow/shared/subagents-and-handoff.md#review-independence); `CONCERNS` and `FAIL` stay inside repair, disposition, or reopen work. Implementation acceptance and closeout follow the root-owned review contract in the implementation phase.

Read [Artifact Model](spec-first-workflow/shared/artifact-model.md) only for persistence, status, ownership, or resume decisions. Read [Autonomous Pre-Review Challenge](spec-first-workflow/shared/autonomous-pre-review-challenge.md) only for that protocol. Read [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) only for delegation, independent review, resume, or handoff mechanics. Read `docs/repo-architecture.md` before design that affects repository boundaries or generated-source ownership.

## Phase Movement

Move forward when the next phase can work without inventing a decision owned by the current phase. Reopen the smallest owner when that is not true.

### Implementation-Input Closure

Before moving forward, every input required by the next phase must be exactly defined by a current canonical source, mechanically derivable from approved sources without a semantic choice, or recorded as an external input with its owner, authoritative source, required shape, and earliest required checkpoint. Prose promises and future fixtures without deterministic content do not count as closure. Before implementation, apply closure to every mandatory task and proof on every dependency path through the accepted completion, not only the first executable task. A known unavailable external input may remain only when its dependent task and protected claim are excluded from the current completion and routed to a later ledger; otherwise readiness is `FAIL` and reopens the smallest owner.

A request authorizing end-to-end implementation may continue through the needed phases and reviews in one session. Stop only when:

- the user explicitly named that boundary;
- a required user or external decision is missing;
- the next action requires new authority;
- current evidence shows that an earlier decision must change;
- the remaining work needs durable resume or coordination that has not yet been recorded.

Review, repair, and re-review of non-implementation artifacts stay with the artifact owner until the shared convergence condition is met. Implementation moves only under the [phase-owned assignment, acceptance, review, correction, and closeout contract](spec-first-workflow/phases/implementation-validation-closeout.md#worker-assignment-and-acceptance). [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) owns non-implementation review and handoff mechanics.

At a true macro-phase boundary, follow [Handoff](spec-first-workflow/shared/subagents-and-handoff.md#handoff).

## Prompt Maintenance

Current reference: OpenAI's [Using GPT-5.6](https://developers.openai.com/api/docs/guides/latest-model) and [Prompting guidance for GPT-5.6](https://developers.openai.com/api/docs/guides/prompt-guidance-gpt-5p6).

Preserve outcome, permission boundaries, success criteria, proof, and stop conditions. Change one instruction group at a time, run the [workflow behavior evals](spec-first-workflow-evals.md), and keep reductions only when task success and evidence completeness do not regress.
