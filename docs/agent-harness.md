# Agent Harness

The workflow instructions in this repository are harness-neutral. This document owns harness detection and the mapping from neutral workflow concepts to the native controls of each supported harness. `AGENTS.md` still owns authorization; the workflow phases still own when a control may be used.

## Read When

- Deciding which harness-native tool implements a workflow concept: durable execution control, implementation worker, read-only subagent lane, model selection, or reasoning effort.
- Dispatching a worker or subagent and selecting its model and reasoning effort.
- Writing instructions, briefs, or handoffs that must work in both harnesses.

## Harness Selection

- Identify the current harness from the session environment before dispatching any harness control: the Codex App or Codex CLI runs the Codex-native path; Claude Code (CLI, desktop, web, or IDE) runs the Anthropic-native path; Qwen Code (CLI or IDE) runs the Qwen-native path.
- Use only the current harness's native controls. Do not emulate another harness's controls, shell out to its CLI, or mix the two control planes within one outcome.
- In the Codex App, this repository's term **implementation Worker** means a separate top-level App task/chat started in **Worktree** mode through the App's native task/thread control, so it owns a Codex-managed worktree ([OpenAI Worktrees](https://learn.chatgpt.com/docs/environments/git-worktrees.md)). A user request to use `Workers` for implementation is an explicit request for those separate App tasks; it is not a request for a nested agent whose role happens to be named `worker`.
- Treat every Codex App `collaboration.spawn_agent` lane as a built-in read-only subagent, including one dispatched with `agent_type: "worker"`; never use it for an implementation write lane or as a substitute for the separate App task above ([OpenAI Subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents.md)).
- When the current session lacks the needed native control, do the work root-locally under the owning phase's local-execution rules and state the missing control; do not invent a substitute.

## Instruction Loading Map

Keep cross-harness gates in `AGENTS.md`, because every supported harness loads
that file through its native bootstrap. Keep phase, shared, and harness method
behind the conditional read gate; loading every owner permanently spends
context and attention without making its trigger more precise.

| Harness | Guaranteed bootstrap | Conditional content | Current verification |
| --- | --- | --- | --- |
| Codex | Codex builds the `AGENTS.md` chain before work, once per run, subject to `project_doc_max_bytes` ([AGENTS.md discovery](https://developers.openai.com/codex/guides/agents-md)). | Skill name/description metadata is discoverable first; full `SKILL.md` loads only after selection. Custom subagents exist only when their `[agents.<name>].config_file` registry entry resolves ([Skills](https://learn.chatgpt.com/docs/build-skills), [Config reference](https://learn.chatgpt.com/docs/config-file/config-reference#configtoml)). | Start a fresh run with `codex --ask-for-approval never "List the active instruction sources and summarize their loading gate."`; use the session JSONL or opt-in TUI log to inspect the actual chain. |
| Claude Code | Root `CLAUDE.md` imports `AGENTS.md` with `@AGENTS.md`; imports enter startup context. | Skills remain discoverable until invoked. Ordinary custom subagents receive the loaded `CLAUDE.md` hierarchy; use a role's `skills` field only when its full skill content must be present at lane startup ([Memory](https://code.claude.com/docs/en/memory), [Subagents](https://code.claude.com/docs/en/sub-agents)). | `/memory` shows loaded instruction files. Use the `InstructionsLoaded` hook when a trace must prove which file loaded, when, and why. |
| Qwen Code | Qwen loads project `AGENTS.md`, `QWEN.md`, and local context at session start ([Memory](https://qwenlm.github.io/qwen-code-docs/en/users/features/memory/)). | Skills are model-invoked from their descriptions and their bodies load when selected; project subagents are discovered from `.qwen/agents/` ([Skills](https://qwenlm.github.io/qwen-code-docs/en/users/features/skills/), [Subagents](https://qwenlm.github.io/qwen-code-docs/en/users/features/sub-agents/)). | `/memory` lists loaded instruction files; `/skills` and `/agents` expose the discoverable adapters. |

Structural checks prove bootstrap files, imports, registries, and mirrors. They
cannot prove that a stochastic model selected a conditional pointer. Use
[Workflow Behavior Evals](spec-first-workflow/shared/workflow-behavior-evals.md)
for the trace boundary, including the order between the required read and the
first governed action.

## Control Map

| Workflow concept | Codex App | Claude Code | Qwen Code |
| --- | --- | --- | --- |
| Durable execution control (implementation only) | `/goal <objective>` behind the `features.goals` flag; the evaluator inspects real files, tests, logs and artifacts. One root control spans the outcome | `/goal <condition>` carries the outcome's completion condition and the evaluator sees only the transcript; the task list (`TaskCreate`/`TaskUpdate`) is its step ledger. One root control spans the outcome | Task list (`todo_write`, or team `task_create`/`task_update`); one root task tree spans the outcome |
| Implementation worker with isolated worktree | Separate top-level App task/chat started in Worktree mode; never `collaboration.spawn_agent` | Background subagent: `Agent` tool with `run_in_background: true`, `isolation: "worktree"`, and `subagent_type` set to a `worker-*` role that fixes the lane's effort | Background subagent: `Agent` tool with `run_in_background: true` and `isolation: "worktree"` |
| Correct a worker without losing its context | Message the same top-level App task/chat | `SendMessage` addressed to the worker's **agent ID**; a completed worker auto-resumes with full history | `SendMessage` to the same agent |
| Read-only research/challenge/review lane | Project subagents in `.codex/agents/*.toml` | `Agent` tool lane: built-in `Explore`, `Plan`, or `general-purpose`, or a project agent in `.claude/agents/*.md` | `Agent` tool lane: built-in `Explore` or `general-purpose`, or a project agent in `.qwen/agents/*.md` |
| Per-lane model selection | Per-worker/subagent model control in the App | `model` parameter on the `Agent` tool call (dispatch-time override) over `model` frontmatter in `.claude/agents/*.md` (role default) | `model` frontmatter in `.qwen/agents/*.md` (`inherit`, `fast`, a model ID, or `authType:modelId`); exact model IDs are provider-specific |
| Per-lane reasoning effort | Per-worker/subagent effort control in the App | `effort` frontmatter in the agent definition (`low`, `medium`, `high`, `xhigh`, `max`); no per-dispatch parameter — unset inherits the session effort | Not yet available in agent frontmatter; a lane inherits the session effort |
| Worker completion signalling | Native completion and status events | Background-task completion notifications; continue an existing worker with `SendMessage` | Background-task completion notifications; continue an existing worker with `send_message` |

## Model And Effort Selection

The dispatch policy lives in the [implementation phase](spec-first-workflow/phases/implementation-validation-closeout.md#worker-execution) and in [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md): the root explicitly and independently selects the best-suited available model and an acceptance-unit-matched reasoning effort for every worker and read-only lane, and never inherits a parent epic's tier when the controls exist. This table owns the per-harness tiers:

| Task class | Codex App | Claude Code | Qwen Code |
| --- | --- | --- | --- |
| Clear mechanical work | Luna | Sonnet (`claude-sonnet-5`) | `fast` (the configured `fastModel`) |
| Ordinary implementation and review | Terra | Sonnet (`claude-sonnet-5`) | `inherit` (the session model) |
| Closed-route complex, cross-cutting, or high-consequence implementation | Terra with higher effort | Opus (`claude-opus-5`) | `inherit`, or a stronger configured model ID |
| Root orchestration or work with unresolved complex reasoning | Sol | Opus (`claude-opus-5`) | `inherit`, or a stronger configured model ID |
| Explicit user request for the most capable model | — | Fable (`claude-fable-5`) | The strongest configured model ID |

- **Codex App implementation is Terra-first after clear mechanical work.** Once behavior, mechanism, ownership, editable boundary, proof, and stop condition are closed, use Terra at `medium`. Raise effort only when this leaf unit still contains a named reasoning pressure and representative evidence shows a quality gain; cross-cutting scope, protected-domain labels, file count, parent-epic importance, and a previous failure are not sufficient alone. Use Sol or `max` only for the highest-consequence unresolved boundary after brief and route defects are excluded and lower tiers are shown insufficient. Keep unresolved architecture, cause, or route discovery root-local instead of delegating it to a stronger Worker.
- **Claude Code runs a two-model ladder.** Sonnet 5 carries every mechanical and ordinary lane; Opus 5 carries the root session and every complex, cross-cutting-refactoring, or high-consequence lane. Haiku is no longer a default tier — select it only for a trivial lane when current task evidence or representative evaluation shows no material quality loss, and say why. Run the root on Opus 5 (`--model opus`, or the app's model selector) whenever it orchestrates workers or coordinates a structured or orchestrated outcome.
- Qwen Code model tiers are provider-specific: the `model` frontmatter in `.qwen/agents/*.md` accepts `inherit`, `fast`, a bare model ID, or `authType:modelId`. `fast` resolves to the configured `fastModel` and falls back to `inherit` when none is set; `inherit` (or an omitted field) uses the session model. Pick exact model IDs from the models configured for the active provider rather than hardcoding them. Qwen Code does not yet expose a per-agent reasoning-effort field, so a lane inherits the session effort.

- Claude Code accepts the aliases `haiku`, `sonnet`, `opus`, and `fable` on the `Agent` tool and in agent frontmatter; the exact model IDs are for SDK and API dispatch.
- **Defaults are fallbacks, overrides are the contract.** The `model:` frontmatter in `.claude/agents/*.md` records each role's tier default; it never substitutes for the per-dispatch choice. Claude Code resolves a lane's model as: `CLAUDE_CODE_SUBAGENT_MODEL` env var → the `model` parameter on the `Agent` tool call → the definition's `model` frontmatter → the session model. The root passes a dispatch-time `model` whenever task difficulty, evidence volume, or consequence departs from the role default; a lower tier is valid only when current task evidence or representative evaluation shows no material quality loss. The parameter also sticks for follow-up messages to that lane.
- Reasoning effort has no per-dispatch parameter. It resolves as: `CLAUDE_CODE_EFFORT_LEVEL` env var → the definition's `effort` frontmatter → the session effort → the model default. **The root therefore selects effort by selecting the role**, which is why the tiers exist as separate definitions rather than as one worker with a parameter: write lanes `worker-mechanical` (`low`), `worker-standard` (`medium`), `worker-critical` (`high`); review lanes `evidence-agent` (`low`), `task-acceptance-agent` (`medium`), and `critical-reviewer-agent` or `critical-adjudicator-agent` (`xhigh`). A role that leaves `effort` unset inherits the session level, so dispatching every write lane through the generic `claude` agent silently runs mechanical work at whatever the session happens to be set to.
- Map reasoning effort to the task, not habit ([OpenAI guidance](https://developers.openai.com/api/docs/guides/latest-model?model=gpt-5.6#prompting-best-practices), [Anthropic guidance](https://support.claude.com/en/articles/8664678-change-the-model-effort-and-thinking-settings)):

| Effort | Use for |
| --- | --- |
| `low` | Clear, bounded mechanical work whose route and proof are already known and whose lower effort has no material quality loss on current task evidence or representative evaluation. |
| `medium` (default) | Ordinary implementation, review, and document analysis. Use this as the balanced starting point. |
| `high` / `xhigh` | Complex debugging, broad synthesis, or high-consequence reasoning when task evidence or a representative evaluation shows that extra reasoning improves the outcome. |
| `max` | The hardest architecture, research, or formal-reasoning work when lower effort is demonstrably insufficient. |

- A wrong result is evidence to improve the diagnosis, brief, or route, not by itself a reason to raise effort. Implementation correction follows its phase-owned frozen-finding contract and never raises effort merely to keep a repair loop active.
- Required re-review remains at least as capable (model and effort) as the review that found the issue. Implementation correction uses delta-only root verification; when independent implementation review remains triggered, the fixed candidate enters a fresh one-shot lane.

## Goal Mechanics

Both harnesses expose a durable execution control typed as `/goal`. The name and the repository policy are shared; the vendor contracts are not. Write a goal from the section for the harness you are in, never from the other one.

### Shared repository policy

This part is owned here, not by either vendor, and applies to both:

- Set a goal only for a genuinely long-running, multi-step, or resumable implementation outcome, never during a non-implementation phase.
- One root control spans the outcome. The task list is its step ledger, not a second control.
- Carry the accepted stop condition and the invariants that must not change into the goal text. The stop condition is the phase-owned one — the outcome closed with its mapped proof, or the honest blocker ([Stop Rule](spec-first-workflow/phases/implementation-validation-closeout.md#stop-rule)) — and the text must let either ending satisfy it, because a condition that recognizes only success cannot end a genuinely blocked run. `Implementation complete; verification incomplete` is such a blocked ending: close the durable control as blocked with the unverified claim, narrower evidence, and next proof or reopen owner rather than leaving it active or marking it complete.
- Never bound a goal by a turn, step, or iteration count. A counter measures spending, not completion: it cuts an honest run off mid-outcome and lets a stuck one run to the same number. What it stands in for is a run that has stopped converging, and the signal for that is evidence, not arithmetic — a turn that changes no file, no ledger entry, and no command result is a stall, and a stall ends the goal by reporting the blocker.
- The goal text is the directive in both harnesses: setting a goal starts work immediately and no separate prompt follows. A brief that will not fit the harness's goal text means the outcome is too broad for one goal, never a reason to send a second message.

### The difference that changes how you write it

The two evaluators do not have the same powers, so the same goal text is not equally provable in both.

| | Codex | Claude Code |
| --- | --- | --- |
| Evaluator input | Concrete evidence: files, tests, logs, benchmark output, generated artifacts. It can rerun commands across turns | The conversation only. It calls no tools and reads no files |
| What the goal must name | The verification surface that proves the outcome | The command whose printed result proves the outcome |
| Failure mode when written wrong | A vague finish line the evidence cannot settle | A true claim the transcript never demonstrated, closed on assertion |

In Codex, name the artifact and let the evaluator go look. In Claude Code, name the command and require its output in the transcript, because nothing outside the transcript is visible to the evaluator.

### Codex Goal Mechanics

Vendor authority: [Follow a goal](https://learn.chatgpt.com/use-cases/follow-goals) and [Using Goals in Codex](https://developers.openai.com/cookbook/examples/codex/using_goals_in_codex).

- `/goal` is behind a feature flag. If it is absent from the slash-command list, enable `features.goals` in `config.toml` or run `codex features enable goals`. Available in the Codex CLI and the desktop app.
- `/goal <objective>` starts work immediately toward that objective. `/goal` alone reports the current goal; `/goal pause`, `/goal resume`, and `/goal clear` control the run.
- The vendor pattern is `/goal <desired end state> verified by <specific evidence> while preserving <constraints>. Use <allowed inputs, tools, or boundaries>.` A complete goal states outcome, verification surface, constraints, boundaries, iteration policy, and blocked stop conditions.
- Completion is evidence-based: an objective is complete only after it is checked against the relevant files, tests, logs, benchmark output, or generated artifacts, never because the model believes it is probably done.
- No character limit is documented. Length is bounded by usefulness, not by the tool.

### Claude Code Goal Mechanics

Vendor authority: [Keep Claude working toward a goal](https://code.claude.com/docs/en/goal). Requires Claude Code v2.1.139 or later.

- `/goal <condition>` sets the condition and starts a turn immediately **with the condition itself as the directive; no separate prompt is sent**. One goal is active per session and a new one replaces it. `/goal` alone reports the condition, elapsed turns, token spend, and the evaluator's last reason. `/goal clear` ends it early; `stop`, `off`, `reset`, `none`, and `cancel` are accepted aliases. There is no pause or resume.
- The condition holds up to 4,000 characters; there is no separate turn limit.
- Evaluation is a session-scoped prompt-based Stop hook. After each turn the configured small fast model, Haiku by default, judges the condition against the conversation and either clears the goal or returns a reason that steers the next turn.
- **The evaluator runs no tools and reads no files.** Write the condition against evidence the run itself surfaces: name the [validation matrix](spec-first-workflow/phases/implementation-validation-closeout.md#validation-matrix) command and the printed result that proves the claim. A condition the transcript cannot demonstrate closes on an assertion instead of proof.
- A goal does not change permissions. Pair it with auto mode for unattended turns: auto mode removes per-tool prompts, the goal removes per-turn prompts.
- `--resume` and `--continue` restore an active goal; its turn, timer, and token baselines reset. An achieved or cleared goal is not restored.
- Non-interactive, `claude -p "/goal <condition>"` runs the loop to completion in one invocation. Add `--output-format stream-json --verbose`, because the default text output prints nothing until the condition is met.
- Requires a trusted workspace and is unavailable when `disableAllHooks` or `allowManagedHooksOnly` is set, because the evaluator is part of the hooks system.

## Claude Code Worker Mechanics

Vendor authority: [Subagents](https://code.claude.com/docs/en/sub-agents), in particular [Resume subagents](https://code.claude.com/docs/en/sub-agents#resume-subagents).

### Dispatch

- When Worker execution is selected, keep one write worker per ready acceptance unit: one background `Agent` lane with `isolation: "worktree"`. Several write workers may run only as members of a positively independent planned wave. The worktree is the isolation boundary; the root owns candidate intake, acceptance, and integration.
- Pass `isolation` as a dispatch parameter rather than moving it into the role's frontmatter. A dispatch-parameter worktree branches from the parent's `HEAD`, which is the accepted integrated base the wave needs; frontmatter isolation follows the `--worktree` base rule and branches from the repository's default branch unless [`worktree.baseRef`](https://code.claude.com/docs/en/worktrees#choose-the-base-branch) is `"head"`.
- Dispatch only after the exact accepted `tasks.md` revision is in the base
  visible to the worker. Pass its path and acceptance-unit or task IDs plus only
  live facts absent from the ledger.
- A background worker keeps every MCP tool but a reduced built-in set: `Read`, `Grep`, `Glob`, `Bash`, `PowerShell`, `Edit`, `Write`, `NotebookEdit`, `WebFetch`, `WebSearch`, `TodoWrite`, `Skill`, `ToolSearch`, `EnterWorktree`, `ExitWorktree`, `Monitor`, `TaskStop`, `SendMessage`, and `Artifact`. That covers implementation; do not narrow it further with a `tools` allowlist unless the task genuinely requires less.

### What crosses into a worker

A worker starts from a fresh context window. It receives the dispatch, files in
its worktree, the repository instructions, a `git status` snapshot taken when
the parent session started, and any skills its role preloads. It does **not**
receive the root's conversation, the command output the root already read, the
root's output style, or the root's auto memory. Its context window is sized by
its own model, so delegating to a smaller model gives that lane a smaller
window.

This is why [Execution-Ready Dispatch](spec-first-workflow/phases/implementation-worker-execution.md#execution-ready-dispatch) is a rule and not a preference: nothing the root learned reaches the lane except through the accepted ledger entry or dispatch. `/subtask` forks are the one exception — a fork inherits the parent conversation and its exact tool pool — but a fork continues one line of reasoning rather than opening an independent lane, so it is not a substitute for a Worker.

### Monitor

- Follow completion notifications and group relevant targets in one native wait
  when available. An unchanged timeout preserves the existing disposition and
  produces no new analysis, narration, message, or candidate inspection.
- A completing worker returns its **agent ID**. Record it: the ID, not the display name, is the reliable address for everything below.
- `/tasks` lists running and finished lanes for a human; the root does not need it to know a lane finished.

### Return work for rework

This is the loop that makes the root an orchestrator rather than a dispatcher.

- After the worker returns a frozen candidate, send one batched correction to the
  **same worker** with `SendMessage`, addressed by its agent ID. The worker
  retains its full history and continues from where it stopped.
- While the worker is active, message only a safety stop or an accepted-input
  invalidation. Review findings wait for its returned frozen candidate.
- Spawning a **new** worker instead of correcting the existing one is a defect, not a shortcut: it discards the context that makes the second attempt cheaper than the first. Replace a worker only for an execution stall that produces no new turn, or for an invalidated base, and then continue the same exact brief from the frozen candidate.
- Address by agent ID, not by name. If a re-spawned worker has taken a name, `SendMessage` refuses the send rather than delivering to the wrong lane, and reports which agent the name now reaches.
- A worker the **user** stopped does not auto-resume; `SendMessage` returns a refusal. That one needs a human to type into its transcript.
- `Explore` and `Plan` are one-shot and return no agent ID, so they can never be corrected or resumed. They are never write lanes.
- A message from any agent is task direction only. It never approves a pending permission prompt and never changes a worker's permission settings, `CLAUDE.md`, or configuration.

### Role and effort selection

The two axes resolve through different channels, so the root steers them differently.

**Model is chosen at dispatch.** Pass `model` on the `Agent` call, using the task-class tiers in [Model And Effort Selection](#model-and-effort-selection). No definition file is needed to express a model choice, and the `model` in a role's frontmatter is only that tier's default for a dispatch that omits it. A per-invocation model also survives a later `SendMessage`.

**Effort is chosen by choosing the role.** There is no `effort` parameter on the `Agent` call and no other in-session channel, so a role definition is the only carrier: `worker-mechanical` (`low`), `worker-standard` (`medium`), `worker-critical` (`high`). An instruction such as "run mechanical work at low effort" is unimplementable without one — the root would read it, find no mechanism, and the lane would silently inherit the session level. That inheritance is how a mechanical regeneration ends up costing what a money-invariant change costs.

Because the axes are independent, the three roles cover the whole grid: `worker-standard` dispatched with `model: "opus"` is Opus at medium effort. Add a role only for a new effort level, never for a new model combination.

The role files therefore carry **no behavior**. Everything a Worker must do already lives in `AGENTS.md` and the implementation phase; a role that restates it creates a second place to keep in sync. Each worker definition is frontmatter plus a pointer to the contract, and nothing more.

These worker roles exist only under `.claude/agents/`. They are deliberately not mirrored to `.codex/agents/` or `.qwen/agents/`: a Codex subagent is `sandbox_mode = "read-only"` by construction and can never be a write lane, and the Codex Worker is a native App control with no definition file. Mirroring them would advertise a lane that harness cannot run.

**Reopen when Claude Code gains an `effort` parameter on the `Agent` call.** These three definitions exist for one reason: effort has no dispatch channel. Give it one — [several open requests ask for exactly that](https://github.com/anthropics/claude-code/issues/39220) — and the files carry nothing a task-class table in this document could not state directly, and should be deleted rather than kept because they exist. Check this whenever the harness version moves: a definition that outlives its only justification is the kind of thing nobody removes, because removing it is nobody's task.

### Read-only lanes

Read-only lanes follow [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md): one distinct decision-changing question per lane, concurrency bounded by current capacity and independence, and read-only boundaries stated in each brief.

Start ordinary research, design, challenge, and review lanes with fresh context
when the harness supports it. In the Codex App, dispatch the selected role with
no inherited root turns; inherit only the smallest recent turn set when a
non-review question depends on irreproducible user context that is unavailable
from the accepted brief or cited sources. In Claude Code and Qwen Code, start a
new lane of the selected role. Pass only the shared Lane Brief, minimal artifact
or source pointers, and irreproducible current facts. Return only the shared
Fan-In envelope rather than replaying the lane transcript.

Triggered independent implementation review opens a new one-shot lane with
fresh context. Dispatch `task-acceptance-agent` for an ordinary bounded unit.
Dispatch `critical-reviewer-agent` only for the highest-consequence boundary
when current unit-specific evidence justifies the critical tier.

An implementation Worker or transcript-inheriting subtask/fork is not an
independent reviewer. Pass an implementation reviewer only the `tasks.md` path
and unit or task IDs, candidate location, and irreproducible external evidence
allowed by the shared implementation-review independence contract. Never
resume that reviewer for a different unit.

Harness-native task lists remain execution controls. They do not replace the
repository `tasks.md` ledger or receive its acceptance receipts.

## Repository Wiring

The repository ships all three harness configurations pre-wired; a fresh clone needs no setup in any harness.

- `CLAUDE.md` at the repository root imports `AGENTS.md` (`@AGENTS.md`), so Claude Code loads the same contract the Codex App reads directly. Qwen Code reads `AGENTS.md` automatically, so the root `QWEN.md` records only Qwen-specific wiring and does not re-import it (re-importing would load the contract twice).
- `.codex/agents/*.toml`, `.claude/agents/*.md`, and `.qwen/agents/*.md` define the same specialist roles for their harnesses. When changing a role, mirror the change in all three files; no file owns the others. The Qwen variants carry Qwen-native frontmatter (a `tools:` list of Qwen tool names and an optional `model:` of `inherit`/`fast`/model ID) over the same harness-neutral body.
- `.agents/skills/` is the canonical skill source. Claude Code discovers each skill through a per-skill symlink `.claude/skills/<name>` → `../../.agents/skills/<name>` — the officially documented pattern (a skill *entry* may be a symlink); do not fork skill content into `.claude/`, and do not replace the per-skill links with one whole-directory symlink (that variant has a history of discovery regressions). Qwen Code discovers `.agents/skills/` natively (it scans both `.qwen/skills/` and `.agents/skills/`), so it needs no symlinks; do not add a `.qwen/skills/` mirror or the skills would load twice.
- Adding or removing a skill in `.agents/skills/` requires resyncing the Claude links: run `make claude-skills-sync`. Qwen Code picks the change up automatically.
- `.claude/settings.local.json` is personal and gitignored. Commit shared Claude Code policy only when it must apply to every clone. The Qwen equivalents (`.qwen/QWEN.local.md`, `.qwen/settings.local.json`) are personal too and stay out of git.
- Windows checkouts need symlink support (`git config core.symlinks true` with Developer Mode or an elevated shell) for the `.claude/skills/*` links; otherwise run `make claude-skills-sync` (or recreate the links manually) after cloning. The Qwen wiring uses no symlinks and is unaffected.

## Programmatic Runs

When this workflow is driven from code rather than an interactive session, use the Claude Agent SDK (`claude-agent-sdk` for Python, `@anthropic-ai/claude-agent-sdk` for TypeScript): `query(prompt, options)` runs the same Claude Code harness with the same subagent definitions (the `agents` option or `.claude/agents/*.md`), per-agent `model` and effort, and background execution. Direct Anthropic API integrations (Messages API, Managed Agents) are separate products outside this repository's workflow contract; do not substitute them for harness controls.
