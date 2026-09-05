# Provider Authentication

Load when grant, OAuth/token lifecycle, scope, credential selection, refresh, or
rotation can change the boundary.

Pin actor, grant, authorization/token endpoints, resource audience, scopes,
token type, expiry, revocation/rotation, clock assumptions, environment, and
provider-supported recovery. Prefer discovery metadata and the official SDK
when they implement the required contract. Bind cached credentials to provider,
environment, tenant/subject, audience, scopes, and client identity. Keep
sandbox/production, redirect URIs, caches, and webhook secrets separate.

Authorization code uses state/nonce, exact redirect binding, and PKCE `S256`;
machine work uses the supported non-user grant. Keep credentials in the existing
secret facility, send them only to the pinned TLS origin, and redact headers,
URLs, bodies, traces, fixtures, and errors.

Refresh before expiry with skew/jitter and one per-binding single flight.
Publish the access token only after the matching refresh generation is durable;
rotating refresh needs compare-and-set or a narrow lock so stale success cannot
overwrite new state. Refresh and replay an eligible `401` at most once and only
when request identity makes replay safe. Repeated `401`, wrong scope/audience,
revocation, disabled account, or `invalid_grant` becomes an owned auth failure,
not a loop.

Proof covers concurrent one-refresh behavior, stale-generation rejection,
eligible one-replay `401`, permanent failures, cancellation/deadline including
refresh wait, redaction, and old/new generation overlap.
