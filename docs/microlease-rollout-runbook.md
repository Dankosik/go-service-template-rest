# Billing Microlease Rollout Runbook

Status: implementation-owned operator runbook for the durable microlease rollout.

## Rollout Modes

- `inert_expand`: schema/config/contracts are present, but paid admission remains closed.
- `shadow_no_spend`: proxy may compare balances and exposure, but no microlease capacity authorizes execution.
- `internal_cohort`: only explicitly enabled internal paid cohorts may issue microleases.
- `migrated`: migrated paid cohorts use billing-issued microleases and proxy child debits.
- `rollback`: no new microleases; already minted valid microleases may be used only until cutoff.

## Required Gates

- Balance parity and active exposure parity must be green before `internal_cohort` or `migrated`.
- Old proxy-local money writer must be disabled before migrated cohorts are admitted.
- Direct per-request reserve fallback must be disabled for migrated cohorts.
- Critical terminal lag, stale exposure, or reconciliation backlog closes paid admission.
- Operators must have readback for lag, stale exposure, reconciliation backlog, rollout mode, and cohort state before enabling spend.

## Rollback

Rollback does not re-enable direct reserve fallback for migrated cohorts.

If a valid microlease already exists, proxy may spend only within its remaining durable child cap until debit cutoff. After cutoff, or when no valid microlease exists, paid admission fails closed until the cohort is moved back through an explicitly approved rollout state.
