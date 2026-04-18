Phase: workflow-planning
Phase status: completed
Execution shape: lightweight local
Research mode: local repo analysis plus `exa` retrieval of official Railway docs
Why no fan-out: task stayed bounded to one delivery/runtime seam; no subagent authorization or multi-lane research was needed.

Artifact expectations:
- `spec.md`: expected
- `research/*.md`: expected
- `design/`: expected
- `tasks.md`: expected
- `rollout.md`: expected
- `test-plan.md`: not expected

Completion marker: routing, artifact expectations, and lightweight-local waiver are explicit enough to proceed through local research/spec/design/planning in one session.
Stop rule: do not let this file absorb spec, design, or task-ledger authority.
Next action: research Railway pre-deploy and GitHub autodeploy behavior, then finalize decisions in `spec.md`.
