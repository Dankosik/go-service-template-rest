# Credential Lifecycle

Load when issue, verification, rotation, recovery, session/token choice, or
revocation latency can change the decision.

For each credential define issue, verify, rotate, revoke, recover, and abuse
limits. Recovery is an authentication path and must not bypass reauthentication,
enumeration resistance, or rate limits. Store API keys and comparable
credentials as scoped, expiring, non-recoverable verifiers when the protocol
allows it.

Choose session or token shape from the tightest accepted revocation window:
server-side sessions permit immediate revocation; self-contained access tokens
remain valid until expiry unless a deny event is checked; short access tokens
plus rotating refresh tokens bound exposure to the access TTL. Refresh reuse
revokes the family. Validate issuer, audience, expiry, signature, and key
rotation overlap; bound clock skew. Claims are readable by the holder and any
role, tenant, or scope claim is stale for up to its token lifetime.

For logout, credential compromise, role/scope downgrade, and tenant removal,
record what stops old access and the maximum delay. Reopen the policy owner when
that delay is not accepted.
