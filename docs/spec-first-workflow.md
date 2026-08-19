# Spec-First Workflow

Stable router for repository work. `AGENTS.md` owns authorization and global invariants; this file owns execution-path selection, phase order, review routing, and movement between phases. Closure is the global execution model: choose the smallest sufficient path, close the current owner's decisions and the next owner's inputs before movement, and reopen only the smallest owner when evidence breaks that closure.

## Choose A Path

Choose the smallest path that can close the accepted outcome; add workflow only when risk, durable decisions, independent evidence, or coordination requires it.

Root [`Direct Work`](../AGENTS.md#direct-work) owns the local path and its
conditions. When it applies, return to that contract; this router owns only the
wider paths below.

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
6. complete Planning and its path/risk-matched readiness review, persisting
   `tasks.md` only when multiple units, dependencies, waves, an actor/session
   boundary, or durable resume state requires it;
7. enter [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) with one direct outcome, one fixed inline structured unit, or the next ready ledger unit or wave; that phase owns execution, review, validation, acceptance, and closeout.

Non-implementation work stays root-local unless a concrete question has an
independently checkable evidence boundary and separate context or independence
can materially improve its answer. Load [Subagents And
Review](spec-first-workflow/shared/subagents-and-handoff.md) only before the
first such dispatch or a triggered non-implementation independent review.

Scoping down research, design, or test design needs one concrete reason in the current artifact or handoff, not a new phase-control file. Specification and planning remain required; independent review follows the shared trigger rather than artifact presence alone.

For review and handoff, the owning macro phases are specification (including any supporting intake and research), technical design (system/integration plus Go ownership), test design, planning, and implementation/validation/closeout. A user-named `research only` boundary makes research the owning macro phase; other supporting-step boundaries stop under their own stop rule without creating an extra review receipt.

One root session owns at most one active macro phase. Supporting work, review,
repair, and the smallest upstream reopen stay inside it. Report and hand off
only after the phase and every triggered review close; an end-to-end request
does not waive this boundary. [Resume And Macro-Phase
Handoff](spec-first-workflow/shared/resume-and-handoff.md#macro-phase-handoff)
owns the emitted prompt and chain of custody.

## Phase Router

Read the matching phase before its first governed action. The link in this
table is a context pointer, not evidence that the target is already loaded.

| Need | Read | Outcome |
| --- | --- | --- |
| Clarify raw or interpretation-sensitive input. | [Intake](spec-first-workflow/phases/intake.md) | A routing-sufficient brief or one blocking question. |
| Resolve evidence that can change a decision. | [Research](spec-first-workflow/phases/research.md) | Supported findings, limits, conflicts, and decision implications. |
| Synthesize accepted intent and evidence into falsifiable product and system behavior. | [Specification](spec-first-workflow/phases/specification.md) | A ready behavioral contract that design can realize without choosing product meaning. |
| Synthesize and select the smallest coherent target-state architecture that realizes the ready behavioral contract. | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) | One evidence-backed architecture with closed components, contracts and sources of truth, material flows, failure/recovery, and rollout. |
| Place the selected architecture in Go while preserving closed system behavior. | [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md) | Evidence-backed responsibility and package/file owners, dependency/composition direction, generated/manual authority, cleanup, and proof ownership. |
| Make non-obvious proof explicit. | [Test Design](spec-first-workflow/phases/test-design.md) | Risk scenarios, proof levels, observables, and commands. |
| Turn decisions into executable work. | [Planning](spec-first-workflow/phases/planning.md) | One fixed executable unit or a small dependency-ordered ledger with completion proof. |
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
| Independently falsify a fixed high-risk implementation acceptance unit. | [Independent Implementation Review](spec-first-workflow/shared/implementation-review.md) | A one-shot `PASS`, `FAIL`, or `NEEDS_PARENT` verdict returned to the acceptance owner. |

Every independent-review decision follows the shared [Review Independence](spec-first-workflow/shared/review-independence.md) trigger. A dispositioned `CONCERNS` verdict may move for non-implementation artifacts; `FAIL` may not. [Implementation Review](spec-first-workflow/shared/implementation-review.md) owns its fixed-unit verdict and return to the acceptance owner; a `tasks.md` entry alone does not trigger it.

### Conditional Read Gate

Load a conditional owner immediately before the first action in its row and
keep it out of context when the trigger is absent:

| Trigger | Read before |
| --- | --- |
| Persist, inspect status/ownership, or resume from task artifacts. | [Artifact Model](spec-first-workflow/shared/artifact-model.md) |
| Dispatch a non-implementation read-only lane. | [Read-Only Lanes](spec-first-workflow/shared/subagents-and-handoff.md) |
| Open a triggered non-implementation independent review. | [Review Findings And Convergence](spec-first-workflow/shared/review-findings-and-convergence.md) |
| Decide whether a fixed artifact or implementation acceptance unit requires independent review. | [Review Independence](spec-first-workflow/shared/review-independence.md) |
| Open a triggered independent implementation review of one fixed acceptance unit. | [Independent Implementation Review](spec-first-workflow/shared/implementation-review.md) |
| Resume after compaction or interruption, or cross a real actor or macro-phase boundary. | [Resume And Macro-Phase Handoff](spec-first-workflow/shared/resume-and-handoff.md) |
| Enter or continue Implementation across a session, terminalize a known Lead, or route an agent-owned upstream reopen. | [Implementation Handoff](spec-first-workflow/shared/implementation-handoff.md) |
| Choose or operate a durable control, Worker/subagent carrier, model, or reasoning effort. | [Agent Harness](agent-harness.md) |

Re-run this gate when phase movement or current evidence introduces a new row;
do not reload an unchanged owner merely to produce a routing receipt.

## Phase Movement

Close before movement: move forward only when the current owner has dispositioned every triggered decision and the next phase can work from closed inputs without inventing meaning, mechanism, ownership, or proof strategy. When new evidence invalidates closure, reopen the smallest owner of the broken decision or input and preserve unaffected dispositions.

### Implementation-Input Closure

Before moving forward, close the inputs required by the next phase action or implementation acceptance unit/wave: each is canonical, mechanically derivable without a semantic choice, or available from a named external owner. Also close any cross-task decision that could invalidate that next work. Later inputs may remain open with an owner and checkpoint when they cannot invalidate the next accepted result; they block only when the current unit would otherwise be unusable or dishonest.

Inside the active macro phase, supporting steps, reviews, repairs, and the
smallest upstream reopen continue automatically while required inputs remain
closed. A candidate awaiting triggered review is internal state, not a result
or handoff. Stop only when:

- the user explicitly named that boundary;
- a required external decision or input is unavailable from its named owner;
- the next action requires new authority;
- the remaining work needs durable resume or coordination that has not yet been recorded.

Otherwise continue instead of asking whether to proceed
([Proceeding](../AGENTS.md#proceeding)). Movement to another macro phase uses a
fresh session and the required handoff. A blocker, interruption, or same-phase
resume emits no next-session prompt; only a completed, review-cleared phase does.
Non-implementation review convergence stays with [Subagents And
Review](spec-first-workflow/shared/subagents-and-handoff.md); Implementation
follows its [phase-owned closure](spec-first-workflow/phases/implementation-validation-closeout.md).

At every macro-phase boundary, follow [Resume And Macro-Phase Handoff](spec-first-workflow/shared/resume-and-handoff.md#macro-phase-handoff).
