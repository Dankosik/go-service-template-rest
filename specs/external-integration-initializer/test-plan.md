# External Integration Initializer Test Design

status: fixed candidate for independent Test Design Review

Routing authority:

- `design/transition.md` SHA-256
  `6a36f06d99af945c4be02071ef35e2f97769ab8b7ad2883494f150d6499447af`

Fixed accepted inputs:

- `spec.md` SHA-256
  `9a54ee75953d242cd37cd27b56e791e2e7f92e1fbdb7e5e528f9917bb50fbbf1`
- `review.md` SHA-256
  `0ce18e168e5f90ddcf631a164567f283e2a732b88e95aef235e6dc1791a71395`
- `design/overview.md` SHA-256
  `ad02cc02cd79dae097850eb241cb8d0f04ce8ee399fc5b2882a68b0255d3c2ac`
- `design/review.md` SHA-256
  `d80db1713d1118e2345b5cb3297b842f6bf9c21e5ca2d7d04a960a5d4dbd2639`

The Specification remains authoritative for behavior and the eight accepted
proof expectations. The reviewed Technical Design remains authoritative for
mechanism, ownership, and proving locations. This artifact owns only
falsifiers, deterministic controls, independent oracles, proving layers, and
validation commands. It does not enter Planning or Implementation and does not
authorize provider, network, credential, deployment, database, or other
external effects.

## Evidence boundary and dispositions

The initializer implementation does not exist as accepted input. Current dirty
HTTP-client, OAuth, initializer, and workflow edits remain evidence only and
are not fixtures, passing proof, or authority for this plan. No behavioral
command is run in Test Design. Every matrix row is a mandatory `planned
scenario` or `non-test falsifier` for later implementation acceptance; there
is no live-provider, deployment, or rollout row because those claims are
explicitly excluded from the accepted outcome.

All `.env` scenarios below use only harness-created entries inside disposable
temporary repositories. The harness must not inspect, source, or modify the
checkout's repository-root `.env`, if one exists.

## Test strategy decision

```text
disposition: proof_only
decision_or_constraint: prove the eight accepted expectations with one disposable command harness, generated package-local contracts, and the existing canonical validation owners; use direct byte and sink oracles rather than source presence or aggregate success
forced_consequences: every expectation has a named fail-before discriminator, deterministic fixture, independent oracle, proving layer, command, required input, owner, and smallest reopen path
proof_or_gap: the 25 Test Plan V1 rows below reconcile bidirectionally to expectations 1 through 8; execution remains planned because accepted implementation does not yet exist
blocker: none
strongest_rejected_alternative: none
rejection_reason: none
reopen_owner_or_condition: the smallest row-specific owner when a falsifier is infeasible or evidence invalidates its accepted input
```

## Proof strategy

The proof ladder stops at the smallest boundary that can expose the wrong
behavior:

- Command grammar, clean-start admission, identity, transaction finality,
  ignored-path custody, byte preservation, no-op, and refresh use
  `scripts/ci/integration-init-check.sh` over disposable locally committed
  initialized repositories. A package unit test or mocked filesystem was
  rejected because it cannot prove Git identity, path containment, patch
  application, ignored-path exclusion, or cleanup.
- Pure config, adapter construction, target denial, auth binding, and close
  behavior use generated package-local tests. A full service process was
  rejected because it can fail for an unrelated dependency and cannot localize
  the owner that admitted the wrong tuple or lifecycle edge.
- OpenAPI and Protobuf source/output parity use their canonical generation and
  drift commands inside the generated fixtures. Generated-file presence or a
  successful package compile was rejected because stale output can still
  compile.
- The fixed HTTP and gRPC contracts used by the harness each expose one
  test-only probe operation. The harness creates a temporary package-local
  `client_contract_test.go` to select that known operation, exercise one
  bounded success and one sanitized failure, and then removes the fixture. It
  does not persist an operation method or infer adopter/provider semantics.
- Non-disclosure uses distinct runtime-generated canaries and byte-scans every
  captured sink. Secret scanning supplements this oracle; it cannot replace it
  because a synthetic canary need not match a scanner rule.
- Repository-wide `make check`, delivery, secret, and change-scope aggregates
  are gates over the focused scenarios, never substitutes for them.

## Deterministic controls and fixtures

- Each command case starts from a fresh local Git repository with one fixed
  `HEAD`, committed contract, regular `template.lock`, resolved profile, and no
  unrelated changes. Harness commits are local fixture setup, not publication.
- HTTP uses `billing` at
  `api/external/billing/openapi.yaml`; gRPC uses `identity` below
  `api/proto/external/identity/`. Their minimal contracts and expected generated
  path inventories are fixed in the harness.
- Tool execution sets `GOTOOLCHAIN=local`, `GOPROXY=off`, and `GOSUMDB=off`.
  Provider exchange fixtures use only package-local fakes, `httptest`, or
  `bufconn`; no public DNS, ambient proxy, live endpoint, or provider account is
  an input.
- The harness records raw bytes and modes for every pre-existing tracked path,
  separate exact bytes for each synthetic ignored `.env`, and exact inventories
  for initializer locks, temporary worktrees, patches, generated paths, and
  manual paths. It compares bytes directly rather than trusting timestamps.
- The mandatory ignored-path custody proof runs the initializer subprocess on
  the repository's Linux CI runner under a locally available `strace`, scoped
  to file syscalls and exact fixture paths. For regular non-symlink entries it
  permits only the metadata syscall used by the one boolean presence helper;
  for symlinks it permits only a no-follow `lstat`/`newfstatat`/`statx` shape.
  It rejects every `open*`, `readlink*`, following stat/access, rename, chmod,
  unlink, or other path operation on `.env`, and every resolution of its
  outside symlink target. A structural check admits exactly that boolean helper
  and its two call sites, so syscall metadata cannot become another branch or
  report. Missing Linux or `strace` is an evidence gap; the harness never
  installs a tool or falls back to canary scanning for this claim.
- Harness-local forwarding wrappers may control only subprocess boundaries:
  one Go wrapper fails the staging validation command after asserting it runs
  in the detached worktree; one Git wrapper creates a synthetic `.env`
  immediately after `git apply --check`; cleanup wrappers copy about-to-be-
  removed temporary material to a harness-owned inspection directory before
  forwarding the real cleanup. Wrappers record their PIDs and exact allowed
  fixture-creation syscalls so the custody trace excludes only those harness
  operations. Each wrapper must assert that its trigger ran exactly once and
  otherwise forward arguments and exit status unchanged. No production test
  hook is added.
- Initial success is committed before no-op or refresh. Refresh changes and
  commits only the canonical contract first; manually owned adapter, test, and
  documentation sentinels are also committed before refresh so the clean-tree
  precondition remains true.
- Presence cases cover a regular file, directory, dangling symlink, symlink to
  an outside canary file, and named pipe. Tests never open the entry. The same
  fixed sanitized rejection must result for every type.
- Unique canaries represent ignored-file values, legacy singleton values,
  named values, client secret, token, target, provider body, raw error, and
  contract data. Test names and assertion messages contain labels only, never
  canary bytes. Captured sinks are stdout, stderr, formatted errors, temporary
  and rollback material, generated files, tracked examples, documentation,
  logs, metrics/traces when emitted, and test diagnostics.
- Every harness case prints its matrix ID only after its oracle passes and ends
  with the exact expected case count. A zero-match Go `-run`, skipped canonical
  generator, untriggered wrapper, or unavailable local tool is failure, not
  passing evidence.

## Scenario matrix

Each row is one Test Plan V1 obligation. Table-driven inputs are merged only
when they share one wrong observable, oracle, proof boundary, and reopen path.

| ID | disposition | claim | wrong_observable | controlled_trigger | independent_oracle | proof_boundary | command_or_procedure | required_input_and_status | owner | reopen_owner |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| E1-HTTP-01 | planned scenario | One HTTP invocation registers the fixed external OpenAPI source, uses the pinned client-only generator and retained bounded HTTP transport, and leaves compiling formatted output. | The external source is absent from canonical generation, another generator or server binding appears, generated code is outside the adapter-internal path, retained HTTP is bypassed, or output does not compile/format. | Clean `billing`, HTTP, `external-https`, `AUTH=none` fixture with the fixed minimal OpenAPI contract and retained `outbound_http = "bounded"`. | Exact record/source/generator/output inventory matches the reviewed topology; `go generate` changes nothing; only the adapter imports generated code; focused packages compile; `make fmt-check openapi-check` passes in the fixture. | Disposable initialized HTTP checkout plus canonical OpenAPI and Go owners. | `make integration-init-check` | Planned harness and pinned local OpenAPI tools; no provider input. | `scripts/ci/integration-init-check.sh`, OpenAPI owners, generated billing packages | Specification if source/transport behavior changes; Technical Design if canonical registration or containment is infeasible. |
| E1-GRPC-01 | planned scenario | One gRPC invocation registers its Protobuf subtree in the existing Buf module, uses pinned generators and retained bounded gRPC transport, and leaves compiling formatted output. | The source/output parity is absent, another module/generator appears, generated code escapes its subtree or adapter import boundary, retained `grpcclient` is bypassed, or output does not compile/format. | Clean `identity`, gRPC, `AUTH=none` fixture with the fixed minimal Protobuf contract and retained `grpc = "enabled"`. | Exact record/source/output inventory matches the reviewed topology; Buf generation changes nothing; depguard admits generated imports only in the named adapter/generated subtree; focused packages compile; `make fmt-check proto-check` passes in the fixture. | Disposable initialized gRPC checkout plus canonical Buf/Protobuf and Go owners. | `make integration-init-check` | Planned harness and cached pinned Buf/Go plugins; no provider input. | `scripts/ci/integration-init-check.sh`, Protobuf owners, generated identity packages | Specification if source/transport behavior changes; Technical Design or Go Ownership if canonical generation or containment is infeasible. |
| E1-RUNTIME-01 | planned scenario | Generated config, adapter, bootstrap construction, and close edges use the retained owners, perform no provider I/O during construction, and close once in reviewed order. | Startup performs DNS/token/resource I/O, publishes a callable operation or health probe, omits partial-start cleanup, closes in the wrong order, or exposes a provider error. | Generated HTTP and gRPC fixtures with unreachable targets, call-counting test seams at retained consumer boundaries, post-construction startup failure, normal shutdown, and repeated close. | Construction call counts stay zero; exported method/API inventory contains no operation or token source; recorder order is partial close once or `drain -> integration close in reverse construction order -> existing dependency close -> telemetry`; errors contain only bounded owner/context. | Generated `internal/config`, named adapters, and `cmd/service/internal/bootstrap` package contracts. | `make integration-init-check` | Planned generated package tests; no live endpoint. | Generated config/adapter/bootstrap tests | Specification for callable/readiness behavior; Technical Design for construction or lifecycle change; Go Ownership if the seam moves. |
| E2-OAUTH-01 | planned scenario | `AUTH=oauth2-client-credentials` gives each named adapter its own retained credential owner and only an authenticated HTTP Doer or authenticated gRPC connection; no secret, token, or token-source API escapes. | The adapter shares singleton credentials, exposes raw credentials/token methods, attaches auth outside the retained owner, accepts a caller auth source, or omits auth cleanup. | First HTTP OAuth integration after absent `.env`, then a later disjoint gRPC OAuth integration; package tests exercise retained HTTP and gRPC credential binding with synthetic acquisitions and repeated close. | Config cardinality is per name; import/type/API checks show only the retained owner and authenticated consumer surface; retained owner tests observe one correct auth attachment, reject competing caller auth, never expose token bytes, and close once before transport. | Generated config/adapters plus existing `oauth2clientcredentials` HTTP/gRPC contracts. | `make integration-init-check` | Planned generated fixtures and existing local OAuth test seams; no credential/provider input. | Harness, generated adapter tests, `internal/infra/oauth2clientcredentials` tests | Specification/Security if auth or custody changes; Technical Design if retained binding cannot realize it. |
| E2-NONE-01 | planned scenario | `AUTH=none` emits no auth-specific field, example, import, owner, close edge, or generated dependency and leaves retained singleton OAuth bytes unchanged. | Any auth path appears or any existing singleton source/example/test byte changes. | HTTP and gRPC `AUTH=none` initial fixtures with the retained OAuth capability both present and absent. | Exact generated inventory and `go list -deps` contain no named auth surface; byte snapshot of every singleton owner is identical before/after; unauthenticated local fake succeeds without a token request. | Disposable command fixture, generated package graph, and exact byte snapshot. | `make integration-init-check` | Planned harness; no provider input. | `scripts/ci/integration-init-check.sh` | Specification if unauthenticated behavior changes; Technical Design if conditional omission is infeasible. |
| E3-INPUT-01 | planned scenario | Invalid names, transport/target/auth choices, missing or extra initializer variables, and dirty caller trees reject before mutation with one bounded reason. | Any owned path changes, a temporary patch reaches the caller, an unknown choice is normalized/ignored, or error output discloses ambient values. | Table of invalid identifiers/keywords/case, missing variables, HTTP without/invalid target, gRPC with target, unknown auth/transport, unknown Make command-line assignment, and staged/untracked/modified caller paths. | Exit is nonzero with exact reason class; `HEAD`, index, tracked/untracked byte inventory, and initializer-owned paths equal entry; no stage/apply wrapper fires; output lacks all canaries. | Parent Make/script admission in disposable clean/dirty repositories. | `make integration-init-check` | Planned harness; no generator/provider input. | `scripts/ci/integration-init-check.sh` | Specification if public grammar or dirty-tree policy changes; Technical Design for parser placement only. |
| E3-PRECONDITION-01 | planned scenario | Missing/conflicting retained choices, absent or invalid `template.lock`, unresolved profile state, collisions, and changed locked identity reject before mutation and name the exact failed authority. | The initializer reconstructs a capability, overwrites a collision, accepts ambiguous profile state, or changes a locked identity. | Table across each retained-choice miss, missing/non-regular lock, unresolved marker/source, reserved output/record collision, extra/mismatched record key, and changed name/transport/target/auth. | Nonzero bounded result identifies the required choice/authority; no generator/stage/apply call occurs; complete caller byte/inventory snapshot is unchanged. | Parent preflight and integration-record admission. | `make integration-init-check` | Planned locally committed fixtures. | `scripts/ci/integration-init-check.sh` | Specification for another accepted migration/precondition; Technical Design for record or preflight mechanism. |
| E3-CONTRACT-01 | planned scenario | Only the exact tracked committed regular non-symlink contract shape is admitted and no path can escape the repository. | An untracked, modified, renamed, symlinked, wrong-location/type, traversal, or invalid contract reaches generation or changes output. | HTTP and gRPC table: untracked, staged/uncommitted, symlink/dangling symlink, directory, wrong extension/location/name, traversal text, outside target, invalid schema/module/import, and valid control. | Every invalid case exits before staging/caller mutation with the contract reason class; generator wrapper count is zero; outside and caller bytes are unchanged; valid controls reach exactly one pinned validator/generator. | Parent contract/path admission and canonical validators. | `make integration-init-check` | Planned local fixtures and pinned tools. | Harness plus OpenAPI/Buf validation owners | Specification if another source shape is needed; Security for another trust boundary; Technical Design for validator placement. |
| E3-ENV-01 | planned scenario | On the first singleton-retiring OAuth initial invocation, any exact root `.env` directory entry rejects before staging using presence only. | Entry type changes behavior/message, any entry is silently opened or followed without mutation/disclosure, staging begins, or a canary reaches output/material. | Separate regular, directory, dangling symlink, outside-canary symlink, and named-pipe fixtures at exact root `.env`; an absent control reaches staging. Under the same oracle, disposable script mutants silently open the regular entry and follow-stat the outside symlink before returning the correct rejection. | Present cases return the same fixed message naming only `.env` and the manual prerequisite; stage/apply counts are zero; raw entry, outside target, repository bytes/modes, and canaries are unchanged/absent from sinks. The Linux file-syscall trace admits only the one presence-helper metadata shape, with no-follow required for symlinks, and contains no open/readlink/follow/target access; both silent-access mutants must fail this trace while the unmutated candidate and absent control pass. The structural owner admits no other `.env` path consumer. | First parent presence admission in disposable first-OAuth fixtures plus syscall/structural custody proof. | `REQUIRE_ENV_CUSTODY_TRACE=1 make integration-init-check` | Planned harness-owned synthetic entries and local Linux `strace`; missing trace is a gap. The checkout `.env` is out of scope. | `scripts/ci/integration-init-check.sh` and `scripts/ci/project-structure-check.sh` | Specification/Security if automatic migration, metadata, or custody is required; Technical Design if non-following presence cannot be realized. |
| E3-ENV-02 | planned scenario | The repeated presence admission rejects an exact root `.env` entry created after staging and before caller patch application without opening or following it. | The candidate applies, silently opens/follows the new entry, or leaves partial repository output while still returning the required rejection. | Harness Git wrapper creates a synthetic canary `.env` after the exact `git apply --check` in one first-OAuth initial run and asserts the earlier admission saw absence; disposable final-admission mutants silently open and follow-stat it before rejecting. | Final result is the same sanitized rejection; apply count is zero; caller repository/index remain at entry bytes; `.env` is byte-identical; captured discarded stage/patch material lacks canary. After excluding only the wrapper PID's asserted creation syscalls, the Linux trace admits only the final presence-helper metadata shape and no open/readlink/follow/target access; both silent-access mutants fail and the unmutated candidate passes; wrappers fire exactly once. | Final parent admission and discarded detached candidate plus syscall custody proof. | `REQUIRE_ENV_CUSTODY_TRACE=1 make integration-init-check` | Planned forwarding wrapper around local Git and local Linux `strace`; missing trace is a gap. No production hook. | `scripts/ci/integration-init-check.sh` and `scripts/ci/project-structure-check.sh` | Specification if initializer must arbitrate concurrent custody; Technical Design if the second non-following admission cannot precede mutation. |
| E4-INITIAL-01 | planned scenario | A staging validation failure after partial initial rendering leaves the clean caller byte-identical, leaves `.env` absent, and removes only initializer-owned temporary state. | Partial scaffold reaches caller, `.env` is created, lock/worktree/patch survives, or caller index/HEAD changes. | First OAuth fixture with absent `.env`; forwarding Go wrapper fails exactly the detached-stage validation command after generated output exists. | Wrapper hit is one; caller `HEAD`, index, tracked/untracked inventory, modes, and bytes equal entry; `.env` remains absent; captured cleanup inventory contains only initializer-owned temporary paths and final cleanup leaves none. | Detached-worktree transaction and cleanup. | `make integration-init-check` | Planned harness-local validation failure; no provider input. | `scripts/ci/integration-init-check.sh` | Technical Design if detached staging/cleanup cannot restore the accepted finality; Specification only if clean-start byte restoration changes. |
| E4-REPEAT-01 | planned scenario | A committed same-identity repeat with unchanged contract is a no-op; a committed contract change updates generated-only bytes. | No-op creates any diff, or refresh rewrites adapter/config/bootstrap/docs/record/tests/manual sentinels. | Commit successful HTTP and gRPC scaffolds; repeat unchanged; then commit one semantics-neutral fixture contract change and run refresh after committed manual sentinels exist. | Unchanged repeat leaves exact clean status and no apply; refresh diff is non-empty and contained exactly in declared generated-only paths; canonical drift passes; every manual sentinel/record byte is identical. | Parent mode classification, registered generators, refresh allowlist, and caller patch. | `make integration-init-check` | Planned committed local fixtures and pinned tools. | `scripts/ci/integration-init-check.sh` plus generator owners | Specification for rename/reconfigure/manual regeneration; Technical Design if generated-only refresh cannot hold. |
| E4-REFRESH-01 | planned scenario | A failed same-identity refresh restores generated repository bytes and leaves a present synthetic `.env` byte-identical and outside all temporary material. | Generated partial output reaches caller, `.env` changes/disappears, presence rejection occurs on refresh, or canary appears in captured material/output. | Commit successful OAuth scaffold and changed contract, create harness-owned `.env`, then fail the detached-stage validation after regeneration; companion no-op refresh succeeds with the entry present. | Caller bytes/modes equal refresh entry; `.env` raw bytes equal entry; no presence-admission branch fires; captured worktree/patch/cleanup and output lack the canary; temporary state is gone. | Refresh classification, detached staging, cleanup, and ignored-path exclusion. | `make integration-init-check` | Planned synthetic fixture; checkout `.env` remains untouched. | `scripts/ci/integration-init-check.sh` | Specification if refresh gains `.env` custody or different consequences; Technical Design for transaction failure. |
| E5-OPENAPI-01 | planned scenario | Canonical OpenAPI generation/drift detects a stale external HTTP client after initializer registration. | A mutated generated external client leaves `make openapi-check` green or regeneration omits it. | In committed HTTP fixture, mutate one generated byte, run the canonical check expecting failure, regenerate, then rerun expecting clean success. | Failure output identifies OpenAPI drift and the external generated path; regeneration restores the committed bytes; passing rerun executes validation, drift, and compile owners. | Generated HTTP fixture and canonical OpenAPI aggregate. | `make integration-init-check` | Planned harness invokes `make openapi-check` and asserts mutation sensitivity. | OpenAPI generation/drift owners | Technical Design if canonical discovery cannot include fixed external paths. |
| E5-PROTO-01 | planned scenario | Canonical Protobuf generation/drift detects stale external gRPC output and record/source/output mismatch. | Mutated/missing/orphan external Protobuf output or record parity leaves `make proto-check` green. | In committed gRPC fixture, separately mutate generated bytes and break record/source/output parity; each must fail before a clean regenerate/pass control. | Each mutant fails with its exact path/parity class; clean `make proto-check` executes format, lint, generation-drift, and parity checks with no diff. | Generated gRPC fixture and canonical Protobuf aggregate. | `make integration-init-check` | Planned harness invokes `make proto-check` and asserts both mutants. | Protobuf generation/drift owners | Technical Design/Go Ownership if the existing module cannot express containment/parity. |
| E5-ROUTING-01 | non-test falsifier | Initializer, record, contract, generator, generated, config, adapter, docs, and proof-path changes route to every owning focused gate. | `ci-change-scope` omits an owning check or an unrelated path triggers the initializer gate. | Self-test table supplies one representative changed path from each new class plus unrelated controls. | Exact required-check set includes initializer, matching OpenAPI/proto, structure, Go, secret, and template-init gates where owned; unrelated controls preserve prior routing. | `scripts/ci/ci-change-scope.sh` self-test. | `make ci-change-scope-check` | Planned path table; no external input. | `scripts/ci/ci-change-scope.sh` | Technical Design if current aggregate routing cannot express dynamic fixed paths. |
| E5-BOUNDARY-01 | planned scenario | Generated config and retained HTTP/gRPC constructors reject every unsupported target/trust shape before provider I/O, and generated structure exposes no dynamic destination or forbidden import. | Invalid scheme/authority/suffix/resolver/TLS shape is admitted, caller data selects a target, a generated package imports outside its boundary, or a provider call occurs. | Generated package tables cover HTTP user info/query/fragment/redirect/proxy/authority change/public-private resolution/suffix mismatch and gRPC non-`dns:///hostname:443`, IP/user info/path/query/fragment/plaintext/nil TLS/service-config/dynamic target; valid construction controls have zero calls. | Exact field/reason class is returned; call counters remain zero for admission failures; valid constructors retain fixed authority/system-root hostname; structure/depguard rejects forbidden imports and callable target seams. | Generated config/adapters, retained `httpclient`/`grpcclient`, and project-structure owner. | `make integration-init-check` | Planned package-local deterministic tables; no DNS/provider input. | Generated package tests and `scripts/ci/project-structure-check.sh` | Specification/Security for a new target/trust class; Technical Design or Go Ownership for mechanism/placement. |
| E5-GATES-01 | planned scenario | Generated results pass project structure, formatting, focused compile/tests, shell validation, secret scanning, template capability cross-product, and the ordinary aggregate. | Any owning gate skips the new dynamic paths, passes with zero tests, or accepts a seeded structural/secret/capability mutant. | Clean HTTP/gRPC fixtures plus one controlled mutant for orphan path, forbidden import, non-empty tracked secret example, absent retained capability, and a shell defect in a disposable copy. | Each mutant fails only its owning focused gate; clean fixtures pass `project-structure-check`, named package tests with observed test events, `fmt-check`, canonical generator checks, secret scan, and `check`; source candidate passes delivery and template-init gates. | Generated fixtures plus repository validation owners. | `make integration-init-check`; then on the fixed source candidate `make delivery-quality template-init-check check` and `bash scripts/ci/secret-scan.sh change origin/main` | Docker/cache required only by delivery ShellCheck; absence is an evidence gap, never a pass. | Harness and existing validation owners | Smallest failed code/gate owner; Technical Design only if an accepted path has no canonical owner. |
| E6-HTTP-01 | planned scenario | The harness's generated HTTP fake proves construction, one bounded exchange, and one sanitized failure without persisting provider semantics or using a live endpoint. | Construction performs I/O, the fixed test operation cannot cross generated binding, success is fake without a request, failure leaks raw body/error, or the fixture survives initialization. | Harness temporarily adds package-local `client_contract_test.go` for the fixed OpenAPI probe; one local success captures method/path/bounds and one canary failure is returned. | Call count is zero after construction and one after each exchange; captured request matches the fixed test contract; success returns fixture data only inside the test; failure is bounded and canary-free; test file is absent from final scaffold and docs/API expose no operation. | Generated billing adapter package with temporary harness-owned local fake. | `make integration-init-check` | Planned loopback/package fake; no provider account or business mapping. | `scripts/ci/integration-init-check.sh` temporary HTTP contract test | Specification if a callable scaffold/provider claim is required; Go Ownership if the test cannot remain fixture-only. |
| E6-GRPC-01 | planned scenario | The harness's generated gRPC fake proves connection construction, one bounded unary exchange, and one sanitized failure without selecting an adopter service or persisting provider semantics. | Construction performs RPC/DNS, generated binding cannot exchange, failure leaks raw status/details, or selected service/method becomes scaffold API. | Harness temporarily adds package-local `client_contract_test.go` for the fixed Protobuf `Probe` using `bufconn`; one success and one canary status failure run under finite contexts. | No call occurs at construction; one call occurs per exchange; success is exact fixture data; failure maps to bounded test-only expectation with no canary in scaffold sinks; temporary test is removed and adapter exports no operation. | Generated identity adapter package with temporary harness-owned gRPC contract test. | `make integration-init-check` | Planned `bufconn` fixture and fixed harness proto only. | `scripts/ci/integration-init-check.sh` temporary gRPC contract test | Specification if a service subset/callable operation is required; Go Ownership if the fixture cannot stay test-only. |
| E7-INITIAL-01 | non-test falsifier | Before and after every initial invocation, all pre-existing unrelated tracked capabilities and manually owned paths are byte-identical. | Any path outside the reviewed initial-mode allowlist changes mode, bytes, existence, or index state. | HTTP/gRPC and auth/no-auth fixtures seed committed distinct sentinels across retained config, features, transports, docs, scripts, examples, and generated/manual paths outside the invocation allowlist. | Full `git ls-files` byte/mode snapshot and untracked inventory compare equal outside the exact allowlist; caller index remains unchanged; a seeded out-of-allowlist write mutant is rejected and fully restored. | Parent changed-path containment, one patch apply, and whole-fixture snapshot. | `make integration-init-check` | Planned locally committed sentinel fixtures. | `scripts/ci/integration-init-check.sh` | Specification if dirty/unrelated path ownership changes; Technical Design if allowlist/patch containment cannot preserve bytes. |
| E7-REFRESH-01 | non-test falsifier | Refresh preserves every manually owned adapter, config, bootstrap, record, test, documentation, and unrelated capability byte. | Contract refresh overwrites or deletes any non-generated path even if final code compiles. | After initial success, commit unique valid manual sentinels in each manual owner and unrelated capability, then commit a contract change and refresh; seed a generator/manual-path mutant in a separate fixture. | Exact pre/post byte/mode snapshot differs only at declared generated-only files; every sentinel and record is identical; mutant is rejected with caller bytes restored. | Refresh allowlist and whole-fixture byte oracle. | `make integration-init-check` | Planned committed fixture. | `scripts/ci/integration-init-check.sh` | Specification for automatic manual regeneration; Technical Design/Go Ownership if generated/manual separation fails. |
| E8-DISCLOSURE-01 | planned scenario | Distinct ignored-file, secret, token, target, contract, provider-body, and raw-error canaries never reach initializer output, errors, rollback material, generated artifacts, docs, logs, metrics/traces, or test diagnostics. | Any canary appears outside its harness-owned input fixture or an error reports `.env` metadata/content. | Drive every initial/refresh rejection, success, staged failure, local fake failure, config denial, close timeout, and legacy-key case with separate runtime-generated canaries; capture cleanup material through forwarding wrappers before deletion. | Byte scan of each enumerated sink finds no canary; `.env` errors are byte-identical across entry types and contain only path/manual prerequisite; test runner output reports labels/IDs only; repository secret scan passes independently. | Command harness, generated packages, observability/error sinks, cleanup capture, and secret-scan owner. | `make integration-init-check` | Planned synthetic canaries only; no checkout `.env`, credential, or provider input. | Harness, generated package tests, secret-scan owner | Specification/Security for a new reader or disclosure sink; smallest leaking Technical Design/Implementation owner otherwise. |
| E8-LEGACY-01 | planned scenario | After first OAuth initialization retires singleton config, unchanged legacy `APP__OUTBOUND_AUTH__*` inputs fail startup admission as unknown keys without value disclosure. | A legacy key is accepted/aliased, failure occurs only after adapter construction, or formatted diagnostics contain its canary value. | Generated OAuth fixture restores a harness-owned legacy environment fixture with a distinct value and invokes the same `internal/config.LoadDetailed` path used before bootstrap; format error/report through all captured forms. | `errors.Is(err, ErrUnknownKey)` and `ErrorType(err) == "unknown_key"`; failed stage is validation before adapter construction; diagnostics name the retired key path only and exclude the value; generic existing unknown-key controls still pass. | Generated `internal/config` startup admission plus retained unknown-key proof. | `make integration-init-check` | Planned synthetic legacy fixture; no actual `.env` is sourced. | Harness and `internal/config` generated tests | Specification if compatibility alias/migration is required; Go Ownership only if unknown-key admission moves. |
| E8-NAMED-01 | planned scenario | A developer/operator-manually mapped complete named environment tuple is accepted locally and constructs the named adapter without provider I/O. | Named keys are unknown/mis-mapped, secret file input is accepted, startup requires legacy keys, construction performs network/token work, or values leak. | Generated HTTP and gRPC OAuth fixtures supply complete distinct named environment tuples; companion missing/empty/incompatible and non-empty YAML secret cases fail. | `LoadDetailed` returns the exact named immutable config only for the complete environment tuple; legacy fields are absent; adapter/bootstrap construction succeeds with zero fake calls; invalid cases identify field/reason only; no canary appears in output. | Generated config, bootstrap mapper, adapter constructor, and secret-source policy. | `make integration-init-check` | Planned environment-only synthetic values and no live endpoint. | Harness plus generated config/adapter/bootstrap tests | Specification/Security if mapping or secret source changes; Technical Design/Go Ownership if named construction cannot use reviewed owners. |

## Validation ladder

Implementation iterates with the named generated package tests and the single
fixture harness. The fixed implementation candidate then runs, in order:

1. `make integration-init-check`
2. On the mandatory Linux custody runner,
   `REQUIRE_ENV_CUSTODY_TRACE=1 make integration-init-check`
3. `go test -vet=off -count=1 ./internal/config ./internal/infra/httpclient ./internal/infra/grpcclient ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap`
4. `make openapi-check proto-check`
5. `make project-structure-check ci-change-scope-check`
6. `make template-init-check`
7. `make delivery-quality`
8. `bash scripts/ci/secret-scan.sh change origin/main`
9. `make check`

The harness owns all dynamic generated integration packages and must report the
executed matrix IDs and exact case count. The source-tree package command cannot
replace that generated-fixture proof. A cached result, zero-match `-run`, skipped
generator, unavailable required local tool, untriggered failure wrapper,
unreadable base for the secret scan, unavailable Linux custody trace, or
missing Docker for `delivery-quality` is an evidence gap, not passing proof. No
race or repetition gate is required: the accepted initializer is
process-serialized and every concurrency boundary above uses an explicit
subprocess barrier rather than scheduler timing.

## Bidirectional reconciliation

| Accepted proof expectation | Scenario IDs |
| --- | --- |
| 1. HTTP/gRPC canonical generation, retained transports, compiling formatted scaffold | E1-HTTP-01, E1-GRPC-01, E1-RUNTIME-01 |
| 2. OAuth retained owner and complete `AUTH=none` absence | E2-OAUTH-01, E2-NONE-01, E1-RUNTIME-01 |
| 3. Pre-mutation fail-closed input, capability, contract, collision, identity, dirty-tree, and first-OAuth `.env` admission | E3-INPUT-01, E3-PRECONDITION-01, E3-CONTRACT-01, E3-ENV-01, E3-ENV-02 |
| 4. Initial failure restoration, no-op, generated-only refresh, and initial/refresh `.env` consequences | E4-INITIAL-01, E4-REPEAT-01, E4-REFRESH-01 |
| 5. Canonical stale-generation detection and aggregate structure/format/compile/denial/secret/routing gates | E5-OPENAPI-01, E5-PROTO-01, E5-ROUTING-01, E5-BOUNDARY-01, E5-GATES-01 |
| 6. Generated local fake construction, bounded success, and sanitized failure only | E6-HTTP-01, E6-GRPC-01 |
| 7. Unrelated capability/manual path byte identity | E7-INITIAL-01, E7-REFRESH-01, E4-REPEAT-01 |
| 8. Non-disclosure, stale singleton rejection, and manually mapped named local runtime | E8-DISCLOSURE-01, E8-LEGACY-01, E8-NAMED-01 |

Every matrix row traces to one accepted expectation, reviewed mechanism, or
required proving owner, and every accepted expectation maps back to at least
one discriminating row. Planning may later choose only dependency order and
implementation placement; it may not weaken, merge away, or reclassify this
matrix.

## Phase and reopen disposition

No accepted behavior, authority, mechanism, or ownership is changed by this
Test Design. A failed or infeasible row reopens only its named smallest owner.
Specification reopens only if the fixed fail-closed behavior, developer or
operator custody boundary, byte-restoration rules, or initial/refresh
consequences cannot be preserved without changing accepted behavior or
authority.

After one fresh independent `PASS` on this exact artifact, Test Design may
return a ready transition receipt. Planning remains next but is not entered or
authorized here.
