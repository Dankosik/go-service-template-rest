# Component Map

Status: approved

## Affected Surfaces

| Surface | Planned Role In This Task | Notes For Planning |
| --- | --- | --- |
| `.codex/agents/challenger-agent.toml` | Update the Codex challenger runtime instruction contract. | Must route three challenge modes and remain read-only/advisory. Keep TOML parseable and preserve `developer_instructions` formatting. |
| `.claude/agents/challenger-agent.md` | Update the Claude challenger runtime instruction contract. | Must stay semantically equivalent to the Codex challenger role while preserving Claude frontmatter and Markdown format. |
| `.codex/agents/observability-agent.toml` | Source for the missing Claude mirror, unless intentional Codex-only policy is discovered. | Existing role is read-only and uses `go-observability-engineer-spec`; no dedicated review skill exists. |
| `.claude/agents/observability-agent.md` | Add missing Claude runtime instruction file if planning keeps the mirror repair slice. | Should mirror the Codex observability semantics in Claude agent-file format. |
| `README.md` | Repair agent inventory and common usage docs where they drift from project-scoped runtime files. | Must include `observability-agent` if the Claude mirror is added. |
| `.codex/agents/*.toml` | Later standardization of return contracts, `Inspect first`, and safe deduplication. | Apply incrementally by role group; avoid broad unrelated rewrites. |
| `.claude/agents/*.md` | Later standardization of return contracts, `Inspect first`, and safe deduplication. | Keep semantic parity with matching Codex files where both runtimes expose the same role. |
| `.codex/config.toml` | Inventory reference for Codex registered agents. | Already registers `observability-agent`; change only if planning finds config drift. |

## Stable Authority Surfaces

| Surface | Stable Role | Change Policy |
| --- | --- | --- |
| `AGENTS.md` | Repository-wide orchestration contract and phase gates. | Do not rewrite unless implementation finds concrete drift caused by this task. |
| `docs/spec-first-workflow.md` | Workflow artifact mechanics and phase handoff details. | Use as context; do not turn this task into workflow redesign. |
| `.agents/skills/*/SKILL.md` | Canonical skill bodies and routing targets. | Do not modify in this task cycle; agent files route to existing skills. |
| `docs/repo-architecture.md` | Stable Go service architecture baseline. | Read for task-local design only; no runtime boundary changes are planned. |

## Role Grouping For Later Planning

Planning should split implementation by low-conflict surfaces:

1. `challenger-agent` pair: `.codex/agents/challenger-agent.toml` and `.claude/agents/challenger-agent.md`.
2. Observability inventory: `.codex/agents/observability-agent.toml`, new `.claude/agents/observability-agent.md`, and README agent table.
3. Shared return contracts: all runtime agent files, applied in small batches.
4. `Inspect first` blocks: all runtime agent files, applied in role-domain batches.
5. Deduplication: all runtime agent files, only after return and inspect-first contracts are stable.
6. Drift-policy checkpoint: README/config/docs only, or a separate reopened task if tooling/CI becomes desired.

## Conditional Artifacts

No task-local conditional design artifacts are triggered. The work touches instruction files and documentation only; it does not alter data ownership, runtime module dependencies, generated contracts, service behavior, deployment policy, or migration choreography.
