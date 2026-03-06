## Context
- The user wants the external subagent pack from `C:\Users\danii\Downloads\subagent-pack\subagent-pack\agents` integrated into the local AI tooling so the agents are actually usable in both Codex and Claude Code.
- The provided pack consists of read-only agent definitions in `*.toml` files with `sandbox_mode` and `developer_instructions`.
- The repository currently has no project-scoped `.codex/` configuration.
- The current user-level Codex config lives at `/home/dankos/.codex/config.toml`, and this repository is not yet listed as a trusted project there.
- Official Codex docs indicate:
  - custom agents are declared under `[agents.<name>]` and can point to an external `config_file`;
  - project-scoped `.codex/config.toml` is loaded only for trusted projects.
- Official Claude Code docs indicate:
  - project subagents live under `.claude/agents/`;
  - each subagent is a Markdown file with YAML frontmatter and instruction body content.

## Scope / Non-goals
- In scope:
  - copy the provided agent TOML files into a project-scoped Codex directory without changing their instructions;
  - add repository-local Codex agent registrations that point to those TOML files;
  - enable this repository to load project-scoped Codex config by adding the required trust entry in the user config;
  - add repository-local Claude Code subagent files that preserve the same instruction body in Claude’s file format;
  - add `CLAUDE.md` as a repository-local copy of `AGENTS.md` for Claude-facing compatibility;
  - validate that Codex and Claude Code accept and can read the new agent setup.
- Non-goals:
  - rewriting or “improving” the agent instruction bodies;
  - changing `AGENTS.md`, repository skills, or external skill mirrors;
  - introducing unrelated global Codex behavior changes outside what is required for this repository.

## Constraints
- Agent instruction content must remain unchanged.
- Repository changes should stay limited to the project-scoped Codex setup.
- User-level Codex changes should be minimal and only support loading this repository’s project config.
- Claude Code support must not depend on ad hoc per-invocation `--agents` JSON blobs; the setup should be project-scoped.
- Validation must use fresh CLI evidence, not assumption.

## Decisions
- The external pack will be copied into `.codex/agents/` inside the repository so the setup is project-local, versionable, and does not depend on the original Downloads path remaining present.
- `.codex/config.toml` will declare one `[agents.<name>]` entry per provided agent and will use `config_file = "agents/<file>.toml"` so the agent instruction files stay unchanged.
- The new `.codex/` directory is intentionally separate from the repository’s existing skill-mirror automation; it is a project-scoped Codex runtime config surface, not a new skill source of truth.
- Claude Code support will live in `.claude/agents/` because that is the project-scoped location Claude Code loads for custom subagents.
- Each Claude Code subagent file will preserve the TOML `developer_instructions` text verbatim as the Markdown body; only Claude-specific YAML frontmatter will be added around it.
- Claude Code subagents will be restricted to read-only research tools via frontmatter to stay aligned with the original `sandbox_mode = "read-only"` intent.
- `CLAUDE.md` will be kept as an exact repository-local copy of `AGENTS.md` rather than a fork, so there is no divergence in workflow instructions between tools.
- Agent registration names will match the provided pack names so the runtime role names stay aligned with the instruction files:
  - `api-agent`
  - `architecture-agent`
  - `concurrency-agent`
  - `data-agent`
  - `delivery-agent`
  - `design-integrator-agent`
  - `distributed-agent`
  - `domain-agent`
  - `performance-agent`
  - `qa-agent`
  - `quality-agent`
  - `reliability-agent`
  - `security-agent`
- The repository will be added to `/home/dankos/.codex/config.toml` under `[projects."<repo-path>"]` with `trust_level = "trusted"` so Codex loads `.codex/config.toml` here.
- Validation will use Codex CLI commands that load the repository config and inspect successful startup/response behavior, plus direct file checks for the copied pack.
- Validation will also use Claude Code CLI with explicit `--agent <name>` selection against the project-scoped `.claude/agents/` files.

## Open Questions / Assumptions
- Assumption: the installed Codex CLI version (`0.111.0`) supports the documented `[agents.<name>]` + `config_file` format and project-scoped `.codex/config.toml`.
- Assumption: agent descriptions can be supplied in the registering config while the source TOML files remain unchanged.
- Assumption: the installed Claude Code CLI supports project-scoped `.claude/agents/*.md` subagents as documented.
- Open question for validation: Codex may not expose a direct “list agents” command, so runtime validation may need to use a targeted `codex exec` prompt that loads project config successfully rather than a dedicated registry dump.

## Implementation Plan
1. Create the project-scoped Codex directory and spec-aligned config skeleton.
   Completion criteria:
   - `.codex/config.toml` exists in the repository;
   - the file registers every provided agent via `[agents.<name>]`.
2. Copy the provided agent pack into the repository without modifying instruction content.
   Completion criteria:
   - every source `*.toml` file exists under `.codex/agents/`;
   - copied file contents match the source files byte-for-byte.
3. Enable project-scoped Codex loading for this repository.
   Completion criteria:
   - `/home/dankos/.codex/config.toml` contains a trusted-project entry for `/mnt/c/Users/danii/IdeaProjects/go-service-template-rest`;
   - no unrelated user-level settings are changed.
4. Validate the setup with fresh Codex CLI evidence.
   Completion criteria:
   - Codex CLI accepts the combined config without parse errors;
   - a targeted non-interactive run succeeds with the project config loaded;
   - copied agent files are confirmed identical to the source pack.
5. Add Claude Code project subagents derived from the same instruction set.
   Completion criteria:
   - `.claude/agents/` contains one Markdown file per provided agent;
   - each file contains Claude frontmatter plus the unchanged instruction body from the source TOML.
6. Validate Claude Code project subagent loading.
   Completion criteria:
   - Claude Code agent files pass static shape checks for frontmatter and preserved body content;
   - if the local Claude account is available, Claude Code accepts `--agent <name>` for at least one project subagent without “agent not found” failure;
   - if the local Claude account is unavailable, record the auth block explicitly rather than claiming runtime success.
7. Add the Claude-compatible top-level instruction file.
   Completion criteria:
   - `CLAUDE.md` exists at the repository root;
   - `CLAUDE.md` matches `AGENTS.md` exactly.

## Validation
- `cmp -s <source-agent-file> .codex/agents/<file>` for each provided TOML
- `codex exec --json --ephemeral --output-last-message /tmp/codex-basic-last.txt "Reply with exactly OK."`
- `codex exec --json --ephemeral --output-last-message /tmp/codex-agent-last.txt "Use the custom architecture-agent to inspect the first line of AGENTS.md, then return exactly ARCH_OK."`
- compare extracted TOML `developer_instructions` body to each generated `.claude/agents/*.md` body
- shell checks for `.claude/agents/*.md` frontmatter shape: `name`, quoted `description`, `tools`, and `---` delimiters
- `claude -p --agent architecture-agent --tools "" "Reply exactly with the first bullet under Mission in your system prompt. If unavailable, reply NO_ROLE."`
- `cmp -s AGENTS.md CLAUDE.md`

## Outcome
- Copied all 13 provided agent TOML files into `.codex/agents/` without changing their `developer_instructions`.
- Added `.codex/config.toml` with one `config_file` registration per provided agent name.
- Added `/mnt/c/Users/danii/IdeaProjects/go-service-template-rest` to `/home/dankos/.codex/config.toml` as a trusted project so Codex loads the repository’s `.codex/config.toml`.
- Added `.claude/agents/` with 13 project-scoped Claude Code subagents derived from the same source instructions.
- Added `CLAUDE.md` as an exact copy of `AGENTS.md`.
- Codex validation already passed:
  - all copied TOML files matched the source pack byte-for-byte;
  - `codex exec` loaded and returned `OK` without config parse errors;
  - a targeted `codex exec` run completed with `ARCH_OK` and emitted a `spawn_agent` collaboration call while operating from the project config.
- Claude Code static validation passed:
  - all 13 generated `.claude/agents/*.md` files preserved the source `developer_instructions` body exactly;
  - all 13 files passed frontmatter shape checks for `name`, quoted `description`, `tools`, and frontmatter delimiters.
  - `CLAUDE.md` matched `AGENTS.md` exactly.
- Claude Code live runtime validation is currently blocked by account access:
  - `claude -p --agent architecture-agent --tools "" -- "Reply exactly with the first line of your active system instructions. If unavailable, reply NO_ROLE."`
  - CLI result: `Your account does not have access to Claude Code. Please run /login.`
- Residual environment note:
  - Codex auto-disabled `js_repl` during validation because the configured Node runtime is `v22.17.0` and the current CLI requires `>= v22.22.0` for `js_repl`; this did not block custom agent loading or multi-agent delegation.
