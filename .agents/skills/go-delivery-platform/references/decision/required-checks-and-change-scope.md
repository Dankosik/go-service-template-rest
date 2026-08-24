# Required Checks And Change Scope

## Load When

Load for required statuses, Rulesets, merge queues, path routing, or a green
check that did not exercise its claimed surface.

## Decide

- `ci.yml` classifies the exact diff once. Its quality, security, secret,
  delivery, and integration leaves are conditional; runtime Go, root/tool
  dependencies, lint config, initializers, database, messaging, process, race,
  migrations, runtime image, and image security remain distinct. One `required`
  job is always reported and rejects any failed or cancelled applicable leaf.
- A skipped leaf is not evidence for that surface. The classifier makes the
  skip explicit. Pull requests, merge groups, and main pushes use their exact
  comparison base; tags, manual runs, and the weekly audit select every surface.
  Generated Go, compose, publication metadata, and each race owner remain
  independently visible.
- OpenAPI and Protobuf breaking comparisons run only on pull requests with the
  event's exact base SHA. A missing base OpenAPI contract means no comparison,
  not proof of compatibility.
- CodeQL reports one stable `codeql-required` context. Go source and dependency
  changes run Go analysis on pull requests and merge groups before admission as
  well as on main, tags, schedules, and manual runs. Actions analysis remains
  conditional on workflow source.
- Both workflows handle `merge_group` before their aggregate contexts are
  required for merge queue events.

## Reject

- Reintroducing a script or jq aggregator that restates job policy instead of
  using native `needs.*.result` conclusions.
- Treating a path-skipped workflow as a passing required context.
- Treating generation, integration, or migration steps that did not run as
  passing evidence.

## Prove

Use the Actions run URL, exact SHA, classifier outputs, applicable job
conclusions, and the `required` and `codeql-required` conclusions.
