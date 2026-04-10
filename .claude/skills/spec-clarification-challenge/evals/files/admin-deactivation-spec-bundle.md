# Scenario

The orchestrator is about to approve `spec.md` for internal admin account deactivation.

## Problem Frame

- Support operators need a quick way to stop abusive or compromised accounts.
- The current workaround is a full-deletion maintenance script.

## Scope / Non-goals

- In scope: internal admin-only deactivate action.
- Out of scope: customer self-service, automated appeal workflow, full account deletion.

## Candidate Decisions

- Add `POST /internal/admin/accounts/{id}/deactivate`.
- Deactivation sets `status=deactivated` on the account row.
- Reactivation is out of scope for v1.
- Existing sessions expire naturally instead of being revoked immediately.
- Integrations owned by the account keep running until they fail naturally.
- Audit logging can be added later if the endpoint proves useful.

## Constraints And Validation Expectations

- The action is internal-only.
- Operators want a one-click action.
- Validation should prove authorization and deactivated-account request denial.

## Known Assumptions / Open Questions

- [assumption] Internal-only status means audit can wait.
- [assumption] Natural session expiry is good enough for compromised accounts.
- [assumption] Support may decide deactivation policy per case without product approval.
- [open] Whether paid-customer deactivation needs a support-manager approval policy is not recorded in repo evidence.

## Research Links

- `research/admin-authz.md`
