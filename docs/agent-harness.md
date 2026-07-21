# Agent Harness

The workflow instructions in this repository are harness-neutral. This document owns harness detection and the mapping from neutral workflow concepts to the native controls of each supported harness. `AGENTS.md` still owns authorization; the workflow phases still own when a control may be used.

## Read When

- Deciding which harness-native tool implements a workflow concept: durable execution control, implementation worker, read-only subagent lane, model selection, or reasoning effort.
- Dispatching a worker or subagent and selecting its model and reasoning effort.
- Writing instructions, briefs, or handoffs that must work in both harnesses.

## Harness Selection

- Identify the current harness from the session environment before dispatching any harness control: the Codex App or Codex CLI runs the Codex-native path; Claude Code (CLI, desktop, web, or IDE) runs the Anthropic-native path; Qwen Code (CLI or IDE) runs the Qwen-native path.
- Use only the current harness's native controls. Do not emulate another harness's controls, shell out to its CLI, or mix the two control planes within one outcome.
- When the current session lacks the needed native control, do the work root-locally under the owning phase's local-execution rules and state the missing control; do not invent a substitute.

## Control Map

| Workflow concept | Codex App | Claude Code | Qwen Code |
| --- | --- | --- | --- |
| Durable execution control (implementation only) | Codex Goal | Task list (`TaskCreate`/`TaskUpdate`); one root task tree spans the outcome | Task list (`todo_write`, or team `task_create`/`task_update`); one root task tree spans the outcome |
| Implementation worker with isolated worktree | Native App Worker with managed worktree | Background subagent: `Agent` tool with `run_in_background: true` and `isolation: "worktree"` | Background subagent: `Agent` tool with `run_in_background: true` and `isolation: "worktree"` |
| Read-only research/challenge/review lane | Project subagents in `.codex/agents/*.toml` | `Agent` tool lane: built-in `Explore`, `Plan`, or `general-purpose`, or a project agent in `.claude/agents/*.md` | `Agent` tool lane: built-in `Explore` or `general-purpose`, or a project agent in `.qwen/agents/*.md` |
| Per-lane model selection | Per-worker/subagent model control in the App | `model` parameter on the `Agent` tool call (dispatch-time override) over `model` frontmatter in `.claude/agents/*.md` (role default) | `model` frontmatter in `.qwen/agents/*.md` (`inherit`, `fast`, a model ID, or `authType:modelId`); exact model IDs are provider-specific |
| Per-lane reasoning effort | Per-worker/subagent effort control in the App | `effort` frontmatter in the agent definition (`low`, `medium`, `high`, `xhigh`, `max`); no per-dispatch parameter — unset inherits the session effort | Not yet available in agent frontmatter; a lane inherits the session effort |
| Worker completion signalling | Native completion and status events | Background-task completion notifications; continue an existing worker with `SendMessage` | Background-task completion notifications; continue an existing worker with `send_message` |

## Model And Effort Selection

The dispatch policy lives in the [implementation phase](spec-first-workflow/phases/implementation-validation-closeout.md#optional-worker-execution) and in [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md): the root explicitly selects the best-suited available model and the lowest reasoning effort likely to succeed for every worker and read-only lane, and never inherits a harness default when the controls exist. This table owns the per-harness tiers:

| Task class | Codex App | Claude Code | Qwen Code |
| --- | --- | --- | --- |
| Clear mechanical work | Luna | Haiku (`claude-haiku-4-5`) | `fast` (the configured `fastModel`) |
| Ordinary implementation and review | Terra | Sonnet (`claude-sonnet-5`) | `inherit` (the session model) |
| Complex or high-consequence work | Sol | Opus (`claude-opus-4-8`) | `inherit`, or a stronger configured model ID |
| Explicit user request for the most capable model | — | Fable (`claude-fable-5`) | The strongest configured model ID |

- Qwen Code model tiers are provider-specific: the `model` frontmatter in `.qwen/agents/*.md` accepts `inherit`, `fast`, a bare model ID, or `authType:modelId`. `fast` resolves to the configured `fastModel` and falls back to `inherit` when none is set; `inherit` (or an omitted field) uses the session model. Pick exact model IDs from the models configured for the active provider rather than hardcoding them. Qwen Code does not yet expose a per-agent reasoning-effort field, so a lane inherits the session effort.

- Claude Code accepts the aliases `haiku`, `sonnet`, `opus`, and `fable` on the `Agent` tool and in agent frontmatter; the exact model IDs are for SDK and API dispatch.
- **Defaults are fallbacks, overrides are the contract.** The `model:` frontmatter in `.claude/agents/*.md` records each role's tier default; it never substitutes for the per-dispatch choice. Claude Code resolves a lane's model as: `CLAUDE_CODE_SUBAGENT_MODEL` env var → the `model` parameter on the `Agent` tool call → the definition's `model` frontmatter → the session model. The root passes a dispatch-time `model` whenever task difficulty, evidence volume, latency/cost, or consequence departs from the role default; the parameter also sticks for follow-up messages to that lane.
- Reasoning effort has no per-dispatch parameter. It resolves as: `CLAUDE_CODE_EFFORT_LEVEL` env var → the definition's `effort` frontmatter → the session effort → the model default. Roles whose consequence fixes their effort declare it in frontmatter (`critical-reviewer-agent` and `critical-adjudicator-agent`: `xhigh`; `evidence-agent`: `low`); all other roles leave `effort` unset so the root steers them through the session effort level.
- Map Anthropic reasoning effort to the task, not habit ([official guidance](https://support.claude.com/en/articles/8664678-change-the-model-effort-and-thinking-settings)):

| Effort | Use for |
| --- | --- |
| `low` / `medium` | Routine mechanical work: small lookups, summaries, drift checks, minor text or code edits. Fastest and cheapest. |
| `high` (default) | Most work: ordinary implementation, review lanes, document analysis. The balance point — start here. |
| `xhigh` | Long agentic and coding outcomes: refactoring, complex debugging, multi-step file work. Deeper than `high`, cheaper than `max`. |
| `max` | Hardest reasoning only: architecture decisions, research- or math-grade problems, or escalation after `high`/`xhigh` produced a wrong result. Slowest and most token-hungry. |

- The practical rule: hold `high`; raise to `xhigh`/`max` only when the model errs or the task is genuinely hard; drop to `low`/`medium` for trivia when limits matter. Escalation after a wrong result is a legitimate trigger — a wrong answer at `high` justifies one retry at `xhigh`/`max` before rerouting.
- Re-review remains at least as capable (model and effort) as the review that found the issue.

## Claude Code Worker Mechanics

- Keep one write worker per outcome: one background `Agent` lane with `isolation: "worktree"`. The worktree is the isolation boundary; the root still owns acceptance and integration per the implementation phase.
- The worker receives the same outcome-first brief the implementation phase requires. Route correction briefs to the same worker with `SendMessage` so its context survives; replace the worker only under the phase's no-progress rule.
- Follow completion notifications instead of polling or narrating unchanged state.
- Read-only lanes follow [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) unchanged: at most three concurrent lanes, one bounded wave, no nested delegation, and read-only boundaries stated in each brief.
- The Claude Code task list follows the same rule as a Codex Goal: create it only for genuinely long-running, multi-step, or resumable implementation, never during non-implementation phases.

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
