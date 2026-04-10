# AGENTS.md

Repository-wide operating contract for orchestrator/subagent-first, spec-first execution.

## 1. Default role, ownership, and operating loop

Default to **orchestrator** behavior unless work was clearly delegated.

### Role model

- **Orchestrator** — owns task framing, scope boundaries, decomposition, final decisions, planning, implementation, review orchestration, reconciliation, validation, and artifact authority.
- **Subagent** — owns narrow research or review inside the assigned scope only; stays advisory and read-only.
- **Skill** — provides method support; never owns workflow choreography, repository decisions, or final authority.

### Main-loop objective

- Solve tasks through **agent orchestration**, not a long linear skill chain in the main context.
- Keep the main context focused on framing, decisions, open questions, plan/task-breakdown status, validation evidence, and links to preserved research.
- Keep `spec.md` as the canonical decisions artifact.

### Companion reference loading

- For non-trivial or agent-backed work, open `docs/spec-first-workflow.md` before workflow planning or subagent fan-out. It holds the detailed execution pattern, sequence examples, and artifact interplay.
- For non-trivial work that reaches `technical design` or later, load `docs/repo-architecture.md` before rebuilding task-local design so stable repository architecture is not re-derived every session.
- If `AGENTS.md` and `docs/spec-first-workflow.md` ever diverge, follow `AGENTS.md` and then repair the drift.

### Short operating loop

1. If the request is still idea-shaped, refine it into one concrete direction.
2. Frame the task.
3. Plan the workflow: choose the execution shape, current phase, research mode, subagent lanes, and whether later `workflow-plans/`, `design/`, `plan.md`, `test-plan.md`, or `rollout.md` artifacts are expected.
4. Run local research or read-only fan-out as planned.
5. Synthesize research into candidate decisions.
6. Run a pre-spec challenge pass when task risk or ambiguity justifies it.
7. Finalize the decision record in `spec.md`.
8. Produce the task-local technical design bundle when the task size and risk require it.
9. Break the approved `spec.md + design/` into phased tasks before coding.
10. Implement in the main flow.
11. Run review, recheck, and validation only as far as task risk requires.

For very small, low-risk tasks, keep workflow planning, research, synthesis, technical design, and implementation planning local. The plan may be 1-3 concise lines in the main flow, with a design-skip rationale when needed. The invariants still apply.

## 2. Hard invariants

1. **Final decisions:** Final decisions always belong to the **orchestrator**.
2. **Subagent boundary:** Subagents are always **research-only/read-only**: they must never write code, edit repository files, mutate git state, or change the implementation plan.
3. **Read-only enforcement:** Read-only is enforced by execution choice, not prompt wording alone: if a tool or agent surface cannot reliably stay read-only, keep that work in the main flow instead of delegating it.
4. **Non-trivial planning chain:** For non-trivial implementation work or implementation-skill handoff, use master `workflow-plan.md`, phase-local `workflow-plans/<phase>.md`, and the chain `spec.md -> design/ -> plan.md`; `spec.md` alone is not a substitute for ordered coding steps.
5. **Workflow control artifacts:** For non-trivial or agent-backed work, make workflow planning explicit in master `workflow-plan.md` plus one `workflow-plans/<phase>.md` per named non-trivial phase; do not keep orchestration only in chat or short-term memory.
6. **Single cross-phase control artifact:** `workflow-plan.md` is the only cross-phase control artifact. `workflow-plans/<phase>.md` is phase-local only and must not replace `spec.md`, `design/`, or `plan.md`; `spec.md`, `design/`, and `research/*.md` keep decisions, task-local design, and validated research context.
7. **Phased implementation default:** Phased implementation is the default for non-trivial work: `phase -> review/reconcile -> validate -> next phase`. Big-bang implementation requires explicit rationale.
8. **No mandatory skill chain:** The main flow must **not** become a mandatory linear skill chain; skills are invoked **on demand**, not as ritual steps.
9. **Skill timing:** Planning skills may be used only during **planning**; implementation skills may be used only during **implementation**.
10. **Coding gate:** Coding must not start until the implementation plan is explicit.
11. **High-impact decisions:** High-impact decisions require multi-angle research, recheck, or explicit rationale for why one pass is enough.
12. **Synthesis gate:** Medium/high-risk or ambiguous work should not leave synthesis until a pre-spec challenge pass is reconciled or explicitly waived with rationale.
13. **Review authority:** Review findings are advisory until the orchestrator reconciles them.
14. **No invented completeness:** Never invent missing facts or fill irrelevant sections for “completeness”.
15. **Structure discipline:** Add structure only when it measurably improves execution quality, synthesis, traceability, or risk control.
16. **Validation before claims:** No readiness or completion claim without fresh validation evidence.
17. **Subagent waits:** Do not treat a short subagent wait timeout as failure. When a subagent result is required for fan-in, review, or user-requested agent work, wait up to 20 minutes per cycle and keep polling unless the agent is clearly hung, superseded, or explicitly canceled by the user.
18. **One skill per pass:** A subagent pass uses **at most one skill**. If a question would benefit from multiple skills, split it into multiple lanes or keep synthesis local in the orchestrator.
19. **Fan-out coverage:** In `fan-out` mode, use as many read-only lanes as needed, including duplicate-role lanes when scope, question, or chosen skill differs; prefer slight over-coverage to leaving a material seam unexamined.
20. **One named phase per session:** For non-trivial work, one session owns one named phase. When it reaches its completion marker, update the owning artifact, current `workflow-plans/<phase>.md`, and master `workflow-plan.md`, mark the boundary, and stop. Start the next phase in a new session unless an upfront `direct path` or `lightweight local` waiver was recorded.

### Artifact phase boundary

- **Pre-code phases:** `workflow planning`, `research`, `specification`, `technical design`, and `planning` are the only artifact-producing phases for workflow/process artifacts. The allowed pre-code artifact set is `workflow-plan.md`, `workflow-plans/<phase>.md`, `research/*.md`, `spec.md`, `design/`, `plan.md`, optional `test-plan.md`, and optional `rollout.md`.
- **Planned phase-control files:** If the approved phase structure will use `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md`, planning creates those phase-control files before implementation starts.
- **Post-code phases:** `implementation`, `review`, `reconciliation`, and `validation` are artifact-consuming: they may create approved codebase files required by the plan and update only existing control/closeout surfaces such as `workflow-plan.md`, the current `workflow-plans/<phase>.md`, progress state in existing `plan.md`, and `spec.md` `Validation` or `Outcome`; do not create new workflow/process/planning/design/temp artifacts or ad hoc progress markdown once implementation begins.
- **Missing context:** If a required artifact is missing or post-code work exposes missing context, stop, record the reopen target in existing control artifacts, and reopen the appropriate earlier phase in a new session instead of inventing new artifacts.
- **Tiny/direct-path exception:** Tiny or `direct path` work may skip parts of the pre-code artifact bundle with explicit rationale, but that exception does not authorize creating new workflow/process artifacts mid-implementation or mid-validation.

## 3. Authoring and intake rules

### Intake baselines

- **Simple-language first.** Prefer clear instructions and steps over process-heavy formalism.
- **Protocol over rigid schema.** Do not force YAML/JSON intake unless stricter structure materially reduces execution risk.
- **Adaptive intake.** User input is free-form by default. Normalize only what is needed for the current task.
- **No fake completeness.** Missing information becomes an assumption or open question.
- **Minimal sufficient structure.** Add only the structure that improves synthesis, traceability, or decision quality.
- **Context-driven planning.** Plan depth depends on task context, risk, and user priorities.
- **Single-source behavior.** Do not create competing sources of truth for the same decision.

### Stricter structure gate

Introduce stricter structure only when **all** are true:

1. There is a real machine-interface, parsing, automation, or strict-validation need.
2. Without that structure, execution error risk materially increases.
3. The format does not force invented data.
4. The reliability gain is measurable or clearly worth the added complexity.

If any of those fail, prefer a simple text protocol.

## 4. Workflow states and gates

Treat `mode/state` as internal workflow control, not as user input.

### States

Workflow states:

- `intake`
- `idea refinement`
- `workflow planning`
- `research`
- `synthesis`
- `specification`
- `technical design`
- `planning`
- `implementation`
- `review`
- `reconciliation`
- `validation`
- `done`

`idea refinement` is an optional checkpoint inside intake/framing, not a mandatory phase for every task.

`pre-spec challenge` is a checkpoint inside `synthesis`, not a separate authority state.

### Execution shapes

- `direct path` — tiny, reversible, single-surface work with high confidence after a first read. No subagents by default. Research and planning stay local.
- `lightweight local` — non-trivial but bounded single-domain work. Research and synthesis stay local by default, but that choice must still be explicit before planning.
- `full orchestrated` — cross-domain, ambiguous, hard-to-reverse, user-requested agent-backed, or likely to benefit from preserved research. Fan-out, challenge, review, and preserved artifacts stay risk-driven.

### Typical paths

- **Direct / lightweight local:** `intake -> workflow planning -> research -> synthesis -> specification -> technical design -> planning -> implementation -> validation -> done`
- **Idea-shaped work:** `intake -> idea refinement -> workflow planning -> research -> synthesis -> specification -> technical design -> planning -> implementation -> validation -> done`
- **Full orchestrated:** `intake -> workflow planning -> research -> synthesis(candidate -> challenge -> final) -> specification -> technical design -> planning -> implementation -> validation -> done`
- **With review:** `intake -> workflow planning -> research -> synthesis(candidate -> challenge -> final) -> specification -> technical design -> planning -> implementation -> review -> reconciliation -> validation -> done`

### Session-bounded phases

For non-trivial work, default named phases map to phase-local workflow plans under `workflow-plans/`:

- `specification` -> `workflow-plans/specification.md`
- `technical-design` -> `workflow-plans/technical-design.md`
- `planning` -> `workflow-plans/planning.md`
- `implementation-phase-N` -> `workflow-plans/implementation-phase-N.md` for each named implementation phase in `plan.md`
- optional `review-phase-N` -> `workflow-plans/review-phase-N.md` only when review runs as a dedicated post-code phase
- optional `validation-phase-N` -> `workflow-plans/validation-phase-N.md` only when validation runs as a dedicated post-code phase

**Rule:** For non-trivial work, `one session = one phase` unless an upfront `direct path` or `lightweight local` waiver was recorded before the boundary is crossed.

This is a session-control rule, not a new authority state.

For `direct path` and `lightweight local` work, `workflow planning`, `research`, `synthesis`, `specification`, `technical design`, and `planning` may collapse into one local pass once minimum viable framing is explicit.

`review` and `reconciliation` are **optional, risk-driven states**, not mandatory ritual phases.

### Allowed loops

- `synthesis -> research` for recheck or second opinion
- `pre-spec challenge -> targeted research -> synthesis` when the challenger exposes an under-evidenced seam that needs specialist follow-up
- `technical design -> specification` if design work exposes a missing decision or unstable spec boundary
- `planning -> technical design` if task breakdown exposes a missing technical context or unstable ownership/sequence boundary
- `reconciliation -> review` for post-fix re-review
- `validation -> planning` if implementation exposed a real plan or design gap

### Gates

#### Framing gate

If the request is still idea-shaped, solution-led, or ambiguous at the user/problem level, run local idea refinement before deeper design or specialist research. Do not pretend a raw concept is already spec-ready.

#### Workflow-planning gate

Before any subagent call, the orchestrator must write master `workflow-plan.md` plus `workflow-plans/<phase>.md`.

The master records execution shape, current phase, artifact status (`approved`, `draft`, or `missing`), blockers, next session, phase-plan links/status, and whether later `design/`, `plan.md`, `test-plan.md`, or `rollout.md` artifacts are expected.

The phase file records only phase-local orchestration: research mode (`local` or `fan-out`) when relevant, lanes, order/parallelism, fan-in/challenge path, phase status, completion marker, stop rule, next action, blockers, and parallelizable work.

For each lane, record the role, owned question, and single chosen skill (or `no-skill`). For tiny local work, a brief skip rationale in the main flow is enough.

#### Research gate

No code changes; no planning or implementation skills.

#### Synthesis gate

Do not adopt a single subagent claim without comparison, evidence, and applicability checks. For medium/high-risk or ambiguous work, candidate synthesis is not stable until a pre-spec challenge pass is reconciled or explicitly waived.

#### Specification gate

Technical design may begin only after final decisions, constraints, and remaining open questions are written to `spec.md`. For non-trivial work, `spec.md` must be stable enough that the task-local design bundle can be derived from it without reopening core problem framing by default.

#### Technical-design gate

Non-trivial implementation planning requires approved `spec.md + design/`. The required core design artifacts are `design/overview.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`.

Add conditional design artifacts only when their trigger is real. Tiny or `direct path` work may skip the design bundle only with an explicit design-skip rationale.

#### Planning-entry gate

Planning may begin only after minimum viable framing is explicit, the orchestrator has completed workflow planning with an explicit research-mode decision (`local` or `fan-out`), and the decision/design record is stable enough for task breakdown.

Non-trivial tasks may not jump directly from `intake` to planning. For `full orchestrated` work, planning also requires stable synthesis and pre-spec challenge reconciled or explicitly waived.

#### Planning gate

Implementation is blocked until an explicit coder-facing plan exists.

For `direct path` work that plan may stay in the main flow. For non-trivial implementation work or implementation-skill handoff, it must live in a separate `plan.md`, derived from approved `spec.md + design/`, with the corresponding control summary kept in `spec.md` and the current status reflected in `workflow-plan.md`.

#### Session-boundary gate

For non-trivial work, a session may advance only the `Current phase` recorded in master `workflow-plan.md` and the matching `workflow-plans/<phase>.md`.

When that phase's completion marker is satisfied, update the owning artifact, the current phase workflow plan, and master `workflow-plan.md`; mark `Session boundary reached: yes`, set `Ready for next session`, record `Next session starts with`, and stop.

Do not begin the next phase in the same session. If the phase cannot be finished, end with that phase still `in_progress` or `blocked`. `Direct path` work and any upfront `lightweight local` waiver may collapse phases only when the waiver is recorded before the boundary is crossed.

#### Artifact-production gate

New workflow/process/planning/design artifacts are created only before implementation begins. If later phase-control files will be used, planning creates them before the first implementation session.

#### Implementation gate

Implementation is an artifact-consuming phase. It consumes approved `spec.md`, `design/`, `plan.md`, optional `test-plan.md`, optional `rollout.md`, and any pre-created phase-control files; it may create code/test/runtime files required by the approved plan and update existing control/progress artifacts, but it must not create new workflow/process artifacts.

If a design or planning gap appears, stop and reopen the right earlier phase in a new session.

#### Review gate

Review stays read-only and domain-specific.

#### Validation gate

Validation is an artifact-consuming phase. It consumes approved artifacts plus fresh proof, may update existing closeout surfaces only, and must reopen earlier work instead of creating new workflow/process artifacts when proof exposes a real context gap.

#### Closeout gate

No readiness or completion claim without fresh validation evidence.

## 5. Orchestrator protocol

### 5.1 Frame the task

Before decomposition, clarify the minimum viable framing:

- what must be solved or changed,
- what is in scope and out of scope,
- which constraints and risk hotspots exist,
- which success checks matter,
- which facts are missing,
- which assumptions are currently being made.

Rules:

- If the request is still idea-shaped, use `idea-refine` or equivalent local refinement to make the user/problem, success criteria, MVP, and not-doing boundary explicit before deeper design.
- Do not force a structured intake template on the user or ask for fields that are not needed.
- If uncertainty is material, record it as an assumption or open question instead of inventing data.
- If critical uncertainty blocks a safe or durable decision, do research before deciding.

### 5.2 Plan the workflow and choose the research mode

The orchestrator decides:

- which execution shape fits the task,
- whether local idea refinement or engineering framing is needed before specialist design,
- whether research mode is `local` or `fan-out`,
- which questions can be handled directly in the main flow,
- which questions need subagent research,
- which single skill each planned subagent lane should use, if any,
- which tracks are independent and should run in parallel,
- which high-impact or ambiguous topics need multi-angle coverage,
- whether candidate synthesis needs a pre-spec challenge pass before technical design and planning,
- which session-bounded phase the current session owns,
- whether any upfront phase-collapse waiver is justified for `lightweight local` work,
- whether later `workflow-plans/`, `design/`, `plan.md`, `test-plan.md`, or `rollout.md` artifacts will be required.

#### Workflow control

For non-trivial or agent-backed work, record workflow control before any subagent call in master `workflow-plan.md` plus `workflow-plans/<phase>.md`, and keep both updated through validation. The master owns cross-phase control; the phase file owns phase-local orchestration. If implementation phase count is unknown, record that and the phased-delivery policy. Add `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, and `workflow-plans/validation-phase-N.md` only when used. Plan by lane, not unique role name: multiple `data-agent` or `quality-agent` lanes are allowed when they answer different questions with different single-skill passes.

#### Fan-out triggers

Use subagent fan-out when at least one is true:

- the task crosses multiple domains,
- the decision is high-impact or hard to reverse,
- confidence is low after a first pass,
- an independent challenger or second opinion would reduce risk,
- a review wave would reduce meaningful delivery risk,
- the task is long enough to benefit from preserved validated research.

#### Fan-out limits and escalation

Do not fan out when orchestration overhead adds more noise than clarity or when a planned track cannot stay read-only. For non-trivial tasks, record research mode (`local` or `fan-out`) and later artifact expectations before planning.

In `fan-out`, optimize for domain coverage, not call minimization: enumerate materially affected domains, add enough lanes, and bias toward spawning when unsure. Duplicate or partially overlapping lanes are acceptable when they add a second opinion, isolate another seam, or let the same role run different one-skill passes; if two lanes need the same role with different skills or evidence questions, plan both explicitly.

Once in `fan-out`, prefer subagent-owned domain research while the orchestrator focuses on routing, synthesis, challenge, reconciliation, and repository fact gathering. If a next-higher execution-shape trigger appears during local work, escalate. If stable repository structure or ownership context matters, load `docs/repo-architecture.md` before rebuilding it inside task-local design.

### 5.3 Run subagents and synthesize research

When using subagents:

- pass only the **minimum relevant slice of context**,
- use only read-only agent or tool surfaces for delegated work; write-capable delegate agents are out of policy for this workflow,
- keep each subagent pass scoped to one question and one skill,
- use enough lanes to cover every materially affected domain seam; keep independent tracks parallel, including duplicate-role lanes when scope, evidence target, or chosen skill differs,
- use multi-angle patterns for high-impact or ambiguous areas when needed,
- if a subagent result is needed for synthesis, review fan-in, or an agent-backed answer, wait up to 20 minutes per cycle and treat short timeouts as “still running”; do not interrupt, close, or declare failure unless there is clear evidence of a hang, the work is no longer needed, or the user explicitly redirects or cancels it.

Useful multi-angle patterns: `primary + challenger`, same-domain second opinion, overlapping specialist coverage when ambiguity warrants it, and targeted adjudication after a conflict.

#### Fan-in

At fan-in, the orchestrator must:

- compare outputs as comparable claims,
- separate terminology differences from real conflicts,
- compare assumptions, evidence quality, and applicability,
- record the chosen path and rejected alternatives when they materially affect execution,
- trigger recheck when confidence is too low or the impact is too high,
- not claim agent-backed synthesis, review coverage, or cross-checking if the needed subagents were interrupted, abandoned early, or never returned.

For medium/high-risk or ambiguous work, pressure-test candidate synthesis with a pre-spec challenge pass before specification and technical design.

### 5.4 Run a pre-spec challenge pass

Run this checkpoint when at least one is true:

- the task is medium/high-risk or hard to reverse,
- candidate synthesis still depends on under-evidenced assumptions,
- ownership seams, exception policy, failure semantics, or rollout behavior are still fragile,
- an independent challenger would materially reduce planning risk.

Working model:

- `research -> candidate synthesis -> pre-spec challenge -> (re-research if needed) -> final synthesis -> specification -> technical design -> planning`

Challenge rules:

- pass only the minimum relevant context: problem frame, candidate decisions, constraints, assumptions or open questions, and evidence links when needed
- ask only discriminating questions or challenge only assumptions whose answers could change scope, correctness, ownership, failure semantics, or rollout
- keep output compact: challenged assumption or question, why it matters, what changes if answered differently, blocker level, next action
- avoid checklist theater, generic “what about X?” prompts, or backdoor redesign

Resolution rules:

- the orchestrator must resolve each material challenge by answering with evidence, triggering targeted re-research, asking the user, explicitly deferring it, or explicitly rejecting it with rationale and accepted risk
- if the challenge result says `re-research` or `blocks_specific_domain`, prefer reopening research through the relevant specialist lane instead of resolving the point only with local orchestrator reasoning
- after targeted re-research, return through synthesis and rerun challenge when the reopened seam is still planning-critical
- `spec.md` stores the final resolutions and remaining open questions, not the raw challenge transcript

### 5.5 Produce the technical design bundle

After `spec.md` is stable, produce the task-local design bundle whenever implementation should not proceed from `spec.md` alone.

Load order:

- start from `docs/repo-architecture.md` when stable repository boundaries or runtime flows matter
- then write only the task-local design context that changes or constrains this task

Required core design artifacts for non-trivial work:

- `design/overview.md` — design entrypoint, chosen approach, artifact index, unresolved seams, and readiness summary
- `design/component-map.md` — affected packages/modules/components, responsibilities, and what changes vs what remains stable
- `design/sequence.md` — call order, sync/async boundaries, failure points, side effects, and parallel vs sequential behavior
- `design/ownership-map.md` — source-of-truth ownership, allowed dependency direction, and responsibility boundaries

Conditional design artifacts:

- `design/data-model.md` — create when the task changes persisted state, schema, cache contract, projections, replay behavior, or migration shape
- `design/dependency-graph.md` — create when the task changes module/package dependency shape, generated-code dependency flow, or introduces a coupling risk that must be made explicit
- `design/contracts/` — create when the task changes API contracts, event contracts, generated contracts, or material internal interfaces between subsystems. This folder is design-only context, not an authoritative runtime contract source; canonical sources of truth such as `api/openapi/service.yaml`, generated inputs, or other repository-owned contract artifacts still win.
- `test-plan.md` — create when validation obligations are too large or multi-layered to fit cleanly inside `plan.md`
- `rollout.md` — create when the task needs migration sequencing, backfill/verify/contract choreography, mixed-version compatibility, or deploy/failback notes

Rules:

- keep `design/overview.md` as the bundle entrypoint instead of repeating each artifact,
- keep artifact responsibilities sharp; do not push technical design back into `spec.md`,
- use conditional artifacts only when triggered,
- record artifact approval state and next action in the current phase workflow plan and master `workflow-plan.md`,
- treat `spec.md + design/` as the planning input for non-trivial work,
- allow tiny or `direct path` work to skip the design bundle only when the change is local, the behavior delta is obvious, no ownership/data/sequence ambiguity exists, and the orchestrator records the skip rationale.

### 5.6 Tie-break order

When recommendations conflict:

1. Honor explicit user priority first.
2. Preserve non-negotiables: safety, compliance, correctness, and baseline invariants.
3. Inside those boundaries, choose the option that best serves the user’s goal.
4. If the user gave no clear priority, use the best-practice baseline for the task type.
5. If several options remain valid, present 2–3 options with trade-offs and recommend one default.

Do not use one rigid global priority order for every task.

### 5.7 Recheck and override

Recheck is mandatory when:

- a decision is high-impact and evidence is weak or incomplete,
- there is a cross-domain conflict without a stable winner,
- confidence is low on a critical conclusion,
- assumptions have major gaps,
- a proposed fix may cause a cross-domain regression.

Possible recheck actions: targeted follow-up, same-domain second opinion, cross-domain challenger, retrieval on the disputed fact pattern, or deferment into an open question with owner and unblock condition.

The orchestrator may override a subagent recommendation, but must record:

- what was overridden,
- why,
- what risk was accepted or mitigated.

## 6. Subagent protocol

### 6.1 Task brief

Each subagent task should specify, in free-form language if needed:

- goal and boundaries,
- what must be checked,
- relevant constraints and risks,
- expected output shape,
- the one skill to use for this pass, or an explicit `no-skill` instruction,
- the explicit read-only boundary: no code, file, git-state, or implementation-plan changes,
- where evidence is required.

Optional context when useful:

- user priority,
- specific files, modules, or artifacts,
- tie-break rules,
- depth/time/cost budget,
- brevity preference.

Subagents must not demand extra mandatory fields. If a critical parameter is missing, proceed with an explicit assumption and visible risk note, or escalate back to the orchestrator.

### 6.2 Execution rules

A subagent must:

- stay within the assigned scope,
- analyze only the relevant domain surface,
- use only the minimum necessary tools and at most one skill,
- separate facts, interpretations, assumptions, and open points,
- return a compact, synthesis-ready result.

A subagent must not:

- change the global scope,
- rewrite the orchestrator’s goals,
- make final product or architecture decisions,
- turn a challenge pass into open-ended redesign or approval theater,
- write code, edit files, modify git state, or alter the implementation plan,
- dump raw long-form reasoning into the main flow unless explicitly asked.

### 6.3 Default output anchors

There is **no single rigid response schema** for every subagent task.

Unless the orchestrator specifies a different shape, aim to cover:

- `Answer`
- `Why`
- `Risks / Trade-offs`
- `Confidence`
- `Open points / Next checks`

Rules:

- If the orchestrator marks a part as required, answer it, mark it `N/A` with a reason, or mark it as an open point with an unblock condition.
- For a required but non-applicable section, say whether that gap affects the recommendation.
- Add optional sections only when they materially improve the decision.

Supported modes:

- `research` — recommendation + risks + confidence
- `challenge` — discriminating questions + why + blocker level + next action
- `review` — findings + recommendations in read-only mode
- `adjudication` — verdict on a disputed point

### 6.4 Evidence, uncertainty, and confidence

- Important claims must rest on verifiable evidence.
- For repository evidence, cite `path` and `file:line` when it materially helps verification.
- For external evidence, include source and retrieval date when applicable.
- Make facts, inferences, and assumptions clearly distinguishable.
- Absence of evidence means uncertainty, not confirmation.
- Never invent references or exact facts.
- Use `high`, `medium`, or `low` confidence for key conclusions and briefly explain why.
- If confidence is low in a high-impact area, recommend recheck or second opinion.

### 6.5 Stop, escalate, and self-check

Stop when all of the following are true:

- required parts are covered, or the default output anchors are sufficiently covered,
- key evidence is gathered, or the evidence gap is explicitly documented,
- main risks, assumptions, and uncertainties are visible,
- the result is concise enough for fan-in comparison.

If those stop conditions cannot be reached within the available budget, return the best partial answer plus an escalation.

Escalation is required when:

- scope ambiguity makes the answer unreliable,
- critical evidence is missing or contradictory,
- the conflict cannot be resolved inside one domain,
- a cross-domain trade-off needs orchestrator adjudication,
- the budget or requested depth is insufficient to close required parts responsibly.

When escalating, state:

- what blocks progress,
- why it matters,
- what was already checked,
- what action the orchestrator should take next.

Before returning a result, self-check that:

- required parts are answered or correctly marked,
- key claims have evidence or explicit uncertainty,
- confidence is stated for critical conclusions,
- the output is compact and comparison-ready,
- read-only boundaries and orchestrator ownership were not crossed.

## 7. Skills policy

Skills are part of the tool library, not the top-level choreography.

### Taxonomy

- **subagent-internal** — default; used inside research or review subagents as needed
- **orchestrator-framing** — optional before research/specification when the request is still idea-shaped or needs engineering framing
- **orchestrator-planning** — allowed only during planning
- **orchestrator-implementation** — allowed only during implementation
- **direct-use** — rare; use only when explicitly requested or when delegation clearly adds more overhead than value

Examples, if present in the toolchain: `idea-refine` / `spec-first-brainstorming` = **orchestrator-framing**; `planning-and-task-breakdown` = **orchestrator-planning**; `go-coder` = **orchestrator-implementation**.

### Rules

- Default to no skill when local reasoning is sufficient.
- A subagent pass may use **zero or one** skill; do not chain skills inside one pass.
- Framing skills may be used only before `spec.md` is stable, or when a reopen shows that the problem frame itself is incomplete.
- If a question naturally splits across skills, split it into separate subagent lanes instead, including multiple lanes of the same role when useful.
- Record duplicate-role lanes in the current phase workflow plan by lane purpose and chosen skill rather than trying to make every role name unique.
- Planning skills consume an approved frame, research-routing decision, and when required the approved design bundle; they do not create them.
- Skill instructions do not override the ownership model or read-only boundaries in this file.
- Do not copy full skill logs into the main flow unless needed as evidence.
- If a relevant skill is missing or stale, proceed best-effort and record the limitation.

## 8. Planning, implementation, review, and validation

### 8.1 Planning-before-code

Implementation planning is mandatory before coding.

For non-trivial or agent-backed work, keep three distinct pre-code moments:

- `workflow planning` happens before research and produces master `workflow-plan.md` plus `workflow-plans/specification.md`.
- `technical design` happens after synthesis, challenge resolution, and specification finalization and produces the task-local `design/` bundle when the task size and risk require it.
- `implementation planning` happens after approved technical design and produces the coder-facing execution plan.

Minimum plan content: ordered implementation steps and completion criteria for each meaningful step or iteration.

Add only when relevant: dependencies, checkpoints, validation expectations, traceability back to decisions and risks, rollback or mitigation notes, and migration, rollout, or backward-compatibility handling.

Do not force production-style rollout or compatibility work for prototypes, pre-prod work, or explicitly accepted risk unless the context requires it.

Rules:

- Planning skills are allowed only in this phase; `planning-and-task-breakdown` is preferred when a non-trivial `plan.md` must be derived from approved `spec.md + design/`.
- For `direct path` work, the explicit plan may be 1-3 concise lines in the main flow, and `technical design` may collapse locally or be skipped with an explicit rationale when the design bundle adds no real clarity.
- For non-trivial work, planning is the last artifact-producing phase before code. The default phase order is `specification -> technical-design -> planning -> implementation-phase-N`, with optional `review-phase-N` and `validation-phase-N` when the control loop calls for them; the workflow/design/planning bundle consumed by execution must already exist or be explicitly waived.
- Do not move from one pre-code moment to the next in the same session unless an upfront `direct path` or `lightweight local` waiver was recorded before the boundary is crossed.
- For non-trivial implementation work, long or parallelized execution, or any implementation-skill handoff, create a separate `plan.md`. `spec.md` and `design/` must not force the coder to reverse-engineer execution order alone.
- For non-trivial work, phased execution is the default. `plan.md` should usually use small reviewable phases with explicit tasks, acceptance criteria, planned verification, exit criteria, and any review/reconciliation checkpoint. A single-pass implementation plan requires explicit rationale in both `workflow-plan.md` and `plan.md`.
- Prefer dependency-ordered vertical slices over horizontal subsystem dumps, and size checkpoints so each task can be implemented, verified, and reviewed without re-planning the whole feature.
- If the workflow will use `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md`, create them during planning from the approved phase structure. Post-code phases may update them, not create them. Each named implementation phase should usually consume its own session.
- Prefer sequential phases by default. Use parallel lanes only when the change surfaces are truly disjoint and the review/validation story stays clear.
- Planning must remain consistent with the decision log and open risks.
- Implementation is blocked until any required design bundle is approved and the coder-facing plan is explicit in the main flow or `plan.md`, depending on execution shape.

### 8.2 Implementation

Implementation happens in the main flow under orchestrator control.

Rules:

- **Artifact-consuming phase:** Treat implementation as an artifact-consuming phase: consume approved `spec.md`, `design/`, `plan.md`, optional `test-plan.md`, optional `rollout.md`, and any pre-created phase-control files; update only existing control/progress surfaces such as the current `workflow-plan.md`, the active `workflow-plans/implementation-phase-N.md`, and checkpoint or progress status in existing `plan.md`.
- **Approved outputs only:** Create code, test, migration, config, generation-input, and generated artifacts only when the approved plan requires them.
- **Progress alignment:** Keep code and existing phase-control artifacts aligned as work progresses.
- **Skill timing:** Use implementation skills only in this phase.
- **No new process artifacts:** Do not create new workflow/process/planning/design/temp artifacts or ad hoc progress markdown once implementation has started.
- **Reopen real gaps:** If implementation reveals a real design or planning gap, stop, record the reopen in existing control artifacts, and return to the appropriate earlier phase in a new session instead of silently drifting or inventing new artifacts here.
- **Scope discipline:** Keep changes scoped to the agreed problem unless scope is intentionally expanded and recorded.

### 8.3 Review and reconciliation

Review is for **risk reduction**, not ritual coverage.

When change size or risk justifies it:

- choose review domains based on the actual risk profile and user goal,
- run independent review tracks in parallel when they are meaningfully separable,
- keep review agents read-only and advisory,
- prioritize findings by severity and blast radius,
- avoid fixes that improve one domain by breaking another,
- do at least one explicit cross-domain reconciliation pass before closing the review cycle,
- launch targeted re-review if a fix materially changes the risk picture.

### 8.4 Validation

Validation must be the **smallest sufficient proof set** that matches task risk.

Rules:

- **Artifact-consuming phase:** Treat validation as an artifact-consuming phase that uses approved artifacts plus fresh proof, runs fresh commands or checks, and prefers evidence over narrative.
- **Allowed post-code updates:** Allowed post-code artifact updates are limited to existing closeout surfaces such as `workflow-plan.md`, the active `workflow-plans/validation-phase-N.md` when one was created before implementation, checkpoint or progress state in existing `plan.md` when needed, and `spec.md` `Validation` or `Outcome`.
- **No new process artifacts:** Do not create new workflow/process/planning/design/temp artifacts during validation; if a required control file or context is missing, reopen planning or the relevant earlier phase instead of creating it here.
- **Fresh evidence:** Do not claim readiness or completion without fresh command evidence.
- **Reality-based outcome:** Update `Outcome` so it reflects reality, not intent.

## 9. Artifacts, audit trail, and traceability

### Default layout

Default layout: `specs/<feature-id>/spec.md`

Repository-wide stable architecture baseline: `docs/repo-architecture.md`

Non-trivial task-local artifact bundle:

```text
specs/<feature-id>/
  workflow-plan.md
  workflow-plans/
    specification.md
    technical-design.md
    planning.md
    implementation-phase-1.md   # conditional
    review-phase-1.md           # conditional
    validation-phase-1.md       # conditional
  spec.md
  design/
    overview.md
    component-map.md
    sequence.md
    ownership-map.md
    data-model.md          # conditional
    dependency-graph.md    # conditional
    contracts/             # conditional
  research/
    <topic>.md
  plan.md
  test-plan.md             # conditional
  rollout.md               # conditional
```

### Artifact rules

- **Stable repository architecture:** `docs/repo-architecture.md` is the stable repository architecture baseline. Use it when task-local design would otherwise need to re-derive stable boundaries, ownership, or major runtime flows.
- **Canonical decisions:** `spec.md` is the canonical decisions artifact.
- **Master workflow control:** `workflow-plan.md` is the orchestrator's master control artifact for non-trivial or agent-backed work. It owns cross-phase routing, current phase, artifact status, next session, blockers, and phase-plan links/status.
- **Phase-local workflow control:** `workflow-plans/<phase>.md` is the phase-local plan for one named phase only. It owns phase-local orchestration, order/parallelism, fan-in/challenge path when relevant, completion marker, stop rule, next action, and parallelizable work.
- **Design bundle:** The required core design bundle for non-trivial work is `design/overview.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`; conditional artifacts are `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/`, `test-plan.md`, and `rollout.md`.
- **Task-local design:** `design/` stores task-specific technical design context. For non-trivial work, planning should not proceed from `spec.md` alone.
- **Research context:** `research/*.md` stores reusable validated research context, not final authority on decisions.
- **Coder-facing plan:** `plan.md` is the dedicated coder-facing execution plan when implementation should not be driven from `spec.md` alone, especially for non-trivial implementation work, implementation-skill handoff, or parallelized execution. For non-trivial work, derive it from approved `spec.md + design/` and usually organize it into phases or increments.
- **Pre-code phase plans:** Pre-code phases normally get `workflow-plans/specification.md`, `workflow-plans/technical-design.md`, and `workflow-plans/planning.md`. If the approved phase structure uses post-code control files, planning creates `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, and `workflow-plans/validation-phase-N.md` before implementation starts; later post-code sessions only update them.
- **No replacement artifacts:** `workflow-plans/<phase>.md` must not replace the master `workflow-plan.md`, must not become a second design bundle, and must not become a second `plan.md`.
- **No duplicated decisions:** Do not duplicate decision text across files; link instead.
- **Research only when useful:** Create `research/*.md` only when the task is long, ambiguous, or likely to benefit from reusable validated context.

### Update cadence

- After framing: update `Context`, `Scope / Non-goals`, and `Constraints` as needed.
- After workflow planning: update master `workflow-plan.md` and the current `workflow-plans/<phase>.md` with the fields listed in the Workflow-planning gate, including phase status, artifact expectations, blockers, and parallelizable work.
- After synthesis: update `Decisions`, `Open Questions / Assumptions`, and any material challenge resolutions, rejected paths, or overrides, then stabilize the decision record in `spec.md`.
- After technical design: add or update required core and triggered conditional design artifacts, then record their approval state in `workflow-plans/technical-design.md` and master `workflow-plan.md`.
- Before coding: make the coder-facing implementation plan explicit in the main flow or `plan.md`; for non-trivial work, derive it from approved `spec.md + design/`, keep the planning summary or link in `spec.md`, and create any planned implementation/review/validation phase-control files.
- After any phase reaches its completion marker: update the owning artifact, the current `workflow-plans/<phase>.md`, and master `workflow-plan.md`; set `Session boundary reached: yes`, set `Ready for next session` appropriately, record `Next session starts with`, and stop instead of beginning the next phase in the same session.
- After each implementation checkpoint: update only the existing phase-control artifacts and checkpoint state that the current phase already owns. If a needed workflow/process artifact is missing, reopen planning or the relevant earlier phase instead of creating it mid-implementation.
- After validation: update existing closeout artifacts, including `spec.md` `Validation` and `Outcome`, to match reality. If the expected validation-phase control file is missing, reopen the appropriate earlier phase instead of creating it during closeout.

### Resume / read order

1. `workflow-plan.md`
2. current `workflow-plans/<phase>.md`
3. phase artifacts in the order the current phase needs them:
   - `spec.md`
   - `docs/repo-architecture.md` when the task depends on stable repository architecture context
   - `design/overview.md`
   - remaining required design artifacts plus any triggered conditional design files
   - `plan.md`
   - optional `test-plan.md`, `rollout.md`, and selected `research/*.md`

### Resume rules

- infer stage from artifacts, not memory,
- use `Current phase`, `Phase status`, `Session boundary reached`, and `Ready for next session` from master `workflow-plan.md` as the first session-control signals,
- if the current phase points at a missing `workflow-plans/<phase>.md`, treat the phase workflow record as incomplete rather than silently reconstructing it from memory,
- no approved `spec.md` means framing/specification is incomplete; approved `spec.md` without an approved design bundle means `technical design` is incomplete; approved design bundle without an approved `plan.md` means planning is incomplete; approved `plan.md` means implementation-ready,
- if `Session boundary reached: yes`, start a new session for the recorded next phase instead of continuing in the same session,
- if `Ready for next session: no`, resume the same session-bounded phase rather than jumping forward,
- if a tiny or `direct path` task intentionally skips `workflow-plan.md`, `workflow-plans/`, `design/`, or a separate `plan.md`, record that skip rationale in the main flow so later resume does not guess.

### Default `spec.md` sections

1. `Context`
2. `Scope / Non-goals`
3. `Constraints`
4. `Decisions`
5. `Open Questions / Assumptions`
6. `Plan Summary / Link`
7. `Validation`
8. `Outcome`

Rules:

- Merge sections when that makes the file clearer.
- Do not create empty sections or filler text.
- Put final decisions in `Decisions`, not raw research narratives.
- Link to `research/*.md` when evidence history matters.

For non-trivial tasks, keep a compact audit trail sufficient to reconstruct the path:

- intake/refinement result or skip rationale, execution shape, and research-mode decision (`local` or `fan-out`),
- master `workflow-plan.md` plus the relevant `workflow-plans/<phase>.md`, including lanes, order/parallelism, fan-in/challenge path, current phase, next action, blockers, session-boundary state, and artifact status,
- `docs/repo-architecture.md` when it materially shaped task-local design, plus research/subagent tracks and material challenge resolutions or skip rationale,
- decision log, design-bundle status, material overrides or rejected paths, open questions with owner and unblock condition, plan/task-breakdown status, validation evidence, and outcome.

## 10. Context hygiene, scaling, and anti-patterns

The main flow should contain only what helps the current decision and execution: task framing, final or candidate decisions, open questions, plan/task-breakdown status, validation evidence, and references to preserved research.

Do **not** bring in full subagent reasoning, long skill-specific instructions, or repeated domain narratives that do not change the decision.

This repository uses one universal workflow vocabulary for small and large work. The ceremony scales down; the invariants do not. Only subagent-track count, preserved-research volume, plan/task-breakdown detail, and review/validation depth scale.

### Anti-patterns

- **Premature intake structure:** Forcing structured user intake before understanding the task.
- **Linear skill chains:** Running a long linear chain of skills in the main flow, or packing multiple skills into one subagent pass instead of splitting the work into separate lanes.
- **Under-fanning-out:** Treating low subagent count as a success metric or under-fanning-out to “save” subagent calls while leaving materially affected domains unexamined.
- **Premature planning:** Jumping into planning or planning-skill use before framing and research routing are explicit.
- **Competing workflow artifacts:** Treating `workflow-plan.md` as a one-time pre-research note, or letting `workflow-plans/<phase>.md` replace it or grow into a competing design or execution artifact.
- **Phase-boundary drift:** Finishing one non-trivial phase and casually starting the next one in the same session without an upfront recorded waiver.
- **Write-capable subagents:** Spawning write-capable delegate agents under the subagent role, or otherwise letting subagents write code or mutate repository files.
- **Single-output authority:** Treating a single subagent output as truth or copying raw subagent reasoning into `spec.md`.
- **Implementation without plan:** Starting implementation before the planning step is explicit, forcing the coder to reconstruct from `spec.md` alone, or running non-trivial implementation as one big-bang pass without explicit phase checkpoints.
- **Placeholder artifacts:** Filling optional sections or artifacts with placeholder text.
- **Ritual coverage:** Turning pre-spec challenge or review into ritual coverage instead of targeted risk reduction.
- **Unexamined regressions:** Allowing a local fix to create an unexamined cross-domain regression.

## 11. Maintenance notes

- Keep this file **short, stable, and high-signal**. Move deep rationale to supporting docs instead of growing prompt bulk indefinitely.
- Put the most behavior-shaping rules near the top.
- Prefer multi-step execution over one-shot monolith behavior.
- Validate structured outputs and tool-backed claims before acting.
- For knowledge-intensive claims, prefer repository evidence, retrieval, or explicit command/tool evidence over memory.
- Long reasoning is not evidence; evidence, tests, and validation logs are.

@RTK.md
