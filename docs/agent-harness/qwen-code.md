# Qwen Code Harness Adapter

Use installed Qwen agent controls as native authority. The ordinary nested
route is supported by Qwen Code 0.21.9; recheck native controls after runtime
changes.

## Native Map

- `/orchestrator` binds the current session as Ledger Orchestrator. Dispatch
  mutually independent ready units through `agent` with
  `subagent_type: "acceptance-unit-lead"`, within current capacity. The native
  [Lead carrier](../../.qwen/agents/acceptance-unit-lead.md) loads the existing
  role skill and owns its fixed unit through proof, required review, and its
  acceptance verdict. The root lands only `Accepted` candidates serially from
  [Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
  and records the verdict without re-adjudicating or implementing the unit.
  `todo_write` tracks execution; repository `tasks.md` owns acceptance.
- The installed schema names the tool `agent` (display name `Agent`), not
  `task`. Ordinary named agents can nest. Apply
  [Nested Execution](../agent-harness.md#nested-execution); project setting
  `model.maxSubagentDepth: 5` selects the installed default of five levels.
  Teammates, forks, and workflow-spawned agents cannot nest regardless of
  that setting. Missing tools or exhausted depth return an exact parent gap.
- The Lead may implement directly. Use `isolation: "worktree"` when
  [Agent Harness](../agent-harness.md) selects isolation. Workers may share
  the Lead checkout when writable responsibilities and locks are disjoint.
- Top-level ordinary agents run in the background by default. Nested agents
  run in the foreground and reject `run_in_background: true`; omit it or set
  it to `false`. Caller-owned `working_dir` launches are also foreground-only.
  Do not promise review/proof overlap from a blocked nested parent: let the
  root broker an independent background reviewer from the Lead's fixed brief
  when that overlap is useful, returning its result to the Lead.
- Agent Teams remain an explicitly selected, already configured route. Since
  teammates cannot nest, the root brokers descendants from teammate Lead
  briefs and returns their results; acceptance stays with the Lead. Team
  `task_create`/`task_update` mirror execution only. Full-ledger work does not
  require enabling Teams or changing user or machine settings.

## Models And Dispatch

Project agent frontmatter accepts `inherit`, `fast`, a model ID, or
`authType:modelId`. Use `fast` for closed mechanical work when configured,
`inherit` for ordinary work and a closed, strongly owned Lead unit, and the
strongest configured model when uncertainty, protected risk, or high
consequence requires it. There is no per-agent reasoning-effort field; lanes
inherit session effort. Preserve a user-selected model.

Apply [Context And Lifetime](../agent-harness.md#context-and-lifetime) for
freshness and ledger-controlled Lead reuse. A new small task starts a fresh
named agent without `subagent_type: "fork"` or continuation. Pass the
[delegation interface](../agent-harness.md#delegation-interface) through native
fields where available and supply facts absent from the named canonical
inputs. Retain the returned `task_id`; `list_agents` identifies retained
background agents, and `send_message` with that `task_id` continues related
work or a same-brief review correction. A completed background agent may
resume from its resident runtime or retained transcript. Foreground nested
results return inline; if their runtime has no retained continuation, return
that exact gap to the parent rather than inventing a resume field.

[Review](../spec-first-workflow/shared/review.md) selects whether independent
review is required. When required, bind a fresh `reviewer-agent` to
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
and the fixed candidate. Start review before independent focused proof only
where the native async route permits it; keep the candidate unchanged and
observe the proof budget. The root binds integrated-candidate review only
when Review requires that boundary. Messages and execution status do not
replace proof receipts or the Lead's acceptance result.
