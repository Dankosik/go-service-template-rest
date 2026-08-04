# Authentication and credential lifecycle

Use this reference only when the provider boundary uses OAuth, renewable tokens, API credentials, scoped access, or credential rotation.

## Pin the security contract

Identify the actor and grant: service account, application, delegated user, installation, tenant, or webhook sender. Pin the authorization server, token endpoint, resource audience, supported grant, scopes, token type, expiry, revocation, rotation behavior, clock assumptions, and environment. Prefer provider discovery metadata and an official SDK over hand-built protocol code when they cover the required contract.

Grant the smallest scopes and resource audience that allow the operation. Keep sandbox and production credentials, redirect URIs, token caches, webhook secrets, and provider accounts separate. Bind every cached credential to its provider, environment, tenant/subject, audience, scopes, and client identity so a token cannot cross boundaries accidentally.

Authorization-code clients use PKCE with `S256`; public clients require it and confidential clients benefit from it. Bind authorization state/nonce and redirect URI to the initiating session, and prefer the code flow over exposing access tokens in authorization responses. Machine-to-machine work uses the provider-supported non-user grant rather than simulating a user.

## Store and expose less

Keep client secrets, API keys, access tokens, and refresh tokens in the repository's existing secret facility with encryption and access audit. Send them only to the pinned TLS origin and never through query strings. Redact headers, URLs, bodies, traces, exception objects, fixtures, and support bundles before they cross a trust boundary.

Treat refresh tokens as higher-value long-lived credentials. Restrict who can read or rotate them, retain only necessary generations, and make revocation and reauthorization an explicit recovery state.

## Coordinate refresh

Cache access tokens by their full security binding and refresh before expiry with enough skew for clock and request duration. Add small jitter across many tenants so they do not refresh simultaneously.

Use single-flight or an equivalent per-binding coordination mechanism: one worker refreshes while peers await the result. Publish a new access token only after its accompanying refresh-token generation is durably stored. With rotating refresh tokens, compare-and-swap the expected generation or hold a narrow lock so a late response cannot overwrite a newer token.

On a provider-authenticated `401`, invalidate and refresh only when the provider contract indicates an expired or invalid access token. Replay the original request at most once and only if its request identity makes replay safe. A repeated `401`, revoked grant, insufficient scope, wrong audience, disabled account, or invalid refresh token becomes an owned authentication failure rather than a refresh loop.

If rotation replay detection or an `invalid_grant` response leaves token ownership uncertain, stop writes for that binding, preserve redacted evidence, and require reauthorization or provider-supported recovery. Concurrent refresh success is not permission to reuse the displaced refresh token.

## Rotate credentials safely

Credential changes are external actions requiring separate authorization. Prefer provider-supported overlap:

1. Create or activate the new credential with the same or narrower scope.
2. Deploy readers that can select the new generation and, for inbound signatures, verify the documented overlap set.
3. Prove authenticated requests and refresh behavior in the authorized environment.
4. Revoke the old credential, then prove it is no longer used.

When overlap is unavailable, declare the outage/freeze window and rollback before the change. A rollback can restore local configuration only while the previous provider credential remains valid.

Observe token age, time-to-expiry, refresh attempts/results/latency, single-flight waiters, `401` after refresh, `invalid_grant`, scope/audience mismatch, credential generation in use, and rotation completion. Never put token material, authorization codes, user identifiers, or unbounded tenant IDs in metrics.

## Failure checks

Use a fake authorization server or scripted transport to prove:

- many concurrent requests near expiry cause one refresh;
- a rotated refresh token is committed atomically and a stale refresher cannot overwrite it;
- one eligible `401` refreshes and replays once, while repeated `401` stops;
- permanent scope, audience, revocation, and `invalid_grant` failures do not retry forever;
- cancellation and the end-to-end deadline include refresh wait and token endpoint latency;
- logs, traces, and errors contain no credential material;
- old/new credential overlap and old-key revocation follow the provider contract.

## Primary sources

- [RFC 9700: Best Current Practice for OAuth 2.0 Security](https://www.rfc-editor.org/rfc/rfc9700.html)
- [RFC 8414: OAuth 2.0 Authorization Server Metadata](https://www.rfc-editor.org/rfc/rfc8414.html)
- [RFC 7636: Proof Key for Code Exchange](https://www.rfc-editor.org/rfc/rfc7636.html)

RFC 9700 drives PKCE, privilege/audience restriction, sender-constrained or rotating refresh-token protection, and secure refresh-token storage. Provider documentation remains authoritative for the grant and rotation behavior actually supported.
