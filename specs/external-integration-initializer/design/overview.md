# Technical design — External Integration Initializer

status: ready

Realizes the current [../spec.md](../spec.md).
The accepted behavior, exclusions, proof expectations, and reopen conditions
remain authoritative. This design adds no provider operation, retry policy,
business port, deployment step, credential value, or live-provider claim.

Current repository authority is PR base
`94dc45411c99413739a75a435aa37b25befeba77` plus this fixed implementation
candidate. Unrelated worktree edits are not implementation inputs.

## Drivers and selected mechanism

| Driver | Forced consequence |
| --- | --- |
| The command is build-time, local, and must not contact a provider or network | `make integration-init` delegates to one repository Bash script, admits only document-local HTTP `$ref` values, and runs pinned tools with `GOTOOLCHAIN=local`, `GOPROXY=off`, and `GOSUMDB=off`; an unavailable local tool fails before the caller worktree changes. |
| The input contract and start state are immutable | Preflight requires one clean worktree, the same `HEAD` throughout the run, a regular committed contract at the accepted path, a regular `template.lock` with canonical `state = "complete"`, resolved profile choices, and no unresolved profile markers. |
| OAuth config must have one runtime owner | Retaining the profile keeps only the reusable credential package; each OAuth invocation creates its named tuple directly and never installs a root singleton. |
| A multi-file failure must restore exactly the clean start | Rendering, generation, formatting, and validation occur in one temporary detached Git worktree at the caller's `HEAD`; only its validated cached diff is applied to the caller. `git apply` is the sole caller-worktree mutation. |
| Manual adapter work must survive regeneration | Initial creation and same-identity refresh are separate modes. Refresh runs only the registered generator and validation; it never renders the adapter, config, bootstrap, docs, record, or proof files again. |
| HTTP and gRPC already have retained owners | HTTP reuses `internal/infra/httpclient` and the pinned oapi-codegen tool. gRPC reuses `internal/infra/grpcclient`, the current Buf module, `buf.gen.yaml`, and pinned Go plugins. OAuth reuses `internal/infra/oauth2clientcredentials`. |
| One invocation must not earn a framework | Each integration has a concrete adapter, concrete config field, concrete bootstrap call, one fixed record, and explicit generator registration. There is no runtime registry, map, factory, plugin surface, provider interface, or manifest interpreter. |
| Provider semantics are unknown by contract | The scaffold constructs transport and generated bindings but exposes no callable operation. A later provider method stays manual and must own its request/response mapping, errors, budget, retry eligibility, and business injection. |

The selected command owner is `scripts/integration-init.sh`, invoked by a thin
Make target. The script uses the existing shell, Git, Make, Go-tool, oapi-codegen,
and Buf paths. It is not a new executable entry point and adds no module.

The accepted `OUTBOUND_HTTP` prerequisite is not yet present in authoritative
`HEAD`; its current dirty implementation is evidence only. This design assigns
the production decision to `scripts/init-module.sh`: it admits exactly
`OUTBOUND_HTTP=none|bounded`, records `outbound_http` in `template.lock`, retains
`internal/infra/httpclient` for `bounded` and whenever another selected retained
capability still imports it, and removes it only when no selected capability
does. `make template-init-check` through
`scripts/ci/init-module-contract-check.sh` proves the selector/profile
cross-product. The
initializer reads only that committed lock field and never reconstructs the
package.

Rejected alternatives:

| Alternative | Rejection |
| --- | --- |
| A new Go command or template engine | Adds an executable/dependency and a second filesystem-generation owner for work already owned by the repository shell tooling. |
| Mutate in place and trap `git restore`/`git clean` | Makes rollback path inventories authoritative and risks deleting or retaining a path the failed run did not classify correctly. |
| Generic integration registry, config map, provider interface, or manifest DSL | Contradicts the Specification and makes runtime behavior depend on inferred provider semantics. |
| Provider SDK or custom generator | Neither is selected by the contract; both add dependencies and provider behavior outside v1. |
| Generate callable placeholder operations | A generated call would have to invent a service subset, mapping, budget, error policy, or fake result. |

## Invocation and transaction

The Make target passes the five accepted variables as quoted script arguments.
It inspects GNU Make's command-line variable origins and rejects every
command-line assignment except `NAME`, `TRANSPORT`, `CONTRACT`, `TARGET`, and
`AUTH`; this catches misspelled initializer variables instead of treating them
as ambient configuration. The script accepts no undocumented public flag.

`NAME` is admitted by `^[a-z][a-z0-9_]*$`, absence of the reserved `__`
environment-key delimiter, `go/token.IsIdentifier`, and `!token.IsKeyword`; the
token check is performed by the installed Go toolchain, not by a maintained
keyword copy. HTTP requires the exact path
`api/external/<NAME>/openapi.yaml`. gRPC requires a `.proto` entry point below
`api/proto/external/<NAME>/`. Exact accepted paths make traversal and
repository escape unrepresentable; Git and filesystem checks additionally
require a tracked `HEAD` blob, a regular working-tree file, and no symlink.

The command takes an exclusive initializer lock below `git rev-parse
--git-common-dir` and owns only that lock and its temporary worktree. Its
sequence is:

1. Resolve the repository root; reject a non-clean caller tree, a missing or
   non-regular or incomplete `template.lock`, another initializer, unresolved profile sources
   or markers, a changed `HEAD`, a missing retained choice, an invalid contract,
   a record collision, and an output collision. Classify initial versus refresh.
2. Validate the contract without network. Before any OpenAPI validator or
   generator receives the HTTP document, a YAML-node check rejects every
   `$ref` that is not document-local (`#...`). HTTP then uses the pinned local
   OpenAPI validator. gRPC uses the pinned cached Buf binary and current module
   config. A missing tool/cache fails here.
3. Add a temporary detached worktree at the exact caller `HEAD` and invoke the
   same script's private staging mode with the already validated tuple. The
   private mode is not a public command surface.
4. In initial mode, render only the exact files below, register the generator,
   generate derived Go, format manual Go, and run the focused structural and
   compile admissions. In refresh mode, regenerate derived output only and run
   the same admissions.
5. Stage the temporary worktree only to calculate a binary diff. Reject the
   candidate if its changed paths exceed the mode-specific allowlist or if a
   non-generated path changed during refresh.
6. Cache the patch, remove the temporary worktree, recheck the caller `HEAD` and
   clean status, run `git apply --check`, then one ordinary `git apply`. The
   caller index remains untouched. An empty patch is the required no-op.
7. Remove the private lock and temporary material on every exit. No validation
   runs after apply because the applied bytes are the already validated staged
   candidate.

Git applies the patch as one unit; a failed check or apply leaves caller bytes
unchanged. A concurrent caller edit is outside the clean-start transaction and
is caught by the final status/HEAD recheck. The command never resets, cleans,
commits, stages, or deletes caller-owned work.

## Integration record and mode selection

Each integration owns `integrations/<NAME>.toml` with this fixed schema and
ordering:

```toml
schema = 1
name = "billing"
transport = "http"
contract = "api/external/billing/openapi.yaml"
target = "external-https"
auth = "oauth2-client-credentials"
generator_source = "internal/infra/billing/internal/openapi/doc.go"
```

For gRPC, `target` is absent and `generator_source = "buf.gen.yaml"`. These are
the only two record shapes. The record contains no URL, hostname, scope,
audience, client identifier, secret, token, provider body, or provider error.
The generator source is a stable repository authority locator; exact versions
remain pinned by that source and `tools/go.mod`.

- No record selects initial mode. Every reserved output must be absent, except
  the accepted contract.
- An exact byte-for-byte canonical regular non-symlink record selects refresh
  mode. `schema`, `name`, `transport`, `contract`, applicable `target`, `auth`,
  and `generator_source` are all checked; duplicate, reordered, or extra fields
  are rejected. `name`, `transport`, `contract`,
  applicable `target`, and `auth` are the locked integration identity. A new
  committed contract blob at the same path is allowed.
- A present non-exact record, duplicate name, reserved-path collision, missing
  generator source, or extra record key fails before mutation. V1 never removes,
  renames, retargets, or changes authentication mode.

The record is provenance and collision authority, not a registry. Runtime,
bootstrap, generators, and Make never enumerate it to construct dependencies.

### Ignored `.env` boundary

The initializer never reads, checks, stages, patches, or reports
repository-root `.env`. Named OAuth values come only from generated tracked
examples and runtime `APP__INTEGRATIONS__<NAME>__OAUTH__*` inputs. Legacy
singleton environment keys remain unknown to `internal/config`.

## Generated and manual topology

| Surface | HTTP | gRPC |
| --- | --- | --- |
| Manual contract | `api/external/<NAME>/openapi.yaml` | Accepted entry point below `api/proto/external/<NAME>/` and its repository-owned imports |
| Generator registration | `internal/infra/<NAME>/internal/openapi/doc.go` plus `oapi-codegen.yaml` | Existing `buf.yaml` module plus `buf.gen.yaml`; the record/source parity check explicitly admits the new subtree |
| Generated-only Go | `internal/infra/<NAME>/internal/openapi/client.gen.go` | Generator-determined files below `internal/gen/proto/external/<NAME>/` |
| Manual adapter | `internal/infra/<NAME>/doc.go` and `client.go` | Same |
| Typed config | `internal/config/<NAME>_integration_config.go` under `integrations.<NAME>` | Same |
| Composition | `cmd/service/internal/bootstrap/startup_<NAME>.go` and one explicit construction/close edge in `run.go` | Same |
| Documentation | `docs/integrations/<NAME>.md` | Same |
| Identity | `integrations/<NAME>.toml` | Same |

HTTP's generated subpackage is Go-internal to the concrete adapter. It uses
oapi-codegen `models,client` only; it generates no server, embedded contract,
business port, or operation wrapper. The adapter is the only package that can
hold the generated client. It constructs one fixed-authority bounded client,
optionally binds one OAuth owner, and passes that Doer and its validated base
URL to the generated client. The exported concrete adapter has private fields
and no operation method after initialization.

gRPC remains in the existing Buf module and output root. A generated gRPC
contract may contain several services, so the initializer must not select or
construct a service stub. The concrete adapter owns the validated bounded
`grpc.ClientConnInterface`; a later manual operation in that same package may
construct exactly the generated service client it needs. A generated depguard
rule denies imports of `internal/gen/proto/external/<NAME>` outside that adapter
and the generated subtree. This preserves adapter ownership without inventing a
service subset.

`Makefile` discovers only the fixed external OpenAPI contract and generator
package shapes for canonical `openapi-generate`, validation, and drift. The
Make's `openapi-drift-check` snapshots every discovered external generated file
before `go generate`. The root `proto-generate` and `proto-drift-check` targets
generate and snapshot the complete `api/proto` module. The separate
`scripts/ci/integration-record-check.sh` binds each exact record to its contract,
generated output, adapter, config, bootstrap, and documentation. The
changed-surface owner routes initializer, record, contract, generator, and
generated paths into those canonical checks.

Generated Go is never formatted or edited as manual source. Manual adapter,
config, bootstrap, documentation, record, generator config, and proof files
are written only in initial mode. A repeat may change only the declared
generated-only Go paths.

## Configuration and trust construction

The root immutable snapshot gains one concrete `IntegrationsConfig` struct, not
a map. Each invocation adds one field tagged with the exact `NAME`; the Go field
spelling uppercases only the first ASCII byte and otherwise preserves the
accepted identifier. A narrow generated lint suppression is allowed only when
an underscore in an otherwise valid package identifier triggers the repository
naming linter. Each integration's type, empty defaults, normalization, and pure
validation stay together in `<NAME>_integration_config.go`; shared root files
only expose the aggregate field, merge its defaults, and call its validator.

The generated runtime keys are exactly:

| Selection | Keys |
| --- | --- |
| HTTP, `external-https` | `integrations.<NAME>.base_url` |
| HTTP, `private-https` | `integrations.<NAME>.base_url`, `integrations.<NAME>.private_dns_suffix` |
| gRPC | `integrations.<NAME>.target` |
| OAuth | nested `oauth.token_url`, `oauth.client_id`, `oauth.client_secret`, and `oauth.scopes` below the integration section |

`AUTH=none` emits no OAuth field, default, environment example, import, owner,
or close edge. For OAuth, `client_secret` is an empty
environment-only example;
the existing secret-source policy rejects a non-empty YAML/file value. Other
fields have empty defaults and startup fails closed until deployment supplies a
complete tuple. The initializer records no runtime value and does not infer a
base URL, target, token URL, scope, or provider-specific token parameter.

HTTP config validation admits one absolute HTTPS target and applies the fixed
selected public/private class. `private-https` requires the non-empty DNS suffix
and `external-https` has no suffix field. The adapter revalidates through
`httpclient.NewExternalHTTPS` or `NewPrivateHTTPS`.

gRPC config validation trims the required value. The generated adapter is the
single transport-policy owner and admits exactly `dns:///hostname:443`: no
authority, user info, IP literal, encoded or extra path, query, fragment, invalid
ASCII DNS label, or other resolver. It returns the canonical target and hostname,
builds system-root TLS with that `ServerName`, and calls
`grpcclient.New`, which retains resolution, reconnect, `pick_first`, transparent
transport retry, service-config denial, and shared message/header bounds.

OAuth config maps to one `oauth2clientcredentials.Client` owned by this
adapter. Its token authority is external HTTPS only. Feature code and generated
code receive only the authenticated HTTP Doer or authenticated gRPC connection;
they never receive a token source or token.

`OUTBOUND_AUTH` is only the `template.lock` choice that retains or removes
`internal/infra/oauth2clientcredentials`. It creates no root config field,
default, validator call, environment example, or bootstrap fixture. Every OAuth
invocation adds a disjoint named section and never shares credentials.

## Bootstrap and lifecycle

`startup_<NAME>.go` is the sole config-to-adapter mapper. Construction performs
only validation and local client/connection creation. It performs no DNS
lookup, token acquisition, resource RPC, health call, listener mutation, or
background start.

`run.go` constructs the concrete adapter after the immutable snapshot and
telemetry exist and before handlers are composed. It immediately registers a
partial-startup safety close. The adapter value is a local composition-root
owner available for an explicit later feature injection; it is not placed in a
registry or exported from bootstrap. With no provider operation, the initial
scaffold injects it nowhere and cannot be called accidentally.

The ordered terminal sequence is:

```text
readiness off -> HTTP/gRPC server drain -> supervised background join
  -> each concrete integration close in reverse construction order
  -> existing dependency close -> telemetry flush
```

The adapter close is idempotent. OAuth retirement runs first so new credential
acquisition is rejected and the provider context is canceled. Then the gRPC
connection or HTTP idle connections close. The deferred safety path uses the
same owner and order
for any post-construction startup failure. No generated integration contributes
a readiness probe, liveness state, or generic provider health signal.

## Material flows and failure finality

### Initial creation

```text
developer + committed contract + template.lock
  -> Make/script initial-or-refresh classification
  -> preflight (caller remains clean)
  -> detached stage at exact HEAD
  -> manual scaffold + registered pinned generation
  -> local validation and changed-path containment
  -> one patch apply
  -> caller-visible working-tree delta or unchanged caller on failure
```

Git `HEAD` is the start-state truth, the supplied contract is the contract
truth, and the integration record is the identity truth after success. No
provider participates. Failure before or during staging deletes only the
temporary worktree. A failed final apply is non-partial. Success finality is the
presence of the exact local delta; commit ownership remains with the developer.

### Same-identity refresh

```text
exact record + same path + committed contract
  -> regenerate registered output in detached stage
  -> reject any non-generated delta
  -> apply generated-only patch, or no-op when bytes match
```

Manual edits are neither inputs nor outputs of refresh. A generator that needs
to change a manual seam is an explicit migration and stops v1.

### Runtime construction and later HTTP use

```text
typed immutable config -> concrete adapter New (no I/O)
  -> future manual provider method
  -> generated OpenAPI client
  -> optional retained OAuth Doer
  -> retained fixed-authority HTTP client
  -> configured provider authority
```

The later provider method owns its finite context, operation mapping, response
bound and parse, error mapping, and retry eligibility. Target denial, OAuth
failure, cancellation/deadline, transport unavailability, and provider business
rejection remain distinct. The scaffold creates none of those caller-visible
translations.

### Runtime construction and later gRPC use

```text
typed immutable target -> system-root TLS + bounded ClientConn
  -> future adapter method selects one generated service client
  -> optional retained PerRPCCredentials
  -> grpc-go DNS/reconnect/pick_first/transparent transport behavior
  -> configured provider authority
```

The later method owns the RPC name, request/response mapping, deadline, status
mapping, streaming lifecycle, and any application retry. The initializer owns
no RPC retry or resolver service config.

### Failure dispositions

| Failure | Owner and finality |
| --- | --- |
| Invalid variable, profile, path, symlink, contract, collision, record, or dirty tree | Parent preflight; definitive rejection before caller mutation. |
| Missing cached pinned tool | Parent preflight; definitive local prerequisite failure, no network fallback. |
| Render, generation, format, compile, structure, or leak check failure | Temporary worktree; caller bytes remain at `HEAD`. |
| Caller changes during staging | Final identity/status recheck; candidate discarded. |
| Final patch does not apply | `git apply`; caller remains unchanged. |
| Runtime config invalid | Existing config bootstrap; readiness is never published and adapter construction does no I/O. |
| Partial startup after adapter construction | Immediate deferred close; bounded, sanitized terminal result. |
| Normal shutdown | Post-drain integration close before base dependencies and telemetry. |

## Proof ownership without Test Design

This phase selects proving surfaces only; it does not define the later scenario
matrix or enter Test Design.

| Specification claim | Selected proving surface |
| --- | --- |
| Command grammar, capability/path/reference admission, clean transaction, collisions, identity, no-op/refresh, failure restoration, and unrelated-byte preservation | `scripts/ci/integration-init-check.sh` over fresh committed initialized fixtures; the harness owns fixture-byte comparison, a metadata-style remote-reference falsifier, and distinct non-disclosure canaries |
| HTTP source/generator/output drift | per-integration `go:generate` plus Make `openapi-generate` and `openapi-drift-check` |
| gRPC source/generator/output drift | current Buf module and `buf.gen.yaml` through Make `proto-generate` and `proto-drift-check` |
| Typed config, target denial, OAuth absence/presence, constructor no-I/O, and cleanup order | package-local config/adapter/bootstrap proof beside each owner |
| Generated binding and construction split proof | `scripts/ci/integration-init-check.sh` proves no-I/O concrete-adapter construction, then emits fixture-only `internal/infra/<NAME>/client_contract_test.go` for one local generated-binding exchange; retained transport/auth package tests remain the independent bounded-denial and credential-failure owners because the scaffold exposes no operation to drive end to end |
| Generated code containment and import direction | Go `internal`, generated depguard rule for gRPC, `integration-record-check`, and package compile |
| Secret and raw-value non-disclosure | existing config secret-source policy and repository secret-scan owner plus focused generated-output and command-output sinks |
| Aggregate routing after dynamic paths exist | `changed-surfaces-check` and the canonical record/OpenAPI/proto/format/compile aggregates |

The later Test Design owner must choose the smallest deterministic HTTP/gRPC
fixture matrix and falsifiers for the eight locked proof expectations. This
design does not treat a green aggregate, generated-file presence, or unit stub
as proof of provider semantics or live compatibility.

## Ownership Map V1 — responsibilities

| responsibility | affected path | current evidence | semantic owner | exact package/file action | dependency/composition/generated boundary | cleanup | proof owner | reopen condition |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Make command and accepted variable surface | `Makefile` | `template-init` is already a thin Make-to-script target | repository build command | add `integration-init` and `integration-init-check`; reject extra command-line assignments | no runtime import or new executable | none | command harness | reopen Specification if another public invocation shape is required |
| Initial/refresh transaction | `scripts/integration-init.sh` | Git and Bash already own template mutation | initializer | one parent/stage script with detached-worktree patch application | may invoke only current pinned repository tools | owns lock/worktree removal | command harness | reopen System Design if Git clean-start staging cannot cover an adopter |
| OpenAPI reference admission | `scripts/openapi-ref-check.go` | pinned tools otherwise allow external URI references | initializer | parse YAML nodes and admit only document-local `#...` references before validation/generation | reuses `go.yaml.in/yaml/v3` already pinned by `tools/go.mod`; no network-capable loader | none | E3 contract row with local control and remote mutant | reopen Specification to support repository-local multi-file contracts |
| Integration identity | `integrations/<NAME>.toml` | `template.lock` uses fixed line-oriented TOML-like records | named integration | add once; exact parse on repeat | not consumed at runtime or used as registry | no removal in v1 | command harness/project structure | reopen Specification for rename/remove/reconfigure |
| External contract authority | accepted `api/external` or `api/proto/external` path | current OpenAPI and Buf contracts are canonical | service developer | never rewrite; register only | contract -> generated output -> adapter | none | canonical generator drift | reopen Specification for SDK/custom generator/dynamic source |
| HTTP generation | adapter-internal OpenAPI package and canonical scripts | current service OpenAPI uses `go:generate` + pinned tool | named adapter plus generator | add client-only config/directive/output; extend discovery/drift | generated package is Go-internal to adapter | none | OpenAPI drift/compile | reopen ownership if generated client cannot stay internal |
| gRPC generation | existing Buf module and `internal/gen/proto/external/<NAME>` | current Buf config already owns recursive module output | Buf/Protobuf generator | keep source-relative output; add explicit record/source/output parity | generated imports denied outside named adapter/generated subtree | none | proto drift/compile/depguard | reopen ownership if contract layout defeats adapter containment |
| Per-integration runtime config | `internal/config` | tagged structs own decode/known keys; section file owns defaults/validation | named integration config | add one concrete aggregate field and one per-name config file | no map/registry; bootstrap is first runtime consumer | immutable snapshot | config package proof | reopen Specification for dynamic integration sets or reload |
| Retained outbound HTTP selection | `scripts/init-module.sh` and `template.lock` | authoritative `HEAD` lacks the field; current dirty selector is evidence, not input | template initialization | add exact `none|bounded` choice, lock write/readback, dependency-aware httpclient retain/remove | only `template-init` selects physical capability; integration initializer reads and fails closed | none | `make template-init-check` / `scripts/ci/init-module-contract-check.sh` cross-product | reopen Specification if another selector value or reconstruction is required |
| OAuth config cardinality | each OAuth invocation adds only `integrations.<NAME>.oauth.*` | retained capability has no runtime tuple | named adapter for OAuth; retained package for later invocation | retain package through `template.lock` and generate one named tuple | no shared credential owner and no root compatibility path | adapter closes its owner | profile/config/adapter proof | reopen Specification for a dynamic integration set |
| Concrete adapter | `internal/infra/<NAME>` | repository architecture places providers under `internal/infra` | named integration | add `doc.go` and `client.go`; no initial operation | only adapter imports generated client/connection and retained transport/auth | owns transport/auth close | adapter package proof | reopen Specification for provider SDK, stream lifecycle, or callable generated behavior |
| Composition and partial startup | `cmd/service/internal/bootstrap` | `run.go` explicitly constructs and closes dependencies | service bootstrap | add `startup_<NAME>.go` and explicit `run.go` edges | feature receives adapter only through later explicit injection | reverse-order close post-drain/pre-dependencies | bootstrap lifecycle proof | reopen System Design if another binary adopts the integration |
| Integration documentation | `docs/integrations/<NAME>.md` | provider inputs and owners are documented beside current integrations | named integration | add once; refresh never overwrites | no secrets/live values/provider claims | documents lifecycle only | command harness/secret scan | reopen Specification for provider-specific setup automation |
| Canonical drift and aggregate routing | Make/CI scripts | current OpenAPI/proto checks own generation drift | repository validation | discover fixed paths and route their changes | no second generator or alternate aggregate | temp snapshots removed | existing check owners | reopen Technical Design if current aggregate cannot express dynamic fixed paths |

The selected reuse rung for every new mechanism is existing repository tooling:
Git worktrees and patches for the transaction, current Bash/Make for command
composition, current config structs, current bootstrap, current transport/auth
packages, and pinned generators. The strongest viable rejected source is a new
Go/template framework; parity is proved by the command harness and canonical
generation checks. Upgrade only when a real adopter cannot satisfy the clean
Git boundary or a pinned generator cannot represent its accepted contract.

## Ownership Map V1 — Go files

The retained OAuth profile adds no root runtime config. `internal/config` keeps
the reusable named tuple type and rejects legacy singleton keys. The Go
placement, dependency, generated/manual, lifecycle, and proof decisions follow.

| path | responsibilities | one present reason to exist | declarations/visibility | call-path role | lifecycle/error ownership | allowed dependencies | forbidden responsibilities |
| --- | --- | --- | --- | --- | --- | --- | --- |
| change `internal/config/types.go` | root `Integrations` field only after a named invocation | root immutable shape exposes concrete integration config, never a placeholder singleton | exported root field and concrete tagged aggregate only | config decode -> bootstrap | none; config errors stay section-owned | config sibling types/stdlib | adapter imports, provider values, maps, validation |
| change `internal/config/defaults.go` | merge each concrete integration default | one canonical defaults composition already lives here | calls only | defaults -> snapshot | none | `maps`, sibling functions | integration validation or values |
| change `internal/config/validate.go` | call each concrete integration validator | one validation order owner already exists | calls only | snapshot -> startup admission | return first sanitized config error | sibling functions | transport construction |
| retain `internal/config/outbound_auth_config.go` | reusable four-field named OAuth tuple and validation | every named OAuth integration shares the same standard input grammar | exported tuple, private validator | named config -> startup admission | sanitized key-only errors | stdlib | root field/defaults, provider extensions, token behavior |
| retain `internal/config/outbound_auth_config_test.go` | standard tuple validation and legacy-key rejection | shared config owner needs local proof | test only | config proof | no production lifecycle | config test support | provider compatibility |
| change `internal/config/snapshot_contract_test.go` | add concrete named leaf sentinels | current reflection contract requires every generated field | test-only maps | config proof | no production lifecycle | config/test deps | provider operation semantics |
| change `internal/config/testhelpers_test.go` and `internal/config/configtest/configtest.go` | install exact named required tuple | generated config makes those values required | test-private setup | config tests -> load | none | testing/stdlib | provider fake |
| add/change `internal/config/integrations_config.go` | concrete aggregate fields only | nested `integrations.<NAME>` needs one tagged struct without a map | `IntegrationsConfig`; exact per-name fields | decode shape | none | no imports | registry iteration, defaults, validation |
| add `internal/config/<NAME>_integration_config.go` | exact transport/auth fields, empty defaults, required target presence, pure validation through the shared OAuth tuple | one named config owner | unexported section type/functions; exported leaf fields only as bootstrap needs | config decode/validate -> bootstrap | closed key-only errors | stdlib | gRPC resolver policy, I/O, generator, adapter construction, business policy |
| add `internal/config/<NAME>_integration_config_test.go` | routine decode/validation and secret-source parity | production config file needs local proof | test only | config proof | none | config test support | broad initializer transaction matrix |
| change `cmd/service/internal/bootstrap/run.go` | explicit construction, later injection point, partial and ordered close | composition root is the only valid runtime owner | local concrete variable/close guard; no registry | config -> adapter -> future feature; drain -> close | closes once after background join | config, named adapter, existing bootstrap owners | provider operation, target parsing, generic integration collection |
| add `cmd/service/internal/bootstrap/startup_<NAME>.go` | config-to-adapter mapping | keeps generated wiring out of the already broad lifecycle function | private `init<NAME>` | run -> adapter New | wrap closed construction error only | config, named adapter, telemetry collaborators only if selected owner needs them | readiness, retries, feature semantics |
| add `cmd/service/internal/bootstrap/startup_<NAME>_test.go` | routine no-I/O construction mapping and close parity | exact mapping has one package owner | test-private fakes only at consumer seam | bootstrap proof | partial-close observables | bootstrap/test deps | provider emulation or Test Design matrix |
| conditionally change `cmd/service/internal/bootstrap/run_test.go` | add exact named integration keys | shutdown/lifecycle tests need one valid required config tuple after generation | test-private setup only | bootstrap test setup | none | stdlib/testing | adapter or provider behavior |
| add `internal/infra/<NAME>/doc.go` | package audience, authority, absent seams | concrete adapter has generated/manual and lifecycle audiences | package documentation only | none | documents Close owner | none | behavior, generator directive |
| add `internal/infra/<NAME>/client.go` | transport-specific Config, concrete Client, New, idempotent close; private generated client/connection; exact gRPC resolver/TLS target admission | one concrete adapter is required now | exported concrete `Config`, `Client`, `New`, transport/auth-specific `Close`; no operation method | bootstrap -> adapter -> retained clients | validates the dial boundary, owns auth/transport close and sanitized construction errors | stdlib, retained httpclient or grpcclient, optional oauth, generated package only as allowed | feature/domain imports, retry, readiness, provider mapping, token API |
| add `internal/infra/<NAME>/client_test.go` | routine constructor, target, no-I/O, no-auth-path and idempotent close proof | production adapter needs local contract proof | test only | adapter proof | cleanup observable | adapter/test deps | live provider or generated business semantics |
| fixture-only `internal/infra/<NAME>/client_contract_test.go` created by `scripts/ci/integration-init-check.sh` | no-I/O adapter construction plus one separate local generated-binding exchange over the harness contract | Specification proof 6 needs honest split proof without inventing an adapter operation | temporary test only; never an initializer output or exported seam | concrete adapter construction; separately generated fixture client -> local fake | owns and joins only its fixture listeners/connections | adapter package, generated binding, stdlib/current gRPC test support | adopter/provider semantics, live endpoint, persisted scaffold file |
| add `internal/infra/<NAME>/internal/openapi/doc.go` (HTTP) | exact `go:generate` authority | external HTTP client needs a canonical package generator | directive/package doc only | contract -> generator | none | repository script | manual adapter behavior |
| add generated `internal/infra/<NAME>/internal/openapi/client.gen.go` (HTTP) | client/models derived from exact contract | pinned oapi-codegen output | generated declarations only | adapter private dependency | generated errors remain inside adapter | generator-selected runtime deps already installed | manual edits or business mapping |
| add generator-determined `internal/gen/proto/external/<NAME>/**/*.pb.go` and `*_grpc.pb.go` (gRPC) | messages and service client interfaces derived from exact contracts | current Buf source-relative output | generated declarations only | named adapter private dependency when a later method selects a service | grpc-go generated semantics only | protobuf/grpc generated deps | adapter, config, lifecycle, manual edits |

An implementation-local contract may determine the exact gRPC generated file
set; the deterministic placement rule is every Buf-created Go file whose source
is below `api/proto/external/<NAME>` must remain below
`internal/gen/proto/external/<NAME>` and in the generated-only refresh allowlist.
No other Go path is implementation-local.

## Non-Go file map and cleanup

| action | owner and present reason |
| --- | --- |
| change `Makefile` | public target, help, fixed external OpenAPI discovery, and focused check target |
| add `scripts/integration-init.sh` | sole initial/refresh transaction and render owner; `.env` is not an input or precondition |
| add `scripts/ci/integration-init-check.sh` | deterministic command/rollback/identity, non-disclosure and byte-comparison owner, and sole generator/runner of the temporary package-local successful/sanitized-failure fake; exact oracle selection remains Test Design |
| change `Makefile` OpenAPI/protobuf targets | discover fixed external OpenAPI packages and keep root Buf generation/drift canonical |
| add `scripts/ci/integration-record-check.sh` | enforce exact record/contract/config/adapter/bootstrap/docs/generated parity and reject orphaned owners |
| change `scripts/ci/changed-surfaces.sh` | route initializer and every record-bound source/output/owner delta to its gates |
| change `scripts/ci/init-module-contract-check.sh` through `make template-init-check` | prove initializer retention and the `OUTBOUND_HTTP`/`GRPC`/`OUTBOUND_AUTH` prerequisite shapes in initialized fixtures |
| change `scripts/init-module.sh` | production owner for exact `OUTBOUND_HTTP=none|bounded`, `template.lock` persistence/readback, and dependency-aware retention/removal of `internal/infra/httpclient`; it does not create integration config or reconstruct a removed profile |
| change `.golangci.yml` | template adapter-ownership rule plus each generated gRPC integration's exact adapter-only import denial |
| conditionally change `env/config/local.yaml` and `env/.env.example` | every invocation adds only its exact empty tracked named integration inputs; `AUTH=none` adds no OAuth fields |
| change `README.md` and add `docs/external-integration-initializer.md` | durable command contract and boundary after task artifacts eventually leave |
| change `docs/outbound-machine-authentication.md` | document retained package plus per-integration ownership; no concrete provider values |
| change `docs/repo-architecture.md`, `docs/architecture/boundaries.md`, `docs/architecture/integration.md`, `docs/project-structure-and-module-organization.md`, and `docs/configuration-source-policy.md` | canonical source, adapter, generated/manual, config, secret, and lifecycle ownership |
| add `integrations/<NAME>.toml` | one fixed integration identity/provenance record |
| add `internal/infra/<NAME>/internal/openapi/oapi-codegen.yaml` (HTTP) | client-only generator settings beside their generated package |
| add `docs/integrations/<NAME>.md` | contract authority, exact runtime keys, trust/auth choice, lifecycle, and remaining manual provider work |

No `go.mod`, `go.sum`, binary, migration, database, deployment, provider,
credential, network-policy, readiness, liveness, business feature, or task/test
plan file is added by this design.

## Reopen conditions

Reopen Specification and stop when an adopter requires a dirty-worktree merge,
automatic removal/rename/reconfigure, initializer custody of `.env`, a provider SDK, custom
generator, dynamic or caller-controlled target, generated callable placeholder,
service/subset inference, streaming-specific lifecycle, unsupported
authentication, or root compatibility configuration.

Reopen Security when a private token authority, proxy, custom roots, plaintext,
mTLS, workload identity, API key, provider signing, new secret source, or new
disclosure sink is required. Reopen System / Integration Design when another
binary, runtime registry, readiness policy, discovery mechanism, background
refresh, rollout mechanism, or new boundary is required. Reopen Go Ownership
when generator evidence cannot preserve the exact manual/generated containment
or bootstrap-only lifecycle. Ordinary private helpers, shell syntax, generated
declaration names, and behaviorally equivalent Go statements remain
Implementation choices.
