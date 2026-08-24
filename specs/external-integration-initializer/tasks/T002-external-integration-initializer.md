# T002 — Complete External Integration Initializer

Outcome:
The source template implements initial, no-op, and generated-only refresh for
named HTTP/gRPC integrations. OAuth retains only the reusable package until a
named integration exists; `x/oauth2` owns token cache, expiry skew, and refresh
synchronization.

Consumes:
- T001 retained outbound HTTP choice.
- Current `../spec.md`, `../design/overview.md`, and `../test-plan.md`.
- Local pinned OpenAPI, Buf, Go, formatting, and shell tools.

Provides:
- Named config, concrete adapters, bootstrap construction/close, canonical
  generation/drift, documentation, and focused proof.
- No root OAuth runtime tuple, singleton migration, provider extension fields,
  provider operation, dependency addition, live credential, network action, or
  `.env` custody.

Mutable owners:
- `internal/infra/oauth2clientcredentials` and its package tests.
- Named integration config/adapter/bootstrap renderers.
- Initializer transaction, OpenAPI reference admission, exact record-parity
  checkers, and 23-row disposable harness.
- Make/CI routing, integration documentation, and current task artifacts.

Accept when:
- `ALLOW_HEAVY=1 make integration-init-check` reports all 23 current IDs.
- The claim-matched Validation ladder in `../test-plan.md` is terminal-success.
  An unavailable required gate records one explicit `Blocked:` receipt with its
  tool or external owner; it does not accept the unit.
- Focused Go proof confirms safe token projection, library cache/refresh,
  acquisition timeout, retirement, fixed egress, secret redaction, and
  competing-auth denial.
- Integrated review finds no surviving correctness, security, lifecycle,
  ownership, or avoidable-complexity finding.

Candidate:
Base `94dc45411c99413739a75a435aa37b25befeba77` plus ordered bounded
implementation/spec/test manifest SHA-256
`5c8da93a73ade008a6f5fb1c6f40867042f8013d99db19da0093a099e004b41f`.
Receipt and ledger files are excluded from that self-independent manifest.

Evidence Result V1:

| claim | scope | command | environment | duration | result | status | gap_or_next_owner |
| --- | --- | --- | --- | --- | --- | --- | --- |
| all 23 production initializer obligations | disposable initialized fixtures | `ALLOW_HEAVY=1 make integration-init-check` | Darwin arm64, Go 1.27 local, pinned tools, no provider | 8m00s | pass, `case count: 23` | verified | none |
| repaired E3/E5 proof oracles | precondition, contract, and Protobuf fixtures | `make integration-init-check INTEGRATION_INIT_ROWS='row_e3_precondition row_e3_contract row_e5_proto'` | same production bytes; current proof harness | 1m15s | pass, `case count: 3` | verified | unaffected row receipts reused under fresh Test Design review |
| retained config/transport/auth/bootstrap packages | five focused Go packages | `go test -vet=off ./internal/config ./internal/infra/httpclient ./internal/infra/grpcclient ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap` | Darwin arm64, Go 1.27 | 6.4s | pass, 416 tests | verified | none |
| canonical HTTP/gRPC generation, validation, runtime and drift | OpenAPI and Protobuf owners | `make openapi-check proto-check` | pinned local Go tools; cached Redocly invocation | 13.1s | pass | verified | none |
| record parity and changed-surface routing | record/routing owners | `make integration-record-check changed-surfaces-check` | clean source template without generated integration records | 0.1s | pass; record profile not applicable, routing self-test pass | verified | none |
| template adoption contract | disposable profile matrix | `ALLOW_HEAVY=1 make template-init-check` | local temporary initialized modules | 5m16s | pass, `initializer contract passed` | verified | none |
| shell ownership | three changed shell owners | `make shellcheck SHELL_FILES='scripts/integration-init.sh scripts/ci/integration-init-check.sh scripts/ci/integration-record-check.sh'` | ShellCheck 0.11.0 container, read-only mount, network none | 4.3s | pass | verified | none |
| secret non-disclosure | repository worktree and diff | `make secret-scan` | local pinned gitleaks | 2.2s | pass, 0 leaks | verified | none |

Supplemental exact-candidate gates: `make actionlint-fast`, direct
document-local OpenAPI reference control, and `git diff --check` passed.

Acceptance Result V1:

unit: T002
verdict: Accepted
candidate: base and bounded manifest above
evidence: every Test Plan ladder claim is verified in the table above
review: `../implementation-review.md` SHA-256 `4fcf36be376d47fb3f7384583e699e8e75acfe799846e302f0680c7f389fe6e9` — PASS; current Specification, Technical Design, and Test Design reviews also PASS
provides: fail-closed deterministic HTTP/gRPC integration initialization with exact record, named config, bounded transport/auth, generated containment, and local proof

Provider compatibility, credentials, live network, deployment, and rollout
remain outside this acceptance.

Reopen if:
Reopen Specification for dynamic integrations, root compatibility config, or
initializer `.env` custody. Reopen Technical Design for another transaction,
generator, target, or lifecycle mechanism. Reopen Go Ownership when
generated/manual containment or bootstrap-only lifecycle cannot hold.
