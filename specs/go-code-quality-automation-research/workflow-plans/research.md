# Research Phase Plan

## Status

- Phase: research
- Status: complete
- Mode: read-only fan-out plus local synthesis
- Completion marker: lane outputs reconciled into a concise evidence-backed proposal, with preserved research note updated.
- Stop rule: do not write `spec.md`, `design/`, `plan.md`, `tasks.md`, or implementation changes.

## Lanes

| Lane | Owner | Question | Skill | Evidence Target | Status |
| --- | --- | --- | --- | --- | --- |
| A | subagent | What Go code quality automation is currently configured locally, in CI, and nightly, and where are the obvious repo-local gaps? | no-skill | `Makefile`, `.golangci.yml`, `.github/workflows/ci.yml`, `.github/workflows/nightly.yml`, `scripts/ci/*`, `scripts/dev/docker-tooling.sh`, docs | complete |
| B | subagent | Which additional ready-made Go linters or tools are good candidates, and which are too noisy for beginner/default use? Do not perform a general Go code review; evaluate tool candidates, default/noise risk, and recommended run location only. | `go-idiomatic-review` | current `.golangci.yml`, `go tool golangci-lint linters` if useful, current tool docs where needed | complete |
| C | subagent | How strong is local/CI/Docker parity for non-expert template users, and what workflow/documentation changes would reduce mistakes? | `go-devops-review` | `Makefile`, `docs/build-test-and-development-commands.md`, CI/nightly workflows, Docker tooling wrapper, branch-protection script | complete |
| D | subagent | Are test, coverage, race, fuzz, and generated-code drift gates arranged well for Go code health, and what should be quick vs full vs nightly? | `go-qa-review` | test-related Make targets, CI/nightly jobs, docs, coverage behavior | complete |

## Fan-In

- Compare lane findings against local inspection before treating them as recommendations.
- Classify recommendations as `do now`, `maybe later`, or `skip for now`.
- For each recommendation, include rationale, benefit, cost/noise risk, likely files or commands to change later, and where it should run.
- Result: reconciled in `research/go-code-quality-automation.md`; final chat report should summarize that note, not raw lane dumps.

## Adequacy Challenge

- Required: yes.
- Status: complete.
- Reconciliation: handoff_ok; one non-blocking Lane B focus concern was tightened in the lane question.

## Parallelism

- Lanes A-D can run in parallel after the adequacy challenge is reconciled.
- Local static inspection can continue in parallel with subagent work as long as it does not duplicate lane decisions.

## Blockers

- None known.

## Completion

- Completion marker met: yes.
- Stop rule honored: no `spec.md`, `design/`, `plan.md`, `tasks.md`, or implementation changes were created.
- Next action: user decision on whether to implement one or more recommendations.
