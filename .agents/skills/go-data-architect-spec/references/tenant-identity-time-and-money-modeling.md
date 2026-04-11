# Tenant, Identity, Time, And Money Modeling

## When To Load
Load this when the task involves tenant isolation, public IDs, partner references, idempotency keys, local business dates, event/effective/processed time, money, balances, credits, quotas, or user-visible amounts.

Use it to make identity and domain types explicit before table shape hardens. Tenant, time, and money mistakes are expensive because they leak across constraints, indexes, pagination, retention, reporting, and rollback.

## Decision Examples

### Example 1: Shared-table tenant isolation
Context: A service stores customer accounts for many tenants in shared tables. Account names only need to be unique inside one tenant, and support operators must not accidentally see another tenant's rows.

Selected option: Include `tenant_id` in invariant-bearing keys and indexes, such as `UNIQUE (tenant_id, account_slug)`. Propagate tenant identity into foreign keys where the child row belongs inside the same tenant boundary. Use row-level security only when the service can reliably set tenant context and test fail-closed behavior.

Rejected options:
- Global uniqueness for tenant-local concepts such as account slug or external reference.
- Application-only tenant filters with no database-level guard on critical tables.
- Cross-tenant foreign keys that make tenant isolation depend on convention.

Migration and rollback consequences:
- Add `tenant_id` additively, backfill from a trusted owner, validate coverage, then add tenant-scoped unique constraints.
- Enabling RLS is a compatibility checkpoint: old code paths and migration roles must be checked before `FORCE ROW LEVEL SECURITY`.
- Rollback is conditional once constraints or RLS policies start rejecting cross-tenant rows that older code might still write.

### Example 2: Public, partner, and internal identity
Context: A payment intent has an internal row ID, a public reference shown to customers, a provider payment ID, and an idempotency key from the caller.

Selected option: Use a stable internal primary key for joins and ownership. Store public references, provider references, and idempotency keys as separate columns with scoped unique constraints that match their authority, for example `(tenant_id, provider, provider_payment_id)` and `(tenant_id, idempotency_key)`.

Rejected options:
- Use mutable provider IDs as primary keys.
- Reuse correlation IDs as idempotency keys.
- Put partner payload IDs only in `jsonb` when they define uniqueness or reconciliation.

Migration and rollback consequences:
- Backfill new identity columns from deterministic evidence and quarantine rows with ambiguous mapping.
- Add unique constraints only after duplicate detection and cleanup.
- Rolling back an identity change may be restore-based if external clients have already observed new public references.

### Example 3: Money and time semantics
Context: Billing stores usage charges, plan prices, invoice dates, provider callbacks, and customer-local billing periods.

Selected option: Use exact numeric modeling for money, either integer minor units plus currency or `numeric(precision, scale)` with a documented rounding policy. Use `timestamptz` for real instants, separate `business_date` or effective-time fields for policy dates, and track provider event time separately from processing time when late delivery matters.

Rejected options:
- Floating-point columns for money or billable usage.
- One `created_at` timestamp to mean event time, processing time, effective time, and local business date.
- Database `money` type without an explicit currency and rounding policy.

Migration and rollback consequences:
- Converting money representation requires expand/backfill/verify/contract with parity checks by currency and aggregate.
- Time-model changes need dual-read or comparison windows because reports may change around timezone and late-arrival boundaries.
- Once invoices or customer-visible balances are issued from the new representation, rollback usually requires forward correction rather than blind schema revert.

## Source Links Gathered Through Exa
- PostgreSQL, "Row Security Policies": https://www.postgresql.org/docs/current/ddl-rowsecurity.html
- PostgreSQL, "Constraints": https://www.postgresql.org/docs/current/ddl-constraints.html
- PostgreSQL, "Date/Time Types": https://www.postgresql.org/docs/current/datatype-datetime.html
- PostgreSQL, "Numeric Types": https://www.postgresql.org/docs/current/datatype-numeric.html
- Stripe, "Idempotent requests": https://docs.stripe.com/api/idempotent_requests

