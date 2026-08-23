# Goal
status: blocked
Completion: T001 and T002 are accepted on one integrated candidate; the fixed Test Design validation ladder is terminal-success for that exact candidate; the required integrated Implementation Review passes; and no provider, network, credential, deployment, actual `.env`, or other external-effect claim is made.
Blocked stop: T002 acceptance requires a Linux runner with local `strace` and a cached `ghcr.io/zizmorcore/zizmor` image for delivery-quality; both are unavailable on this Darwin checkout. Same T002 unit resumes when those gates exist.
Global constraints: Preserve `test-design-transition.md` SHA-256 `8c4afa98856011370f897cdb5bcee903d985104d2d39cb2148e8b44ffd4dffc3` and its fixed Specification, Technical Design, Test Plan, and independent reviews. Treat all pre-existing dirty checkout bytes as evidence only and exclude them from every candidate unless separately accepted. Do not inspect, source, copy, or modify the checkout's actual `.env`; harness-created entries may exist only in disposable repositories under the fixed Test Plan. Use no network, provider account, credential, deployment, database, dependency addition, or other external effect.

## Tasks

- [x] T001: Template initialization retains bounded outbound HTTP as one explicit locked capability without changing any existing profile outcome.
  - Depends on: none
  - Provides: accepted `outbound_http = "none" | "bounded"` lock/readback and dependency-aware retention of the existing bounded HTTP owner
  - Packet: tasks/T001-retained-outbound-http-choice.md
Accepted: T001; evidence: `make template-init-check` exit 0, outbound-http cross-product omitted=none explicit=none bounded=present invalid=unchanged; review PASS; candidate: working-tree vs 8967a4ac06d4fce0515703b15ffa5db35e5378ae scripts/init-module.sh sha256 16a57f749549590e88f1eb0c3bbd205955ddcbc901b5dd8c7be00010a9647d36, scripts/ci/template-init-check.sh sha256 539fc20cbc87f59a69be0ddc8de13b2be8744d0e41dc48dcd4bfbc70feece113, README.md sha256 098f1fdcd2ad55d12bbbff81dc72f6b2a528cd1e0a01f1f1372314ce1ee42cae
- [ ] T002: The source template provides the complete fail-closed External Integration Initializer and satisfies all 25 fixed Test Design rows without inventing provider semantics or taking `.env` custody.
  - Depends on: T001 accepted outbound HTTP choice/lock/retention surface; mandatory local proof gates named in the packet are available
  - Provides: locally accepted initial/no-op/generated-only-refresh HTTP and gRPC scaffold initializer with named config, retained transports/auth, canonical generation/drift, lifecycle, documentation, and proof
  - Packet: tasks/T002-external-integration-initializer.md
Blocked: T002; unverified: nine-command ladder terminal-success including Linux `strace` custody and Docker delivery-quality; evidence: 25 matrix IDs printed; locally runnable commands 1,3-6,8,9 pass; step 2 Darwin/no-strace gap; step 7 zizmor pull denied; review not dispatched; next owner: Implementation same T002 when Linux local strace and cached zizmor image exist; candidate: worktree /Users/daniil/Projects/Opensource/go-service-template-rest.impl-external-integration-t002 branch impl/external-integration-t002 vs 8967a4ac06d4fce0515703b15ffa5db35e5378ae, scripts/integration-init.sh sha256 db0ce9fceb55bb4d1b2e4f3c3f8bac3d8d90ddc19a1fef7e6fbe0d10cb69210e, scripts/ci/integration-init-check.sh sha256 b5571d075c3752453601ebf4916a6d3b622b2404960000e1e9c0bb6c4a768cf0

## No-implementation dispositions

- Actual `.env` reading, migration, backup, mutation, restoration, locking, or arbitration remains developer/operator-owned and outside every task.
- Provider operations, provider compatibility, credentials, registration, live network use, deployment, rollout, readiness policy, and business mapping remain scope exits that require their own accepted authority and phases.
