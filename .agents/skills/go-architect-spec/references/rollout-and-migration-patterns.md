# Rollout And Migration Patterns

## When To Load
Load this when a design moves ownership, extracts a service, changes source of truth, introduces a new runtime, requires mixed-version compatibility, or needs rollout and rollback boundaries.

Use it to choose a safe evolution path: expand, shadow or dark read, dual-read, canary, cut over, contract, and remove. Keep the focus on compatibility, authority, observability, irreversible checkpoints, and rollback limits. Do not drift into migration script mechanics.

## Decision Examples

### Example 1: Pricing extraction with six weeks of mixed versions
Context: `pricing` is moving out of `catalog`. Old checkout nodes, new pricing nodes, and lagging admin tooling will coexist for about six weeks. A proposed plan uses temporary dual writes from both old and new paths plus a permanent compatibility topic.

Selected option: Name one write owner for each phase. Use additive compatibility first, then shadow or dark reads, dual-read comparison if useful, canary traffic, and a cutover checkpoint. Contract legacy writes only after all old nodes and admin tooling stop depending on them. Bound any bridge/topic with removal criteria.

Rejected options:
- Dual writes from both old and new paths, because they create competing write authority.
- Permanent compatibility topics or shims "just in case", because they become hidden architecture.
- Big-bang cutover when mixed versions are already known.

Evidence that would change the decision:
- Legacy nodes can be upgraded in one controlled window and no mixed-version period exists.
- The business requires a hard cutover and accepts downtime or manual freeze.
- The new service cannot be made compatible with old clients, forcing a different extraction boundary or phased adapter.
- Measurement shows shadow/dual-read comparison cannot produce attributable signals.

Failure modes and rollback implications:
- After write authority moves, rollback may require forward repair, not simply redeploying old catalog.
- Dual writes can diverge; if used as a temporary migration mechanism, one side must be explicitly authoritative and reconciliation must be owned.
- A permanent compatibility topic hides ownership; removal date and consumers must be tracked before rollout starts.

### Example 2: Componentizing tax calculation with no merchant impact
Context: A monolith moves tax calculation into a component with a new entrypoint. Behavior must remain identical for existing checkout traffic.

Selected option: Introduce the new component behind the old path. Run an experiment or dark path that computes results through both old and new code and discards the new result while measuring differences. Roll out to a small population, observe, then increase gradually.

Rejected options:
- Immediate switch to the new code path for all traffic.
- Extracting a tax service before the component contract and behavior are stable.
- Comparing only unit tests when production inputs are diverse and domain-heavy.

Evidence that would change the decision:
- The old code path is unsafe to run twice or causes side effects.
- Input populations cannot be segmented safely.
- Differences are expected because the business is intentionally changing tax behavior; then the rollout needs product and support disclosure rather than invisible parity.

Failure modes and rollback implications:
- Dark path side effects can double-call external providers unless the experiment is read-only or isolated.
- If discrepancies appear, keep old code authoritative and iterate until the component matches or the intended delta is approved.
- Rolling back before data/authority moves is usually a traffic or flag change; after authority moves, define repair for any in-flight state.

### Example 3: Canary for a new service runtime
Context: A new runtime handles part of a production request flow, and the team wants to limit blast radius.

Selected option: Use a canary or strangler-style facade only when traffic can be segmented and compared. Tie canary metrics to SLIs, make them attributable to the new runtime, and define advance, halt, and rollback actions.

Rejected options:
- Before/after comparison as the only validation when traffic patterns vary over time.
- Canarying with metrics that aggregate old and new populations together.
- Blue/green switch without checking mixed-version and data compatibility.

Evidence that would change the decision:
- The system cannot split traffic or isolate canary/control signals.
- The change is data-destructive or schema-incompatible, so rollback is not a router flip.
- A smaller shadow test or dark read is safer before live canary.

Failure modes and rollback implications:
- A proxy/facade can become a bottleneck or single point of failure; include its failure behavior.
- Canary and control can share backend failure domains, making metrics noisy; use absolute SLO checks too.
- Rollback is limited by state changes made during canary. Keep irreversible operations behind the old owner until the cutover checkpoint.

## Source Links Gathered Through Exa
- Martin Fowler, "Strangler Fig": https://martinfowler.com/bliki/StranglerFigApplication.html
- Azure Architecture Center, "Strangler Fig pattern": https://learn.microsoft.com/en-us/azure/architecture/patterns/strangler-fig
- AWS Prescriptive Guidance, "Strangler fig pattern": https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/strangler-fig.html
- Google SRE Workbook, "Canarying Releases": https://sre.google/workbook/canarying-releases/
- Google Cloud Deploy, "Use a canary deployment strategy": https://cloud.google.com/deploy/docs/deployment-strategies/canary
- Shopify Engineering, "Componentizing Shopify's Tax Engine": https://engineering.shopify.com/blogs/engineering/componentizing-shopify-tax-engine
- AWS Builders' Library, "Avoiding fallback in distributed systems": https://aws.amazon.com/builders-library/avoiding-fallback-in-distributed-systems/

