# Required Checks And Change Scope

## Load When

Load for required statuses, Rulesets, merge queues, path routing, or a green
check that did not exercise its claimed surface.

## Decide

- `ci.yml` classifies the exact diff once. Its quality, security, secret,
  delivery, and integration leaves are conditional; one `required` job is
  always reported and rejects any failed or cancelled applicable leaf.
- A skipped leaf is not evidence for that surface. The classifier makes the
  skip explicit, while tag dispatch marks Go and release surfaces applicable so
  integration cannot be path-skipped.
- OpenAPI and Protobuf breaking comparisons run only on pull requests with the
  event's exact base SHA. A missing base OpenAPI contract means no comparison,
  not proof of compatibility.
- CodeQL reports one stable `codeql-required` context over its conditional Go
  and Actions analyses. Main, tag, schedule, and manual runs execute both.
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
