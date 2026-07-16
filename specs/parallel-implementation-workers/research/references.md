# Parallel implementation references

valid as of: 2026-07-16

## Open items

### What prompt shape does current OpenAI guidance support?

- Classification: contract evidence for instruction structure.
- Fact: current GPT-5.6 guidance recommends precise outcome-level instructions, explicit autonomy/approval boundaries, thorough validation, progress tracking for long-running agentic work, and keeping each durable rule compact, in one place, and stated once.
- Sources: OpenAI, [Model guidance](https://developers.openai.com/api/docs/guides/latest-model#define-autonomy-and-approval-boundaries) and [Prompt engineering — coding](https://developers.openai.com/api/docs/guides/prompt-engineering#coding), fetched 2026-07-16 through the official Docs MCP.
- Implication: keep planning data structural and compact, let implementation consume the reviewed schedule with a lightweight drift check, persist only resume-critical state, and require focused Worker proof plus root acceptance without duplicating the phase contract in every brief or skill.

### Can Codex isolate concurrent implementation tasks?

- Classification: evidence question for technical design and implementation workflow.
- Fact: Codex-managed worktrees are dedicated per task and are documented as a way to run multiple independent tasks in the same project without interference. Each worktree has its own checkout and normally starts detached from the selected branch HEAD.
- Fact: OpenAI's current multi-agent guidance recommends parallelism first for read-heavy work and cautions that concurrent code edits can add conflicts and coordination cost.
- Sources:
  - OpenAI, [Worktrees](https://developers.openai.com/codex/environments/git-worktrees), fetched 2026-07-16.
  - OpenAI Codex manual, `Execution Model and Workflows / Multi-agent operations`, current local manual fetched 2026-07-16.
- Implication: isolated worktrees make concurrent write lanes feasible, but task independence and integration still need positive proof; worktree isolation alone does not prove semantic or merge independence.

### What is reusable from `obra/superpowers`?

- Classification: evidence question for task decomposition and dispatch policy.
- Fact: upstream commit `d884ae04edebef577e82ff7c4e143debd0bbec99` (release v6.1.1) dispatches parallel agents only for independent domains without shared state or sequential dependencies.
- Fact: its planning guidance defines a task as the smallest independently testable unit worth a review gate, folds setup/config/docs into the deliverable that needs them, and records exact cross-task interfaces.
- Fact: its implementation workflow explicitly forbids multiple implementation subagents in parallel because they share a working branch and may conflict.
- Fact: merged optimization PR #1717 reports roughly 20–25% typical time, token, and cost reduction from leaner subagent-driven execution; this is upstream project evidence, not a Codex App benchmark.
- Fact: commit `8e1262a` moved planning toward task right-sizing, exact `Global Constraints`, and cross-task `Interfaces` instead of repeating full implementation detail in every task.
- Source: [`obra/superpowers`](https://github.com/obra/superpowers) at the commit above, inspected 2026-07-16; relevant files are `skills/dispatching-parallel-agents/SKILL.md`, `skills/writing-plans/SKILL.md`, and `skills/subagent-driven-development/SKILL.md`.
- Sources: [PR #1717](https://github.com/obra/superpowers/pull/1717) and [commit `8e1262a`](https://github.com/obra/superpowers/commit/8e1262a), inspected 2026-07-16.
- Implication: reuse the independence test and task right-sizing, not the shared-branch prohibition. Codex managed worktrees remove that specific filesystem/branch interference, while the integration and semantic-conflict risks remain.

### What do upstream user reports expose?

- Classification: anecdotal operational evidence for convergence and task-contract design.
- Observation: issues [#512](https://github.com/obra/superpowers/issues/512), [#694](https://github.com/obra/superpowers/issues/694), [#895](https://github.com/obra/superpowers/issues/895), and [#1860](https://github.com/obra/superpowers/issues/1860) repeatedly report plans becoming too procedural or duplicative; keep this ledger outcome-first and carry only execution-changing constraints and interfaces.
- Observation: issues [#1120](https://github.com/obra/superpowers/issues/1120), [#1152](https://github.com/obra/superpowers/issues/1152), and [#1251](https://github.com/obra/superpowers/issues/1251) report review amplification; keep one root acceptance path and avoid parallel reviewer lanes.
- Observation: issue [#1988](https://github.com/obra/superpowers/issues/1988) reports an oversized Codex task taking about four hours and repeating the same failure class; add an outcome-domain sizing preflight and causal-class recovery rather than arbitrary time or retry caps.
- Observation: issue [#597](https://github.com/obra/superpowers/issues/597) shows that worktrees can still share databases and ports; make exclusive resources structural planning data.
- Observation: issues [#1725](https://github.com/obra/superpowers/issues/1725) and [#1835](https://github.com/obra/superpowers/issues/1835) expose resume and fan-out limitations; persist only compact active-wave state in the existing ledger and retain sequential fallback.
- Limit: these are user reports, many from Claude Code workflows. They identify plausible failure modes but do not prove Codex App behavior or a universal performance effect.

## Evidence limits

- No source proves that arbitrary write tasks are safe to parallelize.
- No current source supplies a universal optimal concurrency limit; the implementation workflow must stay bounded by available App capacity and fall back to sequential execution when independence is uncertain.
- Live behavior still needs repository eval coverage; documentation evidence is not behavioral proof.
- Upstream timing and token claims do not establish this template's wall-clock improvement; compare representative live runs before making a speed claim.

## Stop rationale

The authoritative platform contract and the strongest named reference agree on the decision boundary: parallelize only positively independent work, isolate each write lane, and validate the combined result. More sources are unlikely to reveal a materially different viable policy family.
