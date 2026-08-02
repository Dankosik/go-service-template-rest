# Durable Messaging V1

status: complete

Current phase: Complete

Active artifact: none. Research, specification, technical design, test design,
planning, implementation, independent reviews, validation, and closeout are
closed.

Accepted outcome: one concrete opt-in durable-messaging initialization pack. `MESSAGING=none` must physically remove every messaging-owned source, dependency, configuration, binary, command, test, CI/profile wire, and document. The selected profile must provide bounded producer and pull-consumer operation, explicit acknowledgement/redelivery/retry/DLQ behavior, trace and correlation propagation, reconnect/readiness behavior, graceful drain plus forced shutdown, and a separate consumer worker composition path.

Authority boundary: local research, workflow artifacts, capability-owned edits, non-destructive proof, and local commits are authorized. No push, PR, deployment, purchase, cloud-resource creation, or other external write is authorized. All work remains in `/Users/daniil/.codex/worktrees/durable-messaging-v1` on `codex/durable-messaging-v1`, based on `bb9ea48d8634b62ca88da1f87ab819cb41389be6`.

Constraints: select exactly one transport; no lowest-common-denominator broker abstraction; no outbox or inbox persistence; feature event semantics and payloads remain feature-owned; broker-semantic claims require a real broker process or Testcontainers; broad Go or Docker gates are serialized.

Repository facts: initialization already removes optional PostgreSQL, gRPC, authentication, and outbound-HTTP profile surfaces. Runtime composition is owned by `cmd/service/internal/bootstrap`; reusable lifecycle supervision by `internal/background`; readiness by `internal/health`; telemetry by `internal/infra/telemetry`; optional integration adapters by `internal/infra/<integration>`; real dependency tests already use Testcontainers.

Bounded assumption: the capability is a template-owned initialization pack, not a fleet rollout. Reopen Research if no transport can dominate without an unavailable fleet-owned infrastructure decision.

Accepted validation tree: `39b01a3bfbc51a15e818dde82e3f28ea94f808a1`.
The reviewed production tree was `12397a8be8fbca0ba58b75e5ffbea465e8a8c08d`;
the later changes were test-only coverage oracles and this closeout record.

Completion proof: accepted real-broker semantics, race/leak and lifecycle proof, producer-only and worker initialized profiles, `MESSAGING=none` purity, pre-mutation invalid-profile rejection, repeated-initialization byte stability, independent high-risk implementation review, clean task-owned local commit, and no unresolved material reliability or lifecycle finding.
