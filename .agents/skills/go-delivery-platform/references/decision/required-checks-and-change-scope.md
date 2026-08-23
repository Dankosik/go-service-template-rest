# Required Checks And Change Scope

## Load When

Load for required statuses, Rulesets, merge queues, path routing, or a green
check that did not exercise its claimed surface.

## Decide

- `ci.yml` exposes three stable always-reported contexts: `quality`,
  `security`, and `delivery`. Require those contexts instead of a
  script-maintained job inventory.
- `quality` always runs `make check`. Template initializer and instruction
  eval steps inside that job are path-gated on pull requests. A skipped step
  is not evidence for the skipped surface; push and `workflow_dispatch` still
  run them.
- `integration.yml` uses native path filters for PostgreSQL, migration, image,
  and integration-test owners. It is not an always-reported context. Ruleset
  policy must require it only where the platform can match the same scope;
  otherwise remove the filters and run it universally.
- OpenAPI and Protobuf breaking comparisons run only on pull requests with the
  event's exact base SHA. A missing base OpenAPI contract means no comparison,
  not proof of compatibility.
- CodeQL reports its own Go and Actions analysis contexts. Keep it independent
  from repository CI.
- Add `merge_group` before requiring these contexts for merge queue events.

## Reject

- Reintroducing a jq aggregator that restates every job name.
- Treating a path-skipped workflow as a passing required context.
- Treating generation, integration, or migration steps that did not run as
  passing evidence.

## Prove

Use the Actions run URL, exact SHA, applicable job conclusions, and the event or
path scope that caused every required workflow to run.
