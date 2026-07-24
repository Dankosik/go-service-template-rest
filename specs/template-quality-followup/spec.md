# Make the template's examples safe and easy to extend
status: ready

## Scope and non-goals

Implement every accepted follow-up finding from the repository quality review:

- distinguish an absent migration directory from an unreadable or invalid one;
- rehearse the complete PostgreSQL migration chain, not only the latest step;
- preserve migration-runner cleanup failures;
- expose the bounded outbound HTTP client through `Do` without returning its
  mutable `*http.Client`;
- finish the concrete PostgreSQL bootstrap flow by removing its residual
  generic result envelope and fictional fallback inputs;
- reject whitespace-only explicit configuration paths consistently across the
  public loader and CLI;
- split the largest mixed-responsibility implementation and test files along
  their existing package-owned behaviors.

This work does not add dependencies, packages, configuration fields, public
routes, migration formats, or reusable frameworks.

## Behavior and ownership

- Migration discovery treats only `fs.ErrNotExist` as "no migration files";
  every other directory read failure reaches the caller.
- PostgreSQL rehearsal applies all migrations, rolls all of them back, then
  reapplies all of them. A cleanup failure remains inspectable even when the
  migration operation also fails.
- `httpclient.Client` implements the standard `Do(*http.Request)` seam and
  keeps its configured transport, redirect policy, and timeout private.
- Bootstrap returns the optional PostgreSQL pool directly and constructs the
  single readiness probe at the composition root. PostgreSQL rejection helpers
  accept only real PostgreSQL inputs.
- `LoadOptions.ConfigPath == ""` remains the optional zero value. A non-empty
  whitespace-only base path, or any empty/whitespace overlay entry, fails at the
  load-file stage.
- File moves remain inside their current packages and do not change runtime or
  test behavior. Generated files remain untouched.

## Proof obligations

- Focused migration tests cover missing and invalid source paths, full-chain
  rehearsal order, operation failures, and joined cleanup failures.
- A real PostgreSQL integration fixture with multiple migrations proves that
  an earlier down migration is exercised.
- HTTP client tests compile and run through `Client.Do` while preserving the
  bounded transport behavior.
- Bootstrap tests cover disabled, successful, failed, and timed-out PostgreSQL
  startup without the generic envelope or fallback cases.
- Public config-loader tests reject whitespace-only base and overlay paths.
- Focused package tests, changed-surface race proof, formatting, lint, broad
  repository checks, and GitHub required checks pass before merge.

## Reopen conditions

- Reopen dependency-bootstrap abstraction only after a second production
  dependency has equivalent lifecycle behavior.
- Reopen HTTP customization only for a concrete consumer requirement that
  cannot use the bounded `Do` seam.
