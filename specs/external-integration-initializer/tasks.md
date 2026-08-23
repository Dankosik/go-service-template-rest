# Goal

status: in_progress

Completion: T001 remains accepted; T002's user-authorized maintainability
repair is terminal-success for the current 23-row Test Design matrix, focused Go
proof, required integrated review, and no provider/network/credential/deployment
claim is made.

## Tasks

- [x] T001: Template initialization retains bounded outbound HTTP as one
  explicit locked capability.
- [ ] T002: The source template provides the fail-closed External Integration
  Initializer with named-only OAuth config, library-owned token cache/refresh,
  retained security boundaries, and all 23 current matrix rows.
  - Depends on: T001 and local pinned tools.
  - Provides: initial/no-op/generated-only-refresh HTTP/gRPC scaffold with named
    config, retained transports/auth, canonical drift, lifecycle, docs, and
    proof.
  - Packet: `tasks/T002-external-integration-initializer.md`.

## No-implementation dispositions

- Actual `.env` reading, migration, backup, mutation, restoration, locking, or
  arbitration remains outside the initializer.
- Provider operations, compatibility, credentials, registration, live network,
  deployment, rollout, readiness policy, and business mapping remain separate
  authorized work.
