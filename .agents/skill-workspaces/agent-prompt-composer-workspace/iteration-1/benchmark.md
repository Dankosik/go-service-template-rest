# Skill Benchmark: agent-prompt-composer

**Model**: gpt-5.4-mini via codex exec (reasoning low)
**Date**: 2026-04-12T12:09:36Z
**Evals**: 0, 1, 2, 3, 4 (1 runs each per configuration)

## Summary

| Metric | New Skill | Old Skill | Delta |
|--------|------------|---------------|-------|
| Pass Rate | 100% ± 0% | 100% ± 0% | +0.00 |
| Time | 23.0s ± 5.5s | 20.8s ± 5.6s | +2.2s |
| Tokens | 3631 ± 1441 | 3098 ± 633 | +533 |

## Notes

- Single-run old-vs-new benchmark: deterministic checks show no pass-rate delta (new_skill 100%, old_skill 100%).
- The two new hardening evals both passed in old_skill and new_skill, so this benchmark does not prove measurable output improvement from the wording change.
- The edits are still directionally useful as explicit guardrails for wrapper/instruction-noise separation and evidence labels, but a larger or more adversarial eval set is needed to demonstrate a statistically reliable lift.
- After correcting the grader, all 5 evals pass for both configurations; the result is best interpreted as no regression plus no proven quantitative gain.
