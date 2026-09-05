# Cached-Value Contract

Load when value meaning, key scope, freshness, or invalidation can change the
decision.

For each value record authority and revision, positive/negative/derived
representation and serializer version, complete key dimensions, fresh and
maximum age, fill owner and publish condition, mutation invalidation path,
duplicate-fill/fencing policy, degraded behavior, and falsifier.

Equal keys must be interchangeable for the requesting principal. Include
tenant, authorization/policy, locale, representation, and any response-varying
input; canonicalize equivalent inputs and version incompatible namespaces. Keep
secrets out of key material and logs. Prefer caching authority-independent data
and applying current policy after retrieval when final-response policy variants
cannot be invalidated reliably.

Classify serves as `fresh`, `allowed-stale`, `forbidden-stale`, or
`unknown-age`. Age begins at authoritative generation/validation, not local
insertion. Negative results are distinct from errors and need create-time
supersession. Commit authority before invalidation. Reject an old in-flight fill
after a newer mutation using revision/generation; TTL is only a bounded
backstop. When missed invalidation can exceed the staleness limit, require a
durable delivery and reconciliation owner.
