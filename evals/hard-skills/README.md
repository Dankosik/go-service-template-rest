# Hard-Skill Evaluation Runbook

This catalog isolates the decision method of every promoted `model/method`
skill. Coverage is derived from current frontmatter: each catalog entry needs
trigger, non-trigger, decision, completion, and an incident edge from the
canonical [neighbor graph](../../.agents/contracts/specialist-neighbors.json).
Every graph edge needs a contrastive collision case. Skills with selectors also
need a reference-selection case; the mutation suite holds concrete plausible
wrong defaults.

Validate structure with:

```bash
bash scripts/ci/instruction-evals-check.sh
```

Run the general evaluator with the same target-harness command documented by
the [instruction evaluation runbook](../instructions/README.md):

```bash
INSTRUCTION_EVAL_HARNESS=codex-cli \
INSTRUCTION_EVAL_MODEL=<model> \
INSTRUCTION_EVAL_REASONING_EFFORT=<effort> \
INSTRUCTION_EVAL_TOOL_PROFILE=repository-default \
INSTRUCTION_EVAL_COMMAND_LABEL=<label> \
INSTRUCTION_EVAL_TRACE_FORMAT=codex-jsonl \
INSTRUCTION_EVAL_FILE=evals/hard-skills/evals.json \
INSTRUCTION_EVAL_REPEATS=3 \
INSTRUCTION_EVAL_VARIANTS=baseline,candidate,routing_ablation,method_ablation,reference_ablation \
scripts/dev/instruction-evals-run.sh -- <agent-command>
```

Run at least three identical repeats for baseline, candidate,
routing ablation, and method ablation; selector cases also run reference
ablation. Grade every expectation before promotion. Tokens, latency, and tool
calls count only after behavior passes; structural validation proves no model
improvement.
