# Rollout And Migration Patterns

## Behavior Change Thesis
When loaded for ownership moves or mixed-version rollout, this file makes the model choose phased compatibility with one authoritative writer per phase, instead of big-bang cutover, indefinite dual writes, or vague rollback claims.

## When To Load
Load when a design moves ownership, extracts a service, changes source of truth, introduces a new runtime, requires mixed-version compatibility, or needs rollout and rollback boundaries.

## Decision Rubric
- Name one write authority for each phase before choosing traffic routing.
- Prefer additive compatibility first, then shadow/dark read, dual-read comparison, canary, cutover, contract, and removal when each step has useful evidence.
- Use dual-write-like comparison or bridge writes only as bounded migration mechanisms with one authoritative side, drift metrics, reconciliation owner, and deletion criteria.
- Keep irreversible operations behind the old owner until the cutover checkpoint when rollback must be a router flip.
- Separate deploy rollback from authority rollback. After data authority moves, rollback may mean forward repair or route-back with reconciliation.
- Bound bridges, compatibility topics, facades, and shims with owner, consumers, exit metric, and removal task.

## Imitate

### Pricing Extraction With Mixed Versions
Context: `pricing` is moving out of `catalog`. Old checkout nodes, new pricing nodes, and lagging admin tooling will coexist for weeks.

Choose: name one write owner per phase. Add compatibility first, use shadow or dual-read comparison if it produces attributable signals, canary traffic, cut over write authority once old dependencies are gone, then contract legacy writes and bridges.

Copy: this prevents two systems from becoming active writers of the same pricing truth.

### Tax Componentization With No Merchant Impact
Context: a monolith moves tax calculation into a component with a new entrypoint, and behavior must remain identical.

Choose: put the new component behind the old path. Run a side-by-side dark calculation that discards the new result, measure differences, then ramp with old path authoritative until parity or intended deltas are approved.

Copy: this uses production-shaped evidence without moving authority too early.

### New Runtime Canary
Context: a new runtime handles part of a production request flow and the team wants limited blast radius.

Choose: canary only when traffic can be segmented and metrics are attributable to new vs old populations. Define advance, halt, and rollback actions before starting.

Copy: this avoids a before/after comparison that can be swamped by traffic variation.

## Reject
- "Dual-write both paths until we are confident." Bad because confidence is not an authority model.
- "Permanent compatibility topic just in case." Bad because temporary bridges become hidden architecture.
- "Blue/green deploy is enough rollback." Bad when data authority, schema compatibility, or in-flight work changes.
- "Canary with aggregate metrics." Bad because the new runtime's signal is not attributable.

## Agent Traps
- Do not say "rollback" without saying what it can and cannot undo.
- Do not contract legacy fields, events, or writes until mixed-version readers and admin tooling are gone.
- Do not add shadow reads or dual reads unless someone will compare, alert, and act on the signal.
- Do not drift into migration script mechanics; keep this at authority, compatibility, observability, and checkpoint level.
