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

Use `orchestrated` when coordination itself is a real problem: broad or multi-owner scope, hard-to-reverse decisions, conflicting evidence, explicit multi-agent work, dirty-checkout isolation, separate context, or likely multi-session execution. Orchestrated work may still omit research, design, test-plan, or rollout artifacts when their questions are not present.

A bounded read-only lane selected by the shared fan-out policy does not by
itself change a structured path into an orchestrated path; coordination must
still be a material problem.

Re-evaluate the path only when evidence changes risk, ownership, reversibility, or proof. A path is not a quality tier.

### Required Spine

Structured and orchestrated work evaluates the phase router in order:

1. establish the accepted outcome at intake;
2. resolve decision-changing evidence, or state why research is unnecessary;
3. complete specification and its path/risk-matched review;
4. complete system and Go-ownership design when implementation would otherwise choose mechanism or placement, then apply path/risk-matched review;
5. complete test design when proof is non-obvious, then apply path/risk-matched QA review;
6. complete `tasks.md` and its path/risk-matched readiness review;
7. enter [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) with one direct outcome or the next ready planned acceptance unit or wave; that phase owns execution-carrier selection, root-local execution, triggered Worker execution, root candidate intake, acceptance and integration, risk-triggered independent implementation review, adaptation to execution drift, and validation.

Before substantive work in each non-implementation macro phase, apply the
shared [Delegation Decision](spec-first-workflow/shared/subagents-and-handoff.md#delegation-decision).
Research and Technical Design use its lane-eligible fan-out as their default
execution shape; the other phases use it for eligible discovery, challenge, and
review questions.

Scoping down research, design, or test design needs one concrete reason in the current artifact or handoff, not a new phase-control file. Specification and planning remain required; independent review follows the shared trigger rather than artifact presence alone.

For review and handoff, the owning macro phases are specification (including any supporting intake and research), technical design (system/integration plus Go ownership), test design, planning, and implementation/validation/closeout. A user-named `research only` boundary makes research the owning macro phase; other supporting-step boundaries stop under their own stop rule without creating an extra review receipt.

## Phase Router

Read the matching phase before its first governed action. The link in this
table is a context pointer, not evidence that the target is already loaded.
Direct `change`, `build`, and `fix` requests start by reading Implementation /
Validation / Closeout even when their route is already obvious.

| Need | Read | Outcome |
| --- | --- | --- |
| Clarify raw or interpretation-sensitive input. | [Intake](spec-first-workflow/phases/intake.md) | A routing-sufficient brief or one blocking question. |
| Resolve evidence that can change a decision. | [Research](spec-first-workflow/phases/research.md) | Supported findings, limits, conflicts, and decision implications. |
| Synthesize accepted intent and evidence into falsifiable product and system behavior. | [Specification](spec-first-workflow/phases/specification.md) | A ready behavioral contract that design can realize without choosing product meaning. |
| Synthesize and select the smallest coherent target-state architecture that realizes the ready behavioral contract. | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) | One evidence-backed architecture with closed components, contracts and sources of truth, material flows, failure/recovery, and rollout. |
| Place the selected architecture in Go while preserving closed system behavior. | [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md) | Evidence-backed responsibility and package/file owners, dependency/composition direction, generated/manual authority, cleanup, and proof ownership. |
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
| Independently falsify a fixed high-risk implementation acceptance unit. | [Independent Implementation Review](spec-first-workflow/shared/implementation-review.md) | A one-shot `PASS`, `FAIL`, or `BLOCKED` verdict returned to root acceptance. |

Every independent-review decision follows the shared [Review Independence](spec-first-workflow/shared/review-independence.md) trigger. A dispositioned `CONCERNS` verdict may move for non-implementation artifacts; `FAIL` may not. [Implementation Review](spec-first-workflow/shared/implementation-review.md) owns its fixed-unit verdict and return to the root-owned acceptance contract; a `tasks.md` entry alone does not trigger it.

### Conditional Read Gate

Load a conditional owner immediately before the first action in its row and
keep it out of context when the trigger is absent:

| Trigger | Read before |
| --- | --- |
| Persist, inspect status/ownership, or resume from task artifacts. | [Artifact Model](spec-first-workflow/shared/artifact-model.md) |
| Enter a non-implementation macro phase, delegate, or open a non-implementation independent review. | [Subagents And Review](spec-first-workflow/shared/subagents-and-handoff.md) |
| Decide whether a fixed artifact or implementation acceptance unit requires independent review. | [Review Independence](spec-first-workflow/shared/review-independence.md) |
| Open a triggered independent implementation review of one fixed acceptance unit. | [Independent Implementation Review](spec-first-workflow/shared/implementation-review.md) |
| Resume after compaction or interruption, or cross a real actor or macro-phase boundary. | [Resume And Macro-Phase Handoff](spec-first-workflow/shared/resume-and-handoff.md) |
| Choose or operate a durable control, Worker/subagent carrier, model, or reasoning effort. | [Agent Harness](agent-harness.md) |
| Design changes repository boundaries or generated-source ownership. | [Repository Architecture](repo-architecture.md) |

Re-run this gate when phase movement or current evidence introduces a new row;
do not reload an unchanged owner merely to produce a routing receipt.

## Phase Movement

Close before movement: move forward only when the current owner has dispositioned every triggered decision and the next phase can work from closed inputs without inventing meaning, mechanism, ownership, or proof strategy. When new evidence invalidates closure, reopen the smallest owner of the broken decision or input and preserve unaffected dispositions.

### Implementation-Input Closure

Before moving forward, close the inputs required by the next phase action or implementation acceptance unit/wave: each is canonical, mechanically derivable without a semantic choice, or available from a named external owner. Also close any cross-task decision that could invalidate that next work. Later inputs may remain open with an owner and checkpoint when they cannot invalidate the next accepted result; they block only when the current unit would otherwise be unusable or dishonest.

A request authorizing end-to-end implementation may continue through the needed phases and reviews in one session. Stop only when:

- the user explicitly named that boundary;
- a required external decision or input is unavailable from its named owner;
- the next action requires new authority;
- current evidence shows that an earlier decision must change;
- the remaining work needs durable resume or coordination that has not yet been recorded.

Absent one of those conditions, movement is automatic: enter the next phase, task, or wave, and report what it produced instead of asking whether to enter it ([Proceeding](../AGENTS.md#proceeding)).

Review, repair, and re-review of non-implementation artifacts stay with the artifact owner until the shared convergence condition is met. Implementation moves only under the [current phase-owned execution, review, correction, acceptance, integration, and closeout contract](spec-first-workflow/phases/implementation-validation-closeout.md). [Review Independence](spec-first-workflow/shared/review-independence.md) owns the shared trigger, [Subagents And Review](spec-first-workflow/shared/subagents-and-handoff.md) owns non-implementation convergence, and [Independent Implementation Review](spec-first-workflow/shared/implementation-review.md) owns the fixed-unit review branch.

### Phase Lock

Planning readiness on the current `tasks.md` candidate commits the next transition to its first executable acceptance unit or real parallel wave. Status checks and compaction resume from artifacts without changing the phase. Concrete new evidence that invalidates a named accepted input or readiness disposition reopens only its smallest owner and preserves every unaffected disposition.

At a true macro-phase boundary, follow [Resume And Macro-Phase Handoff](spec-first-workflow/shared/resume-and-handoff.md#macro-phase-handoff).

## Prompt Maintenance

Current references: Matt Pocock's [Building Great Skills glossary](https://github.com/mattpocock/skills/blob/main/skills/productivity/writing-great-skills/GLOSSARY.md) owns the vocabulary for predictability, information hierarchy, steering, and pruning; OpenAI's [model and prompting guidance for GPT-5.6](https://developers.openai.com/api/docs/guides/latest-model?model=gpt-5.6#prompting-best-practices) owns current Codex model guidance; Anthropic's [Claude Code documentation](https://code.claude.com/docs) and [prompt-engineering guidance](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/overview) own the Claude Code vendor contract; Anthropic's account of [removing ~80% of the Claude Code system prompt for the Claude 5 generation](https://x.com/trq212/article/2080710971228918066) owns the rightsizing posture — constraint mass tuned for a prior model generation is re-derived against the current one, not inherited. [Agent Harness](agent-harness.md) owns which harness's native controls apply.

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

Change one instruction group at a time, and prefer removal: when a behavior
already has an owner, delete the weaker statement instead of adding a
clarifying one. Retain examples and style guidance only when they encode a
product requirement or close a measured gap, then review realistic trigger,
near-miss, and completion cases. Apply the disclosed [Workflow Behavior
Evals](spec-first-workflow/shared/workflow-behavior-evals.md) Run Contract to
every agent-instruction change and its matching phase or orchestration cases.
This repository owns the cases and trace assertions, not a fake agent runner or
judge; without an externally owned live evaluation system, invocation and
model-behavior claims remain explicitly unproven.

Instruction edits prove only an instruction-level mitigation. Claim changed
model behavior only after an external live evaluation exercises the relevant
target model, harness, trigger, and completion case. A new target model
generation reopens accepted no-op, constraint, and example decisions: re-run
the removal-first pass and its representative cases against the new generation
instead of carrying prior-generation constraint mass forward.

### Instruction Ownership

- Keep repository-wide rules in `AGENTS.md` and phase-specific method in
  `docs/spec-first-workflow/phases/`.
- Keep template-owned instruction paths free of repository-specific content:
  service names, module paths, deployment targets, owners, and
  service-specific invariants belong in the repository-owned records named by
  [Template Sync](template-sync.md). `template-owned.paths` mirrors its paths
  verbatim into derived repositories, so portability is part of their contract.
- [Agent Harness](agent-harness.md) owns harness detection and the mapping from
  workflow concepts to native Codex App and Claude Code controls: durable
  execution controls, workers, subagent lanes, model selection, and reasoning
  effort.
- An instruction that summarizes an external tool links the vendor contract
  beside the claim and requires reading it before reliance. Treat only
  documented behavior as authoritative; omitted vendor clauses remain evidence
  gaps.
- [Skill Authoring](skill-authoring.md) owns the lean behavioral-adapter
  contract.
- [Artifact Model](spec-first-workflow/shared/artifact-model.md) owns
  persistence; [Subagents And Review](spec-first-workflow/shared/subagents-and-handoff.md)
  owns built-in subagent delegation and non-implementation review convergence;
  [Review Independence](spec-first-workflow/shared/review-independence.md) owns
  the shared review trigger;
  [Independent Implementation Review](spec-first-workflow/shared/implementation-review.md)
  owns its fixed-unit branch; [Resume And Macro-Phase Handoff](spec-first-workflow/shared/resume-and-handoff.md)
  owns context rollover and chain of custody.
