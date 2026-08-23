# Instruction Evaluation Runbook

Read only for an authorized live baseline/candidate comparison. Prompt
Maintenance owns whether live evaluation is required and what its result may
claim.

Use [`evals.json`](evals.json) for the minimum reusable comparison set. Run
baseline and candidate in disposable copies of the same repository state with
the same target harness, model, reasoning effort, tools, and task inputs. Grade
every expectation against the instruction and skill loads, tool trace, retained
diff, proof, approval stops, and final answer. Compare bootstrap and total
tokens, tool calls, and latency only after behavior passes. Run without
production credentials and deny external writes; authority-boundary cases pass
by stopping before the effect.

Use the separate [hard-skill suite](../hard-skills/README.md) when the claim is
about one model-invoked method rather than repository workflow behavior.

Validate the case schema with `scripts/ci/instruction-evals-check.sh`. That
check proves the eval surface, not model behavior.

Run a live comparison with `scripts/dev/instruction-evals-run.sh`. It
materializes disposable copies, applies the shared fixed fixture, strips
ordinary environment credentials, and captures each side's trace, retained
diff, status, run metadata, metrics, duration, and exit code. The agent command
follows `--` and reads the case prompt from stdin:

```bash
INSTRUCTION_EVAL_BASELINE_REF=HEAD \
INSTRUCTION_EVAL_CANDIDATE=worktree \
INSTRUCTION_EVAL_CASES=1,5,7 \
INSTRUCTION_EVAL_HARNESS=codex-cli \
INSTRUCTION_EVAL_MODEL=gpt-5.6-terra \
INSTRUCTION_EVAL_REASONING_EFFORT=medium \
INSTRUCTION_EVAL_TOOL_PROFILE=repository-default \
INSTRUCTION_EVAL_COMMAND_LABEL=codex-terra-medium \
INSTRUCTION_EVAL_TRACE_FORMAT=codex-jsonl \
scripts/dev/instruction-evals-run.sh -- <agent-command>
```

Use a target-harness command configured with the same model, reasoning effort,
tools, sandbox, and approval policy on both sides. `codex-jsonl` traces produce
token, tool-call, and skill-load metrics; other formats retain explicit
unavailable values. The runner leaves `behavior_verdict` as `ungraded`; inspect
`case.json`, every repeat/variant trace, status, patch, and final answer and
grade every expectation before claiming behavior. Its default remains one
`baseline,candidate` repeat; the hard-skill runbook selects routing, method, and
reference ablations plus the required repeat count through the same runner. A
repository comparison keeps the same global harness instructions on every
variant; evaluate a global-bootstrap candidate with fixed baseline and
candidate harness profiles.
