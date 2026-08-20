# Integration Boundaries

Load when a system neighbor, outbound dependency, provider contract, or
cross-service evidence path changes.

| Neighbor | Role on path | Canonical contract | Checkout or clone | Runtime evidence | Owner |
| --- | --- | --- | --- | --- | --- |
| _(none in the template)_ | caller, provider, broker, job, or managed dependency | repository, generated contract, published spec, or live endpoint | path or clone URL | query/dashboard/command plus correlation field | accountable team or person |

Record a neighbor when this service calls it, is called by it, or shares durable
state with it. Point to the real contract and the concrete runtime evidence path
joined by request ID or W3C trace context. Store access shape, never credentials,
tokens, or customer data.

<!-- profile:outbound-auth-oauth2-client-credentials:start -->
`internal/infra/oauth2clientcredentials` owns one process-local OAuth
client-credentials boundary for one fixed dependency. Feature code receives an
authenticated bounded client, never a token or credential source. Configuration
is immutable and the secret remains environment-only.
<!-- profile:outbound-auth-oauth2-client-credentials:end -->

Provider adapters live under `internal/infra/<integration>` and start with
`net/http`; reuse the bounded client for fixed-authority transport safety when
enabled. The provider adapter owns authentication, budgets, retry eligibility,
provider errors, and generated clients. Bootstrap owns wiring and cleanup. A
dynamic or caller-controlled URL requires a separate security decision.

Before enabling a runtime dependency, close its contract source, trust and
egress policy, criticality, timeout/retry budget, readiness participation,
partial-startup cleanup, telemetry labels, and bootstrap proof. New executable
surfaces use their own `cmd/<binary>` composition root.
