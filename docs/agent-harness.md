# Agent Harness

The repository workflow is harness-neutral. This index owns harness detection
and shared carrier invariants; each adapter owns its native controls.
`AGENTS.md` owns authorization and workflow phases own when a control is used.

## Read When

Read this index only when selecting or operating a durable execution control,
Worker, read-only lane, model, or reasoning effort. Then load exactly one
adapter:

| Current harness | Adapter |
| --- | --- |
| Codex App or Codex CLI | [Codex](agent-harness/codex.md) |
| Claude Code CLI, desktop, web, or IDE | [Claude Code](agent-harness/claude-code.md) |
| Qwen Code CLI or IDE | [Qwen Code](agent-harness/qwen-code.md) |

Do not load another adapter, emulate its controls, shell out to its CLI, or mix
control planes inside one outcome. A native task, subagent, model, or worktree
is a carrier; role authority comes only from the Implementation [Role
Tree](spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree).

## Shared Carrier Contract

A Role Tree write lane uses the selected adapter's isolated Implementation
Worker. Its direct parent supplies the fixed brief, the child dispatches no
grandchildren, and it returns only its bound role's result.

### Write-Carrier Gate

Before an `IMPLEMENTATION_WORKER` writes, its Lead records the carrier required
by the selected adapter, the actual native identity returned by that control,
and the isolated checkout identity. A role label, writable tool set, or
successful dispatch is not carrier proof.

A mismatch before writing cancels the lane. A mismatch after writing freezes
its bytes and proof as diagnostic evidence only; rerun the slice from its
original base through a valid carrier or record the exact capability blocker.
External or ignored inputs cross as read-only locators with identities and are
validated before the first edit and before `DONE`.

### Read-Only Lane Carrier

Research, challenge, and review use a fresh built-in read-only lane in the
current root task. An independent review always starts with fresh context. The
lane returns evidence or a verdict; the root retains synthesis, correction,
acceptance, and completion. If a required independent carrier is unavailable,
the boundary remains unaccepted. Never substitute a write Worker or peer
session.

## Model And Effort Selection

Select the least capable available configuration that representative evidence
shows can close the fixed brief. Preserve an exact user-requested model. Start
ordinary implementation, review, and document work at the adapter's balanced
tier; use low effort for closed mechanical work and high or xhigh for complex
or high-consequence reasoning. The Acceptance-Unit Lead is the reserved
quality-first workload and uses `xhigh`, falling back to `high` only on a
recorded unsupported or rejected override; that effort never propagates to its
leaves. This workflow never selects `max`.

Role names and tree depth do not select effort. A wrong result first reopens the
diagnosis, brief, or route; it does not by itself justify more reasoning. Keep a
running lane's selected configuration across correction. Re-evaluate model and
effort defaults on each model-generation change rather than carrying the prior
generation's maximum forward.

The selected adapter owns exact model names, supported fields, inheritance, and
fallback behavior.

Harness-neutral specialist semantics live in `.agents/roles`; the generated
carrier adds only this adapter's sandbox, tools, model, and file format.

## Goal Mechanics

Use a durable goal only for a genuinely long-running, multi-step, or resumable
Implementation outcome. The Codex Ledger Orchestrator is the routing-only
exception. One goal owns one thread-local stage and names its observable end
state, proof surface, preserved constraints, and blocked stop condition. Never
use a turn or iteration count as completion.

The goal directive starts work; do not send a second prompt that restates it.
An Acceptance-Unit Lead goal ends only at its canonical receipt or blocker. A
Ledger Orchestrator goal stays active through authorized agent-owned recovery.
The selected adapter owns evaluator visibility and exact goal controls.
