# Sequence

Status: approved

## Planning And Implementation Order

1. Planning session consumes the approved `spec.md` and this design bundle, then writes `plan.md`, `tasks.md`, and any phase-control files it chooses to use.
2. Implementation starts with the `challenger-agent` pair because it is the only blocking runtime-contract drift currently identified.
3. The next slice repairs the observability mirror and README inventory drift.
4. The next slices standardize return contracts and `Inspect first` blocks across the portfolio in small batches.
5. Deduplication runs after the stricter role-local sections exist, so removing repeated global policy does not weaken individual agent boundaries.
6. Drift-policy and optional ergonomics remain checkpoints or separate follow-up work unless planning explicitly keeps them as non-coding backlog tasks.

## Validation Sequence

Later planning should keep validation close to each slice:

- After challenger edits, verify both runtime files mention and route all three challenge skills.
- After observability mirror edits, verify the Claude agent file exists and README inventory includes the agent if mirrors are still expected.
- After return-format edits, verify representative research/adjudication and review agents contain the agreed output fields.
- After `Inspect first` edits, verify each role has concise source-of-truth starting surfaces.
- After TOML edits, run a TOML parse check against `.codex/agents/*.toml` and `.codex/config.toml`.
- After Markdown edits, run a lightweight text/reference check for `.claude/agents/*.md` and README links.

No Go tests are expected for the instruction-only task unless a later scope change touches Go runtime files.

## Failure And Reopen Points

- If a slice needs new review skills, reopen specification; do not fake review support inside existing agents.
- If mirror maintenance becomes too risky to do manually, stop at the drift-policy checkpoint and open a separate canonical-source or CI drift-check task.
- If a runtime format has constraints that prevent equivalent semantics, return to technical design before planning or implementation continues.
- If implementation discovers `AGENTS.md` or `docs/spec-first-workflow.md` contradicts the approved agent changes, reopen specification instead of silently editing the workflow contract.

## Parallelism Notes

The planning session may mark some later agent-batch edits parallelizable only after it defines disjoint file ownership. The first two slices should stay sequential because the challenger fix and observability inventory repair establish the contract shape that later standardization should follow.
