Phase: specification
Phase status: completed

Candidate decisions reconciled:
- Use Railway `preDeployCommand` as the single production migrator for this template.
- Do not run migrations from normal service startup or Docker build steps.
- Ship a dedicated migration binary inside the runtime image so pre-deploy works with the existing distroless Dockerfile.
- Treat GitHub autodeploy and `Wait for CI` as documented operator prerequisites, not repo-managed settings.
- Require mixed-version-safe schema releases while Railway overlap/draining remain enabled.

Local clarification pass:
- Hidden assumption: can this be solved by app startup migrations? Resolved `no`; multi-replica/startup concurrency would violate one-migrator ownership.
- Hidden assumption: can the template enable GitHub autodeploy entirely in repo code? Resolved `no`; Railway service connection and `Wait for CI` stay operator-owned.
- Hidden assumption: do automatic pre-deploy migrations make destructive schema steps safe? Resolved `no`; same-deploy schema changes still need expand/contract compatibility because old and new deployments overlap.

Spec readiness: approved
Completion marker: `spec.md` captures stable decisions, constraints, assumptions, and validation obligations.
Stop rule: do not turn this file into technical design or task breakdown.
Next action: produce the design bundle and rollout notes needed for planning.
