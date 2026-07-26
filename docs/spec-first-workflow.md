# Spec-First Workflow

Stable router for repository work. `AGENTS.md` owns authorization and global invariants; this file owns execution-path selection, phase order, review routing, and movement between phases. Closure is the global execution model: choose the smallest sufficient path, close the current owner's decisions and the next owner's inputs before movement, and reopen only the smallest owner when evidence breaks that closure.

## Choose A Path

Choose the smallest path that can close the accepted outcome; add workflow only when risk, durable decisions, independent evidence, or coordination requires it.

Use `direct` only when all of these are true:

- the request is clear enough to act on;
- the change is small, reversible, and has one obvious owner;
- no unresolved contract, data, security, money, performance, concurrency, delivery, or cross-service decision exists;
- validation is obvious and bounded;
- durable resume state and independent evidence are unnecessary.

Direct work that satisfies these conditions enters the [implementation phase](spec-first-workflow/phases/implementation-validation-closeout.md#local-execution) with one accepted outcome. That phase owns root-local execution, root review, and bounded validation.

Use `structured` for the normal non-trivial case. Keep only the `spec.md`, `tasks.md`, design, or test artifacts whose decisions must survive; root self-review is sufficient unless the independent-review trigger below applies.

Use `orchestrated` when coordination itself is a real problem: broad or multi-owner scope, hard-to-reverse decisions, conflicting evidence, explicit multi-agent work, or likely multi-session execution. Orchestrated work may still omit research, design, test-plan, or rollout artifacts when their questions are not present.

Re-evaluate the path only when evidence changes risk, ownership, reversibility, or proof. A path is not a quality tier.

### Required Spine

Structured and orchestrated work evaluates the phase router in order:

1. establish the accepted outcome at intake;
2. resolve decision-changing evidence, or state why research is unnecessary;
3. complete specification and its path/risk-matched review;
4. complete system and Go-ownership design when implementation would otherwise choose mechanism or placement, then apply path/risk-matched review;
5. complete test design when proof is non-obvious, then apply path/risk-matched QA review;
6. complete `tasks.md` and its path/risk-matched readiness review;
7. enter [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) with one direct outcome or the next ready planned ledger task or wave; that phase owns direct root-local execution, default Worker execution for ready ledger tasks, root acceptance, integrated review, adaptation to execution drift, and validation.

Scoping down research, design, or test design needs one concrete reason in the current artifact or handoff, not a new phase-control file. Specification and planning remain required; independent review follows the shared trigger rather than artifact presence alone.

For review and handoff, the owning macro phases are specification (including any supporting intake and research), technical design (system/integration plus Go ownership), test design, planning, and implementation/validation/closeout. A user-named `research only` boundary makes research the owning macro phase; other supporting-step boundaries stop under their own stop rule without creating an extra review receipt.

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

Independent review, when triggered, is an internal method of the artifact-owning phase:

| Review need | Read | Outcome |
| --- | --- | --- |
| Falsify a standalone research synthesis. | [Research](spec-first-workflow/phases/research.md#review) | Evidence findings and verdict returned to research. |
| Falsify a completed spec. | [Specification Review](spec-first-workflow/phases/specification-review.md) | Findings and `PASS`, `CONCERNS`, or `FAIL` returned to specification. |
| Test technical design and ownership readiness. | [Technical Design Review](spec-first-workflow/phases/technical-design-review.md) | Findings and verdict returned to technical design. |
| Falsify non-obvious scenarios and proof feasibility. | [Test Design](spec-first-workflow/phases/test-design.md#review) | Independent QA findings and verdict returned to test design. |
| Test whether a ledger is executable. | [Task Review / Readiness](spec-first-workflow/phases/task-review-readiness.md) | Findings and verdict returned to planning. |
For every triggered non-implementation review, phase movement follows the shared [Review Independence](spec-first-workflow/shared/subagents-and-handoff.md#review-independence) rule. A dispositioned `CONCERNS` verdict may move; `FAIL` may not. Implementation acceptance and closeout follow the current root-owned review contract in the implementation phase.

Read [Artifact Model](spec-first-workflow/shared/artifact-model.md) only for persistence, status, ownership, or resume decisions. Read [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) only for delegation, independent review, resume, or handoff mechanics. Read `docs/repo-architecture.md` before design that affects repository boundaries or generated-source ownership.

## Phase Movement

Close before movement: move forward only when the current owner has dispositioned every triggered decision and the next phase can work from closed inputs without inventing meaning, mechanism, ownership, or proof strategy. When new evidence invalidates closure, reopen the smallest owner of the broken decision or input and preserve unaffected dispositions.

### Implementation-Input Closure

Before moving forward, close the inputs required by the next phase action or implementation task/wave: each is canonical, mechanically derivable without a semantic choice, or available from a named external owner. Also close any cross-task decision that could invalidate that next work. Later inputs may remain open with an owner and checkpoint when they cannot invalidate the next accepted result; they block only when the current task would otherwise be unusable or dishonest.

A request authorizing end-to-end implementation may continue through the needed phases and reviews in one session. Stop only when:

- the user explicitly named that boundary;
- a required external decision or input is unavailable from its named owner;
- the next action requires new authority;
- current evidence shows that an earlier decision must change;
- the remaining work needs durable resume or coordination that has not yet been recorded.

Absent one of those conditions, movement is automatic: enter the next phase, task, or wave, and report what it produced instead of asking whether to enter it ([Proceeding](../AGENTS.md#proceeding)).

Review, repair, and re-review of non-implementation artifacts stay with the artifact owner until the shared convergence condition is met. Implementation moves only under the [current phase-owned execution, acceptance, review, correction, and closeout contract](spec-first-workflow/phases/implementation-validation-closeout.md). [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) owns non-implementation review and handoff mechanics.

### Phase Lock

Planning readiness on the current `tasks.md` candidate commits the next transition to its first executable task or real parallel wave. Status checks and compaction resume from artifacts without changing the phase. Concrete new evidence that invalidates a named accepted input or readiness disposition reopens only its smallest owner and preserves every unaffected disposition.

At a true macro-phase boundary, follow [Handoff](spec-first-workflow/shared/subagents-and-handoff.md#handoff).

## Prompt Maintenance

Current references: for Codex sessions, OpenAI's [model and prompting guidance for GPT-5.6](https://developers.openai.com/api/docs/guides/latest-model?model=gpt-5.6#prompting-best-practices); for Claude Code sessions, Anthropic's [Claude Code documentation](https://code.claude.com/docs) and [prompt-engineering guidance](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/overview). [Agent Harness](agent-harness.md) owns which harness's native controls apply.

Use the repository [Task Contract](../AGENTS.md#task-contract) as the
outcome-first prompt schema. State each durable instruction once in its
narrowest owner and link to it elsewhere. Phrase it as an observable trigger,
action, completion criterion, or stop condition. Prefer the allowed behavior;
reserve prohibitions for safety, authorization, or a decisive exclusion. Avoid
broad tone or brevity labels; name the required content and what may be omitted.

Keep each policy compact and in one location. Restating an approval or
confirmation rule produces unnecessary approval requests, so link the owning
rule instead of repeating it. Hold every skill, subagent, and tool description
to the machine contract in [Skill Authoring](skill-authoring.md#invocation) —
leading word, distinct triggers, owned outcome, decisive exclusion — and expose
only material the current task can act on; a long session amplifies every
repeated prompt and tool description. Reasoning effort and response verbosity
are harness controls owned by [Agent Harness](agent-harness.md); set them there
instead of asking a model in prose to think harder or answer at length.

Change one instruction group at a time. Retain examples and style guidance only
when they encode a product requirement or close a measured gap, then review
realistic trigger, near-miss, and completion cases. This repository does not own
a fake agent runner or judge; without an externally owned live evaluation
system, invocation and model-behavior claims remain explicitly unproven.

Instruction edits prove only an instruction-level mitigation. Claim changed
model behavior only after an external live evaluation exercises the relevant
target model, harness, trigger, and completion case.
