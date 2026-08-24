# Goal

status: done

Completion: T001 remains accepted. T002's user-authorized maintainability
repair is terminal-success for the current 23-row Test Design matrix, static
owner gates, and independent Specification, Technical Design, Test Design, and
Implementation reviews. No provider, credential, deployment, or live-network
claim is made.

## Tasks

- [x] T001: Template initialization retains bounded outbound HTTP as one
  explicit locked capability.
- [x] T002: The source template provides the fail-closed External Integration
  Initializer with named-only OAuth config, library-owned token cache/refresh,
  retained security boundaries, and all 23 current matrix rows.
  - Depends on: T001 and local pinned tools.
  - Provides: initial/no-op/generated-only-refresh HTTP/gRPC scaffold with named
    config, retained transports/auth, canonical drift, lifecycle, docs, and
    proof.
  - Packet: `tasks/T002-external-integration-initializer.md`.

Accepted: T002; evidence: full initializer matrix 23 plus focused repaired-oracle rows 3, focused Go 416, OpenAPI/Proto PASS, template initializer contract PASS, record/routing PASS, containerized ShellCheck PASS, secret scan 0 leaks, independent Specification/Technical Design/Test Design/Implementation reviews PASS; candidate: commit `cdfd44fc1744de009dda593f46833e807b31ac9a`

## No-implementation dispositions

- Actual `.env` reading, migration, backup, mutation, restoration, locking, or
  arbitration remains outside the initializer.
- Provider operations, compatibility, credentials, registration, live network,
  deployment, rollout, readiness policy, and business mapping remain separate
  authorized work.
