# Security, Network, Service Auth, JWKS, And Proxy Contract

Status: targeted full-rollout research complete
Date: 2026-06-02

## Questions

What service-auth/JWKS and private-network proof must specification reopen
decide before billing-service can be enabled for full production authority?

## Billing-Service Auth Findings

Billing-service requires `service_auth.issuer`, `service_auth.audience`, and
`service_auth.jwks_url` when service auth is enabled
(`internal/config/validate.go:226`). The runtime verifier accepts only Bearer
JWTs signed with RS256, requires a non-empty `kid`, fetches keys from the
configured JWKS URL, and verifies issuer and audience
(`internal/infra/http/service_auth.go:83`).

OpenAPI route scopes include:

- `billing.accounts.resolve`;
- `billing.balances.read`;
- `billing.usage.read`;
- `billing.usage.write`;
- `billing.microleases.read`;
- `billing.microleases.write`;
- `billing.operations.read`;
- `billing.reconciliation.read`;
- `billing.admin.read`.

The protected internal route scope mappings are in
`api/openapi/service.yaml:112` through `api/openapi/service.yaml:647`.

## Gonka-Proxy Contract Findings

The sibling `gonka-proxy` checkout is not a clean provider contract. Relevant
billing-service files are modified or untracked, including
`src/config/env/schema/billing.ts`, `src/plugins/billing.ts`,
`src/services/billing/shared-balance-live.ts`, and untracked
`src/services/billing/service-auth-signer.ts`.

The draft proxy signer creates RS256 JWTs with audience `billing-service`,
subject `svc:gonka-proxy`, TTL 60 seconds, max TTL 120 seconds, `scope`, and
`kid` (`src/services/billing/service-auth-signer.ts:7`). The proxy env schema
adds issuer, audience, subject, private key PEM, key id, and TTL fields
(`src/config/env/schema/billing.ts:19`), but `.env.example` still documents only
the older `BILLING_SERVICE_AUTH_KEY` for the billing-service section
(`.env.example:247`).

The proxy plugin requires issuer, private key, and key id when shared-balance
cutover is enabled (`src/plugins/billing.ts:85`). Current scoped billing calls
use only `billing.usage.write` and `billing.operations.read`
(`src/services/billing/shared-balance-live.ts:76`), and the live shared-balance
client still retains a legacy unscoped operator-adjustment path and legacy
auth-key fallback (`src/services/billing/shared-balance-live.ts:572`,
`src/services/billing/shared-balance-live.ts:684`).

Focused searches found no committed JWKS publication route for billing-service
verification and no billing microlease event producer in proxy. Existing proxy
drafts therefore do not yet answer JWKS custody, public-key publication, key
rotation, or terminal/checkpoint/close event production.

## Private Networking And Metrics Findings

Railway private networking docs state that services in the same project and
environment can reach each other by `SERVICE_NAME.railway.internal`, without
public exposure. Live read-only inventory shows the `billing-service` app
service exists in the same production project as `gonka-proxy` and has no
separate billing public-domain requirement recorded in service config.

Billing-service bootstrap treats `NETWORK_*` as a separate operator-owned
network policy channel, not normal app config
(`docs/configuration-source-policy.md:19`). Missing
`NETWORK_PUBLIC_INGRESS_ENABLED` is not equivalent to `false` in non-local
wildcard-bind deployments (`docs/configuration-source-policy.md:23`).

If public ingress is enabled, startup rejects operational metrics exposure
because `/metrics` is not yet private or protected
(`cmd/service/internal/bootstrap/network_policy_enforcement.go:23`). Internal
or private hosts, including `.internal`, are treated as non-public egress
targets (`cmd/service/internal/bootstrap/network_policy_enforcement.go:153`).

## Evidence Limits

Raw private keys, JWKS contents, bearer tokens, request bodies, Railway
variables, and dynamic proof URLs were intentionally not read or printed.

## Handoff Implications

Specification reopen must decide:

- JWKS publication owner and endpoint, including whether it is served by
  billing-service, proxy, a deployment-managed static source, or another
  approved internal service;
- key rotation order, overlap window, and proof without exposing key material;
- exact proxy scopes needed for microlease issue/read/close, usage write/read,
  operations read, and any admin/reconciliation use;
- whether legacy `BILLING_SERVICE_AUTH_KEY` and legacy operator-adjustment path
  are excluded from the full production target;
- private URL shape for proxy-to-billing calls;
- secret-free proof that billing-service has no public metrics exposure and
  that private service-to-service reachability works without a public `/metrics`
  probe.
