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

- [x] T5: close the principal ownership audit findings without widening the
  concrete NATS capability
  - Source: `spec.md` M02 through M10 and invariants; `test-plan.md` TD-02
    through TD-15; the current base-to-candidate ownership audit of
    `a93162d9e7168e82a04575c2c81a3c294e4bf443`.
  - Owner/surface/resources: `internal/infra/natsjs/`, service and worker
    bootstrap, messaging config/defaults, selected-only messaging docs and
    workflow artifacts, and their existing unit/integration/process fixtures;
    one serialized local NATS/Docker proof owner. Preserve direct `nats.go`,
    one consumer/handler per worker, feature-owned event semantics, physical
    `MESSAGING=none` removal, and the no-outbox/inbox boundary.
  - Depends on: T4 — accepted candidate and exact audit base — needed to start.
  - Deliverable: terminal handler failures cannot be lost; inbound deliveries
    enforce the admitted header envelope; served readiness fails immediately
    during messaging loss and recovers only after a fresh probe; worker feature
    construction can consume loaded config and owns cleanup before broker
    admission; terminal diagnostics retain safe location/identity evidence;
    reconnect documentation matches pinned SDK behavior; test-only defaults,
    dead state, and duplicate metadata are removed; ACK/DLQ/capacity/admission
    and drain proofs observe authoritative completion rather than handler entry
    or scheduler luck.
  - Proof: focused package/component tests for each repaired contract; focused
    race/goleak for NATS and bootstrap lifecycle; serialized
    `REQUIRE_DOCKER=1 make test-integration`; messaging profile initialization
    and repository gates only after the focused owners pass. Expected
    observables include preserved `ErrTerminal`, no handler call for an
    oversized inbound header, immediate HTTP 503 during disconnect, confirmed
    zero ACK-pending/empty final-success DLQ, released publish capacity after
    ambiguous cancellation/deadline, rejected missing topology/config drift,
    bounded safe panic diagnostics, and clean physical removal.
  - Reopen if: a repair requires a generic broker abstraction, more than one
    consumer per worker, outbox/inbox semantics, or a changed feature error or
    idempotency policy; reopen the narrow Design or Specification owner.

- [x] T6: close the residual lifecycle and proof gaps found by the follow-up
  ownership audit
  - Source: the follow-up principal audit of current branch base
    `75e03c097444165725b73d5b29684c03d1f506dc`; T5 evidence remains historical
    and does not substitute for this repair.
  - Owner/surface/resources: `internal/infra/natsjs/`, worker bootstrap,
    selected Makefile/docs/profile blocks, and their existing unit and real-NATS
    fixtures. Preserve one worker per client and direct `nats.go` ownership.
  - Deliverable: connected topology failures cannot enter reconnect recovery;
    `NewWorker` rejects a second sequential or concurrent registration before
    broker mutation; failed `NakWithDelay` is terminal; `AckWait` derives from
    the actual settlement budget; bootstrap joins every owned runtime goroutine
    and never races feature cleanup with an uncooperative forced handler; one
    process grace deadline bounds drain, diagnostics, background joins, feature
    cleanup, and telemetry flush;
    abrupt close is no longer a public application API; the SDK's asynchronous
    drain is not wrapped in another goroutine; drain linearizes handler admission;
    the feature builder receives the admitted producer; async broker errors retain
    a bounded safe class; worker logs retain trace correlation and telemetry
    degradation cause; process fixtures use the working tree; real valid and
    rejected `.creds` paths are exercised; selected integration concurrency runs
    under `-race` and is physically removed with `MESSAGING=none`; runtime image
    proof executes the real fail-closed `/worker` artifact.
  - Proof: focused unit/race and real-NATS regressions,
    `make test-messaging-race`, `make check-full`, and
    `TEMPLATE_INIT_PROFILE=messaging make template-init-check` all pass
    serially; the aggregate covers lint, NilAway, coverage, full race, runtime
    image execution, migration rehearsal, and container security.
  - Reopen if: a second consumer must share one client, feature cleanup must run
    after an uncooperative handler exceeds the process deadline, or settlement
    grows another broker round trip; reopen the narrow lifecycle design owner.

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

### T5 audit-repair closure

- Independent acceptance review passed staged tree
  `d94f310ec67efaa503a8744406114591ba30dc08` after correcting terminal
  propagation, immediate served readiness, feature/telemetry construction and
  cleanup order, safe terminal diagnostics, reconnect documentation, and every
  deterministic proof gap named by the ownership audit.
- `go test -vet=off -race ./internal/infra/natsjs
  ./cmd/worker/internal/bootstrap ./cmd/service/internal/bootstrap -count=1`
  passed 188 tests; focused telemetry, handler-construction, process-readiness,
  reconnect, admission, publish-ambiguity, ACK/DLQ, trace, and drain cases also
  passed while the repair was narrowed.
- `REQUIRE_DOCKER=1 make test-integration` passed the serialized real-broker
  owners; the final `make check-full` completed with exit code 0 and reran that
  aggregate together with lint, NilAway, repository race, runtime image,
  migration, security, and container proof.
- `TEMPLATE_INIT_PROFILE=messaging make template-init-check` completed with
  exit code 0 on the final repaired tree, including physical `MESSAGING=none`
  removal, selected minimal/maximal composition, cross-selection rejection, and
  repeated byte stability.
- The runtime-generated operator/JWT fixture now proves both valid `.creds`
  readiness and broker rejection for a readable credential issued to an unknown
  account; no credential material is retained in the repository.
- No push, PR, deployment, purchase, cloud resource, or other external write
  was performed.

### T6 ownership-audit closure

- Fresh independent T6 acceptance review passed the current `75e03c0 + dirty`
  working tree after checking transport, worker lifecycle, process-fixture,
  profile, logging, and image-proof owners; no material finding remains in the
  accepted repair scope.
- The focused `natsjs`/worker-bootstrap suite passed 88 tests under `-race`.
- `make test-report` passed 1208 tests with two intentional skips and effective
  coverage 80.0%; `make check-full` completed with exit code 0 and included lint,
  NilAway, the full untagged race suite, generated/module/structure checks,
  real-broker/process tests, runtime image, migration, and container security.
- `REQUIRE_DOCKER=1 make test-integration` passed the complete serialized
  real-NATS/process owners (root 207.135s, package 15.838s, worker 1.862s) and
  its tagged race subset, including authentication, singleton admission,
  reconnect, saturation, process composition, and both drain paths.
- `TEMPLATE_INIT_PROFILE=messaging make template-init-check` passed none,
  selected-minimal, selected-maximal, cross-selection, repeatability, broad
  residue scanning, and removal of every `github.com/nats-io/` dependency.
- Runtime image proof executed `/worker` with a read-only filesystem and no
  network, migration rehearsal exited 0, and Trivy reported zero HIGH/CRITICAL
  findings in `service`, `worker`, and `migrate`.
- The repair candidate passed final local acceptance as a working-tree change
  based on `75e03c097444165725b73d5b29684c03d1f506dc`; commit, PR, and merge
  identities remain publication evidence and are not embedded in this record.

## Closeout sequence

After T4 acceptance, record the exact final evidence and mark every workflow
artifact/task complete in the working tree; remove any execution-only artifact
that is not part of the durable record; run `git diff --check` and review the
complete diff/status; create one local commit containing source plus final
artifacts; then verify the branch and worktree are clean. Goal completion occurs
only after that clean post-commit check. No file is edited after the
implementation commit. Push, PR, merge, deployment, infrastructure
purchase/resource, or another external write requires separate explicit
authorization.

## Independent readiness review

The fixed ledger passed independent readiness review after closing T1's full
broker-proof scope, root-fixture ownership, and the non-circular final
ledger/commit sequence. Local Docker and Go prerequisites are available; no
material implementation-readiness blocker remains.
