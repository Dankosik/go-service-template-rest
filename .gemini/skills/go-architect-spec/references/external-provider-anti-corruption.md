# External Provider Anti-Corruption

## Behavior Change Thesis
When loaded for external provider or partner lifecycle pressure, this file makes the model normalize provider evidence behind a local lifecycle owner instead of importing vendor statuses, retries, webhooks, and failure vocabulary as internal source-of-truth semantics.

## When To Load
Load when a design depends on an external provider, partner webhook, vendor status model, ambiguous third-party response, provider retry policy, callback ordering, or external lifecycle vocabulary.

## Decision Rubric
- Treat providers as semi-trusted evidence sources. The local service owns internal lifecycle truth, acceptance decisions, retries, reconciliation, and operator repair.
- Normalize provider statuses at an anti-corruption adapter before they enter domain state, public API status, metrics, or alerts.
- Keep internal state monotonic and explicit about ambiguity: `submitted`, `pending_confirmation`, `confirmed`, `failed`, `requires_repair`, or similar local states when the business needs them.
- Use provider identifiers as correlation evidence, not as the primary idempotency or lifecycle key unless the provider contract truly guarantees it.
- Define webhook ordering, duplicate handling, timeout, retry, and replay assumptions before trusting callbacks.
- Treat callbacks as evidence only after authenticity, account or tenant context, replay, and recency handling are assigned to the local boundary.
- Fail closed when provider ambiguity could authorize money movement, entitlement, identity verification, tenant access, or irreversible notification.
- Reconcile independently of provider push events when missing, duplicate, delayed, or contradictory callbacks are plausible.
- Keep the adapter in-process unless separate ownership, scale, isolation, or release control justifies the extra service boundary.

## Imitate

### Payment Capture With Ambiguous Provider Response
Context: capture returns a timeout after the local system submits a request. The provider may later send a webhook saying captured, declined, or unknown.

Choose: keep `payments` as lifecycle owner. Record a local submitted/pending state with idempotency key and provider correlation id, reconcile by poll/webhook, and expose a pending or repairable status until the provider result is normalized.

Copy: this avoids treating HTTP timeout as either success or failure and prevents double capture.

### KYC Provider Status Vocabulary
Context: provider statuses include `green`, `yellow`, `manual_review`, and `expired`, while the product needs local onboarding decisions.

Choose: map provider evidence into local decision states such as `verified`, `needs_review`, `rejected`, or `evidence_expired`, with local review ownership and audit trail. Keep raw provider payload as evidence, not domain language.

Copy: this prevents vendor vocabulary from leaking into internal invariants and public API promises.

### Shipping Carrier Webhooks
Context: carrier webhooks can arrive out of order and sometimes skip intermediate statuses.

Choose: model carrier updates as evidence for a local shipment timeline. Use monotonic state transitions, dedupe by webhook/event identity, reconcile missing milestones, and avoid making carrier callbacks the only repair path.

Copy: this keeps the local system operable when provider delivery is delayed or inconsistent.

## Reject
- "Use the provider status enum directly in our domain model." Bad because provider vocabulary changes the local lifecycle and leaks across API, data, and support flows.
- "If the partner call times out, mark it failed and retry with a new request." Bad when the first request may have committed externally.
- "The provider retries webhooks, so we do not need reconciliation." Bad because provider retries do not cover local handler bugs, lost credentials, ordering ambiguity, or support repair.
- "Expose raw partner failure reasons to clients." Bad when they are unstable, sensitive, or inconsistent with local remediation semantics.
- "Let the provider decide tenant access or entitlement." Bad because local authorization and tenant isolation must remain fail-closed under ambiguity.

## Agent Traps
- Do not design public status values by copying partner states.
- Do not assume webhook ordering unless the prompt gives that contract.
- Do not treat provider idempotency as local idempotency without retention, scoping, and same-payload behavior.
- Do not forget the operator path for `unknown` or `requires_repair` outcomes.
- Do not make the adapter a generic pass-through client; it owns vocabulary translation and evidence normalization.
