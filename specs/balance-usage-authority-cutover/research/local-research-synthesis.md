# Local Research Synthesis

Scope: fan-in synthesis for L1-L9 local research on
`balance-usage-authority-cutover`.

## What Is Known

- The prior microlease packet is useful preserved context, not an active
  implementation handoff for this broader cutover.
- Billing-service has strong durable schema/repository/event building blocks for
  USD atom balances, idempotency, immutable ledger entries, usage operations,
  microleases, child debits, event inbox/outbox, and reconciliation.
- Billing-service does not currently have the full HTTP/runtime surface needed
  by the requested cutover: account resolve, balance read, generic usage
  reserve/finalize/write-off/reversal, admin/reconciliation readbacks, concrete
  HTTP app service wiring, and real worker task wiring.
- Gonka-proxy still has live local money authority in normal completion and
  web-search paths, guarded only by feature flags/cutover checks in selected
  code paths.
- Gonka-proxy has two partially overlapping target concepts:
  an older shared-balance bridge to generic `/api/v1/usage/*` endpoints, and
  newer durable microlease code that is not wired into live completion paths.
- Current provider/consumer contracts do not match without specification work:
  route paths, auth model, scopes, status/result envelopes, usage identity, and
  microlease versus generic usage lifecycle semantics differ.

## Specification Inputs

The specification phase should decide these before technical design:

1. Authoritative API model and route compatibility.
2. Account resolve and balance read semantics, including migrated account import
   and active reserve/microlease exposure.
3. Usage lifecycle command model: reserve, finalize, write-off, reversal,
   operation readback, and reconciliation/admin readback.
4. Microlease relationship to generic usage commands.
5. Idempotency, replay, conflict, and stored outcome contract.
6. Service authentication, route scopes, account binding, and privacy-safe
   metadata requirements.
7. Worker/event ownership and recovery semantics.
8. Proxy cutover policy and whether cross-repo implementation is in scope.
9. Proof obligations, including cross-repo contract and negative-path proof.

## Risks To Keep Visible

- Enabling proxy shared-balance cutover against current billing-service routes
  would not match the current provider API.
- Treating proxy local balance reservations as fallback for migrated cohorts
  would violate the workflow plan's explicit out-of-scope direct reserve fallback
  rule.
- Treating current billing-service microlease OpenAPI as runtime-ready would be
  wrong without handler service injection and worker task wiring.
- Treating `User.balanceNgonka` and `BalanceTransaction` as live mutable state
  after account migration would create dual money authority unless the spec
  defines historical/read-only behavior.
- The operation-readback scope mismatch must be resolved before generated
  contract and service-auth proof can be clean.

## Ready For Specification

Research is sufficient to start specification. No research blocker remains, but
the specification phase must not silently choose API shape, auth compatibility,
proxy write scope, fallback policy, or worker runtime ownership during
implementation.
