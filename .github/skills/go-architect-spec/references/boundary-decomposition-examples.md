# Boundary Decomposition Examples

## Behavior Change Thesis
When loaded for unclear boundary placement, this file makes the model choose invariant and ownership boundaries before topology, instead of splitting by table, entity name, handler package, or read-query convenience.

## When To Load
Load when the hard question is where a module, runtime, or service boundary belongs, who owns write truth, whether direct data dependency is acceptable, or whether Go package/module structure is being mistaken for service architecture.

## Decision Rubric
- Start with the invariant-bearing truth: the owner is the boundary that can accept or reject the business state change.
- Use domain capability, data ownership, team ownership, and transaction boundary as the decomposition axes; package names are enforcement aids, not proof.
- Prefer an in-process domain module when ownership is still one team, one datastore, and one transaction boundary.
- Treat steady-state cross-service DB reads as architecture coupling, even when "read-only".
- Justify sensitive-data isolation with exclusive data authority, narrow API, threat model, and operability, not with local-call convenience.
- Record extraction posture: `stay in-process`, `separate runtime`, `candidate for extraction`, or `service now`.

## Imitate

### Tax Logic In A Growing Monolith
Context: Tax calculation is spread across checkout, cart, and order creation. One organization owns the flow, and no independent runtime or data-isolation requirement exists yet.

Choose: create an internal tax domain module with one public entrypoint and explicit ownership of tax logic. Keep it in-process until compliance isolation, team ownership, runtime scaling, or stable extraction evidence appears.

Copy: this separates domain truth from caller sprawl without pretending network isolation is the first fix.

### Category Composition With Stale Reads
Context: `catalog` owns product mutation. Category pages need product, price, and inventory summaries that can be stale, while checkout cannot use stale price or inventory.

Choose: keep product, price, and inventory write truth with their owners. Serve category pages from a derived read path or BFF with a freshness contract. Keep checkout on owner commands or correctness-critical reads.

Copy: this rejects direct DB joins because the issue is composed reads, not a new write owner.

### Sensitive Vault Boundary
Context: a broad product monolith handles sensitive identity or payment material that should not flow through unrelated product code.

Choose: consider a separate service or isolated runtime only if the sensitive capability has clear owner, narrow API, exclusive data authority, independent threat model, and accepted operational cost.

Copy: this isolates the sensitive truth first and avoids splitting adjacent workflows just because they are nearby.

## Reject
- "Create one service per table." Bad because table shape is not ownership, transaction scope, or runtime isolation.
- "Put shared business structs in `common` so modules can move faster." Bad because it hides the domain boundary and invites cross-module invariants.
- "A read service can join private service databases because it never writes." Bad because schemas become integration contracts and releases coordinate through the database.
- "Extract now and leave the old monolith writing the same data until later." Bad because it creates competing write authority unless one side is explicitly non-authoritative and bounded.

## Agent Traps
- Do not overfit to Go directory layout. Use `internal/` packages and module boundaries to enforce decisions after ownership is clear.
- Do not treat high read volume as service-extraction proof. Read scale can often use projections, caches, BFFs, or worker runtimes without moving write truth.
- Do not collapse orchestration into a peer module when one application layer coordinates multiple module-owned truths.
- Do not call a boundary "independent" while it still shares direct database access or coordinated release requirements.
