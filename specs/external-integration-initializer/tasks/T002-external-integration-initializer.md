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
- Initializer transaction and 23-row disposable harness.
- OAuth docs and current task artifacts.

Accept when:
- `make integration-init-check` reports all 23 current IDs.
- The eight-command Validation ladder in `../test-plan.md` is terminal-success
  or every unavailable aggregate is returned as an explicit evidence gap.
- Focused Go proof confirms safe token projection, library cache/refresh,
  acquisition timeout, retirement, fixed egress, secret redaction, and
  competing-auth denial.
- Integrated review finds no surviving correctness, security, lifecycle,
  ownership, or avoidable-complexity finding.

Reopen if:
Reopen Specification for dynamic integrations, root compatibility config, or
initializer `.env` custody. Reopen Technical Design for another transaction,
generator, target, or lifecycle mechanism. Reopen Go Ownership when
generated/manual containment or bootstrap-only lifecycle cannot hold.
