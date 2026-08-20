# Authorization Enforcement

Load when permission semantics, object access, enforcement placement, or tenant
isolation can change the decision.

Write rules as `principal, action, resource, context -> allow/deny`. Use RBAC
for stable permission bundles, ABAC for state/context decisions, and ReBAC when
access follows relationships. An unregistered action denies by default; avoid
scattered role strings that cannot be enumerated or revoked.

Tenant scope is a mandatory partition derived from the verified principal, not
from a client-supplied user or tenant ID. Apply it to reads, writes, cache keys,
jobs, and messages. Authorize the object and caller-controlled fields, not only
the route. Put the decision at the effect-owning boundary so internal callers
and background work cannot bypass it. When permission may change between check
and effect, recheck inside the effect transaction or use the accepted
concurrency owner.

The falsifier crosses tenant and object IDs, attempts horizontal and vertical
escalation, exercises default deny and background/internal entry points, and
measures revocation within its accepted window. Reopen the data or concurrency
owner when enforcement requires a new durable constraint or atomic boundary.
