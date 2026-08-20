# Agent Harness

Harness-neutral index for native execution controls. `AGENTS.md` owns
authorization; workflow phases own when a control is used; adapters own its
native mechanics.

## Read When

Read only when selecting or operating a durable task, Worker, read-only lane,
Goal, model, or reasoning effort. Select exactly one adapter:

| Current harness | Adapter |
| --- | --- |
| Codex App or Codex CLI | [Codex](agent-harness/codex.md) |
| Claude Code CLI, desktop, web, or IDE | [Claude Code](agent-harness/claude-code.md) |
| Qwen Code CLI or IDE | [Qwen Code](agent-harness/qwen-code.md) |

Do not load another adapter, emulate its controls, shell out to its CLI, or mix
control planes inside one outcome. A native task, subagent, model, or worktree
is a carrier; role authority comes only from the Implementation [Role
Tree](spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree).

## Conditional Contracts

Load only the contract activated by the next native action:

| Trigger | Shared contract |
| --- | --- |
| An `IMPLEMENTATION_WORKER` is about to write. | [Write Carrier](agent-harness/shared/write-carrier.md) |
| Research, challenge, or review needs a read-only lane. | [Read-Only Carrier](agent-harness/shared/read-only-carrier.md) |
| A native lane needs model or effort selection. | [Model And Effort Selection](agent-harness/shared/model-selection.md) |
| A durable Goal may be warranted. | [Goal Mechanics](agent-harness/shared/goals.md) |

Then apply only the selected adapter's matching conditional control.
