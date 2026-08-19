# Qwen Code Harness Adapter

## Read When

Read after the [Agent Harness](../agent-harness.md) selects Qwen Code and a task
list, background Worker, read-only lane, model, or effort decision is needed.

## Native Map

- Use `todo_write`, or team `task_create` and `task_update`, as the one root
  execution tree. Repository `tasks.md` remains the acceptance ledger.
- Bind one Acceptance-Unit Lead per structured root session. Qwen has no Ledger
  Orchestrator carrier.
- Dispatch an isolated background implementation agent with
  `isolation: "worktree"`; send corrections to the same agent.
- Use built-in `Explore`, `general-purpose`, or project agents for fresh
  read-only lanes.

## Model And Effort

Project agent frontmatter accepts `inherit`, `fast`, a model ID, or
`authType:modelId`. Map the shared selection policy to `fast` for closed
mechanical work when configured, `inherit` for ordinary work, and a stronger
configured model for complex or high-consequence work. Qwen exposes no
per-agent reasoning-effort field, so a lane inherits session effort.
