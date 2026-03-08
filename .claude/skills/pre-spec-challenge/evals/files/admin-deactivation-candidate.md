# Scenario

We want an internal admin flow to deactivate a customer account.

## Problem Frame

- Support operators sometimes need to stop abusive or compromised accounts quickly.
- The current system only supports full deletion via a maintenance script.

## Candidate Decisions

- Add an admin-only endpoint `POST /internal/admin/accounts/{id}/deactivate`.
- Deactivation sets `status=deactivated` on the account row.
- Existing sessions will naturally expire; we do not need to revoke them immediately in v1.
- Integrations owned by that account can continue running until they fail naturally.
- Reactivation is not planned initially.
- Audit logging can be added later if the endpoint proves useful.

## Constraints

- This is an internal tool, not a public API.
- Operators want a one-click action.
- We want to ship the first version quickly.

## Open Assumptions

- [assumption] Internal-only means lighter audit requirements.
- [assumption] Natural session expiry is good enough for compromised-account cases.
- [assumption] Side effects on downstream integrations do not need explicit policy yet.
- [assumption] Irreversibility is acceptable because support can always run a DB fix manually.

## Task

Run a pre-spec challenge pass on these candidate decisions. Focus on what could still change planning safely.
