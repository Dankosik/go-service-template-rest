Phase: technical-design
Phase status: completed

Required design artifacts:
- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`

Conditional artifacts:
- `rollout.md`: triggered because deploy sequencing and mixed-version migration safety affect correctness
- `design/data-model.md`: not expected because schema ownership stays in `env/migrations/` and no new data model is introduced
- `test-plan.md`: not expected because proof obligations stay compact

Design readiness: approved
Completion marker: planning can derive executable tasks without inventing missing rollout or ownership detail.
Stop rule: do not start coding from this phase file.
Next action: write `tasks.md` with explicit proof obligations and implementation-readiness status.
