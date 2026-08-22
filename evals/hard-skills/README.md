# Hard-Skill Evaluation Runbook

This catalog isolates the decision method of promoted `model/method` skills.
[`coverage.json`](coverage.json) binds every skill to one trigger, non-trigger,
neighbor-collision, decision, and completion case; [`evals.json`](evals.json)
uses the shared disposable baseline/candidate runner.

Validate structure with:

```bash
bash scripts/ci/instruction-evals-check.sh
```

Run the same command documented by the [instruction evaluation
runbook](../instructions/README.md), adding:

```bash
INSTRUCTION_EVAL_FILE=evals/hard-skills/evals.json
```

Compare `HEAD` to an isolated candidate first. Then compare an ablated baseline
whose target `SKILL.md` keeps its metadata but omits the method body to the same
candidate. Keep model, effort, tools, fixture, and prompt identical. Grade
behavior before tokens or latency; structural validation alone proves no model
improvement.
