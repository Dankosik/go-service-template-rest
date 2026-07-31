# OIDC/JWT Authentication Capability

status: accepted

## Accepted intake

Add one structurally optional initialized-service profile, `AUTHN=none|oidc-jwt`.
The `none` profile must contain no capability-owned source, configuration,
dependency, documentation, test, or bootstrap residue. The `oidc-jwt` profile
must authenticate OpenAPI-secured HTTP operations and, when `GRPC=enabled`,
gRPC calls with equivalent fail-closed JWT validation and publish the resulting
identity through `reqctx.Principal`.

Authentication ends at identity establishment. Authorization, RBAC, tenant
policy, provisioning, sessions, and domain ownership stay outside this
capability. No development or production bypass is permitted.

Local inspection, workflow artifacts, implementation, non-destructive
validation, and commits are authorized. Pushes, pull requests, deployments,
purchases, and other externally observable writes are forbidden.

## Routing

Current phase: Validation and Closeout

Path: orchestrated. Security-boundary decisions, two transports, generated
profile purity, new dependency selection, time-sensitive standards evidence,
and explicit independent reviews make a smaller path insufficient.

Active artifacts:

- `research/synthesis.md` (`accepted`; material first-review concerns repaired,
  fresh re-review concern was routing-only and is dispositioned here)
- `spec.md` (`accepted`; independent review `PASS` after three blocking forks were
  repaired)
- `design/overview.md` (`accepted`; independent review and adversarial lifecycle
  challenge both `PASS` after repairing key ambiguity, initial gRPC readiness,
  metrics classification, pre-Run cleanup, and failed-refresh cadence)
- `test-plan.md` (`accepted`; independent QA review `PASS`)
- `tasks.md` (`accepted`; independent task-readiness review `PASS`,
  implementation complete, claim-scoped implementation re-review `PASS`)

Blockers / assumptions:

- The worktree is isolated at base `41107602349dc46be942bfbcee3ad87454897252`.
- `AUTHN=none` remains the initializer default; reopen Specification only if
  repository authority establishes a different default.
- The selected design retains the existing OpenAPI fail-closed seam,
  `reqctx.Principal`, bounded outbound HTTP owner, and native gRPC lifecycle.
  Reopen Design only if deterministic proof cannot exercise one of those
  concrete seams without changing accepted behavior.

Implementation and validation result: T1 is complete. Focused, full-repository,
fuzz, race, profile-generation, OpenAPI, protobuf, formatting, lint, module,
security, secret-scanning, and production-binary redaction checks passed.
Independent claim-scoped implementation re-review returned `PASS` after the
bootstrap-order, finite HTTP-failure mapping, and literal refresh/readiness
proof gaps were repaired. No known material security finding remains within
the recorded evidence boundary.

Next action: create the authorized local task commit, verify its exact identity,
and prove the isolated worktree clean. This is carrier closeout only; a push,
pull request, merge, deployment, purchase, or other external write remains
forbidden.

Completion proof: `tasks.md` records the exact accepted proof. The final
task-owned commit must contain this accepted capability and canonical
documentation, and the post-commit worktree status must be clean.
