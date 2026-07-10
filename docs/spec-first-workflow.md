# Spec-First Workflow

Stable router for repository work. `AGENTS.md` owns global policy; this file explains how to choose and load only the workflow detail that can change the current result.

## Principles

- Start from the desired outcome, constraints, success criteria, and stop conditions.
- Use the smallest path and artifact set that preserves correctness and proof.
- Treat phases as decision boundaries, not mandatory ceremonies or separate chats.
- State a rule once. Link to its owner instead of paraphrasing it elsewhere.
- Persist information only when another phase, actor, or session needs it.
- Let risk determine review and validation depth.

## Choose A Path

Use `direct` only when all of these are true:

- the request is clear enough to act on;
- the change is small, reversible, and has one obvious owner;
- no unresolved contract, data, security, money, concurrency, delivery, or cross-service decision exists;
- validation is obvious and bounded;
- durable resume state and independent evidence are unnecessary.

Use `structured` for the normal non-trivial case. Create only the artifacts needed to carry decisions or execution.

Use `orchestrated` when coordination itself is a real problem: broad or multi-owner scope, hard-to-reverse decisions, conflicting evidence, explicit multi-agent work, or likely multi-session execution. Orchestrated work may still omit research, design, test-plan, or rollout artifacts when their questions are not present.

Re-evaluate the path only when evidence changes the risk, ownership, reversibility, or proof shape. A path is not a quality tier: all paths must satisfy the accepted outcome.

## Phase Router

| Need | Read | Outcome |
| --- | --- | --- |
| Clarify raw or interpretation-sensitive input. | [Intake](spec-first-workflow/phases/intake.md) | A routing-sufficient brief or one blocking question. |
| Resolve evidence that can change a decision. | [Research](spec-first-workflow/phases/research.md) | Supported findings, limits, conflicts, and decision implications. |
| Define behavior, invariants, scope, and proof expectations. | [Specification](spec-first-workflow/phases/specification.md) | A compact decision record, reviewed to the risk of the change. |
| Choose runtime/system behavior. | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) | Contracts, sources of truth, sequence, failure, data, and rollout decisions. |
| Choose Go ownership and placement. | [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md) | Package/file owners, dependency direction, cleanup, and test ownership. |
| Make non-obvious proof explicit. | [Test Design](spec-first-workflow/phases/test-design.md) | Risk scenarios, proof levels, observables, and commands. |
| Turn decisions into executable work. | [Planning](spec-first-workflow/phases/planning.md) | A small, dependency-ordered ledger with completion proof. |
| Change, review, validate, and close out. | [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) | Working changes and evidence-clamped completion. |

### Optional Review Methods

Review is an internal method of the artifact-owning phase, not a separate phase or automatic user handoff:

| Review need | Read | Outcome |
| --- | --- | --- |
| Falsify a completed spec. | [Specification Review](spec-first-workflow/phases/specification-review.md) | Findings and `PASS`, `CONCERNS`, or `FAIL` returned to specification. |
| Test design and ownership readiness. | [Technical Design Review](spec-first-workflow/phases/technical-design-review.md) | Findings and verdict returned to technical design. |
| Test whether a ledger is executable. | [Task Review / Readiness](spec-first-workflow/phases/task-review-readiness.md) | Findings and verdict returned to planning. |

Read [Artifact Model](spec-first-workflow/shared/artifact-model.md) only when deciding what to persist, how to resume, or which artifact owns current state. Read [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) only when delegation, independent review, resume, or a handoff is relevant. Read `docs/repo-architecture.md` before design that affects repository boundaries or generated-source ownership.

## Phase Movement

Move forward when the next phase can work without inventing a decision owned by the current phase. Reopen the smallest owner when that is not true.

A request that authorizes end-to-end implementation may continue through the needed phases in one session. Stop at a macro-phase boundary only when:

- the user explicitly named that boundary;
- a required user or external decision is missing;
- the next action requires new authority;
- current evidence shows that an earlier decision must change;
- the remaining work needs durable resume or coordination that has not yet been recorded.

Review, repair, and re-review stay with the artifact owner. When they are internal checkpoints of an active macro phase, the owning root runs them in the same session and emits no next-session prompt. An explicitly user-requested standalone review stays read-only and stops at that requested boundary. A fresh independent review is required when the work is hard to reverse, materially high-impact, ambiguous, explicitly review-gated, or when the author lacks a credible way to falsify their own result. Otherwise perform a focused self-review and validate the changed surface. [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md#handoff) owns the exact prompt gate.

## Artifact And Handoff Rules

- Do not create empty workflow files or copy full chat prompts into artifacts.
- Use one short `status: draft | ready | blocked | done` field when durable state is useful; prose explains the blocker or evidence.
- `spec.md` owns decisions. `tasks.md` owns implementation progress after it exists. `workflow-plan.md` owns only cross-session coordination.
- A handoff contains the next outcome, first artifact to read, material constraints, required proof, and blocker/reopen rule. Omit repository-wide instructions already recoverable from `AGENTS.md`.
- If the work is done, return evidence and remaining gaps; do not invent another phase or prompt.

## Prompt Maintenance

Current reference: OpenAI's [Using GPT-5.6](https://developers.openai.com/api/docs/guides/latest-model) and [Prompting guidance for GPT-5.6](https://developers.openai.com/api/docs/guides/prompt-guidance-gpt-5p6).

When changing this workflow:

1. preserve outcome, safety/permission boundaries, success criteria, proof, and true stop conditions;
2. remove repeated process instructions, examples that do not change behavior, and universal rules for judgment calls;
3. prefer decision rules over keyword matrices and mandatory templates;
4. change one instruction group at a time when representative evals exist;
5. run the compact [workflow behavior evals](spec-first-workflow-evals.md);
6. compare task success and evidence completeness before counting token or line reduction as a win.
