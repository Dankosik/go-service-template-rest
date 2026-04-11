# Tenant, Identity, Time, And Money Modeling

## Behavior Change Thesis
When loaded for a task where tenant scope, public references, partner IDs, idempotency keys, business dates, event time, processed time, balances, credits, or money could blur together, this file makes the model choose explicit scoped domain types instead of likely mistake "reuse one identifier, one timestamp, or one numeric column everywhere."

## When To Load
Load this for tenant-scoped identity, external references, time semantics, money, quota, credit, or user-visible amount decisions.

## Decision Rubric
- Use stable internal surrogate keys for joins and ownership; model public references, partner references, correlation IDs, and idempotency keys as separate concepts.
- Scope uniqueness to the authority that owns it: usually `(tenant_id, key)` or `(tenant_id, provider, external_id)`, not global by habit.
- Put `tenant_id` in invariant-bearing uniqueness, child ownership, and access-path indexes for shared-table tenancy.
- Use row-level security only when tenant context is reliably set, tests prove fail-closed behavior, and migration/admin roles are accounted for.
- Model real instants, effective time, provider event time, processing time, and user-local business dates separately when policy or reporting depends on the distinction.
- Use exact money representation with currency and rounding policy. Do not use floating-point types for money, credits, quotas, or billable usage.

## Imitate

### Shared-table tenant isolation
Context: Account names are unique only within a tenant, and support operators must not accidentally see another tenant's rows.

Choose `UNIQUE (tenant_id, account_slug)`, tenant-aware foreign keys where child rows belong to the same tenant, and optional RLS only after context propagation and fail-closed behavior are testable.

Copy this because tenant scope belongs in the invariant and access path, not only in application filters.

### Public, partner, and internal identity
Context: A payment intent has an internal row ID, a public customer reference, a provider payment ID, a caller idempotency key, and a request correlation ID.

Use the internal ID as primary key. Store public references, provider references, and idempotency keys in separate columns with scoped uniqueness such as `(tenant_id, provider, provider_payment_id)` and `(tenant_id, idempotency_key)`.

Copy this because each identifier answers a different authority question.

### Money and time semantics
Context: Billing stores usage charges, plan prices, invoice dates, provider callbacks, and customer-local billing periods.

Use integer minor units plus currency, or `numeric(precision, scale)` plus currency and rounding policy. Use `timestamptz` for real instants, separate `business_date` or effective-time fields for policy dates, and separate provider event time from processing time for late callbacks.

Copy this because customer-visible balances and invoices need explainable precision and date semantics.

## Reject
- "Provider payment ID is the primary key." Provider identifiers may be mutable, reused by scope, or absent before provider handoff.
- "Use a correlation ID as the idempotency key." Correlation traces a request; idempotency proves safe replay for a semantic operation.
- "All rows filter by tenant in code, so database constraints can stay global or tenant-free." That makes isolation depend on convention.
- "Store provider IDs in `jsonb`; reconciliation can search payloads." If the ID defines uniqueness or repair, make it relational.
- "Use `created_at` for provider event time, invoice business date, and processing time." That destroys late-arrival and reporting semantics.
- "Use float for money because values are small." Small values still need exact representation and rounding.

## Agent Traps
- Do not mark every unique key as global just because it is easier to describe.
- Do not propose RLS without naming how service, migration, and support roles set or bypass tenant context.
- Do not assume UTC instants solve user-local business-date policy.
- Do not collapse balances, credits, and quotas into one generic numeric type if their rounding, currency, or lifecycle differs.

## Validation Shape
- Identity proof lists each identifier, its authority, uniqueness scope, exposure level, and reconciliation use.
- Tenant proof includes constraints or indexes with `tenant_id`, fail-closed access tests, and migration-role behavior when RLS is used.
- Money/time proof includes precision, currency, rounding, event/effective/processed time mapping, and parity checks for representation changes.
