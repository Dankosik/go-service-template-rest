# Implementation, Validation, And Closeout Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when implementing from an approved ledger, running post-code review/reconciliation, validating, or closing out.

## Read When

- Implementation, review, validation, or closeout is next from an approved `tasks.md`.
- Post-code proof, negative legacy-surface proof, generated or mirror drift proof, or ledger-owned closeout updates are required.
- Validation exposes a missing decision, missing proof path, or old surface that may require reopening an earlier phase.

## Inputs

- Approved `tasks.md` first, then artifacts named by that ledger, including `test-plan.md` when the ledger references scenario IDs.
- Existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when the ledger explicitly names that pre-created phase file.
- Repository-owned validation commands and current workspace state.

## Outputs

- Code, tests, config, generated output, or docs required by the approved ledger.
- Updated existing `tasks.md` progress/evidence and ledger-owned closeout surfaces.
- Fresh validation evidence, or a recorded blocker with exact reopen target.

## Stop Rule

Do not create or approve missing pre-code workflow artifacts after implementation starts. If proof or validation exposes a missing decision, missing `test-plan.md` scenario, or missing artifact, record the blocker in the ledger or closeout surface permitted by `tasks.md` and reopen the owning earlier phase.

## Ledger-First Execution

Implementation starts from an approved, reviewed `tasks.md` or it does not start. Read `tasks.md` before `workflow-plan.md`, phase-control files, prior chat, or broad repository search. Then read only the artifacts the ledger names under `Read before coding` and the task-specific artifacts it names under `Read before relevant tasks`.

Before the first edit, read current workspace state and separate pre-existing unrelated changes from this session's work. Do not use unrelated dirty files as task evidence, do not clean them up without an explicit request, and do not let their presence widen the ledger's approved scope.

Before the first edit, set or verify the Codex Goal from the `tasks.md` Goal Contract and implementation handoff. The goal must cover every required ledger task through final validation, not only the first task or checkpoint. Also confirm the ledger records implementation readiness as `PASS`, eligible `CONCERNS`, or eligible `WAIVED`; identifies the first executable task or checkpoint; and separates successful completion from blocked-stop behavior. If readiness, the Goal Contract, `test-plan.md` scenario mapping when referenced, or the completion condition is missing, stale after repair, `FAIL`, or too vague to determine the next executable task and final proof, stop and reopen planning, test design, specification review, technical design review, or the owning earlier phase named by the gap.

At the start of each task, bind the task ID to its source anchor, implementation obligations when present, dependencies, owner package/file, test-design scenario ID when present, proof, evidence fields, and stop or reopen condition. For protected-domain work or any task referencing `test-plan.md`, expand every referenced scenario/proof-obligation row into concrete assertions: source anchor -> invariant -> forbidden regression or side effect -> owning code path -> required proof. Do not infer hidden architecture, ownership, dependency, rollout, scenario class, proof level, or validation choices from chat memory. If a required field or assertion is absent, broad, or contradicts approved artifacts, leave the task unchecked and reopen planning, test design, or the owning earlier phase instead of deciding during implementation.

Execute tasks in dependency order through the ledger's final proof unless blocked. After each task or checkpoint, update the existing `tasks.md` checkbox and evidence fields with the command or read performed, result, key output or evidence reference, changed proof files when relevant, and any residual blocker or narrower claim. A task remains unchecked when any named obligation lacks proof, or when proof is skipped, unavailable, stale, failing, cached in a way that does not prove the claim, or narrower than the task's stated behavior.

On resume, read current workspace state and `tasks.md` first, then continue at the first unchecked task whose dependencies and checkpoint gates are satisfied. Re-run only the proof needed to detect drift unless the ledger, changed surface, or failing evidence requires broader validation.

## Implementation Goal Orchestrator Role

During an implementation Goal, the orchestrator is the integration controller for the approved ledger. It does not replace `tasks.md` with a private plan, accept worker output on confidence, or reopen design questions silently. It delegates ledger-bounded implementation bundles, verifies patches against approved obligations, and updates authoritative progress only after integration proof.

Orchestrator control must be strict and evidence-based:

- Hold `tasks.md` as the acceptance contract. Every worker assignment, rejection, repair prompt, patch-intake action, and completion claim names the task ID, source anchor, implementation obligation, expected proof, and approved boundary it depends on.
- Delegate only reviewable diff-story bundles already allowed by `Implementation execution mode`. Do not ask a worker to "figure out" missing scope, ownership, contract semantics, dependency choices, rollout shape, cleanup disposition, or validation policy.
- Reject or send back incomplete worker patches when an approved obligation is missing, proof is skipped or too narrow, generated/manual authority is wrong, forbidden files changed, unrelated cleanup appears, ownership placement is unapproved, or the diff invents a decision.
- Do not nitpick worker output on taste alone. Blocking quality findings must name a concrete task quality bar, repository pattern, owner responsibility, proof gap, or changed-surface maintainability risk. If behavior, ownership, proof, and maintainability constraints satisfy the ledger, accept the patch even when a different local style would also work.
- Give no vague repair feedback. A repair prompt must cite the exact task ID, file or evidence anchor, failed obligation, observed gap, required patch/proof, and unchanged boundaries. "Make it better", "clean this up", or "finish properly" is not a valid worker instruction.
- Use the shortest valid intake path: accept patch; resume same worker for narrow same-bundle repair; rerun/split worker when the patch is structurally off; otherwise record blocker and reopen the owning phase. Do not manually author implementation patches.
- Keep workers under pressure until the assigned bundle is done or honestly blocked. Do not mark a task complete because the worker reports success, because most code exists, or because remaining gaps look small.
- Run required proof in the integration workspace before checking tasks or closeout surfaces. Worker-local proof is supporting evidence only.

## Isolated CLI Worker Execution

Isolated Codex CLI workers are an implementation tactic for reducing context bias and constraining write scope. They do not replace the implementation Goal, task-ledger authority, or orchestrator closeout responsibility.

Use an isolated worker only when all are true:

- `tasks.md` has passed task-review/readiness with `PASS`, eligible `CONCERNS`, or eligible `WAIVED`;
- the approved ledger or implementation handoff marks `Implementation execution mode` as `isolated-cli-worker required` and names worker boundaries;
- the worker unit is one reviewable diff story or an explicitly coupled task bundle, not an arbitrary single checkbox split that would separate proof-first tests from the implementation they prove;
- owner package/file, generated or mirrored source authority, proof, evidence fields, and stop/reopen condition are already approved for the worker unit;
- a separate worktree, fresh checkout, container workspace, or equivalent isolated filesystem is used so the worker cannot mutate the orchestrator's integration workspace while producing the patch.

Do not add isolated-worker mode during implementation. If the execution mode or worker boundaries are missing, stale, vague, or authority-confusing, reopen planning or task-review/readiness. When worker launch or resume fails before a usable patch, stop hard: report the failed command, key output, task bundle, and smallest repair target to the user. Do not author the implementation yourself. Do not use isolated write workers when the next step needs a missing specification, design, test-design, planning, readiness, dependency/OSS, Pattern Fit, source-of-truth, or validation decision. Do not use parallel write workers for overlapping owner files, generated outputs, migrations, schemas, fixtures, package initialization, shared test helpers, global config, or checkpoint-dependent tasks. When overlap is possible, run workers serially or keep the work in the orchestrator.

The orchestrator owns worker setup. Default launch shape:

```bash
codex exec \
  --cd "$WORKTREE" \
  --sandbox workspace-write \
--ask-for-approval never \
--strict-config \
--json \
--output-schema "$WORKER_SCHEMA" \
-o "$WORKER_OUT/final.md" \
- < "$WORKER_PROMPT" > "$WORKER_OUT/events.jsonl"
```

Do not require a named profile or hard-code user-specific tools in the template. By default, do not pass `--ignore-user-config` or `--ignore-rules`; let Codex inherit the user's active config, MCP servers, plugins, hooks, feature flags, and local tooling. Do not force `--model` or `-c model_reasoning_*` overrides unless the approved ledger names a task-specific rationale; avoid downgrading user defaults accidentally. The worker may use any inherited tool that helps the assigned task, but an unavailable optional tool is not a blocker unless the approved ledger made it required.

Use `--output-schema "$WORKER_SCHEMA"` when the launch harness can provide a schema; otherwise require the same fields in the worker's final message and record that schema enforcement was unavailable. Schemas must be strict: top-level `additionalProperties: false`, and every property listed in `required`. The minimum result fields are patch summary, changed files, commands run, key output, residual risks, blockers, and ready-for-intake status. Use `--add-dir` only for ledger-named extra paths. Do not use deprecated `--full-auto`. Do not use `--yolo` or `--sandbox danger-full-access` unless the worker runs inside an externally hardened container, VM, or CI runner with no ambient secrets and the approved ledger explicitly names that broader access. Do not use `--ephemeral` for task-bearing workers because patch intake needs the run history.

If patch intake finds a narrow in-scope defect, resume the same worker session instead of starting over:

```bash
codex exec resume "$WORKER_THREAD_ID" --cd "$WORKTREE" --sandbox workspace-write --ask-for-approval never --strict-config --json --output-schema "$WORKER_SCHEMA" -o "$WORKER_OUT/final-repair.json" - < "$REPAIR_PROMPT" > "$WORKER_OUT/events-repair.jsonl"
```

Use resume only for same task bundle, same isolated workspace, and same approved boundaries. Repair prompts are transient like worker prompts. Do not resume worker to choose new scope, ownership, contract, design, dependency, rollout, cleanup disposition, or validation policy; record blocker/reopen target instead.

Worker prompt file lifecycle:

```bash
WORKER_PROMPT="$(mktemp -t codex-worker-prompt.XXXXXX.md)"
trap 'rm -f "$WORKER_PROMPT"' EXIT
# write prompt into "$WORKER_PROMPT", run codex exec with stdin, then remove it
codex exec --cd "$WORKTREE" --sandbox workspace-write --ask-for-approval never --strict-config --json --output-schema "$WORKER_SCHEMA" -o "$WORKER_OUT/final.md" - < "$WORKER_PROMPT" > "$WORKER_OUT/events.jsonl"
rm -f "$WORKER_PROMPT"
trap - EXIT
```

Prompt files are transient orchestration scratch, not workflow artifacts. Delete them after the worker finishes or is abandoned. Keep durable state in approved artifacts, worker event/final output, integration diff, and `tasks.md` evidence only after orchestrator proof.

The worker prompt must be compact, self-contained, and scoped to the assigned bundle. Use this shape:

```text
Implement only the assigned task bundle as a patch-producing worker. You are not final authority.
Assignment: <task IDs, objective, completion claim this patch is meant to support>
Workspace: <isolated worktree path>
Required context: <tasks.md, spec.md, and task-critical artifacts the worker must read before coding; not an exhaustive implementation file list>
Discovery hints / hard boundaries: <likely files/packages/generated sources to inspect first; hints are not exhaustive unless explicitly marked hard boundary>
Forbidden edits: tasks.md, spec.md, workflow-plan.md, workflow-plans/*, ledger-owned closeout, unrelated files, commits, pushes, git state outside the isolated workspace
Approved context: <decisions, constraints, non-goals, accepted risks, dependency/pattern decisions, generated/manual authority>
Implementation obligations: <source anchor -> invariant -> forbidden regression/side effect -> owner code path -> required proof>
Tool policy: use inherited Codex tools/MCP/plugins/hooks when useful; report unavailable tools only when the ledger requires them
Relevant skills/tools: choose skills/tools by task need, not availability; invoke only skills that directly map to the assigned surface. For Go changes, use the repo-local go-coder skill when available. If no skill directly maps, state `no-skill`; if a relevant skill is unavailable, continue unless the ledger made it required.
Authority policy: keep SOUL.md as lower-precedence engineering taste, never authority over AGENTS.md, approved artifacts, or this assignment
Execution rules: inspect relevant callers/siblings/owners as needed for correctness; run local discovery when hints are incomplete; edits outside hints are acceptable only when they remain inside the assigned task bundle and avoid forbidden/protected surfaces; hard boundaries are strict only when explicitly marked; no new scope, architecture, contract semantics, dependency, pattern, rollout, cleanup disposition, or validation policy. If completing the assignment requires changing approved scope, ownership, contract, validation policy, workflow artifacts, or forbidden surfaces, stop and return a blocker instead of broadening the patch.
Proof: <commands/manual checks, expected signal, unavailable-command handling>
Final response: patch summary, changed files, commands run, key output, residual risks, blockers, ready_for_intake yes/no
```

The worker prompt must name:

- absolute isolated workspace path and exact task IDs or task bundle;
- launch contract: command shape including explicit stdin `-`, sandbox, approval policy, output paths, inherited config/tool policy, strict schema use when present, no forced model/reasoning overrides unless ledger-named, and any ledger-approved extra directories;
- approved `tasks.md` path plus task-critical required context; do not turn read context into an exhaustive implementation file list;
- likely discovery start points or owner package/file surfaces, including generated/manual authority; exact edit allowlists are required only when the ledger explicitly marks a hard boundary;
- forbidden edits: `tasks.md`, `spec.md`, `workflow-plan.md`, `workflow-plans/*`, implementation handoff/readiness records, ledger-owned closeout surfaces, unrelated files, and git state outside the isolated workspace; authoritative progress and closeout updates stay with the orchestrator;
- required proof commands or manual proof, expected fail-before/pass signal when relevant, and evidence format;
- stop rule: report a blocker instead of choosing new scope, owner, architecture, contract semantics, dependency, pattern, rollout, generated authority, cleanup disposition, validation policy, workflow-artifact edit, or forbidden surface change;
- relevant skill/tool policy, including task-need-based skill invocation and `no-skill` when no skill directly maps;
- output shape: patch summary, changed files, commands run, key output, residual risks, blockers, and whether the patch is ready for orchestrator intake.

Worker output is evidence, not completion. Treat `ready_for_intake=yes` as advisory only. The worker must not mark ledger tasks complete, edit authoritative workflow artifacts, commit, push, merge, stage orchestrator-owned files, or claim final readiness. If the worker edited forbidden files, made an unapproved decision, broadened scope, skipped required proof without a blocker, or mixed unrelated cleanup into the patch, reject the patch or split it before integration.

Patch intake is an orchestrator duty:

1. Inspect the worker diff before applying it to the integration workspace. Confirm every changed file maps to the assigned task bundle, generated/mirrored outputs follow the approved source order, no forbidden artifact changed, and no new dependency, pattern, architecture, public contract, data behavior, security behavior, rollout behavior, or validation policy was invented. For code-quality objections, name the concrete task quality bar, repository pattern, owner responsibility, proof gap, or changed-surface maintainability risk.
2. Apply or merge only the accepted patch. Resolve only mechanical conflicts that stay inside the approved ledger; otherwise rerun the worker, repair the patch manually inside approved scope, or record a blocker/reopen target.
3. Run the task proof and any checkpoint/final validation in the integration workspace after applying the patch. Worker-local proof is useful evidence but cannot by itself satisfy task evidence, checkpoint gates, or closeout.
4. Update existing `tasks.md` progress/evidence only after integration-workspace proof covers the task claim. Name worker evidence as supporting context when useful, but do not substitute it for fresh integration proof.

For narrow same-bundle patch-intake defects, prefer `codex exec resume` on the same worker session and workspace before starting a new worker. Repair prompts must name one failed obligation and request the smallest patch that fixes only that obligation while preserving accepted in-scope changes. This is repair, not a new delegation scope.

When multiple isolated workers run during one implementation Goal, keep a visible worker map in the implementation scratch context or existing ledger evidence:

```text
Worker | Session id | Task bundle | Isolated workspace | Discovery hints / hard boundaries | Status | Patch intake | Integration proof | Ledger update
```

This map is not a new workflow artifact and must not become a second task ledger. It exists only to prevent overlapping writes and make closeout auditable.

## Patching, Review, Reconciliation, And Validation

Implementation patching consumes the approved task handoff. The implementation path is worker-produced patches plus orchestrator intake; orchestrator-authored implementation edits are not allowed. A worker patch may create or edit code, tests, migrations, configs, generation inputs, generated output, or docs required by the task ledger.

Before adding substantial code to an existing hand-written source file, inspect its current responsibility, sibling files in the package, and package owner. Record or satisfy the ledger's owner-file/package placement decision before editing. If the new code is a distinct concern, abstraction level, mapping, validation, lifecycle, adapter, or test-helper policy, place it in a focused same-package seam file or the correct owner package instead of enlarging a catch-all file. If that split would change approved architecture, public contract, dependency direction, generated-source ownership, or another protected decision, stop and reopen the owning phase.

An implementation patch may use the selected dependency or custom approach recorded in approved artifacts. If implementation discovers that the chosen approach needs a new runtime dependency, custom infrastructure, or a material helper/abstraction not covered by dependency/OSS due diligence, stop and reopen specification, technical design, or planning according to where the decision belongs. Do not add the dependency or build the custom substitute silently inside implementation.

An implementation patch may implement the selected design or system-design pattern recorded in approved artifacts. If implementation discovers that the chosen shape needs a different pattern, a previously rejected pattern, or a custom design not covered by Pattern Fit Diligence, stop and reopen research, specification, technical design, or planning according to where the decision belongs. Do not introduce a new pattern or pattern-like abstraction during implementation just because it seems cleaner locally.

An implementation patch may use local code-level patterns only to simplify approved behavior. Good candidates include table-driven tests for several meaningful cases, guard clauses, same-package policy seams, first-class function strategy, narrow consumer-owned interfaces, map-driven dispatch, middleware or decorator only at an existing composition seam, and functional options only when optional construction has real combinatorial pressure. If the pattern adds files, interfaces, callbacks, option bags, or indirection without reducing duplication, branch complexity, ownership ambiguity, or test burden, inline the code or use the stdlib/repo idiom instead.

Cleanup made necessary by the approved task is part of implementation scope. An implementation patch removes stale old-path code and adjacent artifacts, refactors old code into the active path when that is the approved target state, or stops at the smallest reopen target when retention/removal would change public contract, data behavior, security, reliability, rollout, generated contracts, or another protected domain.

If implementation discovers an old surface not named by the approved spec or ledger, classify it before editing: in-scope and safe to remove/refactor, intentionally retained by an existing approved artifact, or requiring reopen because removal or retention changes contract, data, security, reliability, rollout, generated-source, or another protected-domain behavior.

Implementation Goal sessions may continue across the approved `tasks.md` items and the ledger's named proof checks. They must not use implementation momentum to create or approve missing specification, design, test-design, planning, review, or validation-phase artifacts.

Post-code work is ledger-driven. It may update only:

- existing `tasks.md` checkbox/progress state;
- ledger-owned `spec.md` `Validation` and `Outcome` when the approved `tasks.md` requires closeout and the task has a spec;
- existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when the approved `tasks.md` explicitly names that pre-created phase file as part of the post-code checkpoint.

Do not update `workflow-plan.md` or phase-control files merely because they exist. After `tasks.md` is approved, those files are not the implementation source of truth.

Do not create new workflow/process artifacts after implementation starts. Reopen the earlier phase that owns the missing artifact instead.

Review is read-only and risk-driven. Review findings are advisory until the orchestrator reconciles them. During orchestrator reconciliation or implementation, fix findings that are inside the approved ledger and proof path. If a finding requires a new decision, missing test-design scenario, missing artifact, broader validation policy, changed dependency choice, generated-source authority change, or retention/removal decision outside the approved ledger, record the blocker in the ledger or closeout surface permitted by `tasks.md` and reopen the owning earlier phase instead of creating a new pre-code artifact after coding starts.

Review should flag unexplained surviving replaced or unused code, tests, fixtures, configs, docs, generated artifacts, skills, agents, or mirrors as merge-risk findings unless an approved artifact records why the surface remains with owner, reason, proof, and exit condition.

Review should also flag custom implementations, newly added dependencies, or meaningful helper abstractions that lack approved stdlib, repository-pattern, and OSS due diligence. Severity depends on ownership risk, security/license exposure, transitive dependency cost, and whether a mature maintained library or standard-library path appears to satisfy the same contract.

Review should also flag architecture, workflow, integration, resilience, data-flow, or abstraction shapes that lack approved Pattern Fit Diligence when they appear invented, cargo-culted, or inconsistent with the selected pattern. Severity depends on whether the missing comparison could change ownership, interfaces, failure behavior, validation, rollout, or idiomatic Go implementation shape.

Review should also flag verbose local code that missed an obvious Go-native simplification or small code-level pattern, and pattern-shaped code that adds indirection without reducing duplication, branch complexity, ownership ambiguity, or test burden.

Review should also flag hand-written source files that grew into mixed-responsibility, multi-abstraction-level, or hard-to-review catch-all modules when the approved artifacts did not justify that placement. Severity depends on whether the file now hides ownership, couples unrelated concerns, blocks focused tests, or makes future changes likely to land in the wrong owner.

Severity for unexplained surviving replaced paths is risk-based: `high` when the old path can still execute, import, generate, or validate; `medium` for test, fixture, doc, config, skill, agent, or mirror drift; `low` only when the surface is clearly unreachable, non-authoritative, and unlikely to mislead future work.

Post-code review findings close only when the task evidence names the finding, the action taken, the proof that covers it, and the residual risk or narrower claim. A reviewer report, subagent summary, cached command, or green unrelated check is not proof by itself.

Validation uses fresh evidence matched to the changed surface and the ledger's completion condition. A closeout claim is valid only when the commands or manual proof actually cover that claim, including targeted behavior proof, repository-owned validation commands, targeted negative searches or reads for retired identifiers and references where text proof is reliable, retained-surface proof when old artifacts remain, generated or mirror drift proof when owning sources changed, and whitespace/drift checks for changed docs or tooling.

Negative proof must name the retired identifiers, paths, commands, config keys, generated files, fixtures, docs, skills, agents, or mirrors searched. A generic search such as `rg legacy` is not sufficient unless the retired surface is literally named `legacy`.

Generated or mirrored-surface proof must name the authoritative source, generator or sync command, expected derived paths, and drift check. If the source changed but the generated or mirrored output cannot be regenerated or proven current, leave the relevant task unchecked and record the missing command, failing output, or unavailable environment blocker. A drift command that fails because task-owned generated files are dirty is still failing proof unless the approved ledger named narrower deterministic freshness proof plus a final clean-tree gate; record the narrower claim and leave the clean-tree claim unchecked. Do not hand-edit generated or mirrored output unless the approved ledger explicitly authorizes the path.

Validation evidence should be no broader or narrower than the claim. A package-level test can prove a package claim; repository readiness needs the repository-owned command set for the changed surfaces; generated API, migration, sqlc, docs, agent, skill, or mirror changes need their drift checks when those surfaces are in scope. When a required command is unavailable, record the unavailable command, why it could not run, the narrower evidence that did run, and the residual unverified claim.

Before closeout, map every changed file from the final diff to a `tasks.md` task, checkpoint, or ledger-owned closeout surface, and verify that the task evidence names the proof covering that file. Any changed file that cannot be mapped is either unrelated pre-existing work to leave alone, an accidental edit to remove, or a blocker/reopen signal because the approved ledger did not cover it.

Closeout is complete only when all required ledger tasks and checkpoint gates are checked with current evidence, required validation passes, ledger-owned `spec.md` `Validation` and `Outcome` updates are current when the task has a spec, and any pre-created review or validation phase file explicitly named by `tasks.md` is updated. Do not claim completion from a recorded blocker, unchecked task, stale proof, or proof that validates a neighboring surface instead of the changed one.

Final responses must clamp the claim to the recorded evidence. Use `done`, `complete`, `passed`, or equivalent success language only when the ledger completion condition is satisfied with fresh proof; otherwise report `blocked`, `partially verified`, or `not verified` with the exact failing or missing proof and reopen target.

If implementation or validation discovers legacy cleanup that cannot be completed inside the approved scope, record the blocker in the allowed ledger or closeout surface and reopen the smallest owning phase: planning for missing tasking/proof, technical design for new ownership/tooling semantics, or specification for changed scope, public contract, protected-domain behavior, or retention policy.

Blocker records must be specific enough for the reopened phase to act without chat history. Include the task ID or checkpoint, failing or missing command/read, exact missing decision or artifact, affected surface, evidence already gathered, tasks left unchecked, narrower claim if any, and the owning reopen target. A blocker is a valid stop, not a successful closeout claim.
