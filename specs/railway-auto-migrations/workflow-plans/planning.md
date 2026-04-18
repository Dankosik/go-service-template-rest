Phase: planning
Phase status: completed
Artifact outputs:
- `tasks.md`
- `rollout.md`

Implementation readiness: PASS
Readiness rationale:
- design is stable enough for a small implementation slice
- no unresolved architecture/API/data/security blocker remains
- verification scope is clear: targeted package tests, guardrails, migration rehearsal, and a live run of the new migrator against ephemeral Postgres

Accepted concerns:
- automatic pre-deploy does not remove the need for mixed-version-safe migrations while Railway overlap remains enabled

Completion marker: approved task ledger exists and proof obligations are explicit enough for implementation to start in the same session under the recorded lightweight-local waiver.
Stop rule: do not add new design decisions to `tasks.md` during coding.
Next action: execute `T001` through `T004`.
