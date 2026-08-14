# PostgreSQL-backed HTTP idempotency implementation ledger

status: done

Completion: the reusable `HTTP_IDEMPOTENCY=postgres` pack is implemented in the
health-only template, every repository-local TD-IDEM-001 through TD-IDEM-015A
oracle passes on its accepted carrier, `none|postgres` generation is deterministic
and dependency-clean, and the existing production-image/migration aggregate proves
both the health-only and active-registration source identities. No completion claim
includes a concrete endpoint, live writer/failover topology, restore, activation, or
external effect.

Blocked stop: leave the affected task unchecked and record the unavailable claim,
current narrower evidence, and reopen owner. Reopen Technical Design when a
required deterministic carrier or fixed owner/file boundary is infeasible. Reopen
Specification only for the recorded R2 raw-framing-OWS, R3 authorization-input, R6
zero-cross-replica-wait, or R9 exact-commit-epoch behavioral falsifier. Missing
adopter inputs do not block this reusable-template ledger; they remain the
TD-IDEM-016 scope exit below.

The frozen T5 coverage candidate and T5 remain blocked until their recorded
conditions are met. The only executable frontier is the noncanonical prerequisite
recovery below; it creates no Accepted:/Blocked: transition for T1-T5.

Global constraints:

- HTTP transport validates, carries generated operation identity, maps classified
  outcomes, and freshly renders retained semantic results; it never begins,
  commits, retries, or otherwise owns the feature transaction.
- The endpoint/application adapter calls the existing `postgres.Pool.InTx` once.
  Acquire, feature mutation, optional outbox append, result encoding/bound check,
  and completion share that exact caller-owned `pgx.Tx`; completion is the final
  application statement and baseline execution has no hidden retry.
- Initial arbitration, any absence that may authorize execution, replay,
  reconciliation, expiry, cleanup eligibility, and commit epoch use only the
  current authoritative writer. The process-local publication group carries only
  a completion signal and never becomes correctness authority.
- The reusable guarantee ends at the exact PostgreSQL commit. Direct outbound
  effects remain forbidden from the covered sequence; a concrete endpoint owns
  outbox/downstream idempotency, reconciliation, or compensation and its final
  result semantics.
- The 20-row `test-plan.md` matrix is the proof authority. Implementation may add
  its named tests and fixtures but does not add scenarios, substitute mock-only
  proof for PostgreSQL arbitration/visibility, replace owned event gates with
  sleeps, or add a production fault seam.
- No numeric endpoint or deployment value becomes a template default. No new
  dependency, deployable, cache, broker, scheduler, cleanup binary, elected leader,
  store interface, or fake OIDC trust path is introduced.
- Canonical OpenAPI, migration, and SQL sources precede generated output. The
  health-only image and disposable active-registration image are distinct source
  identities, each built once and reused for only its accepted claims.

## Obligation reconciliation

| Accepted obligation | Ledger disposition |
| --- | --- |
| R1 complete fail-closed opt-in and no generic guarantee | T1 owns declaration/registration rejection; T3 owns exact empty/nonempty runtime activation; T4 owns removable profile closure. |
| R2 exact opaque wire key and bad-key action | T1. |
| R3 authenticated scope, current authorization, admission, and response precedence | T1 owns the HTTP order and authority seam; T2 consumes the resulting scoped identity only on the writer. |
| R4 recorded-version semantic fingerprint | T1 owns canonical/versioned value contracts; T2 owns persisted-version comparison and writer replay. |
| R5 exact caller-owned atomic boundary | T2 owns reusable acquire/mutation/optional-outbox/result/completion composition. Concrete endpoint external-effect recovery is the TD-IDEM-016 scope exit. |
| R6 one owner, bounded classification, rollback/death, and whole-attempt retry policy | T2. |
| R7 bounded original semantic replay with fresh transport metadata | T1 owns the envelope/renderer contract; T2 owns committed result authority and fresh-process replay. |
| R8 bound-before-commit and every authoritative ambiguity branch | T2. |
| R9 exact commit horizons, retention, erasure, and cleanup | T2 owns exact epoch capture and fail-closed loss; T3 owns maintenance, capacity closure, and lifecycle. Live restore/activation is the TD-IDEM-016 scope exit. |
| R10 stable Problem/client actions and disclosure boundary | T1 owns the catalog and HTTP mapping; T2 and T3 supply only closed Store/lifecycle dispositions. |
| R11 writer-only correctness | T2 owns every repository-local writer/non-writer falsifier. Live promotion/failover certification is the TD-IDEM-016 scope exit. |
| R12 privacy, telemetry, and cached readiness | T1 owns HTTP/result redaction; T3 owns closed telemetry, aggregate observations, and readiness. Adopter privacy/erasure/restore authority is the TD-IDEM-016 scope exit. |
| R13 profile, migration, mixed-version safety, and rollback | T3 owns zero-registration runtime behavior; T4 owns generated-profile purity; T5 owns additive migration and production-image rejection/compatibility proof. Activation remains the TD-IDEM-016 scope exit. |

## Matrix reconciliation

| Test Design row | Ledger disposition |
| --- | --- |
| TD-IDEM-001 | T1. |
| TD-IDEM-002 | T1. |
| TD-IDEM-003 | T1 owns canonical vectors, formatting/default equivalence, one mismatch per declared semantic-manifest field, and codec rules; T2 owns recorded-fingerprint-version persistence/replay. |
| TD-IDEM-004 | T1 owns HTTP authorization/admission ordering; T2 consumes its authority-A/B setup in pool-headroom proof. |
| TD-IDEM-005, TD-IDEM-005A | T2. |
| TD-IDEM-006, TD-IDEM-007 | T2. |
| TD-IDEM-008 | T1 owns fresh replay rendering; T2 owns bounded commit/replay and post-commit process death. |
| TD-IDEM-009 | T2. |
| TD-IDEM-010 | T2 owns exact epoch capture/materialization; T3 owns epoch-loss task failure, cached readiness, and drain. |
| TD-IDEM-011, TD-IDEM-011A | T3. |
| TD-IDEM-012 | T2 owns the local writer/non-writer oracle; the target failover procedure is the TD-IDEM-016 scope exit. |
| TD-IDEM-013 | T1. |
| TD-IDEM-013A, TD-IDEM-014 | T3. |
| TD-IDEM-015 | T4. |
| TD-IDEM-015A | T5. |
| TD-IDEM-016 | Scope exit: `spec.md` limits this bundle to a reusable health-only capability and `test-plan.md` makes endpoint quantities, real issuer, writer/failover topology, privacy/restore authority, traffic gate, downstream oracle, and external-effect recovery adopter-owned. A concrete adopting endpoint/deployment reopens those named endpoint, database, privacy, and deployment owners; only its recorded R2, R3, R6, or R9 behavioral falsifier reopens Specification. It creates no reusable-template implementation task or completion dependency. |

## Current-surface reconciliation

- `internal/infra/postgres/transaction.go` remains the sole caller transaction
  owner; T2 composes through `Pool.InTx` and retains
  `TestInTxCommitOutcomes` as the independent lost-result boundary.
- `internal/infra/postgresoutbox/store_append.go` remains the optional
  transaction-bound durable-intent owner, and
  `internal/infra/postgresoutbox/store_receipt.go` remains unchanged evidence for
  writer-only reconciliation. T2 may exercise Append in the synthetic transaction;
  no idempotency package owns publication or a downstream effect.
- `internal/infra/http/router.go` and `api/openapi/service.yaml` are the current
  envelope/contract producers and T1 couples them to their generated binding and
  focused tests. `request_errors.go` remains the unchanged owner of earlier
  generated validation/authentication rejection.
- `internal/background`, cached `internal/health`, and the shutdown sequence in
  `cmd/service/internal/bootstrap/run.go` remain lifecycle authorities; T3 only
  registers the conditional probe/task and preserves their existing drain/join.
- `/migrate`, canonical `migrations/`, PostgreSQL query sources, and SQLC output
  retain their current authority order. T2 adds one additive no-backfill relation;
  T5 proves the service never migrates at startup and old/profile-off code remains
  compatible while opted traffic is gated.
- `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, existing profile
  markers/inventories, `Makefile`, CI change-scope routing, and the runtime-image
  owner remain the only template/build surfaces. T4 and T5 consume the complete
  fixed T1-T3 path set; no second generator or migration/image harness is added.

## Noncanonical prerequisite recovery

This recovery unit is a prerequisite to the frozen Coverage Correction Lead and
T5, not an implementation delta for a canonical T1-T5 obligation. It is the
only planned recovery representation, and its later receipt must not add or
replace a canonical Accepted:/Blocked: transition for T1-T5.

- [x] PR-GOSEC-TOOLCHAIN-01: Repository-controlled toolchain authorities require Go 1.26.6 before the Gosec aggregate can be revalidated
  - Source: `design/toolchain-security-reopen.md` SHA-256 `5c985e243a4504b538250a760b2e5044a5066e5e515664d351cef7d25539b5bd`, Decision, Implementation ownership and proof, and Exclusions and reopen condition.
  - Owner/surface/resources: one fresh serial Local Acceptance-Unit Lead is the only writer/integrator for `go.mod`, `tools/go.mod`, and `build/docker/Dockerfile`: set both module directives to `go 1.26.6` and the first Docker `FROM` exactly to `golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36`. The selected host Go toolchain, Docker builder image, and disposable local `service:pr-gosec-toolchain` image tag/Docker daemon are exclusive proof resources; remove that tag after its proof. Do not change `Makefile`, GitHub Actions, scanner configuration, advisory/threshold suppression, application source, `scripts/ci/s3-source-receipt.sh`, or any `specs/s3-compatible-object-storage/` artifact; preserve all unrelated dirt.
  - Depends on: none.
  - Handoff: accepted coherent root/tools/Docker Go 1.26.6 pins and their host/security/container receipts. Before resuming the Gosec prerequisite, archive/freeze its terminal retry and dispatch one fresh replacement Lead for `PR-GOSEC-01` to rerun its complete recorded proof, one serialized `make check-full`, and fresh independent implementation review; this unit neither accepts nor replaces `PR-GOSEC-01` or T1-T5.
  - Proof: `test "$(GOTOOLCHAIN=go1.26.6 go version | awk '{print $3}')" = go1.26.6`; `test "$(awk '/^go / {print $2; exit}' go.mod)" = 1.26.6` and the same check for `tools/go.mod`; `test "$(awk '$1 == "FROM" { for (i = 1; i <= NF; i++) if ($i ~ /^golang:/) { print $i; exit } }' build/docker/Dockerfile)" = golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36`; `GOTOOLCHAIN=go1.26.6 make mod-tidy-check`; `GOTOOLCHAIN=go1.26.6 go test -vet=off -run '^$' ./...`; `GOTOOLCHAIN=go1.26.6 make govulncheck`; and `make runtime-image-build RUNTIME_IMAGE=service:pr-gosec-toolchain`, then remove the local tag. Observable: the selected host is exactly `go1.26.6`, both module authorities agree and stay tidy, every package compiles, the real Docker build uses the exact pinned builder identity, and govulncheck no longer reports GO-2026-6218, GO-2026-6091, GO-2026-6090, GO-2026-6089, GO-2026-6088, GO-2026-5972, or GO-2026-5026; no scanner/advisory suppression is added.
  - Reopen if: Go 1.26.6 changes an accepted runtime/image contract, the pinned index cannot resolve, or a consumer does not derive its version from these three authorities — Technical Design; if the S3 candidate needs its source receipt against the upgraded image — S3 Technical Design.
  - Accepted: PR-GOSEC-TOOLCHAIN-01; evidence: exact Go 1.26.6 host/directive/Docker-pin assertions, `GOTOOLCHAIN=go1.26.6 make mod-tidy-check`, `GOTOOLCHAIN=go1.26.6 go test -vet=off -run '^$' ./...`, `GOTOOLCHAIN=go1.26.6 make govulncheck` (0 reachable vulnerabilities), and `make runtime-image-build RUNTIME_IMAGE=service:pr-gosec-toolchain` with tag cleanup passed; fresh independent Implementation Review PASS; candidate: current bounded diff.

- [x] PR-GOSEC-01: Gosec prerequisite preserves bounded conversions and telemetry observability
  - Source: `design/gosec-prerequisite-recovery.md` SHA-256 `2406de9eea0cbcd5c666414e33ac8844e94e1bbb07a23ffa29f721d3948c9933`, Fixed boundary, Selected recovery, Proof obligations, and Correction routing and reopen conditions; `design/t5-coverage-recovery.md` SHA-256 `ba5d357f791e1b43fca213b2658e16a5b0afe3a4e199129f825d38b4a4365ec0`, Execution shape and reopen conditions.
  - Owner/surface/resources: one fresh Acceptance-Unit Lead executes serially in Local and is the only writer/integrator for `internal/jobs/transition.go`; `internal/infra/postgresjobs/{store_claim.go,store_finalize.go,store_rescue.go,telemetry.go,telemetry_test.go}`; `internal/httpidempotency/{identity.go,result.go}`; `internal/infra/http/idempotency_registration.go`; and `internal/config/configtest/configtest.go`. No schema, SQL, migration, transaction, exported API, scanner policy/baseline, blanket/path suppression, public/wire contract, coverage filter, or Makefile change is writable. The shared Local checkout and the final host aggregate are exclusive resources; retain the frozen T5 candidate's `store_rescue.go` overlap and all unrelated dirt.
  - Depends on: PR-GOSEC-TOOLCHAIN-01 — accepted toolchain handoff — needed to start and prove.
  - Handoff: one integrated bounded conversion/telemetry/security-scanner recovery with all focused proofs and the one final aggregate; the frozen Coverage Correction Lead and T5 may resume only after this unit's fresh review accepts it. It does not accept, replace, or release T1-T5.
  - Mechanism: add finding-local `#nosec G115` rationale only beside existing guards: Jobs jitter `[1,1000]` with modulo at most `2000`; Jobs claim/rescue limits `[1,math.MaxInt32]`, rejected negative scope generation before unsigned conversion, and `validateAttemptIdentity` rejection above `math.MaxInt64`; HTTP identity/result `math.MaxUint32` and header/value-count `math.MaxUint16`; and the positive `Contract.KeyMaxBytes` required-key comparison. Add one private Jobs telemetry count-to-`int64` boundary that saturates a `StateObservation.Count` over `math.MaxInt64`, with the checked local cast annotated and a cached-snapshot `math.MaxUint64` test observing a non-negative saturated point. Add only the adjacent `#nosec G101` explanation for deterministic `testing.TB.Setenv` outbound-auth fixture placeholders. Keep validators, typed errors, SQLC query shapes, Session ownership, generation fencing, telemetry attributes/callback I/O/readiness, byte framing, canonical vectors, header allowlist, decode validation, and fixture values unchanged; do not add a generic converter, alter `StateObservation`, types, queries, or scanner configuration.
  - Proof: SEC-G115-01: `go test -vet=off ./internal/jobs -run '^TestJobsTransition$' -count=1` and `go test -vet=off -race ./internal/infra/postgresjobs -run '^(TestStoreMappingRejectsMalformedDatabaseValues|TestStoreClaimMappingPreservesOnlyValidRows|TestStoreDurationAndTimeMappingRejectInvalidValues|TestStoreRescueMappingRejectsMalformedRows)$' -count=1`; oracle: existing range validators and typed mapping errors, while TD-JOBS-010/TD-JOBS-014 retain their PostgreSQL fencing/transaction carriers. SEC-G115-02: `go test -vet=off -race ./internal/infra/postgresjobs -run '^TestTelemetryExportsCachedObservationWithoutStoreCalls$' -count=1`; oracle: manual OTel reader sees exact `math.MaxInt64` and callback makes no Store I/O, while TD-JOBS-015 retains the SQL count/oldest carrier. SEC-G115-03: `go test -vet=off ./internal/httpidempotency -run '^(TestHTTPIdempotencyCanonicalVectors|TestEncodeResultRejectsResponsesThatCannotBeSafelyReplayed|TestDecodeResultRejects(CorruptOrOverlongRetainedData|InvalidRetainedFields)|TestResultReaderRejectsTruncatedRetainedFields)$' -count=1` and `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyRegistrationContract$' -count=1`; oracle: existing vectors, rejection matrix, and required-key contract, while TD-IDEM-003/005-012/015A retain PostgreSQL carriers. SEC-G101-01: `make gosec`; oracle: only the precise `TB.Setenv` fixture annotation is exempt and live credentials remain scanned. Before SEC-AGG-01, assert the immutable Makefile/coverage identity: `test "$(shasum -a 256 Makefile | awk '{print $1}')" = '15db97ad2b79eaa3a13821fdc023e4f666ef721a39ed7184806c0211bfad0413'`; then run one serialized `make check-full`; oracle: unchanged Makefile policy, race/static gates, `COVERAGE_MIN=80.0`, exclusions, configured integration/container surfaces, and effective coverage `>=80.00%`. Do not reuse the saved 80.10% result after source changes. Retain, not substitute, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotency(RecordedFingerprintVersion|FirstPublicationAndPoolHeadroom|ClassificationBudget|OwnerRecovery|CallerTransactionAtomicity|ReplayResultBoundAndPostCommitDeath|CommitReconciliation|RetentionAndCleanupRaces|WriterAuthority)$' -count=1`, `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresJobs(Acceptance|Claim|Session|OperationBudget|Finalize|Recovery)$' -count=1`, and `make test-messaging-race` as their named non-substitutable carrier obligations.
  - Fresh review: after the integrated candidate and all proof above, obtain one fresh independent Implementation Review before handing the receipt to the frozen Coverage Correction Lead or T5; no partial slice or focused proof authorizes resumption.
  - Cleanup: keep all recovery local to the listed findings; remove no retained guard, test oracle, carrier, scanner policy/baseline, or frozen-candidate content, and add no generic utility, fake PostgreSQL/NATS claim, live-looking credential, or test-support package.
  - Reopen if: a cited guard is removed/weakened, a `configtest` caller can construct an outbound client or publish the fixture value, a live credential is introduced, or a scanner exception lacks its adjacent invariant — Go Security; a shared converter, exported API, changed query/schema/transaction owner, or another package becomes semantic owner — Go Ownership; a required oracle would fake PostgreSQL/NATS or move an existing real integration claim into a unit test — Test Design; telemetry saturation changes its closed observable contract or caller-owned/writer-only boundary — Technical Design; final unchanged effective coverage is below `80.00%` — reopen `design/t5-coverage-recovery.md` with the exact filtered uncovered-statement report.
  - Accepted: PR-GOSEC-01; evidence: all fresh focused/non-substitutable carriers and `make gosec` passed at `/private/tmp/pr-gosec-01-r4.9JCbcn/focused.log` (SHA-256 `304f77fed199b85d867bf3f124b7b739d077486bb5c73367d85075d8859be474`); the immutable Makefile SHA matched; the sole serialized `make check-full` passed at `/private/tmp/pr-gosec-01-r4.9JCbcn/check-full.log` (SHA-256 `098130f1f1ab70bf5b6ad3dfa1efb76c030578cbe660cd1a662b1618ad1c9f45`), including effective coverage `80.00%`, Gosec, PostgreSQL/NATS integration, both runtime-image identities, repaired active-image non-writer cleanup, and container security; fresh independent Implementation Review PASS; candidate: HEAD `3e58587695652f48bda2ba2db3a1e617a29d5c8b` with exact unfiltered diff SHA-256 `49261d6efa4eccd796a6c7244352bc620efbfeec0e08f682b64c6b457238da7f`.

- [x] PR-GOSEC-TD-IDEM-015A-01: The active-image non-writer rehearsal restores the disposable app role without weakening its rejection oracle
  - Source: `design/rollout.md` TD-IDEM-015A active-image cleanup reopen and Runtime proof identities; `test-plan.md` TD-IDEM-015A migration/runtime-image carrier; `Makefile` `migration-validate` active non-writer assertion and commit-timestamp case.
  - Owner/surface/resources: one fresh serial Local Acceptance-Unit Lead is the sole writer/integrator for `Makefile`. Replace the current post-`assert_active_failure` reset `docker run` with one disposable `docker run --rm ... --entrypoint psql ...` control-session invocation that passes `-c 'SET default_transaction_read_only = off'` followed by `-c 'ALTER ROLE app RESET default_transaction_read_only'`; these are separate completed commands in the one persistent client session. The disposable Compose project, PostgreSQL volume/network/containers, and distinct active fixture image tag are exclusive proof resources and existing trap cleanup remains the fallback. Do not change image/runtime/fixture/CI/application/test/specification surfaces; do not add a second `docker run`, combine the commands with a semicolon, weaken the read-only falsifier, or introduce a privileged runtime path.
  - Depends on: none.
  - Handoff: an accepted cleanup repair returns the active fixture to a writable role before the existing `track_commit_timestamp=off` rejection, while retaining distinct health-only and active tags, migration-only schema mutation, writer-only correctness, and fail-closed active startup. It invalidates the blocked PR-GOSEC-01 aggregate; archive that attempt and dispatch one fresh replacement only after this receipt. It neither accepts nor replaces T1-T5.
  - Proof: `make runtime-image-build RUNTIME_IMAGE=service:pr-gosec-td-idem-015a-cleanup` then `make migration-validate RUNTIME_IMAGE=service:pr-gosec-td-idem-015a-cleanup`; oracle: health-only up/down/up and migration-only mutation remain intact, the active fixture rejects missing schema, non-writer read-only authority, and disabled commit timestamps before OIDC/serving, the non-writer assertion is retained, cleanup succeeds before the commit-timestamp case, and the disposal trap removes every temporary resource. Run `git diff --check`; inspect the changed Makefile recipe for tabbed make syntax and exactly two separate control-session `-c` commands. No scoped formatter or repository Makefile linter exists; the recipe and real carrier are the formatting/lint boundary.
  - Fresh review: after the fixed candidate and proof, obtain one fresh independent Implementation Review. The Lead alone records one canonical `Accepted:` or `Blocked:` receipt for this noncanonical recovery.
  - Reopen if: the same control session cannot make the role writable before reset, the active non-writer assertion no longer fails before OIDC/serving, cleanup changes either source identity or migration authority, or the Compose disposal fallback cannot contain the role mutation — HTTP-idempotency Technical Design.
  - Accepted: PR-GOSEC-TD-IDEM-015A-01; evidence: `make runtime-image-build RUNTIME_IMAGE=service:pr-gosec-td-idem-015a-cleanup`, `make migration-validate RUNTIME_IMAGE=service:pr-gosec-td-idem-015a-cleanup`, and `git diff --check -- Makefile` passed; fresh independent Implementation Review PASS; candidate: current bounded diff.

- [x] CR-COV-05: Profile-owned runtime-options test coverage remains compilable when generated services omit PostgreSQL and NATS configuration
  - Source: `design/t5-coverage-recovery.md` CR-COV-05; `scripts/init-module.sh` profile-marker removal authority; T5's canonical blocker.
  - Owner/surface: `cmd/internal/runtimeopts/runtimeopts_test.go` only. PostgreSQL-only setup and mapping assertions are removed with the database profile; the messaging test remains profile-owned, retains blank addresses, and compares the complete inferred `natsjs.Config` structurally after explicitly setting every current field.
  - Proof: `go test -vet=off ./cmd/internal/runtimeopts -run '^(TestAdapterOptionsPreserveConfiguredValues|TestMessagingMappingRetainsBlankAddressesForAdapterValidation)$' -count=1`; `TEMPLATE_INIT_PROFILE=minimal make template-init-check`; scoped `goimports`, `gofumpt`, and `golangci-lint ./cmd/internal/runtimeopts`; `git diff --check`.
  - Reopen if: a profile-removed configuration type is again referenced by generated minimal tests, a mapped NATS field or blank URL is no longer structural-equality checked, or template marker removal changes the retained shared test — Go Ownership / Test Design.
  - Accepted: CR-COV-05; evidence: all listed proof passed, including generated `github.com/acme/feature-proof/cmd/internal/runtimeopts`; fresh independent Implementation Review PASS; candidate: current bounded diff.

## Acceptance units

- A-PR-GOSEC-TOOLCHAIN-01: PR-GOSEC-TOOLCHAIN-01 — one serial Local singleton because its three coequal authority pins and the host/Docker security proof must describe one coherent toolchain, while the shared checkout remains dirty.
- A-PR-GOSEC-01: PR-GOSEC-01 — one serial Local singleton because the checkout has broad unrelated dirt, the frozen T5 candidate overlaps `store_rescue.go`, and no partial slice can independently clear the repository-wide Gosec prerequisite.
- A-PR-GOSEC-TD-IDEM-015A-01: PR-GOSEC-TD-IDEM-015A-01 — one serial Local singleton because the one `migration-validate` control session and disposable active-image PostgreSQL carrier form one cleanup/proof boundary.

## Readiness review

Fresh independent Planning Task Review / Readiness **PASS** on candidate SHA-256
`c0482ae347b6b1351c654717ce17c3964d2f65f94504cfdb355cdb075c4bf829` for
`PR-GOSEC-TD-IDEM-015A-01`: the single Makefile-owned recovery reaches acceptance
through the current disposable active-image PostgreSQL carrier. It fixes one
`docker run --rm --entrypoint psql` control session with completed ordered
`SET default_transaction_read_only = off` and role-reset `-c` commands, preserves
the non-writer rejection and cleanup fallback, and hands off only to a fresh
PR-GOSEC-01 replacement. No surviving readiness finding; this receipt authorizes
only the singleton and records no implementation acceptance.

Focused independent Planning Task Review / Readiness **PASS** for
`PR-GOSEC-TOOLCHAIN-01`: the fixed unit reaches acceptance from the reviewed
Go 1.26.6 design with its exact three-file authority boundary, explicit
`GOTOOLCHAIN=go1.26.6` host selection, root/tools consistency and compile
checks, advisory-free `govulncheck`, real pinned-builder image build, exclusive
tag cleanup, and the downstream PR-GOSEC replacement handoff. No surviving
material finding. This review authorizes only the singleton toolchain unit;
PR-GOSEC-01 and T5 remain downstream.

Fresh Planning Task Review / Readiness **PASS** on candidate SHA-256
`e134bad28d7a3f41f51fd6efd9f8dfae5b258dd69378abf3b0e0c93d9029c56c` by an
independent Terra/medium reviewer: PR-GOSEC-01 reaches acceptance from the fixed
ledger with its listed writable surface, exclusive Local aggregate, focused
oracles, immutable Makefile identity, fresh implementation review, and handoff.
The noncanonical representation preserves the sole canonical T1-T5 transitions;
T5 remains unchecked and still requires its own aggregate, template-init proof,
and fresh review. This verdict records no implementation acceptance.

Independent Task Review / Readiness returned **PASS** on candidate SHA-256
`85a1dc163a81e0093704a27e4881811337d180f74d9c436681cb115cc46a0986` after one
Planning-owned repair restored TD-IDEM-003 formatting/default equivalence and one
mismatch per semantic-manifest field to T1. The focused fresh review found no
surviving finding. The T1 dry run reached acceptance from current owners and named
proof without a database, external input, chat history, or hidden choice; later
tasks are dependency-ordered and cannot invalidate that result. This receipt proves
ledger readiness only, not implementation, post-implementation tests, production
topology, endpoint adoption, or activation.

Planning reopen review history: whole-ledger Task Review / Readiness **FAIL** on
candidate `770cfe417d8b35055bc13fa6c66af95bbf0e290f4bb385617736e0a2fd233226`
because T3 owned the tagged bootstrap-internal integration carrier without an
executable acceptance check, while T5 retained an unspecified later write to the
same file. Planning moved the exact positive local composition oracle and complete
carrier ownership into T3; T5 now consumes and may rerun that accepted proof but
does not change the carrier. Focused independent re-review **PASS** on candidate
`11f60891af8f09dcc1586c80c1c28a21dd24ed6e4a38dfee610c793cb88e845b`:
T3 reaches an independently executable acceptance boundary from the fixed ledger,
and later work cannot invalidate it. Target activation remains outside this ledger
and **NOT READY**.

Planning reopened T3 after its canonical blocker. Review-cleared Technical Design,
Go Ownership, and Test Design now fix its split-finality, prompt-drain,
ceiling-boundary, two-order, and restart carriers. The current T3 entry below is
the sole executable replacement. Fresh independent Task Review / Readiness
**PASS** on candidate `afb6e4336383f0bcc1de350772a8d9a64d7535d5317e14633073718775f89c56`
found no surviving finding: the T3 dry run reaches acceptance from the fixed ledger
and cited inputs, while T4/T5 cannot invalidate it. This receipt is the only
post-review edit. TD-IDEM-016 remains a scope exit and target activation remains
**NOT READY**.

- [x] T1: The complete reusable declaration and HTTP envelope fail closed before Store work and render only bounded, versioned, redacted outcomes
  - Source: `spec.md` R1-R4, R7, R10, and R12; `design/overview.md` Exact identity, fingerprint, and result representations, HTTP envelope and opt-in closure, Go responsibility map, and inverse Go file map; `test-plan.md` TD-IDEM-001-TD-IDEM-004, the HTTP delta of TD-IDEM-008, and TD-IDEM-013.
  - Owner/surface/resources: keep `internal/httpidempotency/{doc.go,identity.go,result.go,outcome.go}`, change `internal/httpidempotency/contract.go` and its tests, and add `internal/httpidempotency/reservation.go`; add `internal/infra/http/{middleware_idempotency.go,idempotency.go,idempotency_registration.go,idempotency_response.go}` and their fixed tests; change `internal/infra/http/{router.go,doc.go,router_contract_test.go,openapi_contract_test.go}`; keep `internal/infra/http/request_errors.go` and its tests unchanged; change `internal/problem/{problem.go,problem_test.go}`, canonical `api/openapi/service.yaml`, and generated `internal/openapi/openapi.gen.go`; standard library and existing OpenAPI/HTTP owners only; no mutable resource.
  - Depends on: none.
  - Handoff: accepted versioned `Contract`, `DuplicateRiskPolicy`, `Scope`, `Fingerprint`, `Attempt`, `Reservation`, `ReservationRecovery`, `Result`, and `Decision` values; exact identity/result codecs; one-to-one operation registration; authenticated authorization/admission ordering; and closed Problem/render mapping consumed by T2 and T3. T1 owns the shared carrier declarations; T2 owns their Store round-trip, stale-generation, and recovery behavior proof under TD-IDEM-006, TD-IDEM-007, and TD-IDEM-009.
  - Proof: TD-IDEM-001 complete registration alone serves and every missing/duplicate/version/external-effect mutation fails before Store construction: `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyRegistrationContract$' -count=1`. TD-IDEM-002 proves the exhaustive `tchar`/field-line/byte-bound oracle and zero downstream calls: `go test -vet=off ./internal/httpidempotency -run '^TestHTTPIdempotencyKeyParser$' -count=1` and `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyKeyContract$' -count=1`. TD-IDEM-003 pins both canonical vectors, rejects every excluded transport/header field, proves formatting/default equivalence through a synthetic typed canonicalizer, and produces one mismatch for each declared semantic-manifest field: `go test -vet=off ./internal/httpidempotency -run '^TestHTTPIdempotencyCanonicalVectors$' -count=1`. TD-IDEM-004 proves 431, 401, 403, key 400, authority admission 429/503, then retained-state ordering with scope isolation: `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyAuthorizationAndAdmissionOrder$' -count=1`. The HTTP half of TD-IDEM-008 proves original semantic rendering with fresh transport metadata: `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyReplayRendering$' -count=1`. TD-IDEM-013 proves every Problem status/code/type/title, required or forbidden `Retry-After`, fresh correlation, and sentinel absence: `go test -vet=off ./internal/problem -run '^TestHTTPIdempotencyProblemCatalog$' -count=1` and `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyProblemAndRedaction$' -count=1`. Canonical/generated OpenAPI remains identical under `make openapi-check`.
  - Reopen if: raw framing OWS must remain distinguishable after `net/http` normalization — Specification R2; current client-visible authorization needs body or external input before key validation — Specification R3; otherwise a required declaration, excluded field, or closed disposition cannot be observed/mapped through the fixed envelope — Technical Design.
  - Accepted: T1; evidence: TD-IDEM-001-TD-IDEM-004, HTTP TD-IDEM-008, TD-IDEM-013, and `make openapi-check` passed on the bounded candidate; fresh independent acceptance review PASS; handoff: `DuplicateRiskPolicy`, `Reservation`, and `ReservationRecovery` remain shared declarations only, while T2 stays blocked for Store round-trip and recovery proof.
  - Accepted: T1 lint-repair; evidence: scoped `golangci-lint` for `./internal/httpidempotency ./internal/infra/http ./internal/problem` passed with 0 issues; TD-IDEM-001-TD-IDEM-004, HTTP TD-IDEM-008, TD-IDEM-013, and `make openapi-check` passed; fresh independent acceptance review PASS; candidate: current bounded diff.

- [x] T2: One writer-authoritative row arbitrates one caller-owned transaction and deterministically resolves reservation, rollback, death, replay, ambiguity, and exact commit epoch
  - Source: `spec.md` R3-R9 and R11; `design/overview.md` Selected authority and state model, PostgreSQL request protocol and exact transaction composition, literal authoritative-commit horizons, failure schedules, and PostgreSQL responsibility/file map; `design/rollout.md` additive migration and migration-before-service boundary; `test-plan.md` Store delta of TD-IDEM-003, TD-IDEM-005-TD-IDEM-010, and local TD-IDEM-012.
  - Owner/surface/resources: add canonical `migrations/000003_postgres_http_idempotency.sql` and `internal/infra/postgres/queries/postgres_http_idempotency.sql`, then regenerate only `internal/infra/postgres/sqlcgen/*`; add `internal/infra/postgresidempotency/{doc.go,errors.go,store.go,store_reserve.go,store_acquire.go,store_complete.go,store_release.go,store_reconcile.go,store_epoch.go}` and their fixed tests plus `docs_test.go`; add/extend `test/postgres_http_idempotency_fixtures_integration_test.go` and `test/postgres_http_idempotency_integration_test.go`; change only the matching driver/generated import exemptions in `.golangci.yml`. Mutable proof resources are one disposable PostgreSQL 17 writer with `track_commit_timestamp=on`, bounded pools, relation/row locks, application-name trigger gates, backend PIDs, helper subprocesses, IPC gates, and disposable feature/outbox probe rows; each test owns cleanup.
  - Depends on: T1 — output handoff — needed to start.
  - Handoff: accepted additive schema/query/generated authority, concrete writer Store, reservation publication group, caller-`pgx.Tx` Acquire/Complete primitives, rollback release, writer reconciliation, exact epoch materialization, and deterministic PostgreSQL fixtures consumed by T3.
  - Proof: preserve canonical authority with `make migration-check` and `make sqlc-check`. TD-IDEM-003 recorded v1/current-v2 disagreement replays under retained v1 and rejects v1 removal: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyRecordedFingerprintVersion$' -count=1`. TD-IDEM-005 proves one local publisher, at most one publication connection per Store, writer reclassification, authority-B headroom, and no feature-owner wait with `go test -vet=off -race ./internal/infra/postgresidempotency -run '^TestPublicationGroup$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyFirstPublicationAndPoolHeadroom$' -count=1`, then repeats that integration command with `-race`. TD-IDEM-005A proves one non-resetting `W_in_progress` across coordinator, pool, `ACCESS EXCLUSIVE`-blocked writer read, and publication gate, returning inner 503 and never executing: `go test -vet=off ./internal/infra/postgresidempotency -run '^TestClassificationBudgetDoesNotReset$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyClassificationBudget$' -count=1`. TD-IDEM-006 proves the four live/death schedules, backend/lock disappearance, new generation, and exactly one successor: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyOwnerRecovery$' -count=1`. TD-IDEM-007 proves acquire/mutation/optional-outbox/result/completion atomicity, definite rollback, 40001/40P01, cancellation, and whole-callback-only retry: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyCallerTransactionAtomicity$' -count=1`. TD-IDEM-008 proves exact/overflow result bounds, response loss, the post-commit/pre-render SIGKILL carrier, exact epoch recovery, and one durable feature/result/outbox effect: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyReplayResultBoundAndPostCommitDeath$' -count=1`. TD-IDEM-009 keeps the real caller lost-result boundary and drives every reservation/caller writer branch with two absence successors: `REQUIRE_DOCKER=1 go test -tags=integration ./internal/infra/postgres -run '^TestInTxCommitOutcomes$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyCommitReconciliation$' -count=1`. The Store half of TD-IDEM-010 proves `committed_at = pg_xact_commit_timestamp(xmin)` and no fallback epoch: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyCommitEpoch$' -count=1`. Local TD-IDEM-012 proves stale/read-only absence never reserves, executes, expires, or cleans: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyWriterAuthority$' -count=1`.
  - Reopen if: any fixed reservation, lock, ambiguity, transaction, process-death, or exact-epoch carrier cannot be forced and independently observed — Technical Design; zero cross-replica publication wait/connection is required instead of the accepted replica bound — Specification R6; exact physical commit time cannot be retained or recovered and behavior must change — Specification R9.
  - Accepted: T2; evidence: every exact Proof command passed on its PostgreSQL 17 carrier, including publication race, lock/process-death, caller-transaction, replay, reconciliation, exact-epoch, and writer-authority schedules; fresh independent acceptance review PASS; candidate: current bounded diff.

- [x] T3: Registered capability runtime owns bounded maintenance, telemetry, readiness, and shutdown while zero registrations remain exactly inert
  - Source: `spec.md` R9, R12, and R13; `design/overview.md` Literal authoritative-commit horizons, maintenance/process lifecycle, security/privacy/telemetry/readiness, active-only config quantities, and bootstrap responsibility/file map; `design/rollout.md` deployment admission and maintenance/process lifecycle; `test-plan.md` lifecycle delta of TD-IDEM-010, TD-IDEM-011, TD-IDEM-011A, TD-IDEM-013A, TD-IDEM-014, and the exact Store conversion in TD-IDEM-015A.
  - Owner/surface/resources: change current `internal/infra/http/{idempotency.go,idempotency_test.go,router.go,router_contract_test.go}` only to bind the one private terminal observer: the envelope emits one final admission rejection before returning, while the admitted synthetic application adapter emits exactly one outcome after caller-owned transaction/reconciliation. Change current `internal/infra/postgresidempotency/{store.go,store_test.go,store_reserve.go,store_reserve_test.go}`; add `internal/infra/postgresidempotency/{store_maintenance.go,store_maintenance_test.go,telemetry.go,telemetry_test.go,vocabulary.go,vocabulary_test.go}`. Add `internal/config/{http_idempotency_config.go,http_idempotency_config_test.go}` and change `internal/config/{types.go,snapshot_contract_test.go}`. Add `cmd/service/internal/bootstrap/{startup_idempotency.go,startup_idempotency_test.go,startup_idempotency_integration_test.go}` and change `cmd/service/internal/bootstrap/{run.go,run_lifecycle_test.go,startup_http.go}`. Change current `test/{postgres_http_idempotency_fixtures_integration_test.go,postgres_http_idempotency_integration_test.go}`. Change `docs/project-structure-and-module-organization.md` and `scripts/ci/project-structure-check.sh` only for the tagged bootstrap-internal integration-test exception. Keep every other accepted T1/T2 idempotency surface, migration/query/generated source, `.golangci.yml`, `shutdown.go`, and `shutdown_test.go` unchanged. `HTTPIdempotencyConfig.OwnerRecoveryDelay` has no default, is validated only for a nonempty exact registration slice, and maps once through bootstrap into `postgresidempotency.StoreOptions`; it remains outside `IdempotencyOperation` and OpenAPI. `store_maintenance.go` owns the private `Store.allowsFirstExecution` predicate, terminal snapshot, first-error wakeup, and `Store.ObserveTerminal`; `store_reserve.go` calls `allowsFirstExecution` only after writer-confirmed absence and before publication. Bootstrap is the sole `TerminalErrors` receiver and returns the first request-discovered epoch/integrity error before a cadence tick; it preserves the existing supervisor and drain order. Mutable proof resources are T2's disposable PostgreSQL 17 writer/data volume, controlled restart, writer-clock rows/locks, `testing/synctest`, recording telemetry exporters, cached health, the terminal notification, and the existing background supervisor; each test owns cleanup.
  - Depends on: T2 — accepted output handoff — needed to start; no other dependency.
  - Handoff: accepted complete zero-registration-safe runtime path; active-only, no-default `OwnerRecoveryDelay` flowing exactly from typed config through one bootstrap mapping into concrete Store options and writer `recover_after`; mutually exclusive terminal observation (admission rejection in the envelope or exactly one post-transaction/reconciliation admitted outcome); request-discovered terminal wakeup that begins ordinary drain without a cadence tick; one supervised maintenance loop/probe/snapshot; ceiling-normalized retention; bounded telemetry vocabularies; positive local active composition and authoritative writer readback; and the complete fixed capability path set consumed unchanged by T4 and T5.
  - Proof: TD-IDEM-010 proves request-discovered epoch loss returns the closed 503, preserves one first terminal error, and makes the existing supervised task flip cached readiness and begin normal drain before any cadence advance: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyCommitEpoch$' -count=1` and `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyEpochLossDrains$' -count=1`. TD-IDEM-011 proves writer-clock ceiling normalization, including a positive sub-microsecond duration, result stripping and finite-guard deletion in both lock-forced request-first and cleanup-first orders, bounded batches, and a fresh Store's monotonic restart: `go test -vet=off ./internal/infra/postgresidempotency -run '^TestMaintenanceScheduleAndBounds$' -count=1` and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyRetentionAndCleanupRaces$' -count=1`, then repeats that integration command with `-race`. TD-IDEM-011A proves recoverable cadence, one cycle, safe-read continuation, pre-reserve capacity closure, and stale/terminal unready with `go test -vet=off ./internal/infra/postgresidempotency -run '^TestMaintenanceFailureAndCapacityClosure$' -count=1`, `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyMaintenanceReadiness$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyMaintenanceCapacity$' -count=1`. TD-IDEM-013A proves both finality branches, no provisional reservation/Acquire/Complete terminal event, one post-commit `executed`, reconciliation/render-loss classification, sentinel/label absence, `other` fallback, stale observation, and request-scoped versus readiness-terminal classification with `go test -vet=off ./internal/infra/http -run '^TestHTTPIdempotencyTerminalObservation$' -count=1`, `go test -vet=off ./internal/infra/postgresidempotency -run '^TestHTTPIdempotencyTelemetryAndVocabulary$' -count=1`, `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyReadinessLifecycle$' -count=1`, and `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresHTTPIdempotencyTelemetry$' -count=1`. TD-IDEM-014 proves an empty registration constructs/checks nothing, a nonempty registration rejects missing/nonpositive active config before Store/schema/writer work, and distinct `30s`/`50s` test sentinels map all five config fields exactly: `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyZeroRegistrationIsInert$' -count=1` and `go test -vet=off ./cmd/service/internal/bootstrap -run '^TestHTTPIdempotencyActiveConfigMapping$' -count=1`. TD-IDEM-015A's Store conversion proof rejects dropped, clamped, or constant writer arguments, including sub-microsecond ceiling rounding: `go test -vet=off ./internal/infra/postgresidempotency -run '^TestStoreOptionsOwnerRecoveryDelay$' -count=1`. Its positive local composition oracle runs `REQUIRE_DOCKER=1 go test -p=1 -count=1 -tags=integration ./cmd/service/internal/bootstrap -run '^TestPostgresHTTPIdempotencyActiveBootstrap$'`; valid otherwise-identical `30s`/`50s` typed config reaches initial `Maintain`/probe/HTTP construction, and each authoritative writer `recover_after` lies inside its narrower-than-`5s` shifted writer-clock bracket with disjoint normalized intervals. `make project-structure-check` proves the sole tagged bootstrap-internal integration-test exception.
  - Reopen if: the mutually exclusive finality owners cannot remain observable without a request token, second metric owner, or protected data; request-discovered terminal error cannot wake the existing supervisor before cadence; row locks cannot force both retention serial orders; ceiling arithmetic or bounded Store restart cannot be observed; bounded capacity observation cannot fail closed; empty/nonempty registration cannot remain the sole activation decision; or the exact Store writer argument cannot be derived from the one active config value — Technical Design. Reopen Go Ownership if the tagged bootstrap integration carrier cannot remain one bounded structure-check exception or the frozen observer/Store/bootstrap file map becomes cyclic. Reopen Specification R9 only if replay/duplicate-risk behavior or exact epoch must change.
  - Accepted: T3; evidence: every exact T3 Proof command above passed on the current bounded candidate, including real PostgreSQL retention/cleanup both orders and fresh restart, the positive sub-microsecond ceiling oracle, maintenance capacity, terminal telemetry, and active bootstrap composition; `make project-structure-check` passed; fresh independent acceptance review PASS after the ceiling-normalization correction; candidate: current bounded diff.
- [x] T4: `HTTP_IDEMPOTENCY=none|postgres` generation is deterministic, orthogonal, dependency-clean, and retains exactly the complete capability pack
  - Source: `spec.md` R13; `design/overview.md` Profile generation and absence/presence contract plus repository placement/operator contract; `design/rollout.md` Profile and build gate; `test-plan.md` TD-IDEM-015.
  - Owner/surface/resources: change `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `Makefile`, and the exact profile markers/inventories across every T1-T3 capability-owned path; change `docs/project-structure-and-module-organization.md` and `docs/repo-architecture.md`, add `docs/postgres-http-idempotency.md`, and update only reached README/config/operator surfaces required by the accepted pack. `api/openapi/service.yaml`, generated `internal/openapi/openapi.gen.go`, SQL query/generated files, `.golangci.yml`, config/bootstrap files, migrations, tests, and docs stay canonical/generated/profile-coupled exactly as fixed in T1-T3. Checkout-copy generator fixtures only; no external or persistent mutable resource.
  - Depends on: T3 — output handoff — needed to start.
  - Handoff: accepted `none|postgres` selector, `template.lock` value, immutable repeat behavior, complete on/off inventory, and initialized source identities consumed by T5.
  - Proof: TD-IDEM-015 initializes `none`, postgres with auth on/off and outbox/inbox/messaging combinations, invalid database, explicit empty, unknown, same-lock, and changed-lock cases; successful trees compile with health unopted, failed inputs leave byte-identical trees, equal replay is identical, and off/on inventories are exact. Run `make template-init-check`, `make project-structure-check`, `make openapi-check`, and `make sqlc-check`; every generated/drift command must report no difference.
  - Reopen if: one coherent removable profile cannot preserve shared OpenAPI/SQLC generation — Technical Design; current declarations make the fixed package/file or profile inventory ownership cyclic or give one file two independent reasons to change — Go Ownership.
  - Accepted: T4; evidence: `make template-init-check`, `make project-structure-check`, `make openapi-check`, `make sqlc-check`, and `make lint` (0 issues) passed; `git diff --check` passed; fresh independent acceptance review PASS, including targeted `TEMPLATE_INIT_PROFILE=http-idempotency make template-init-check`; candidate: HEAD `0e9713932a0f97d3560dcf5011b6e49ee08cf94f` plus current bounded diff.

- [x] T5: The existing migration and production-image release gate proves both source identities without promoting local evidence to adopter activation
  - Source: `spec.md` R13; `design/overview.md` Active-registration production-image proof carrier; `design/rollout.md` Runtime proof identities, migration/mixed-version sequence, repository publication gates, and rollback falsifiers; `test-plan.md` TD-IDEM-015A and the repository-local prerequisites of TD-IDEM-016.
  - Owner/surface/resources: add `scripts/ci/fixtures/postgres-http-idempotency-active.patch`; change `scripts/ci/runtime-image-build.sh`, the existing `Makefile` `migration-validate`/runtime-image aggregate and integration-package inventory, `scripts/ci/ci-change-scope.sh` and its self-test, and the existing `.github/workflows/ci.yml` route only as required to make every idempotency profile/fixture-only change take the full migration/image path. Keep T3's accepted `cmd/service/internal/bootstrap/startup_idempotency_integration_test.go` unchanged and consume its positive-composition receipt. Mutable proof resources are disposable Docker/Compose PostgreSQL volumes/networks/containers and two exact local image tags, one per source identity; the creating command registers teardown and reuses each tag without rebuild.
  - Depends on: T4 — output handoff and generated-profile gate — needed to start and prove; PR-GOSEC-01 — accepted Gosec prerequisite handoff — needed to prove.
  - Proof: TD-IDEM-015A first runs `make migration-check` and `REQUIRE_DOCKER=1 go test -p=1 -count=1 -tags=integration ./cmd/service/internal/bootstrap -run '^TestPostgresHTTPIdempotencyActiveBootstrap$'`. The integration carrier uses valid otherwise-identical typed config at `30s` and `50s`, brackets each authoritative writer reservation with `clock_timestamp()` reads narrower than `5s`, and proves each `recover_after` lies in its shifted writer-time interval while the normalized intervals are disjoint. The existing broad `make check-full` is the one host aggregate: within it the health-only `make runtime-image-build RUNTIME_IMAGE=service:ci` result is reused by `make migration-validate RUNTIME_IMAGE=service:ci`, while migration validation builds the distinct disposable active fixture once and reuses that exact tag for missing schema, read-only/non-writer with commit timestamps enabled, and `track_commit_timestamp=off`. The observable is repeatable up/down/up, migrate-only schema mutation, health-only ready/shutdown and previous/profile-off compatibility, active PostgreSQL rejection before OIDC I/O/serving, active initial `Maintain`/probe/HTTP construction, and no positive active production-image claim. Run `make template-init-check` after the aggregate to retain the exact generated-profile oracle; current exact-head CI remains the publication authority.
  - Reopen if: the active patch cannot produce one complete registration, a required PostgreSQL rejection occurs only after OIDC initialization, startup order no longer exposes the design's active rejection carrier, the exact configured delay cannot be observed at the authoritative writer `recover_after` without another production seam, or the existing exact-tag aggregate cannot preserve the two source identities — Technical Design.
  - Accepted: T5; evidence: `make migration-check`, `REQUIRE_DOCKER=1 go test -p=1 -count=1 -tags=integration ./cmd/service/internal/bootstrap -run '^TestPostgresHTTPIdempotencyActiveBootstrap$'`, the sole repaired-candidate `make check-full`, recovered `GOTOOLCHAIN=auto make template-init-check` session 95903 (exit 0; `template initialization contract passed`), `git apply --stat scripts/ci/fixtures/postgres-http-idempotency-active.patch`, and scoped `git diff --check` passed; fresh independent Implementation Review PASS; candidate: current bounded diff.
