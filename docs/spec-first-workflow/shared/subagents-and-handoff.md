# Subagents And Handoff

Detailed shared companion for `docs/spec-first-workflow.md`. Read this when planning subagent lanes, auditing subagent gates, resuming from artifacts, or rendering final handoff prompts.

## Read When

- A phase needs read-only fan-out, local-only or scoped-down rationale, or `Subagent Gate Audit` status.
- A session is resuming from artifacts and must choose the correct read order.
- A non-implementation phase boundary needs a copy-pastable next-session prompt, or implementation handoff must set a Codex Goal.

## Inputs

- Current phase file, task-local workflow-control artifacts, and the relevant reviewed spec/design/tasks context.
- `docs/subagent-contract.md` and `docs/subagent-brief-template.md` for reusable subagent brief shape.
- Active tool availability and explicit subagent authorization state.

## Outputs

- Subagent lane plan, lane summaries, fan-in status, or valid local-only/scoped-down rationale.
- Artifact-first resume order.
- Chat-only next-session prompt or Codex Goal implementation handoff.

## Stop Rule

Use this shared file to support the owning phase. It does not approve specs, design, tasks, implementation, or completion by itself.

## Shared Subagent Gate Audit

Non-trivial workflow-control records should also include a `Subagent Gate Audit` when a phase depends on fan-out, formal clarification, specification review, technical design review, task-ledger review, or an explicit local-only decision:

```text
Subagent Gate Audit:
- Trigger: <why lanes are required, scoped down, waived, or not expected>
- Gate type: <research fan-out | spec-clarification | specification-review | workflow-adequacy | design-authoring fan-out | technical-design-review | task-ledger-review | review/validation fan-out>
- Required lane policy: <default lens set | expanded lane set | scoped-down lane set | local-only rationale>
- Lane table: <lane id, agent, mode, lens/domain, owned question, skill/no-skill, inspect-first target, order/parallelism, read-only enforcement, status>
- Lane result summary: <strongest finding, classification, falsification check or decisive evidence test, recommended handoff, owner or reopen target, evidence pointer>
- Fan-in: <orchestrator resolution, action, owner/artifact updated, unresolved conflicts, accepted risks, proof obligations, why severity is not stronger/weaker when material>
- Gate result: <complete | blocked | waived | not_expected | PASS | CONCERNS | FAIL>
- Readiness consequence: <next phase allowed yes/no, with proof obligations when allowed with concerns>
- Reopen target: <required when blocked or failed>
```

`Subagent Gate Audit` statuses of `complete`, `PASS`, or `CONCERNS` are invalid when a required lane is missing or pending, a lane blocker or material severity conflict is unresolved, or scoped-down/local-only rationale does not explain why omitted lanes cannot change the decision. Missing audit for a non-trivial phase approval keeps the owning phase draft or blocked; later phases must reopen or repair that phase instead of inferring readiness.

`Lens` is metadata for coverage, not a replacement for `spec-clarification-challenge` classifications. Lane outputs use `blocks_spec_approval`, `blocks_specific_domain`, and `non_blocking_but_record`; shared handoffs use the classifications from `docs/subagent-contract.md`.

The orchestrator owns fan-in:

- deduplicate overlapping questions and findings;
- compare conflicting assumptions across lanes;
- classify each surviving point by strongest justified impact: approval blocker, domain reopen, record-only constraint, proof obligation, accepted risk, or no-action item;
- preserve a short fan-in table or equivalent status in the workflow-control file: lens, trigger or source, falsification check or decisive evidence test, strongest finding, classification, action, owner or reopen target, and why severity is not stronger or weaker when material;
- treat lane-level missing input, unresolved blockers, and material blocker-severity conflicts as blocking the relevant approval area until answered, explicitly waived or accepted as risk, or routed to the owning phase;
- update `spec.md` only with final reconciled outcomes, not raw lane transcripts;
- reopen research, design, planning, or a specialist lane when a finding exposes a missing owner decision.

## Subagents

Subagents are the normal read-only evidence surface for non-trivial decision work, not phase ceremony. Lanes must be narrow, question-owned, evidence-oriented, and reconciled by the orchestrator.

Read-only must be enforced by the actual execution choice. If a lane cannot reliably stay read-only, keep that question in the main orchestrator flow instead of delegating it.

Use `docs/subagent-contract.md` and `docs/subagent-brief-template.md` for reusable brief shape.

If the active tool surface exposes subagent spawning only after an explicit user request for subagents, delegation, or parallel agent work, the repository workflow must carry that request in next-session prompts instead of requiring the user to remember it manually. Before declaring spawning unavailable, use tool discovery for subagent or multi-agent spawn tools when none are visible. If a required lane cannot run solely because the current prompt lacks explicit authorization, record `Subagent gate: blocked: missing explicit subagent authorization`; do not downgrade the gate to `local_only`, `scoped_down`, `waived`, or `not_expected`.

This file is the canonical source for the exact authorization line. Phase files and skills should reference this section instead of restating the full prompt text.

Use this exact line in any non-trivial next-session or reopen prompt whose next phase may depend on research fan-out, specification review lanes, clarification challenge lanes, design fan-out, technical design review, task-ledger review, workflow-plan adequacy challenge, review fan-out, or validation fan-out:

```text
Subagent authorization: I explicitly request and authorize read-only subagents, delegation, and parallel agent work for every repository workflow gate that requires or benefits from fan-out in this session. Spawn the required read-only lanes without asking again; the orchestrator retains final authority and reconciles results.
```

Every lane needs:

- goal and exact question;
- scope and constraints;
- lens or specialist domain when multiple challenge lanes share the same artifact;
- expected output;
- evidence requirement;
- skill name or `no-skill`;
- read-only enforcement.

A lane uses at most one skill. If the selected skill defines a stricter deliverable, follow it. Otherwise use the shared envelope from `docs/subagent-contract.md`. Multi-lane challenge improves coverage only when lanes have distinct lenses and an explicit fan-in path.

Lane planning should be a coverage map:

- choose the independent questions that can change the current decision or gate;
- assign each question to the narrowest suitable expert lane;
- include sibling lens names when lanes share the same artifact bundle;
- merge duplicate lanes and record why omitted lenses cannot change the decision;
- preserve only compact lane summaries and reconciled outcomes in authoritative artifacts.

## Resume Order

Resume from artifacts, not chat memory.

If approved `tasks.md` exists and implementation, review, validation, or closeout is next:

1. read `tasks.md`;
2. read the artifacts named by `tasks.md`, usually reviewed `spec.md`, specification-review result, and any required design, test-plan, or rollout context;
3. read `workflow-plans/<phase>.md` only when `tasks.md` explicitly names a pre-created review or validation phase file.

If there is no approved `tasks.md` and `workflow-plan.md` exists:

1. read `workflow-plan.md`;
2. read the current `workflow-plans/<phase>.md` if the task uses one;
3. read the files named in the `Next session context bundle`;
4. then read phase artifacts as needed.

If there is no approved `tasks.md` and no `workflow-plan.md` because the task is direct or lean:

1. read `spec.md` when it exists;
2. read the specification-review record when non-trivial work is moving beyond specification;
3. read `tasks.md` when implementation or validation is next;
4. read optional `research/*.md` or `design/overview.md` only when named or needed.

Treat missing expected artifacts as incomplete unless an explicit waiver covers that exact artifact.

## Final Chat Handoff

When any non-implementation workflow phase reaches a boundary and a next session or reopen target exists, the final chat response must include a copy-pastable recommended next-session prompt derived from recorded artifacts. This is default behavior; the user does not need to ask the agent to stop or produce the handoff prompt.

Render the prompt in the final chat response only. Do not write the full ready-to-paste prompt into `workflow-plan.md`, `workflow-plans/*`, `spec.md`, `tasks.md`, an ad hoc prompt file, generated notes, or any other repository artifact. Artifacts may record the workflow state, next-session start point, context bundle, blockers, accepted risks, and proof obligations needed to regenerate the prompt; they must not become a second source of truth for the prompt text.

Assume the next session is context-blind: it can read repository files, but it cannot see the current chat. The prompt should carry a short task-specific context capsule that explains the current state, why the named next step is next, and what the next session must not lose. It should not become a transcript, broad project summary, or second copy of the artifacts.

Select context by relevance:

- include the current workflow state, accepted objective, and the reason this exact next phase or reopen target is next;
- include exact artifact paths, task IDs, phase names, commands, blocker names, accepted decisions, accepted assumptions, accepted risks, and proof obligations that matter to the next phase;
- include one-line reasons for non-obvious files in the context bundle so the next session knows why to read them;
- omit generic repository rules already covered by `AGENTS.md`, long artifact excerpts, resolved debate history, unrelated prior-session details, and context the next phase can cheaply rediscover from the listed files;
- when uncertain, include a bounded assumption or reopen target instead of padding the prompt with unrelated context.

The recommended prompt should be operational, not just descriptive. Include:

- the exact next phase or reopen target;
- the artifact read order, task-local paths, and short reasons for any non-obvious context files;
- the immediate objective and expected output for that one phase;
- important blockers, accepted assumptions, accepted risks, and proof obligations from recorded state;
- `Subagent authorization:` with explicit permission for read-only subagents, delegation, and parallel agent work for non-implementation next phases that may run subagent/readiness gates, or for implementation only when the approved `tasks.md` explicitly requires same-session read-only review, validation, or adequacy fan-out;
- a stop rule telling the next session to complete only that phase, update workflow state, and produce the following next-session prompt if another phase remains.

When the next phase is a design authoring checkpoint, the prompt must name exactly one checkpoint, usually `system-integration-design` first or `go-code-ownership-design` after system design is complete. It must tell the next session to first record or run the checkpoint-scoped `Design fan-out` and only then write or repair design artifacts for that checkpoint.

For implementation from approved `tasks.md` that has passed task-ledger review/readiness, compose the prompt with `.agents/skills/codex-goal-prompt-composer/SKILL.md`. The prompt must explicitly tell the next session to set a Codex Goal first, then orchestrate implementation of all required tasks in the approved ledger from start to finish. It must not rely on a slash command being parsed from the handoff. It should tell the next session to delegate every code-writing bundle to named isolated CLI workers and run the ledger's named proof without stopping between task IDs. If worker launch or resume fails before a usable patch exists, the session must stop blocked and report the failure instead of editing inline. It must still prohibit creating or approving missing pre-code workflow artifacts during implementation.

Implementation goal handoff rules:

- use `codex-goal-prompt-composer` whenever the recommended next-session prompt sets a Codex Goal;
- apply that skill's Goal Line Quality Gate before returning the prompt;
- start the fenced prompt with `First, set a Codex Goal for this session:` followed by a short durable goal objective;
- the next paragraph must say `After the goal is set, orchestrate implementation of every required task in <tasks.md path> from start to finish`;
- derive `<approved objective>` and `<verifiable completion condition>` from the `tasks.md` Goal Contract and implementation-readiness handoff;
- scope the goal to all executable tasks in the approved ledger, from the recorded first task or checkpoint through final validation, not just the first task ID;
- keep the Codex Goal objective as a durable objective only; do not pack artifact lists, constraints, risks, commands, or detailed execution rules into it;
- put all execution details under `Implementation brief` so the durable goal stays stable while the working instructions remain readable;
- include working directory, artifact read order split into required start context and task-specific context when available, task-local paths, accepted constraints, accepted risks, proof obligations, and named validation commands or manual proof;
- include `Implementation execution mode` from `tasks.md`. For code-writing implementation, include worker boundary summary and launch contract, and state workers are write-capable implementation delegates, while the orchestrator owns patch intake, integration proof, and ledger updates;
- include an `Orchestrator control posture` block that makes acceptance ledger-bound, evidence-based, and non-taste-based;
- include `Subagent/readiness gates: <status, evidence artifact, proof obligations, reopen target if blocked>` whenever the next phase depends on a subagent/review gate or local-only rationale;
- if readiness is `CONCERNS` or `WAIVED`, keep the Codex Goal objective focused on the approved objective and put the concern, waiver rationale, and required proof obligations in the implementation brief;
- tell the next session to update only ledger-owned progress/evidence and closeout surfaces permitted by `tasks.md`;
- if the `tasks.md` Goal Contract is missing, conflates blocked-stop with successful completion, or is too vague to form a verifiable Codex Goal, do not invent a broad objective; reopen planning to repair the Goal Contract or mark the implementation prompt as blocked;
- include a blocked-stop rule: if an implementation-blocking decision, missing artifact, unavailable required command, or failing proof cannot be resolved inside the approved ledger, stop with the Goal blocked, record the blocker in the allowed ledger/closeout surface, leave affected tasks unchecked, and return the exact reopen target instead of inventing new workflow artifacts or marking completion.

Recommended implementation prompt shape:

```text
First, set a Codex Goal for this session:
Complete <approved objective> by orchestrating every required task in `<task-local>/tasks.md` without stopping until <verifiable completion condition>.

After the goal is set, orchestrate implementation of every required task in `<task-local>/tasks.md` from start to finish. Start at <T001 or recorded checkpoint>, delegate every code-writing bundle to named worker bundles; do not make orchestrator-authored implementation edits; continue through the ledger's final validation/proof, and do not redefine success around a smaller slice.

Implementation brief:

Work in `<absolute repo path>`.
Read before coding:
- `<task-local>/tasks.md` because it is the approved implementation ledger and source of truth.
- `<task-local>/spec.md` because it is the canonical decision record.
- <additional required start artifacts named by `tasks.md`, each with a one-line reason>.

Read before relevant tasks:
- <task-specific artifacts named by `tasks.md`, with task IDs and one-line reasons>.

Current state:
- Next phase: implementation.
- Task-ledger review: <PASS | CONCERNS | WAIVED>.
- Implementation readiness: <PASS | CONCERNS | WAIVED>.
- Implementation execution mode: <isolated-cli-worker required | not_expected>.
- Subagent/readiness gates: <status, evidence artifact, proof obligations, reopen target if blocked>.
- First executable task/checkpoint: <T001 or named checkpoint>.
- Accepted concerns or waiver: <none | named concern/waiver plus proof obligation>.
- Isolated worker boundaries: <not_applicable | task bundles, launch contract, launch/resume failure blocker, discovery hints plus explicitly marked hard boundaries, forbidden edits, parallelism rule, worker proof, orchestrator patch intake, integration proof>.

Orchestrator control posture:
- Treat `tasks.md` as the acceptance contract. Every worker assignment, rejection, repair prompt, patch-intake action, and completion claim must name the task ID, source anchor, obligation, proof gap or proof result, and approved boundary.
- Be strict about incomplete scope, missing proof, forbidden edits, unapproved ownership, generated/manual authority drift, hidden design decisions, invented validation policy, and code-quality issues with a concrete task quality bar, repository pattern, owner responsibility, proof gap, or changed-surface maintainability-risk anchor. Do not be strict about unsupported taste preferences when ledger obligations are satisfied.
- Send worker patches back only with concrete repair instructions: exact failed obligation, evidence anchor, expected patch/proof, unchanged boundaries. No vague "clean up" or "finish properly" feedback.
- Intake sequence: accept; resume same worker for narrow same-bundle repair; rerun or split worker when structure is off; otherwise record blocker and reopen owning phase. Do not manually author implementation patches.

Execution rules:
- Orchestrate all required tasks in `tasks.md` in dependency order through the ledger's named proof; do not stop between task IDs unless blocked.
- For code-writing implementation, use isolated CLI workers only for the named reviewable diff-story bundles in isolated worktrees or fresh checkouts. Default launch is `codex exec --cd <isolated-worktree> --sandbox workspace-write --ask-for-approval never --strict-config --json --output-schema <schema> -o <final-output> - < <prompt-file>` with event output captured, strict schema used when available, user config/rules inherited, no required named profile, and no forced model/reasoning overrides unless the approved ledger names a task-specific rationale. Do not use `--full-auto`; use `--yolo` or `danger-full-access` only in a ledger-named externally hardened container/VM/CI runner. If launch or resume fails before a usable worker patch exists, stop with the Goal blocked and report the failed command, key output, affected bundle, and repair target; do not author the implementation yourself. Treat worker output as patch evidence only and `ready_for_intake` as advisory: inspect the diff, reject forbidden edits or unapproved decisions, apply only accepted changes, rerun proof in the integration workspace, and update `tasks.md` only after integration proof passes.
- For narrow same-bundle patch-intake defects, resume the same worker session with `codex exec resume` in the same isolated workspace before starting over. Repair prompts must name one failed obligation and request the smallest patch that fixes only that obligation while preserving accepted in-scope changes. Do not use resume to change scope, ownership, contract, design, dependency, rollout, cleanup disposition, or validation policy.
- Treat worker discovery hints as starting points, not exhaustive edit permissions, unless the ledger explicitly marks them as hard boundaries. Workers may inspect callers, siblings, owners, generated/manual authorities, and adjacent files needed for the assigned task; edits still must stay inside assigned scope and avoid forbidden/protected surfaces. If completing a worker assignment requires changing approved scope, ownership, contract, validation policy, workflow artifacts, or forbidden surfaces, the worker must stop and return a blocker instead of broadening the patch.
- Do not rely on current Codex App thread state, runtime-only tools, chat memory, or unsaved app context reaching CLI workers. Put task-critical context in the worker prompt or approved artifacts.
- Write worker prompts to temporary files, pass them to `codex exec` through explicit stdin marker `- < "$WORKER_PROMPT"`, and delete them after the worker finishes or is abandoned. Do not preserve prompt files as workflow artifacts.
- Tell CLI workers to choose skills/tools by task need, not availability; invoke only skills that directly map to the assigned surface, otherwise state `no-skill`. For Go code changes, use the repo-local `go-coder` skill when available. Treat `SOUL.md` as lower-precedence engineering taste, never authority over `AGENTS.md`, approved artifacts, or the assignment.
- Preserve the accepted constraints, non-goals, risks, and proof obligations recorded in the listed artifacts.
- Apply the task-local implementation quality bar and evidence format from `tasks.md`.
- Before editing each protected-domain or `test-plan.md`-referencing task, expand its `Source`/`Implementation obligations` into concrete assertions: source anchor -> invariant -> forbidden regression or side effect -> owning code path -> required proof. If this cannot be made concrete, stop and reopen planning or test design.
- Do not check tasks or claim completion from any unproven obligation, skipped, unavailable, stale, failing, or too-narrow proof; leave affected tasks unchecked and record `Blocked:` or the narrower claim.
- Do not create or approve missing pre-code workflow artifacts during implementation.
- Update existing `tasks.md` progress/evidence and any ledger-owned closeout surfaces exactly as allowed by `tasks.md`.
- If blocked by a missing decision, missing artifact, unavailable required command, or unresolved failing proof outside the approved ledger, stop with the Goal blocked, record the blocker/evidence/exact reopen target, and do not mark completion.
```

Use no prompt when the workflow is honestly done.

The prompt is chat-only. It is not a workflow artifact and must not become a second source of truth. If an artifact currently contains the full prompt body, trim it back to routing state and context-bundle entries unless the user explicitly asked for a standalone prompt document.

Before returning the prompt, apply the start test: a new session with no chat history should know the single next phase, why it is next, what to read first, what constraints and proof obligations matter, and where to stop. Remove any sentence that does not help that session start or avoid a real mistake.
