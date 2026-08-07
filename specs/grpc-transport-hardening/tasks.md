# gRPC callers receive the same retry, identity, and liveness guarantees the HTTP transport already gives them

status: done
Completion: every rule in [spec.md](spec.md) R1–R9 is observable on the production
path, all four configuration-to-adapter crossings carry every new bound, and
`make check` and `make lint-deep` pass on the integrated tree.

Accepted: T1–T6; evidence: `go test ./...` 1547 passed in 39 packages;
`go test -race` over the changed packages clean; `make lint` 0 issues;
`make fmt-check`, `make project-structure-check`, `make mod-tidy-check` clean;
`deadcode -test -tags=integration ./...` clean; `nilaway` clean over the changed
packages. Candidate: current bounded diff.

Unverified remainder, stated beside the completion claim: repository-wide
`make lint-deep` **fails**, in `internal/infra/postgresoutbox` only — two nilaway
findings in another writer's uncommitted work that this change does not touch.
`make check` was therefore run as its constituent gates rather than as one
command. The design's Go Ownership review gate was overridden after four panel
rounds; `design/overview.md` records that and its residual risk.

Global constraints:

- [design/overview.md](design/overview.md)'s file map is the accepted placement.
  A task may not choose a different file, package, owner, or exported surface; a
  placement that the real code shows is wrong reopens that design owner.
- Every new configuration bound reaches **all four** crossings from
  `config.GRPCServerConfig` into `grpcx.Config`: `startup_grpc.go`,
  `config_parity_test.go`'s `serverConfigFromRuntime`, the benchmark server's
  `settingsFromDefaults`, and `examples/grpc-reference-service/service_test.go`.
  Three in-repo comments still say three; whichever task touches one corrects it.
- Preserved unchanged, and a task that alters one has exceeded its boundary: the
  existing interceptor positions and both error boundaries' trust rules;
  correlation acceptance and minting; the admission budget and its health
  exemption; every existing transport bound; and the client's three metadata
  strip seams.
- Production import direction is unchanged. `internal/infra/grpc` imports only
  `internal/problem`, `internal/reqctx`, and external libraries;
  `internal/infra/grpcclient` imports only `internal/reqctx`. The two test-build
  edges the design declares are the only new ones.
- The design's Go Ownership review gate was deliberately overridden after four
  panel rounds; its header records the residual risk. Implementation is the first
  place a compiler and linter see this plan, so treat a build or lint failure
  that contradicts the design as evidence to reopen it, not as a local fix.

- [x] T1: The policy shape carries the RPC context down to the work below it, with no behavior change
  - Source: [design/overview.md](design/overview.md) S1
  - Owner/surface/resources: `internal/infra/grpc` — `chain.go` (`aroundRPC`,
    `asUnaryInterceptor`, `asStreamInterceptor`), `interceptors.go`
    (`accessLogAround`, `recoveryAround`, `admissionLimiter.around`), `status.go`
    (`handlerErrorBoundary`, `policyErrorBoundary`), and the `performance_test.go`
    call sites. `asStreamInterceptor` wraps the stream only when the context
    changed, reusing the existing `serverStreamWithContext`. No mutable resource.
  - Depends on: none
  - Proof: claim — every existing gRPC behavior is unchanged while the shape
    widens; `go test -race ./internal/infra/grpc/...`; expected observable — all
    current tests pass with no assertion edited.
  - Enables: T3. This task carries no obligation of its own.

- [x] T2: The grpcx test harness owns one streaming registration and a raw-connection stand-up path
  - Source: [design/overview.md](design/overview.md) file-map rows for
    `harness_test.go` and `correlation_service_test.go`
  - Owner/surface/resources: `internal/infra/grpc/harness_test.go` gains
    `registerStreamTestService` mirroring the existing `registerUnaryTestService`,
    and a stand-up path returning the listener address so a caller can dial and
    abandon a raw `net.Conn`; its header's "two ways" enumeration widens.
    `internal/infra/grpc/correlation_service_test.go` keeps its own handler,
    switches to the shared registration, and loses its sole-caller claim. The
    registered method takes its own name; the existing `testStreamFullMethod`
    constant stays a bare string for `grpc.StreamServerInfo` literals. Loopback
    TCP port from the ephemeral range.
  - Depends on: none
  - Proof: claim — the restructure preserves every current assertion;
    `go test -race ./internal/infra/grpc/...`; expected observable — correlation
    and limits tests pass with no assertion edited.
  - Enables: T3. This task carries no obligation of its own.

- [x] T3: An RPC cannot occupy a handler indefinitely, and a dead or idle peer stops holding a connection slot
  - Source: [spec.md](spec.md) R3, R4, R5 and success criteria 3, 4, 5, 9;
    [design/overview.md](design/overview.md) S2, S3, S5 and responsibility rows
    3, 5, 7, 9, 11
  - Owner/surface/resources: `internal/infra/grpc/{chain,interceptors,config,server,doc}.go`;
    `internal/config/{types,defaults,validate}.go`;
    `cmd/service/internal/bootstrap/startup_grpc.go` and `startup_grpc_test.go`;
    `examples/grpc-reference-service/service_test.go` and
    `cmd/benchmark-server/{main,main_test}.go`; new
    `internal/infra/grpc/{deadline,admission,keepalive}_test.go`; changed
    `internal/infra/grpc/{interceptors,config_parity,performance,harness}_test.go`;
    `internal/config/{grpc_config,snapshot_contract}_test.go`; `env/.env.example`;
    `docs/grpc.md`. Loopback TCP ports for the keepalive and admission proofs. The
    nine new fields are flat and unprefixed on both config structs; no new config
    type is created.
  - Depends on: T1 — output handoff — needed to start; T2 — output handoff —
    needed to prove.
  - Handoff: T1 produces `aroundRPC` in its `call func(context.Context) error`
    form, which is what lets `deadlineAround` be one policy for both RPC kinds; T2
    produces the shared streaming registration and the raw-connection stand-up
    path, which the stream and vanished-peer claims consume.
  - Proof:
    - claim — a context-respecting handler invoked without a caller deadline
      answers `DEADLINE_EXCEEDED` inside the unary budget and its admission slot
      is released; `go test -race ./internal/infra/grpc/ -run 'Deadline|Admission'`;
      expected observable — the status, and a subsequent RPC admitted rather than
      shed.
    - claim — the process-wide admission budget survives S3's two-call
      composition; same command; expected observable — filling the budget from
      one RPC kind sheds the other on a server built by `NewServer`.
    - claim — with rotation off a stream outlives the unary budget and neither the
      idle bound nor an answered ping closes its connection, and with a configured
      age the connection is closed and the client recovers;
      `go test -race ./internal/infra/grpc/ -run Keepalive`; expected observable —
      per spec criteria 4 and 5, under shortened bounds.
    - claim — every configuration relation R4 and R5 state is refused at startup;
      `go test ./internal/config/... ./internal/infra/grpc/`; expected observable —
      a negative keepalive duration, a stream budget at or above a configured age,
      and an age with a non-positive grace or a grace below the unary budget each
      fail configuration load and `grpcx.NewServer`.
    - claim — all four crossings carry every new bound;
      `go test ./cmd/service/internal/bootstrap/... ./examples/grpc-reference-service/...`;
      expected observable — the reflection guards report no zero or unmirrored
      field.
  - Reopen if: the real code shows a keepalive bound whose two owners cannot
    state the same conditional rule — reopen design S5.

- [x] T4: A classified domain error reaches a gRPC caller with its retry hint and a machine-readable identity
  - Source: [spec.md](spec.md) R1, R2 and success criteria 1, 2;
    [design/overview.md](design/overview.md) S4, S8
  - Owner/surface/resources: `internal/infra/grpc/{status,config,server,doc}.go`;
    new `internal/infra/grpc/error_details_test.go`; changed
    `internal/infra/grpc/{interceptors,docs,performance}_test.go`;
    `cmd/service/internal/bootstrap/startup_grpc.go`;
    `examples/grpc-reference-service/cmd/benchmark-server/main.go`;
    `.golangci.yml`; `go.mod`; `docs/grpc.md`. The `exhaustruct` entry matches the
    **import path**: `^github\.com/example/go-service-template-rest/internal/infra/grpc\.Options$`,
    anchored, inside the existing `profile:grpc` markers. No mutable resource.
  - Depends on: none
  - Proof:
    - claim — one mapper delay yields an exact `RetryInfo` over gRPC and a
      `Retry-After` no shorter than it over HTTP, for a sub-second, whole-second,
      and fractional delay; `go test ./internal/infra/grpc/ -run ErrorDetails`;
      expected observable — 200ms/1s/1500ms give `RetryInfo` 200ms/1s/1500ms and
      `Retry-After` 1/1/2.
    - claim — two domain errors sharing one gRPC code arrive with distinct
      `ErrorInfo` reasons, an unclassified error carries neither detail nor
      handler text, and every catalog code renders to a conforming reason; same
      command; expected observable — per spec criterion 2, over `problem.All()`.
    - claim — a configured value on `Options` cannot silently go unwired;
      `make lint`; expected observable — removing `ErrorDomain` from
      `newGRPCRuntime`'s literal fails `exhaustruct`.
    - claim — the promoted dependency is declared; `make mod-tidy-check`;
      expected observable — no diff.
  - Reopen if: a consumer is found that rejects unknown status details — reopen
    spec R1/R2's first assumption.

- [x] T5: An outbound client distributes across resolved backends and keeps an idle connection alive
  - Source: [spec.md](spec.md) R6, R7, R8 and success criteria 6, 7;
    [design/overview.md](design/overview.md) S6, S7
  - Owner/surface/resources: new
    `internal/infra/grpcclient/{load_balancing.go,load_balancing_test.go,keepalive_test.go,keepalive_parity_test.go}`;
    changed `internal/infra/grpcclient/{client,propagation,doc}.go`,
    `client_test.go`, `resolver_live_test.go`; `docs/grpc.md`,
    `docs/repo-architecture.md`, `docs/first-production-feature.md`,
    `docs/project-structure-and-module-organization.md`. Two loopback TCP
    listeners, and the process-global resolver registry — register the additional
    scheme from a **sequential top-level test**, because `resolver.Register`
    writes an unsynchronized map that every parallel `grpcclient.New` reads.
  - Depends on: none
  - Proof:
    - claim — a two-address target reaches both backends under the default policy
      and one under first-address selection;
      `go test -race ./internal/infra/grpcclient/ -run LoadBalancing`; expected
      observable — both servers record RPCs, then only one.
    - claim — the shipped client's default ping interval exceeds the shipped
      server's default minimum accepted interval, and an idle connection is
      pinged; `go test -race ./internal/infra/grpcclient/ -run Keepalive`;
      expected observable — the constants compare, and a peer whose enforcement
      policy rejects the ping sends `GOAWAY`. Costs at least 10s of wall clock:
      grpc-go clamps a client ping interval below 10s up to 10s.
    - claim — the client's trust boundary is unweakened by its own default service
      config; `go test -race ./internal/infra/grpcclient/...`; expected observable
      — every existing propagation, resolver, and transparent-retry test passes
      with no assertion edited.
  - Reopen if: a deployment needs a resolver-supplied service config — reopen
    spec R7's non-goal.

- [x] T6: Server construction refuses a nil service registration instead of skipping it
  - Source: [spec.md](spec.md) R9 and success criterion 8;
    [design/overview.md](design/overview.md) responsibility row 8
  - Owner/surface/resources: `internal/infra/grpc/server.go` (`registerServices`
    returns an error, `NewServer` propagates it), `internal/infra/grpc/server_test.go`.
    No mutable resource.
  - Depends on: none
  - Proof: claim — a nil registration fails construction and opens no listener;
    `go test ./internal/infra/grpc/ -run TestNewServer`; expected observable — a
    non-nil error and a nil `*Server`.

## Shared surfaces

No wave is planned: the ready tasks do not have pairwise-disjoint writable
surfaces, so ordering is by `Depends on` and by these overlaps.

- `internal/infra/grpc/server.go` — T1, T3, T4, T6
- `internal/infra/grpc/config.go` — T3, T4
- `internal/infra/grpc/doc.go` — T3, T4
- `internal/infra/grpc/{interceptors,performance}_test.go` — T1, T3, T4
- `docs/grpc.md` — T3, T4, T5
- `cmd/service/internal/bootstrap/startup_grpc.go` — T3, T4

## Obligation reconciliation

| Obligation | Disposition |
| --- | --- |
| R1, R2 | T4 |
| R3, R4 | T3 |
| R5 | T3 |
| R6, R7, R8 | T5 |
| R9 | T6 |
| S1's widened policy shape | T1, enabling change serving T3 |
| The harness restructure S5's and R4's stream proofs need | T2, enabling change serving T3 |
| Spec non-goals — error vocabulary, field-level details, proto validation, pre-decode shedding, client health-checking LB, reflection | No implementation. Each is a recorded non-goal with a reopen condition in [spec.md](spec.md); none is completion evidence. |
| The four crossings' three stale comments | Distributed to whichever task touches each file, per Global constraints |
