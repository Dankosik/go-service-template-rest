# Principal Boundaries

Load when principal types, entry points, delegation, or cross-service identity
can change the decision.

For each end user, operator, API client, service account, background actor, or
anonymous caller, name the stable identifier, credential/issuer, lifecycle
owner, tenant membership, and allowed entry points. Background work uses a named
service principal or durably captured user context, never ambient superuser
authority. Impersonation and on-behalf-of flows carry both actor and subject.

Every hop verifies the identity it consumes. Edge-injected identity headers are
trusted only inside a boundary unauthenticated callers cannot reach. Restrict
token audience; use explicit delegation/token exchange when one service acts
for a user, and do not forward a caller token to an unintended service. Async
payloads carry durable principal context rather than live tokens that may expire
in a queue.

Reopen the system/security owner when a hop lacks a verifiable identity,
delegation meaning, or replay boundary.
