# R1: Shape And Execution Authority

## Question And Scope

What authority and routing model can make execution-shape classification, dedicated workflow planning, direct-path code execution, and subagent authorization coherent without weakening existing safety gates?

Coverage: `B01-F01`, `B01-F02`, and `B01-F07`. This lane was read-only, advisory, and used explicit `no-skill`. Candidate models below are inputs to specification, not approved policy.

## Confirmed Facts

### Authority Baseline

- `AGENTS.md` is the compact repository authority; the router and skills are lower-precedence and must not override it (`AGENTS.md:17-23`).
- Final decisions remain with the orchestrator; subagents are read-only evidence providers, and workers are write-capable implementation delegates rather than authorities (`AGENTS.md:28-30`, `AGENTS.md:41-46`).
- Shape selection requires an accepted Phase 0 brief or explicit clear-input rationale and occurs before subagent calls (`AGENTS.md:27`, `AGENTS.md:191-192`).
- The task-local packet preserves all three R1 findings as unresolved research inputs (`specs/execution-shape-routing-hardening/workflow-plan.md:62-68`).

### `B01-F01`: Direct-Path Code Writer Has No Coherent Route

Confirmed contradiction:

- Direct path is the tiny, normally artifactless shape, permits inline planning, and is described as `first read -> inline plan when useful -> edit -> proof -> done` (`AGENTS.md:68`, `AGENTS.md:89-99`, `AGENTS.md:185`).
- Direct-path workflow artifacts, including `tasks.md`, should not be created for ceremony (`docs/spec-first-workflow/shared/artifact-model.md:88-96`).
- The same artifact model says direct-path code writing must use isolated CLI workers or stop blocked (`docs/spec-first-workflow/shared/artifact-model.md:82-84`).
- Worker eligibility requires an approved `tasks.md`, a passed task-review/readiness gate, a required worker handoff, and a ledger-bounded bundle; the orchestrator must not author patches for that approved-ledger route (`AGENTS.md:45-47`).
- The router likewise introduces workers only through approved-ledger implementation (`docs/spec-first-workflow.md:69-74`).
- The coding skill still anticipates direct-path multi-step work without `tasks.md` and permits eligible direct readiness waiver semantics (`.agents/skills/go-coder/SKILL.md:44-48`, `.agents/skills/go-coder/SKILL.md:76-82`).
- The Goal prompt helper's automatic route is approved-ledger-only, so it does not supply a direct-worker handoff (`.agents/skills/codex-goal-prompt-composer/SKILL.md:15-20`, `.agents/skills/codex-goal-prompt-composer/SKILL.md:32-39`).

Evidence-backed inference: in the intersection of the current rules, neither orchestrator nor worker is eligible to write direct-path code. The advertised route must therefore block or violate one owner rule.

Classification for specification handoff: `blocks_spec`.

### `B01-F02`: Classification And Workflow-Planning Ownership Are Circular

Confirmed overlap:

- `AGENTS.md` owns policy and the orchestrator owns the decision (`AGENTS.md:17-23`, `AGENTS.md:113-123`).
- The artifact model declares execution-shape and artifact-depth decisions as its output (`docs/spec-first-workflow/shared/artifact-model.md:13-25`).
- The router sends shape and artifact selection to that shared model (`docs/spec-first-workflow.md:29-34`, `docs/spec-first-workflow.md:40-53`).
- `workflow-planning-session` says to use it when the orchestrator must choose shape, yet also says to skip tiny direct work and lean work without durable control (`.agents/skills/workflow-planning-session/SKILL.md:21-32`).
- Its required inputs already include whether work is tiny/direct or non-trivial/agent-backed, but its workflow later chooses the shape again (`.agents/skills/workflow-planning-session/SKILL.md:34-43`, `.agents/skills/workflow-planning-session/SKILL.md:118-127`).
- Its tiny-task eval expects invocation of the skill to return the direct skip route without creating artifacts (`.agents/skills/workflow-planning-session/evals/evals.json:17-24`).

Targeted search found no canonical intake-to-classifier-to-dedicated-planning invocation table.

Evidence-backed inference: implementations can decide shape before invoking the skill, inside the skill, or twice, producing inconsistent routes or ceremonial control artifacts.

Classification for specification handoff: `blocks_spec`.

### `B01-F07`: Capability Authorization Can Self-Escalate Shape

Confirmed overlap:

- Full orchestrated includes `user-requested agent-backed` work (`AGENTS.md:70`, `docs/spec-first-workflow.md:42`, `docs/spec-first-workflow/shared/artifact-model.md:51`).
- The escalation list instead uses the broader phrase `user-requested subagents` (`AGENTS.md:74-83`).
- When platform policy requires explicit authorization, the canonical handoff contract requires an exact line that says the user explicitly requests and authorizes subagents and commands required lanes to spawn (`docs/spec-first-workflow/shared/subagents-and-handoff.md:66-74`).
- Non-trivial next-session prompts may be required to carry that line (`docs/spec-first-workflow/shared/subagents-and-handoff.md:138-145`, `AGENTS.md:211-216`).
- The workflow-planning skill uses bare `agent-backed` as a trigger without defining capability-only authorization (`.agents/skills/workflow-planning-session/SKILL.md:21-24`, `.agents/skills/workflow-planning-session/SKILL.md:79-91`).

Targeted negative search found no definition distinguishing capability authorization from substantive user-required agent participation.

Evidence-backed inference: a lean task can become full merely because its handoff contains the repository-required authorization token.

Classification for specification handoff: `blocks_spec`.

## Candidate Repair Models

### Direct-Code Actor Models

#### R1-M1: Explicit Orchestrator Direct-Path Exception

- The orchestrator classifies and retains authority.
- For eligible direct path only, it may author the tiny patch and run fresh proof.
- Mandatory isolated workers remain the rule for approved-ledger code-writing implementation.
- Newly discovered non-direct scope stops the direct edit and enters the reclassification transaction owned by `B01-F04`.

Compatibility: closest to current `edit -> proof`, no-bundle, `go-coder`, and approved-ledger worker wording. It requires removing or narrowing the direct-worker sentence in the artifact model, not weakening ledger-bound worker delegation.

#### R1-M2: Minimal Direct-Worker Exception

- Direct code uses a worker from a small direct execution brief containing accepted framing, one writable surface, forbidden workflow mutations, proof, and orchestrator patch intake.
- Worker eligibility becomes `approved ledger` or `explicitly classified direct execution brief`.

Compatibility: preserves universal worker-only patch production, but adds a new handoff, failure, resume, and status surface that does not exist today. It increases the burden on `B01-F03`, `B01-F04`, and `B01-F10`.

#### R1-M3: Remove Code-Writing Direct Path

- Direct path becomes no-code/read-only only.
- Every code patch is at least lean local, receives durable tasking/readiness, and uses worker delegation.

Compatibility: simplest writer rule but largest ceremony and compatibility change. It invalidates current direct edit/proof wording, the coding skill's direct waiver behavior, and tiny-code examples/evals.

Research recommendation for specification to evaluate first: R1-M1, because it resolves the contradiction with the smallest owner change and preserves approved-ledger worker safety. This is not approval.

### Routing Owner Models

#### R1-R1: Authoritative Preclassification

- `AGENTS.md`: hard policy and trigger owner.
- Orchestrator: decision actor after intake.
- Artifact model: detailed decision algorithm, artifact consequences, and reclassification recording.
- Dedicated workflow-planning skill: conditional procedure that validates/falsifies and persists an already defensible route when durable routing is triggered.

#### R1-R2: Direct Eligibility Precheck

- Orchestrator first decides only obvious direct versus needs routing.
- Workflow planning classifies lean versus full.
- Lean routing may require a chat-only form of the checkpoint.

Research recommendation for specification to evaluate first: R1-R1. It avoids two classifiers and is most compatible with current authority. This is not approval.

### Authorization Models

#### R1-A1: Orthogonal Booleans

- `subagent_capability_authorized`: platform permission only, no shape effect.
- `user_requested_agent_backed_execution`: agent participation/evidence is a substantive task requirement and may trigger full.

#### R1-A2: Typed Intent

- `Agent-backed execution request: substantive | capability_only | absent`.
- The canonical authorization line is `capability_only` unless separate user intent makes it substantive.
- Missing required authorization continues to block the required lane without changing shape or justifying local-only execution.

Research recommendation for specification to evaluate first: R1-A2, because it is explicit for resume and evals. It must compose with the typed status model rather than becoming another overloaded status. This is not approval.

## Rejected Alternatives

- Let subagents or workers decide shape or policy: violates orchestrator authority.
- Keep the current direct route and select a writer ad hoc: preserves `B01-F01`.
- Invoke dedicated workflow planning for every task: contradicts direct/lean artifact minimization and its own skip contract.
- Spawn research before classification: violates the repository order.
- Treat the exact capability authorization line as substantive agent-backed intent: creates automatic escalation and defeats smallest-shape routing.
- Remove `request` from the canonical authorization line without proving platform compatibility: the line exists for runtimes that require an explicit request.
- Reuse the Goal prompt helper for artifactless direct work without changing its trigger contract: it is currently approved-ledger-oriented.

## Compatibility Consequences

- R1-M1 must align `AGENTS.md`, the artifact model, router, `go-coder`, and direct-path evals around one explicit writer exception.
- R1-M2 additionally needs a direct worker brief owner, launch contract, unavailable-worker outcome, patch-intake route, resume semantics, and status fields.
- R1-M3 must normalize every direct-code example and waiver; it is a policy contraction, not a wording fix.
- The routing-owner repair must align router, artifact model, workflow-planning front matter, Use/Skip rules, required inputs, execution steps, and evals.
- The authorization repair must align both authority trigger phrases, shared handoff semantics, subagent contract, workflow planning, adequacy, status, and eval terminology.
- Completed historical bundles remain unchanged; new and actively repaired surfaces must not rely on ambiguous legacy terms.

## Required Proof And Fail-Before Cases

### `B01-F01`

- Eligible tiny one-surface code change with no ledger has exactly one legal writer and one proof route.
- Worker unavailability follows the selected model: R1-M1 remains executable; R1-M2 blocks; R1-M3 reclassifies before editing.
- A protected trigger discovered before or during first read prevents further direct writing.
- A second surface or non-obvious validation path triggers atomic reclassification before more writes.
- Approved-ledger code-writing still rejects orchestrator-authored patches.
- Direct no-code work remains executable with fresh proof.

### `B01-F02`

- Raw input cannot classify before Phase 0 acceptance.
- Clear tiny input reaches direct without durable workflow artifacts.
- Bounded non-trivial input reaches lean without a forced full phase file.
- Full/protected/durable input invokes dedicated workflow planning exactly once.
- Explicit skill invocation on tiny work returns the direct skip route without implementation.
- New trigger evidence falsifies the recorded classification through the `B01-F04` route, not a competing classifier.

### `B01-F07`

- The canonical authorization line alone does not change shape.
- `You may use subagents if needed` is capability-only.
- A substantive requirement to use parallel agents and preserve their evidence triggers the recorded agent-backed rule.
- Missing required authorization blocks the lane and cannot become `local_only`, `waived`, or `not_expected`.
- Strict-boundary and broad-audit triggers remain independent.
- Resume preserves the distinction without treating injected handoff text as new task intent.

## Cross-Finding Constraints

- `F03`: writer mode and authorization intent need typed homes, but must not be mixed into artifact status.
- `F04`: every direct model depends on atomic reclassification and stale-state handling.
- `F06`/`F09`: adequacy must test the chosen actor route, invocation predicate, protected triggers, and authorization distinction.
- `F10`: artifactless direct work must report or explicitly exclude shape, writer eligibility, proof state, and reclassification.
- `F11`: each branch needs semantic guardrail/eval coverage.
- R1-M2 creates the largest risk of reopening `F03`/`F10`; no R1 model independently closes those findings.

## Missing Evidence And Specification Handoff

- No artifactless direct-worker launch/handoff protocol was found.
- No canonical definition distinguishes `agent-backed`, `user-requested agent-backed`, `user-requested subagents`, and capability-only authorization.
- No authoritative intake-to-classifier-to-workflow-planning invocation table was found.
- Scoped evals do not cover direct writer eligibility, unavailable writer, direct escalation, or capability-only versus substantive intent.
- Runtime-specific need for the exact authorization sentence remains a proof obligation, not a policy answer.

Recommended specification destinations:

| Finding | Destination | Reopen implication |
| --- | --- | --- |
| `B01-F01` | `Execution Actors And Direct-Path Code-Writing Eligibility` | Unresolved writer choice blocks spec completion; infeasible direct-worker evidence reopens research. |
| `B01-F02` | `Routing Authority And Invocation Table` | Circular ownership blocks spec completion; later trigger mismatch reopens workflow planning/reclassification. |
| `B01-F07` | `Subagent Capability And Agent-Backed Intent` | Ambiguous intent blocks spec completion; runtime availability may remain a named proof obligation. |
