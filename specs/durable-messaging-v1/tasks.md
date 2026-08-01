# Durable Messaging V1 Implementation Ledger

status: complete

## Goal

Implement the selected direct NATS JetStream pack so an initialized service
contains either the complete bounded producer/worker capability or no messaging
surface at all. Completion requires current real-broker, lifecycle, profile,
and repository proof plus independent review of the fixed high-risk units.

Stop without a readiness/completion claim if the pinned `nats.go v1.52.0`
public API cannot preserve synchronous `PubAck`, confirmed explicit ACK,
single-message pull, exact durable delivery count, or bounded drain semantics;
reopen the narrow ready Design or Test Design owner with the failed observable.

Execution is sequential. The units share configuration/profile authorities and
all Docker-backed or broad Go gates remain serialized.

- [x] T1: implement the concrete bounded NATS JetStream transport and its
  package-owned semantic proof
  - Source: `spec.md` M03 through M10; `design/overview.md` concrete Go,
    producer, pull/ACK/retry/DLQ, readiness, telemetry, and drain decisions;
    `test-plan.md` TD-01, TD-02, TD-04 through TD-12.
  - Owner/surface/resources: `internal/infra/natsjs/`, `go.mod`, `go.sum`, and
    the TD-02/04–12 semantic cases plus shared NATS container helpers in
    `test/nats_messaging_integration_test.go`; direct
    `github.com/nats-io/nats.go v1.52.0`; test-local OTel/slog fixtures; unique
    Testcontainers streams/consumers with immediate cleanup. Do not add a
    transport interface, framework, outbox, inbox, schema registry, or feature
    event model.
  - Depends on: none.
  - Deliverable: immutable feature-owned `Event` input and consumer `Message`,
    bounded synchronous producer, admitted connection/topology lifecycle,
    one-at-a-time pull with bounded active handlers, explicit confirmed ACK,
    durable retry/DLQ policy, trace/correlation and fixed-cardinality telemetry,
    readiness, drain, and forced close.
  - Proof: all focused unit and race/goleak commands from
    TD-01/05/07/10/11/12, then the complete serialized real-broker aggregate
    `REQUIRE_DOCKER=1 make test-integration`. The aggregate includes every T1
    package semantic case plus the initialized service and worker process
    cases; it is the repository-owned oracle, not a hand-maintained regex.
    Expected observables are exact `PubAck` completion, one wire attempt,
    stable identity/lineage, bounded acquisition and resident bytes, durable
    reconnect/attempt state, final-attempt and poison handling, retained
    oversized/ambiguous source, non-serialized key observation, fixed safe
    telemetry, bounded shutdown, and no leak/race.
  - Fixed review unit: all T1 production paths and semantic tests at one exact
    tree; independent review must close data loss, false ACK, hidden retry,
    acquisition/memory, reconnect, and shutdown findings before T4.
  - Reopen if: any selected client primitive requires an unbounded queue or
    background publish retry; Reliability/Performance Design.

- [x] T2: compose producer-only service and separately scalable consumer
  worker without weakening existing lifecycle or health ownership
  - Source: `spec.md` M02, M08, M10; `design/overview.md` configuration,
    connection ownership, readiness, worker composition, and placement;
    `test-plan.md` TD-03, TD-04, TD-12, TD-13.
  - Owner/surface/resources: marker-scoped messaging config in
    `internal/config/` and `env/.env.example`; service wiring in
    `cmd/service/internal/bootstrap/`; new `cmd/worker/main.go` and
    `cmd/worker/internal/bootstrap/`; TD-03/13 composition/process cases added
    sequentially to T1's `test/nats_messaging_integration_test.go`. Reuse
    `internal/background`, `internal/health`, existing telemetry, config loading,
    signal, and process-test owners; do not rewrite T1 semantic cases/helpers.
  - Depends on: T1 — consumes its concrete client, producer, worker, readiness,
    shutdown methods, and root NATS fixture without adding an interface.
  - Deliverable: disabled/service-producer/worker validation paths; service has
    no consumer; worker rejects nil feature registration before any connection;
    topology validation precedes mutation; readiness is live and liveness is
    process-only; shutdown drains within the existing process budget.
  - Proof: TD-03/04/12/13 focused component and real-process commands. Expected
    observables are zero TCP connections from empty worker main, no service
    durable consumer, mutation-free startup failure, fresh reconnect readiness,
    supervised terminal exhaustion, graceful completion, and forced redelivery.
  - Fixed review unit: T2 diff plus the exact consumed T1 tree; independent
    review must close startup ordering, health, supervision, producer-only,
    worker admission, and cleanup-order findings before T4.
  - Reopen if: the current service shutdown budget cannot contain messaging
    drain and telemetry flush; Lifecycle Design.

- [x] T3: make messaging a physically optional, byte-stable initialization
  profile with selected-only build, test, CI, and operator documentation
  - Source: `spec.md` M01; `design/overview.md` package/profile placement;
    `test-plan.md` TD-14 and TD-15.
  - Owner/surface/resources: marker-scoped `Dockerfile`, `Makefile`, README and
    architecture text, CI/profile wiring, `scripts/init-module.sh`,
    `scripts/ci/template-init-check.sh`, `template.lock`, and selected-only
    `docs/durable-messaging.md`. Preserve every sibling profile and
    template-owned path invariant.
  - Depends on: T1 and T2 — their final path/dependency/config/command set is
    the exact deletion manifest.
  - Deliverable: exactly `MESSAGING=none|nats-jetstream`; validate before target
    mutation; selected builds service and worker and runs every messaging
    integration owner; none removes all messaging source, dependency, config,
    binary, command, test, CI wire, and document; repeats are byte-identical.
  - Proof: TD-14 structural matrix plus TD-15 aggregate expansion; expected
    observables are unchanged hashes for invalid/cross-profile re-selection,
    clean negative scans/`go list`/tidy/build/test for none, complete selected
    paths and commands, minimal/maximal compatibility, and repeated byte
    stability.
  - Fixed review unit: initializer/profile deletion map and selected-only build
    wiring at one exact tree; independent review must close residue,
    pre-mutation, and aggregate-omission findings before T4.
  - Reopen if: a capability-owned path is template-owned or cannot be removed
    without deleting a sibling profile owner; Profile Design.

- [x] T4: validate and close the fixed integrated candidate
  - Source: all M01–M10 and TD-01–TD-15; no new behavior or source owner.
  - Depends on: T1, T2, T3, and PASS reviews for their fixed high-risk units.
  - Proof order: focused package/component tests; required real-broker aggregate;
    focused race/goleak; messaging profile matrix; generated/module/structure
    drift; then repository `make check-full`, never overlapping broad Go or
    Docker gates. Rerun the smallest owning proof after every repair and rerun
    the integrated gate only when its inputs changed.
  - Acceptance: all current claim-scoped and integrated proof passes at the
    fixed reviewed tree, and no material reliability or lifecycle finding
    remains. This acceptance unit ends before the final ledger transition and
    commit.
  - Reopen if: any gate fails; route to the earliest owning task rather than
    weakening or skipping the oracle.

## Completion evidence

- Independent implementation re-review passed for all fixed high-risk units.
  The concrete transport/lifecycle and service/worker composition reviewers
  passed staged tree `12397a8be8fbca0ba58b75e5ffbea465e8a8c08d` after
  reconnect, fail-closed drain, panic supervision, process-exhaustion, and
  proof-selector repairs. The profile reviewer passed staged tree
  `994f5a64d5635955f2ac799ed7cb13cc430ada1f`; later changes did not alter
  profile-owned production or initializer paths.
- `REQUIRE_DOCKER=1 make test-integration` passed with the real pinned broker:
  root process cases 220.916s, package semantics 13.250s, worker composition
  1.366s. The repaired terminal fail-closed broker regression also passed.
- `go test -vet=off -race ./internal/infra/natsjs
  ./cmd/worker/internal/bootstrap ./cmd/service/internal/bootstrap -count=1`
  passed 144 tests across three packages after the production repairs.
- `make check-full` passed on staged validation tree
  `39b01a3bfbc51a15e818dde82e3f28ea94f808a1`, including lint, NilAway,
  full race, effective coverage 80.1%, security, real-broker/process,
  generated/module/structure, runtime image with both `/service` and `/worker`,
  migration, and container proof.
- `TEMPLATE_INIT_PROFILE=messaging make template-init-check` passed on the same
  validation tree, including `none`, explicit-empty/invalid, selected minimal,
  selected maximal, cross-selection, and repeated byte-stability oracles.
- No material reliability, lifecycle, composition, or profile finding remains.
  No push, PR, deployment, purchase, cloud resource, or other external write
  was performed.

## Closeout sequence

After T4 acceptance, record the exact final evidence and mark every workflow
artifact/task complete in the working tree; remove any execution-only artifact
that is not part of the durable record; run `git diff --check` and review the
complete diff/status; create one local commit containing source plus final
artifacts; then verify the branch and worktree are clean. Goal completion occurs
only after that clean post-commit check. No file is edited after the commit, and
no push, PR, deployment, infrastructure purchase/resource, or other external
write is authorized.

## Independent readiness review

The fixed ledger passed independent readiness review after closing T1's full
broker-proof scope, root-fixture ownership, and the non-circular final
ledger/commit sequence. Local Docker and Go prerequisites are available; no
material implementation-readiness blocker remains.
