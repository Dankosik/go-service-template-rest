# Boundary Decomposition Examples

## When To Load
Load this when the hard question is where a module, runtime, or service boundary belongs, who owns write truth, whether a direct data dependency is acceptable, or whether Go package/module structure is being mistaken for service architecture.

Keep the output at the architecture-decision level. Use these examples to sharpen ownership, invariant locality, dependency direction, and operational cost. Do not turn them into endpoint payloads, table designs, generated clients, or handler wiring.

## Decision Examples

### Example 1: Read-heavy catalog composition
Context: `catalog` owns product mutation. Category pages need product, price, and inventory summaries. Category reads can be stale for 90 seconds, but checkout cannot. A proposed `catalog-read-service` would directly join `pricing` and `inventory` databases.

Selected option: Keep product, price, and inventory write truth with their owning services. Serve category pages through a derived read path, such as a service-owned projection, an API composition layer, or a BFF that is explicit about staleness. Checkout must query or command the write owners for correctness-critical state.

Rejected options:
- A read service that directly joins other services' databases, because it makes database schemas the integration contract and weakens service autonomy.
- Moving checkout correctness to the category-page projection, because the projection is derived and stale by design.
- Extracting a new service only because read volume is high, because read scale is not the same as write ownership.

Evidence that would change the decision:
- Category pages require strict freshness near checkout semantics and cannot tolerate stale summaries.
- One team becomes the durable owner of the composed read product and accepts projection lag, rebuild, and support obligations.
- Pricing and inventory ownership changes so the proposed boundary becomes a real capability owner, not only a query shortcut.
- Current owner APIs cannot support read load even after projection, caching, or aggregator alternatives are measured.

Failure modes and rollback implications:
- Projection lag can show stale category data; disclose freshness and keep correctness-critical paths off the projection.
- Cross-service schema reads can break independently deployed owners; rollback requires removing DB credentials and replacing direct queries with contracts.
- If a projection corrupts, rebuild from source owners or fall back to a slower read path; do not promote the projection to write authority.

### Example 2: Tax logic in a growing monolith
Context: A large monolith has tax calculation logic spread across checkout, cart, and order creation. The same organization still owns the full flow, and there is no independent runtime or data-isolation requirement yet.

Selected option: Create an internal domain component/module for tax with one public entrypoint, explicit request/response contract, and ownership of tax logic. Keep it in-process until independent deployability, isolation, or scaling evidence exists.

Rejected options:
- Immediate tax microservice extraction, because the first risk is unclear ownership and tangled dependencies, not network isolation.
- Keeping tax behavior scattered under callers, because each caller will keep encoding partial tax truth.
- A generic `common` or `util` package, because that hides the domain boundary and invites unrelated dependencies.

Evidence that would change the decision:
- Tax processing requires separate compliance isolation, secrets handling, release cadence, or runtime scaling.
- A separate team owns tax with independent deployability and support responsibility.
- The module's public contract becomes stable and the remaining in-process dependency graph is narrow enough for extraction.

Failure modes and rollback implications:
- A broad module interface becomes a second monolith inside the monolith; reduce the public surface before extraction.
- If the new module differs behaviorally, run old and new paths side by side and compare before switching.
- Rolling back an in-process componentization is usually a code-path switch or module dependency revert; rolling back a service extraction also involves routing, data ownership, and compatibility windows.

### Example 3: Payments vaulting or sensitive identity material
Context: A monolith handles a sensitive value that should not flow through broad product code, and the capability has stricter isolation and audit expectations than adjacent workflows.

Selected option: Consider a separate service or highly isolated runtime boundary if the sensitive capability has clear ownership, narrow API, independent threat model, and accepted operational cost.

Rejected options:
- Keeping the capability in a broad shared module only for local-call convenience.
- Splitting every adjacent payment workflow at the same time; isolate the sensitive truth first and leave orchestration boundaries explicit.
- Sharing the sensitive datastore directly with the monolith after extraction.

Evidence that would change the decision:
- Security and compliance review finds in-process isolation is enough and the team cannot operate a new runtime safely.
- The boundary still requires chatty synchronous calls across a long request path, indicating the split is premature.
- Data ownership cannot be made exclusive during rollout.

Failure modes and rollback implications:
- A split that keeps shared storage is fake isolation; rollback does not remove exposure until data access is also constrained.
- Extra network hops can put sensitive flows on a fragile critical path; require timeouts, idempotency, and clear degradation.
- If extraction fails after data authority moves, rollback may be a forward fix or route-back with dual-read validation, not a simple deploy revert.

## Source Links Gathered Through Exa
- Go, "Organizing a Go module": https://go.dev/doc/modules/layout
- Go Blog, "Package names": https://go.dev/blog/package-names
- Microservices.io, "Decompose by business capability": https://microservices.io/patterns/decomposition/decompose-by-business-capability.html
- Microservices.io, "Database per service": https://microservices.io/patterns/data/database-per-service.html
- Microservices.io, "Shared database": https://microservices.io/patterns/data/shared-database.html
- Shopify Engineering, "Deconstructing the Monolith": https://shopify.engineering/deconstructing-monolith-designing-software-maximizes-developer-productivity
- Shopify Engineering, "Componentizing Shopify's Tax Engine": https://engineering.shopify.com/blogs/engineering/componentizing-shopify-tax-engine
- AWS Prescriptive Guidance, "Anti-corruption layer pattern": https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/acl.html

