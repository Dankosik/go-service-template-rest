# Modular Monolith Vs Service Extraction

## When To Load
Load this when a prompt asks whether to keep a Go service as a modular monolith, create internal packages/modules, split a separate runtime/worker, or extract a true service.

Use it to test the boundary against domain capability, data ownership, team ownership, transaction boundary, runtime isolation, and rollout cost. Keep package layout guidance secondary to the architecture decision; Go directories can enforce internal code boundaries, but they do not prove a service boundary.

## Decision Examples

### Example 1: Onboarding flow owned by one team
Context: An onboarding service includes applicant profile capture, document verification, sanctions screening, and manual reviewer decisions in one codebase and one Postgres database. One team owns all of it. Another team wants four immediate microservices.

Selected option: Keep a modular monolith. Define modules around invariant ownership and workflow roles, for example `applicant-profile`, `verification`, `screening`, and `review-decision`, with a separate application/orchestration layer if it coordinates the process. Use logical data ownership rules inside the datastore and explicit internal contracts.

Rejected options:
- Four microservices immediately, because team ownership, transaction boundaries, and operational independence are not yet proven.
- A flat package layout by technical layer, because each business change would cut across all layers.
- A shared `common` domain model for all modules, because it hides ownership and encourages cross-module invariants.

Evidence that would change the decision:
- Distinct teams become responsible for different capabilities and need independent release cadence.
- Screening or document verification has materially different scaling, availability, compliance, or provider-isolation needs.
- A module's contract stabilizes, and its data can move behind exclusive ownership without coordinated releases.
- The current monolith cannot meet runtime goals after worker/runtime split, queueing, or module isolation.

Failure modes and rollback implications:
- Too many tiny modules create peer-module chatter; merge closely coupled modules while refactoring is still local.
- If logical ownership is not enforced, the modular monolith decays into folders over a shared data model.
- Service extraction rollback is harder than module refactoring because routing, data authority, and mixed-version compatibility may outlive the code deploy.

### Example 2: Read-only storefront rendering
Context: A high-throughput read-only rendering capability handles a different workload shape from the merchant administration workflow. It can run with narrow data access and strict runtime constraints.

Selected option: A separate runtime or service can be justified if the capability is read-only or derived, has a narrow stable contract, and benefits from independent scaling and performance constraints.

Rejected options:
- Extracting mutable core workflow state together with the renderer.
- Keeping it in the core monolith only because the organization prefers one deployment unit.
- Creating a service that still depends on broad internal state or direct database access from the monolith.

Evidence that would change the decision:
- The renderer needs correctness-critical writes or tightly coupled transactions with core workflows.
- Scaling data shows the monolith can meet throughput with a simpler runtime split or cache.
- The contract is unstable and every product change still requires coordinated deploys across the boundary.

Failure modes and rollback implications:
- A read-only extraction can usually route traffic back if old and new renderers share compatible inputs.
- A hidden shared-data dependency turns the service into a distributed monolith; rollback must include dependency removal.
- Separate runtime constraints create drift unless they are documented and tested as part of release readiness.

### Example 3: Batch/export pressure
Context: A service needs long-running exports that scan large data ranges and threaten request-path latency.

Selected option: Prefer a separate worker runtime, bounded queue, read replica, or stable read fence before extracting a new domain service. The write owner remains unchanged; the worker owns execution and backpressure, not business truth.

Rejected options:
- New export service that owns the same entities as the core service.
- Running the export synchronously on the request path.
- Using a projection as write truth because it is convenient for export formatting.

Evidence that would change the decision:
- Export business rules become independently owned and change on a separate cadence.
- The export requires a distinct data product with its own freshness contract and support owner.
- Worker/runtime isolation cannot protect request latency or storage load after measurement.

Failure modes and rollback implications:
- Queue backlog can become hidden unbounded work; define cancellation, retry, and shedding ownership.
- Read fences may disappoint users if phrased as exact snapshots without proof.
- Rolling back a worker runtime is usually a routing/scheduling change; rolling back a service extraction requires contract and data ownership rollback.

## Source Links Gathered Through Exa
- Go, "Organizing a Go module": https://go.dev/doc/modules/layout
- Go Blog, "Package names": https://go.dev/blog/package-names
- Shopify Engineering, "Deconstructing the Monolith": https://shopify.engineering/deconstructing-monolith-designing-software-maximizes-developer-productivity
- Shopify Engineering, "Under Deconstruction: The State of Shopify's Monolith": https://shopify.engineering/shopify-monolith
- Shopify Engineering, "Componentizing Shopify's Tax Engine": https://engineering.shopify.com/blogs/engineering/componentizing-shopify-tax-engine
- Kamil Grzybek, "Modular Monolith: A Primer": https://www.kamilgrzybek.com/blog/posts/modular-monolith-primer
- GitLab Handbook, "Hexagonal Rails Monolith": https://handbook.gitlab.com/handbook/engineering/architecture/design-documents/modular_monolith/hexagonal_monolith
- Microservices.io, "Decompose by business capability": https://microservices.io/patterns/decomposition/decompose-by-business-capability.html

