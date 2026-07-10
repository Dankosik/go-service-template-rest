# Subagents And Handoff

Detailed shared companion for `docs/spec-first-workflow.md`. Read this when planning subagent lanes, routing model profiles, running phase-internal review/repair loops, resuming from artifacts, or rendering macro-phase handoff prompts.

## Read When

- A phase needs a fan-out decision, a mandatory independent review/challenge, or `Subagent Gate Audit` status.
- An owning macro phase must run review, repair actionable findings, and obtain a fresh re-review verdict without returning control to the user.
- A session is resuming from artifacts and must choose the correct read order.
- A macro-phase boundary needs a copy-pastable next-session prompt, or implementation handoff must set a Codex Goal.

## Inputs

- Current phase file, task-local workflow-control artifacts, and the relevant reviewed spec/design/tasks context.
- `docs/subagent-contract.md` and `docs/subagent-brief-template.md` for reusable subagent brief shape.
- Active tool/model-profile availability and `agent_request`; repository-standing read-only authorization is `capability_only` and does not select execution shape.

## Outputs

- A local decision or a bounded lane plan, compact lane summaries, and root-owned fan-in status.
- A current review-cycle result: reviewed revision, model route, findings, repair disposition, and fresh verdict.
- Artifact-first resume order.
- Chat-only next-session prompt or Codex Goal implementation handoff.

## Stop Rule

Use this shared file to support the owning phase. It does not approve specs, design, tasks, implementation, or completion by itself.

## Shared Subagent Gate Audit

Workflow-control records include a `Subagent Gate Audit` when a phase depends on fan-out, formal clarification, specification review, technical design review, task-ledger review, or an explicit local decision:

```text
Subagent Gate Audit:
- Decision: <local or fan-out, with the independent question test>
- Routing identity / validity: <routing_scope, routing_revision, record_validity observed by this gate>
- Gate type: <research fan-out | spec-clarification | specification-review | workflow-adequacy | design-authoring fan-out | technical-design-review | task-ledger-review | review/validation fan-out>
- Lane table: <lane id, agent, owned question, why separate context helps, skill/no-skill, inspect-first target, order/parallelism, read-only enforcement, status>
- Model route: <agent profile, exact model and reasoning effort selected before launch, task-complexity rationale, and launch surface that enforces the pair>
- Review cycle: <candidate revision, reviewer thread/process, prior finding closure when applicable, and fresh-versus-stale verdict>
- Concurrency: <active lane count; task-specific reason when greater than three>
- Lane result summary: <strongest finding, classification, falsification check or decisive evidence test, recommended handoff, owner or reopen target, evidence pointer>
- Fan-in: <orchestrator resolution, action, owner/artifact updated, unresolved conflicts, accepted risks, proof obligations, why severity is not stronger/weaker when material>
- Procedural gate: <procedural_gate_state>
- Review verdict: <review_verdict when the owning phase produces one; otherwise not applicable>
- Record validity: <current | stale | superseded>
- Readiness consequence: <handoff_readiness and next-phase consequence, with proof obligations when allowed with concerns>
- Reopen target: <required when blocked or failed>
```

`Subagent Gate Audit` statuses of `complete`, `PASS`, or `CONCERNS` are invalid when a required independent lane is missing or pending, a lane blocker or material severity conflict is unresolved, or fan-out uses broad/duplicate questions. Missing audit for a triggered gate keeps the owning phase draft or blocked; later phases must reopen or repair that phase instead of inferring readiness.

`Lens` is metadata for coverage, not a replacement for `spec-clarification-challenge` classifications. Lane outputs use `blocks_spec_approval`, `blocks_specific_domain`, and `non_blocking_but_record`; shared handoffs use the classifications from `docs/subagent-contract.md`.

The orchestrator owns fan-in:

- deduplicate overlapping questions and findings;
- compare conflicting assumptions across lanes;
- classify each surviving point by strongest justified impact: approval blocker, domain reopen, record-only constraint, proof obligation, accepted risk, or no-action item;
- preserve a short fan-in table or equivalent status in the workflow-control file: lens, trigger or source, falsification check or decisive evidence test, strongest finding, classification, action, owner or reopen target, and why severity is not stronger or weaker when material;
- treat lane-level missing input, unresolved blockers, and material blocker-severity conflicts as blocking the relevant approval area until answered, explicitly waived or accepted as risk, or routed to the owning phase;
- update `spec.md` only with final reconciled outcomes, not raw lane transcripts;
- reopen research, design, planning, or a specialist lane when a finding exposes a missing owner decision.

## Phase-Owned Review, Repair, And Fresh Re-Review

Specification, technical design, test design, planning, and implementation each own their required internal review loop. The user starts the macro phase; the root orchestrator completes this loop without asking the user to launch review or repair sessions:

1. Finish a reviewable candidate and identify its exact artifact paths, routing identity, and revision or content anchor.
2. Launch at least one required independent read-only reviewer in a fresh agent thread or independent read-only Codex process. Give it the candidate, accepted criteria, evidence boundary, output schema, and model route; do not ask it to repair files.
3. Reconcile findings as root authority. Separate phase-local actionable defects from accepted risks, downstream proof obligations, upstream reopen findings, and no-action observations.
4. Repair every actionable finding owned by the active macro phase. The root edits authoritative artifacts; a lower-cost read-only agent may propose a mechanical patch, but it never writes or approves it.
5. Mark the prior verdict stale, record the changed revision and finding closure anchors, and launch a fresh reviewer context. Re-review uses the same or a stronger model tier than the review that found the issue.
6. Repeat until the current verdict is `PASS`, or `CONCERNS` contains only explicitly accepted risks or downstream proof obligations that do not hide an in-scope repair. Then close the macro phase and render the next macro-phase prompt.

`FAIL` and actionable `CONCERNS` do not create a user handoff. They return to the root inside the same macro phase. A review author must not repair or approve its own candidate, and reusing the same agent thread without an explicit fresh-context reset is not independent re-review.

Stop the loop as blocked when a required repair belongs to an earlier macro phase, needs user policy or external authority, cannot run because all independent review surfaces are unavailable, or stagnates. Stagnation means the same approval-blocking finding survives two consecutive fresh reviews after bounded repair attempts with no changed evidence, owner decision, or narrower falsification result; record the blocker and smallest reopen target instead of looping indefinitely.

## Task-Aware Model Routing

Child roles are unpinned in `.codex/agents/*.toml`. The root must choose each child or subprocess route immediately before launch from the strict current catalog below and the supported reasoning levels currently exposed by the runtime. The root session's model and app mode remain user-selected and are outside this catalog.

### Current Child Model Catalog

Catalog scope: exact OpenAI model identifiers allowed for Codex subagents and Codex subprocesses launched by the root. This is an allowlist for dynamic per-lane selection, not a fixed role-to-model matrix and not a default for the root session.

Catalog verified: `2026-07-10`. The official Codex models page lists the three GPT-5.6 variants below, describes GPT-5.5 as previous-generation, and labels the remaining Codex Spark option as a research preview. The latest-model guide identifies Sol as the flagship, Terra as the intelligence/cost balance, and Luna as the efficient high-volume choice. This catalog therefore keeps only the latest production family.

| Allowed exact model ID | Official capability position | Dynamic selection pressure |
| --- | --- | --- |
| [`gpt-5.6-sol`](https://developers.openai.com/api/docs/models/gpt-5.6-sol) | Frontier capability for complex professional work. | Prefer when ambiguity, blast radius, difficult reconciliation, or the cost of a wrong judgment justifies maximum capability. |
| [`gpt-5.6-terra`](https://developers.openai.com/api/docs/models/gpt-5.6-terra) | Strong balance of intelligence and cost. | Prefer for substantial everyday evidence, implementation, and review work that does not need Sol's full depth. |
| [`gpt-5.6-luna`](https://developers.openai.com/api/docs/models/gpt-5.6-luna) | Cost-sensitive, high-volume capability. | Prefer for clear, bounded, repeatable extraction, transformation, or scan work with an objective result shape. |

Use only these exact identifiers. Do not substitute a family alias, previous-generation model, research preview, deprecated model, or any other runtime-visible model. Runtime availability narrows this allowlist; it never expands it. If the preferred model is unavailable, choose another catalog model from the lane's actual requirements and record the fallback rationale. If no catalog model is available, record the capability gap and do not claim a routed lane or silently fall back to an older model.

This snapshot becomes stale when OpenAI changes the recommended Codex family, publishes a migration or deprecation affecting an entry, the runtime no longer exposes an entry, or more than 30 days have passed since `Catalog verified`. Reverify it against the official [Codex models](https://learn.chatgpt.com/docs/models), [latest-model guide](https://developers.openai.com/api/docs/guides/latest-model), and [Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents) pages before the next child launch. When the official recommended set changed, update this catalog and its guardrail allowlist together before using new identifiers. If the active task does not authorize that instruction change, report the drift and do not claim a catalog-compliant required lane.

| Route | Project profile | Selection pressure | Boundary |
| --- | --- | --- | --- |
| Evidence | `evidence-agent` | Prefer the fastest and least costly currently available pair that can reliably handle the evidence volume, tools, context, and extraction accuracy required by the lane. | No gate verdict, readiness approval, semantic adjudication, or file write. |
| Semantic | Existing domain/review agents | Increase capability and reasoning with ambiguity, cross-source reconciliation, edge-case depth, and the cost of a wrong judgment. | One bounded question per lane; root owns synthesis and repair. |
| Critical review | `critical-reviewer-agent` | Select a stronger currently available pair only for one named approval-critical, hard-to-reverse, protected-domain, or complex multi-source question. | Not a default route and not a substitute for missing success criteria or evidence. |
| Critical adjudication | `critical-adjudicator-agent` | Select the strongest justified currently available pair only after ordinary review/repair/re-review leaves a material evidence-backed conflict. | Never first pass; return advisory resolution evidence to root. |

Before every launch, the root records the exact catalog model identifier, supported reasoning effort, task-specific rationale, and enforcement path. Agent role or filename never chooses the model by itself. Do not turn the catalog into a static model matrix, inherit the root model silently, or treat a prior lane's choice as the default for the next lane. Re-check the choice when task evidence, scope, or risk changes; a follow-up verdict uses the same or a stronger task-appropriate capability choice than the review that found the issue.

Use the native spawn surface only when it can enforce the selected model and effort for that launch. Otherwise launch a bounded process with explicit overrides, for example `codex --model "$LANE_MODEL" -c "model_reasoning_effort=\"$LANE_REASONING_EFFORT\"" --sandbox read-only exec ...`, and have the prompt load the named role profile plus `docs/subagent-contract.md`. For non-Codex subprocesses, pass the selected pair through that runtime's per-launch flags or environment; do not store provider-wide child-model defaults in repository config. If no path can enforce the selected pair, record the capability gap and do not claim routed review.

Keep `agents.max_depth=1`. A child that discovers another independent question returns it to the root; the root makes a fresh route decision before opening another direct lane.

## Subagents

Use subagents only when work divides into concrete, independent, bounded questions and separate context materially improves speed or quality. Keep work local when it is small, sequential, dependent on one reasoning chain, or would contend over shared mutable state. Execution shape and domain count do not create lanes by themselves.

Read-only must be enforced by the actual execution choice. If a lane cannot reliably stay read-only, keep that question in the main orchestrator flow instead of delegating it.

Use `docs/subagent-contract.md` and `docs/subagent-brief-template.md` for reusable brief shape.

Applicable repository instructions provide standing `capability_only` authorization for required read-only subagents and independent local review processes. Do not ask the user to repeat an authorization line in a handoff. This standing capability never selects `full_orchestrated`; `agent_request=substantive` still requires accepted task intent that makes multi-agent participation part of the result.

If the primary spawn surface is unavailable, try the configured read-only custom-agent or bounded `codex exec` fallback before blocking. Missing runtime capability does not turn an optional lane into a required one and does not permit a required independent review to become local-only.

Every lane follows `docs/subagent-contract.md` and the compact brief template. Merge duplicate questions, run dependent work sequentially, and default to no more than three concurrently active subagent lanes. Exceed three only when the lane plan gives a concrete task-specific reason why the extra independent question cannot wait, merge, or run sequentially. Preserve only compact lane summaries and reconciled root outcomes in authoritative artifacts.

## Resume Order

Resume from artifacts, not chat memory.

If approved `tasks.md` exists and implementation, review, validation, or closeout is next:

1. read `tasks.md` and its recorded routing identity first;
2. require its verdict/readiness to be `record_validity=current` for the active durable route before treating it as execution authority; stale, superseded, missing, or conflicting identity blocks execution;
3. read the artifacts named by `tasks.md`, usually reviewed `spec.md`, specification-review result, and any required design, test-plan, or rollout context;
4. read `workflow-plans/<phase>.md` only when `tasks.md` explicitly names a pre-created review or validation phase file.

If there is no approved `tasks.md` and `workflow-plan.md` exists:

1. read `workflow-plan.md` and its routing identity;
2. read the current `workflow-plans/<phase>.md` if the task uses one, and require the same current routing identity before handoff;
3. read the files named in the `Next session context bundle`;
4. then read phase artifacts as needed.

If there is no approved `tasks.md` and no `workflow-plan.md` because the task is direct or lean:

1. read `spec.md` when it exists;
2. read the specification-review record when non-trivial work is moving beyond specification;
3. read `tasks.md` when implementation or validation is next;
4. read optional `research/*.md` or `design/overview.md` only when named or needed.

Treat missing expected artifacts as incomplete unless an explicit waiver covers that exact artifact.

An artifactless `direct_state_envelope` may support only a status/closeout answer in the same orchestrator session under `STATUS-DIRECT-ENVELOPE`. It expires at session end, is never a resume source, and must not be reconstructed from chat memory or user-quoted text.

## Final Chat Handoff

When a macro phase reaches its boundary and a next macro phase or external/upstream reopen target exists, the final chat response must include a copy-pastable recommended next-session prompt derived from recorded artifacts. Internal review, repair, and fresh re-review never emit this prompt; they return to the owning root session. This is default behavior; the user does not need to ask the agent to stop or produce the handoff prompt.

Render the prompt in the final chat response only. Do not write the full ready-to-paste prompt into `workflow-plan.md`, `workflow-plans/*`, `spec.md`, `tasks.md`, an ad hoc prompt file, generated notes, or any other repository artifact. Artifacts may record the workflow state, next-session start point, context bundle, blockers, accepted risks, and proof obligations needed to regenerate the prompt; they must not become a second source of truth for the prompt text.

## Compact Handoff Contract

This file is the single owner of the chat handoff contract. Phase files and prompt-rendering skills may add phase-specific eligibility checks, but they must reference this contract instead of copying its template, standing-authorization policy, resume order, or execution manuals.

Assume the next session is context-blind but can read repository files. A handoff contains only the fields that can change its first action, result, proof, or stop decision:

- **Goal:** one immediate phase outcome, or one durable implementation objective.
- **Success/completion condition:** one observable condition; blocked stop is never successful completion.
- **Constraints and authorization:** only preserved scope, accepted concerns/waivers, current routing validity, execution mode, or authorization that is not already recoverable from the named artifacts.
- **Read/evidence:** the owning artifact first, then only non-obvious context or proof needed to avoid a real mistake.
- **Expected output:** only when the artifact or response shape is not obvious from the goal.
- **Stop/block rule:** exact blocked behavior and smallest reopen target.

Headings are optional. Omit empty fields, broad project summaries, generic strictness language, repeated prohibitions, long artifact excerpts, resolved history, and mechanics owned by another file.

For a non-implementation macro phase, name exactly one next macro phase or owning reopen target, its current routing identity/validity when durable state exists, the immediate output, minimal read order, material constraints/proof obligations, and the phase stop rule. Do not route the user to specification review, technical design review, task-review/readiness, post-code review, or validation when those are internal checkpoints of the active macro phase.

When the next macro phase is technical design, the prompt starts `technical-design-session`; that session runs system/integration design first, then Go code ownership design, then technical design review. It must not ask the user to start either internal checkpoint separately.

For implementation from a current approved `tasks.md`, use `.agents/skills/codex-goal-prompt-composer/SKILL.md` to render this contract. The generated Goal prompt contains only:

- one durable objective covering every required ledger task through final validation;
- one successful completion condition from the `Goal Contract`;
- the `tasks.md` path first, plus only the minimal additional read order named by the ledger;
- current implementation mode and preserved constraints, concerns, or waivers that affect execution;
- required proof or exact proof owner in `tasks.md`;
- blocked-stop behavior and the exact reopen target.

Do not copy worker launch commands, resume rules, sandbox policy, patch-intake procedure, integration-proof procedure, or a generic orchestrator manual into the generated prompt. Task-specific worker boundaries belong in approved `tasks.md`; repository-wide worker execution mechanics belong only in `docs/spec-first-workflow/phases/implementation-validation-closeout.md`.

Compact implementation shape:

```text
First, set a Codex Goal for this session: <durable objective>.

Completion condition: <one successful completion condition>.

Work in <absolute repository path>. Read `<task-local>/tasks.md` first, then <only non-obvious artifacts named by the ledger>.

Implementation mode and preserved constraints: <mode plus only material constraints, concerns, or waivers>.

Required proof: <named ledger proof or concise command/manual proof set>.

If the ledger cannot be completed because an implementation-blocking decision, artifact, command, or proof is missing or failing outside its approved boundaries, leave affected tasks unchecked, record the blocker in the ledger-owned surface, keep the Goal blocked, and reopen <exact owner>.
```

Use no prompt when the workflow is honestly done.

The prompt is chat-only. It is not a workflow artifact and must not become a second source of truth. If an artifact currently contains the full prompt body, trim it back to routing state and context-bundle entries unless the user explicitly asked for a standalone prompt document.

Before returning the prompt, apply the start test: a new session with no chat history should know the single next phase, why it is next, what to read first, what constraints and proof obligations matter, and where to stop. Remove any sentence that does not help that session start or avoid a real mistake.
