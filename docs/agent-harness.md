# Agent Harness

The workflow instructions in this repository are harness-neutral. This document owns harness detection and the mapping from neutral workflow concepts to the native controls of each supported harness. `AGENTS.md` still owns authorization; the workflow phases still own when a control may be used.

## Read When

- Deciding which harness-native tool implements a workflow concept: durable execution control, implementation worker, read-only subagent lane, model selection, or reasoning effort.
- Dispatching a worker or subagent and selecting its model and reasoning effort.
- Writing instructions, briefs, or handoffs that must work in both harnesses.

## Harness Selection

- Identify the current harness from the session environment before dispatching any harness control: the Codex App or Codex CLI runs the Codex-native path; Claude Code (CLI, desktop, web, or IDE) runs the Anthropic-native path; Qwen Code (CLI or IDE) runs the Qwen-native path.
- Use only the current harness's native controls. Do not emulate another harness's controls, shell out to its CLI, or mix the two control planes within one outcome.
- Treat every Codex App `collaboration.spawn_agent` lane as a built-in read-only subagent, including one dispatched with `agent_type: "worker"`; it is never the Native App Worker with managed worktree.
- When the current session lacks the needed native control, do the work root-locally under the owning phase's local-execution rules and state the missing control; do not invent a substitute.

## Control Map

| Workflow concept | Codex App | Claude Code | Qwen Code |
| --- | --- | --- | --- |
| Durable execution control (implementation only) | Codex Goal | `/goal <condition>` carries the outcome's completion condition; the task list (`TaskCreate`/`TaskUpdate`) is its step ledger. One root control spans the outcome | Task list (`todo_write`, or team `task_create`/`task_update`); one root task tree spans the outcome |
| Implementation worker with isolated worktree | Native App Worker with managed worktree | Background subagent: `Agent` tool with `run_in_background: true` and `isolation: "worktree"` | Background subagent: `Agent` tool with `run_in_background: true` and `isolation: "worktree"` |
| Read-only research/challenge/review lane | Project subagents in `.codex/agents/*.toml` | `Agent` tool lane: built-in `Explore`, `Plan`, or `general-purpose`, or a project agent in `.claude/agents/*.md` | `Agent` tool lane: built-in `Explore` or `general-purpose`, or a project agent in `.qwen/agents/*.md` |
| Per-lane model selection | Per-worker/subagent model control in the App | `model` parameter on the `Agent` tool call (dispatch-time override) over `model` frontmatter in `.claude/agents/*.md` (role default) | `model` frontmatter in `.qwen/agents/*.md` (`inherit`, `fast`, a model ID, or `authType:modelId`); exact model IDs are provider-specific |
| Per-lane reasoning effort | Per-worker/subagent effort control in the App | `effort` frontmatter in the agent definition (`low`, `medium`, `high`, `xhigh`, `max`); no per-dispatch parameter — unset inherits the session effort | Not yet available in agent frontmatter; a lane inherits the session effort |
| Worker completion signalling | Native completion and status events | Background-task completion notifications; continue an existing worker with `SendMessage` | Background-task completion notifications; continue an existing worker with `send_message` |

## Model And Effort Selection

The dispatch policy lives in the [implementation phase](spec-first-workflow/phases/implementation-validation-closeout.md#worker-execution) and in [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md): the root explicitly and independently selects the best-suited available model and a task-matched reasoning effort for every worker and read-only lane, and never inherits a harness default when the controls exist. This table owns the per-harness tiers:

| Task class | Codex App | Claude Code | Qwen Code |
| --- | --- | --- | --- |
| Clear mechanical work | Luna | Sonnet (`claude-sonnet-5`) | `fast` (the configured `fastModel`) |
| Ordinary implementation and review | Terra | Sonnet (`claude-sonnet-5`) | `inherit` (the session model) |
| Root orchestration, complex, cross-cutting refactoring, or high-consequence work | Sol | Opus (`claude-opus-5`) | `inherit`, or a stronger configured model ID |
| Explicit user request for the most capable model | — | Fable (`claude-fable-5`) | The strongest configured model ID |

- **Claude Code runs a two-model ladder.** Sonnet 5 carries every mechanical and ordinary lane; Opus 5 carries the root session and every complex, cross-cutting-refactoring, or high-consequence lane. Haiku is no longer a default tier — select it only for a lane that is both trivial and latency-bound, and say why. Run the root on Opus 5 (`--model opus`, or the app's model selector) whenever it orchestrates workers or owns acceptance for a structured or orchestrated outcome.
- Qwen Code model tiers are provider-specific: the `model` frontmatter in `.qwen/agents/*.md` accepts `inherit`, `fast`, a bare model ID, or `authType:modelId`. `fast` resolves to the configured `fastModel` and falls back to `inherit` when none is set; `inherit` (or an omitted field) uses the session model. Pick exact model IDs from the models configured for the active provider rather than hardcoding them. Qwen Code does not yet expose a per-agent reasoning-effort field, so a lane inherits the session effort.

- Claude Code accepts the aliases `haiku`, `sonnet`, `opus`, and `fable` on the `Agent` tool and in agent frontmatter; the exact model IDs are for SDK and API dispatch.
- **Defaults are fallbacks, overrides are the contract.** The `model:` frontmatter in `.claude/agents/*.md` records each role's tier default; it never substitutes for the per-dispatch choice. Claude Code resolves a lane's model as: `CLAUDE_CODE_SUBAGENT_MODEL` env var → the `model` parameter on the `Agent` tool call → the definition's `model` frontmatter → the session model. The root passes a dispatch-time `model` whenever task difficulty, evidence volume, latency/cost, or consequence departs from the role default; the parameter also sticks for follow-up messages to that lane.
- Reasoning effort has no per-dispatch parameter. It resolves as: `CLAUDE_CODE_EFFORT_LEVEL` env var → the definition's `effort` frontmatter → the session effort → the model default. Roles whose consequence fixes their effort declare it in frontmatter (`critical-reviewer-agent` and `critical-adjudicator-agent`: `xhigh`; `evidence-agent`: `low`); all other roles leave `effort` unset so the root steers them through the session effort level.
- Map reasoning effort to the task, not habit ([OpenAI guidance](https://developers.openai.com/api/docs/guides/latest-model?model=gpt-5.6#prompting-best-practices), [Anthropic guidance](https://support.claude.com/en/articles/8664678-change-the-model-effort-and-thinking-settings)):

| Effort | Use for |
| --- | --- |
| `low` | Clear, bounded, latency-sensitive mechanical work whose route and proof are already known. |
| `medium` (default) | Ordinary implementation, review, and document analysis. Use this as the balanced starting point. |
| `high` / `xhigh` | Complex debugging, broad synthesis, or high-consequence reasoning when task evidence or a representative evaluation shows that extra reasoning improves the outcome. |
| `max` | The hardest architecture, research, or formal-reasoning work when lower effort is demonstrably insufficient. |

- A wrong result is evidence to improve the diagnosis, brief, or route, not by itself a reason to raise effort. Implementation correction follows its phase-owned frozen-finding contract and never raises effort merely to keep a repair loop active.
- Required non-implementation re-review remains at least as capable (model and effort) as the review that found the issue. Implementation correction uses delta-only verification instead of re-review.

## Claude Code Goal Mechanics

`/goal` is the Claude Code equivalent of a Codex Goal and carries the same restriction: set one only for a genuinely long-running, multi-step, or resumable implementation outcome, never during a non-implementation phase. It requires Claude Code v2.1.139 or later. Vendor authority: [Keep Claude working toward a goal](https://code.claude.com/docs/en/goal).

- `/goal <condition>` sets the condition and starts a turn immediately **with the condition itself as the directive; no separate prompt is sent**. One goal is active per session and a new one replaces it. `/goal` alone reports the condition, elapsed turns, token spend, and the evaluator's last reason. `/goal clear` ends it early; `stop`, `off`, `reset`, `none`, and `cancel` are accepted aliases.
- Because the condition is the directive, the whole brief travels inside it: outcome, authorities, boundaries, and finish line in one command. Never split a goal into a task prompt plus a separate `/goal` message. A brief that will not compress into the 4,000-character condition is evidence that the outcome is too broad for one goal, not a reason to send two messages.
- Evaluation is a session-scoped prompt-based Stop hook. After each turn the configured small fast model judges the condition against the conversation and either clears the goal or returns a reason that steers the next turn.
- **The evaluator runs no tools and reads no files.** Write the condition against evidence the run itself surfaces: name the [validation matrix](../AGENTS.md#validation-matrix) command and the result that proves the claim. A condition the transcript cannot demonstrate closes on an assertion instead of proof.
- Carry the accepted stop condition into the goal text, including the invariants that must not change and an explicit bound such as `or stop after 20 turns`. The condition holds up to 4,000 characters; there is no separate turn limit.
- A goal does not change permissions. Pair it with auto mode for unattended turns: auto mode removes per-tool prompts, the goal removes per-turn prompts.
- `--resume` and `--continue` restore an active goal; its turn, timer, and token baselines reset. An achieved or cleared goal is not restored.
- Non-interactive, `claude -p "/goal <condition>"` runs the loop to completion in one invocation. Add `--output-format stream-json --verbose`, because the default text output prints nothing until the condition is met.
- `/goal` needs a trusted workspace and is unavailable when `disableAllHooks` or `allowManagedHooksOnly` is set, because the evaluator is part of the hooks system.

## Claude Code Worker Mechanics

- Keep one write worker per ready ledger task: one background `Agent` lane with `isolation: "worktree"`. Several write workers may run only as members of a positively independent planned wave. The worktree is the isolation boundary; the root still owns acceptance and integration per the implementation phase.
- The worker receives the same outcome-first brief the implementation phase requires. Route correction briefs to the same worker with `SendMessage` so its context survives; replace the worker only for an execution stall or invalidated base, and continue the same exact brief from the frozen candidate.
- Follow completion notifications instead of polling or narrating unchanged state.
- Read-only lanes follow [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md): one distinct decision-changing question per lane, concurrency bounded by current capacity and independence, and read-only boundaries stated in each brief.
- The task list is the goal's step ledger and carries the same implementation-only restriction ([Claude Code Goal Mechanics](#claude-code-goal-mechanics)).

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
