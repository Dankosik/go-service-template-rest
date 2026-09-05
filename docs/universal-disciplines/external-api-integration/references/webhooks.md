# Webhooks

Load when provider callbacks affect authentication, durable receipt, ordering,
replay, or convergence.

Capture bounded raw bytes before parsing. Verify the provider's exact signature
or authentication contract, endpoint/environment binding, algorithm,
canonicalization, key generation, timestamp tolerance, and constant-time
comparison before durable acceptance or effect. An allowlist does not replace
integrity verification. If no signed timestamp exists, do not invent one; use
provider delivery identity and its documented replay window.

Keep the synchronous receiver to: bound/capture, authenticate, atomically retain
raw-body hash plus provider event/delivery identity and audit metadata, then
return the provider-defined response. Storage failure returns the provider's
retry response. A durable authenticated duplicate returns success without
repeating the effect. Semantic processing runs asynchronously.

Assume duplicate and out-of-order delivery unless the pinned contract says
otherwise. Atomically deduplicate by provider receipt identity plus any
documented semantic key. Only provider sequence/version semantics may order
events; receipt time does not. Stale events are no-op, and gaps/unknown schemas
retain evidence and trigger lookup or reconciliation rather than fabricated
state.

Proof mutates one signed byte/secret/environment, checks timestamp bounds,
concurrent duplicate receipt, storage failure, slow async processing,
out-of-order/stale/gap behavior, unknown version quarantine, key overlap, and
dropped callback convergence through lookup/reconciliation.
