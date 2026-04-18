# Chosen Approach

Use Railway's deployment-native pre-deploy hook as the single production migrator and make the runtime image self-sufficient for that hook by shipping a dedicated migration binary plus migration files.

# Artifact Index

- `component-map.md`: approved
- `sequence.md`: approved
- `ownership-map.md`: approved
- `rollout.md`: approved and required because deploy sequencing affects correctness
- `data-model.md`: not expected; schema ownership stays unchanged
- `test-plan.md`: not expected; proof obligations fit in `tasks.md`

# Unresolved Seams

None for this bounded change. The remaining risk is operational discipline for destructive migrations, which is recorded as rollout guidance rather than left implicit.

# Readiness Summary

The design is planning-ready because it answers:
- where migrations execute
- how they get the same DSN/config contract as the service
- how the distroless image remains compatible with Railway pre-deploy
- what stays operator-managed outside repository code
