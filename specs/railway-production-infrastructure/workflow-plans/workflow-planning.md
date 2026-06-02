# Railway App-Only Deployment Workflow Planning

Phase: workflow-planning
Status: complete, repaired after user scope correction
Date: 2026-06-02
Owner: orchestrator

## Routing Decision

The current workflow target is app-only Railway deployment for `billing-service`.

The earlier full production-infrastructure routing is superseded. It was too broad for the user's corrected goal because it included proxy integration, worker, broker, dedicated Postgres, backup/PITR, paid cohorts, and customer-money rollout.

## Current Shape

Mode: full orchestrated, narrowed to one live app deployment because production Railway mutation remains a protected delivery domain.

Required artifacts:

- `spec.md`;
- `design/overview.md` plus small split design files already present in this directory;
- `workflow-plans/technical-design-review.md`;
- `tasks.md`;
- `test-plan.md`;
- `rollout.md`.

No additional subagent fan-out is required for the repaired app-only scope.

## Stop Rule

Workflow planning is complete. Implementation starts only from the reviewed `tasks.md`.

No code, config, generated artifacts, Railway services, variables, deployments, databases, brokers, domains, or live state are changed in workflow planning.
