# Agent Contract Refresh Tasks

Created: 2026-04-12
Status: Phase 7 complete; Validation Phase 1 pending

Implementation readiness: PASS. Phase 7 completed; next session starts with `validation-phase-1`.

## Phase 1: Challenger Three-Mode Contract

- [x] T001 [Phase 1] Update `.codex/agents/challenger-agent.toml` so `Mission`, `Use when`, `Mode routing`, `Skill policy`, `Return`, and `Escalate when` cover `workflow-plan-adequacy-challenge`, `pre-spec-challenge`, and `spec-clarification-challenge`. Depends on: none. Proof: `rtk rg -n "workflow-plan-adequacy-challenge|pre-spec-challenge|spec-clarification-challenge" .codex/agents/challenger-agent.toml` and TOML parse check.
- [x] T002 [Phase 1] Update `.claude/agents/challenger-agent.md` with equivalent three-mode challenger semantics while preserving Claude frontmatter and read-only tools. Depends on: T001. Proof: `rtk rg -n "workflow-plan-adequacy-challenge|pre-spec-challenge|spec-clarification-challenge" .claude/agents/challenger-agent.md`.
- [x] T003 [Phase 1] Recheck README challenger wording and update it only if it conflicts with the fixed runtime role. Depends on: T001, T002. Proof: `rtk sed -n '108,116p' README.md`.
- [x] T004 [Phase 1] Validate the challenger slice. Depends on: T001-T003. Proof: TOML parse check for `.codex/agents/challenger-agent.toml`, three-skill `rtk rg` check for both challenger files, and `rtk git diff --check`.

## Phase 2: Observability Mirror And README Inventory

- [x] T010 [Phase 2] Create `.claude/agents/observability-agent.md` from the semantics in `.codex/agents/observability-agent.toml`, preserving read-only advisory behavior and the current no-dedicated-review-skill boundary. Depends on: T004. Proof: `rtk test -f .claude/agents/observability-agent.md` and `rtk rg -n "go-observability-engineer-spec|not a default review|read-only" .claude/agents/observability-agent.md`.
- [x] T011 [Phase 2] Update README agent inventory to include `observability-agent` with a link to `.claude/agents/observability-agent.md`. Depends on: T010. Proof: `rtk rg -n "observability-agent" README.md`.
- [x] T012 [Phase 2] Compare `.codex/agents` and `.claude/agents` inventories and confirm no unexpected runtime mirror gaps remain. Depends on: T010, T011. Proof: `rtk zsh -lc 'diff -u <(find .codex/agents -maxdepth 1 -name "*.toml" -exec basename {} .toml \; | sort) <(find .claude/agents -maxdepth 1 -name "*.md" -exec basename {} .md \; | sort)'`, plus a short note in `tasks.md` progress if any intentional difference remains.
- [x] T013 [Phase 2] Validate the observability inventory slice. Depends on: T010-T012. Proof: README link check, agent inventory check, TOML parse check if Codex TOML changed, and `rtk git diff --check`.

## Phase 3: Return Contracts For Review-Focused Agents

- [x] T020 [Phase 3] Update Codex review-focused agents with the shared review return contract: `Findings by severity`, `Evidence`, `Why it matters`, `Validation gap`, `Handoff`, and `Confidence`. Scope: `.codex/agents/concurrency-agent.toml`, `.codex/agents/data-agent.toml`, `.codex/agents/domain-agent.toml`, `.codex/agents/performance-agent.toml`, `.codex/agents/qa-agent.toml`, `.codex/agents/quality-agent.toml`, `.codex/agents/reliability-agent.toml`, `.codex/agents/security-agent.toml`. Depends on: T013. Proof: `rtk rg` check for the shared fields in the scoped files and TOML parse check.
- [x] T021 [Phase 3] Update matching Claude review-focused agents with equivalent review return contracts. Scope: `.claude/agents/concurrency-agent.md`, `.claude/agents/data-agent.md`, `.claude/agents/domain-agent.md`, `.claude/agents/performance-agent.md`, `.claude/agents/qa-agent.md`, `.claude/agents/quality-agent.md`, `.claude/agents/reliability-agent.md`, `.claude/agents/security-agent.md`. Depends on: T020. Proof: `rtk rg` check for the shared fields in the scoped files.
- [x] T022 [Phase 3] Recheck that review-focused agents still preserve their one-skill-per-pass and read-only advisory boundaries after return-section edits. Depends on: T020, T021. Proof: targeted `rtk rg` check for `Use at most one skill per pass` and read-only/advisory wording in the scoped files.
- [x] T023 [Phase 3] Validate the review return-contract slice. Depends on: T020-T022. Proof: TOML parse check for touched Codex files and `rtk git diff --check`.

## Phase 4: Return Contracts For Advisory And Mixed-Mode Agents

- [x] T030 [Phase 4] Update Codex advisory and mixed-mode agents with the shared research/adjudication return contract: `Conclusion`, `Evidence`, `Open risks`, `Recommended handoff`, and `Confidence`. Scope: `.codex/agents/api-agent.toml`, `.codex/agents/architecture-agent.toml`, `.codex/agents/challenger-agent.toml`, `.codex/agents/delivery-agent.toml`, `.codex/agents/design-integrator-agent.toml`, `.codex/agents/distributed-agent.toml`, `.codex/agents/observability-agent.toml`. Depends on: T023. Proof: `rtk rg` check for the shared fields in the scoped files and TOML parse check.
- [x] T031 [Phase 4] Update matching Claude advisory and mixed-mode agents with equivalent research/adjudication return contracts, including `.claude/agents/observability-agent.md` from Phase 2. Depends on: T030. Proof: `rtk rg` check for the shared fields in the scoped files.
- [x] T032 [Phase 4] Recheck that `delivery-agent`, `distributed-agent`, and `observability-agent` still say they are not default review lanes. Depends on: T030, T031. Proof: `rtk rg -n "not a default review|no dedicated .*review skill|targeted.*recheck" .codex/agents/delivery-agent.toml .codex/agents/distributed-agent.toml .codex/agents/observability-agent.toml .claude/agents/delivery-agent.md .claude/agents/distributed-agent.md .claude/agents/observability-agent.md`.
- [x] T033 [Phase 4] Validate the advisory return-contract slice. Depends on: T030-T032. Proof: TOML parse check for touched Codex files and `rtk git diff --check`.

## Phase 5: Inspect-First Blocks For Runtime And Domain Roles

- [x] T040 [Phase 5] Add concise `Inspect first` blocks to Codex runtime/domain roles: `api-agent`, `concurrency-agent`, `data-agent`, `domain-agent`, `observability-agent`, `performance-agent`, `reliability-agent`, and `security-agent`. Depends on: T033. Proof: `rtk rg -n "Inspect first" .codex/agents/api-agent.toml .codex/agents/concurrency-agent.toml .codex/agents/data-agent.toml .codex/agents/domain-agent.toml .codex/agents/observability-agent.toml .codex/agents/performance-agent.toml .codex/agents/reliability-agent.toml .codex/agents/security-agent.toml`.
- [x] T041 [Phase 5] Add equivalent `Inspect first` blocks to matching Claude runtime/domain roles. Depends on: T040. Proof: `rtk rg -n "Inspect first" .claude/agents/api-agent.md .claude/agents/concurrency-agent.md .claude/agents/data-agent.md .claude/agents/domain-agent.md .claude/agents/observability-agent.md .claude/agents/performance-agent.md .claude/agents/reliability-agent.md .claude/agents/security-agent.md`.
- [x] T042 [Phase 5] Spot-check inspect-first paths against `docs/repo-architecture.md` and approved spec examples, keeping each list to 3-6 targeted surfaces. Depends on: T040, T041. Proof: manual diff review plus targeted `rtk rg` for core paths such as `api/openapi/service.yaml`, `internal/app`, `cmd/service/internal/bootstrap`, and `internal/infra/telemetry` where relevant.
- [x] T043 [Phase 5] Validate the runtime/domain inspect-first slice. Depends on: T040-T042. Proof: TOML parse check for touched Codex files and `rtk git diff --check`.

## Phase 6: Inspect-First Blocks For Workflow And Meta Roles

- [x] T050 [Phase 6] Add concise `Inspect first` blocks to Codex workflow/meta roles: `architecture-agent`, `challenger-agent`, `delivery-agent`, `design-integrator-agent`, `distributed-agent`, `qa-agent`, and `quality-agent`. Depends on: T043. Proof: `rtk rg -n "Inspect first" .codex/agents/architecture-agent.toml .codex/agents/challenger-agent.toml .codex/agents/delivery-agent.toml .codex/agents/design-integrator-agent.toml .codex/agents/distributed-agent.toml .codex/agents/qa-agent.toml .codex/agents/quality-agent.toml`.
- [x] T051 [Phase 6] Add equivalent `Inspect first` blocks to matching Claude workflow/meta roles. Depends on: T050. Proof: `rtk rg -n "Inspect first" .claude/agents/architecture-agent.md .claude/agents/challenger-agent.md .claude/agents/delivery-agent.md .claude/agents/design-integrator-agent.md .claude/agents/distributed-agent.md .claude/agents/qa-agent.md .claude/agents/quality-agent.md`.
- [x] T052 [Phase 6] Recheck that `challenger-agent` inspect-first guidance distinguishes `workflow-plan-adequacy-challenge`, `pre-spec-challenge`, and `spec-clarification-challenge`. Depends on: T050, T051. Proof: three-skill `rtk rg` check in both challenger runtime files.
- [x] T053 [Phase 6] Validate the workflow/meta inspect-first slice. Depends on: T050-T052. Proof: TOML parse check for touched Codex files and `rtk git diff --check`.

## Phase 7: Safe Deduplication And Drift-Policy Checkpoint

- [x] T060 [Phase 7] Trim repeated global policy from Codex agent files only where role-local mission, boundaries, mode routing, return contract, inspect-first guidance, and escalation rules remain explicit. Depends on: T053. Proof: manual diff review, TOML parse check, and targeted `rtk rg` for read-only/advisory/one-skill-per-pass guardrails.
- [x] T061 [Phase 7] Trim repeated global policy from Claude agent files under the same safety rule. Depends on: T060. Proof: manual diff review and targeted `rtk rg` for read-only/advisory/one-skill-per-pass guardrails.
- [x] T062 [Phase 7] Recheck README and runtime inventory consistency after all agent edits. Depends on: T060, T061. Proof: README `observability-agent` link check and `rtk zsh -lc 'diff -u <(find .codex/agents -maxdepth 1 -name "*.toml" -exec basename {} .toml \; | sort) <(find .claude/agents -maxdepth 1 -name "*.md" -exec basename {} .md \; | sort)'`.
- [x] T063 [Phase 7] Confirm broader backlog items were not silently implemented: canonical-source generation, CI drift checks, new review skills, nickname additions, or model/reasoning overrides. Depends on: T062. Proof: `rtk git status --short` and diff review focused on `.agents/skills`, CI files, `.codex/config.toml`, and runtime agent metadata.
- [x] T064 [Phase 7] Validate the deduplication and drift-policy checkpoint. Depends on: T060-T063. Proof: full TOML parse check, final return/inspect `rtk rg` checks, and `rtk git diff --check`.

## Validation Phase 1: Final Proof And Closeout

- [ ] T900 [Validation 1] Run full instruction-only validation across Codex TOML, Claude Markdown presence, README links, challenger skill routing, return contracts, inspect-first sections, and diff whitespace. Depends on: T064. Proof: commands listed in `plan.md` `Cross-Phase Validation Plan`.
- [ ] T901 [Validation 1] Update existing task-local closeout surfaces with fresh evidence: `spec.md` `Validation` / `Outcome`, `workflow-plan.md`, `tasks.md`, and `workflow-plans/validation-phase-1.md`. Depends on: T900. Proof: updated artifacts cite the actual validation commands and results.
- [ ] T902 [Validation 1] Confirm no prohibited Go runtime, generated code, migration, or skill-body changes landed in this instruction-only task. Depends on: T901. Proof: `rtk git status --short` and diff review.
- [ ] T903 [Validation 1] Mark the task done only if validation passes; otherwise record the exact reopen target. Depends on: T900-T902. Proof: final workflow artifacts show done or the named reopen phase.
