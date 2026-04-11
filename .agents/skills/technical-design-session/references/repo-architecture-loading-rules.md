# Repository Architecture Loading Rules

Use this file to decide whether the stable repository baseline must be loaded before technical design. `docs/repo-architecture.md` is authoritative for this repository; external links are calibration only.

## When To Load
- Load when the design touches repository boundaries, runtime flow, ownership seams, new packages, dependency direction, generated contracts, async work, data flow, or bootstrap/app/infra edges.
- Load when rebuilding task-local design from `spec.md` rather than polishing one narrow existing seam.
- Load when existing design artifacts do not already capture the stable repository baseline.
- Skip only for a narrow continuation where the relevant baseline is already present in the approved design bundle.

## Good Design-Session Outputs
- "Loaded `docs/repo-architecture.md` because the change crosses HTTP transport, app behavior, persistence, and generated OpenAPI bindings."
- "Skipped architecture reload with rationale: continuation-only edit to one ownership note; current approved design bundle already quotes the relevant bootstrap/app/infra baseline."
- "Design keeps `internal/app` transport-agnostic and puts concrete HTTP behavior under `internal/infra/http`."
- "Design keeps generated code derived from `api/openapi/service.yaml` and does not hand-edit `internal/api` as authority."

## Bad Design-Session Outputs
- "I remember the repository shape, so I will not load the architecture baseline."
- "The new app service can import HTTP helpers because it is convenient."
- "The design updates generated bindings directly and treats them as the contract source."
- "The workflow plan says repository baseline required, but the session marks design complete without loading or citing it."

## Conditional Artifact Examples
- New package or adapter boundary after loading repo architecture can trigger `design/dependency-graph.md`.
- OpenAPI source-of-truth or generated-code flow changes can trigger `design/contracts/`.
- New persistence extension under `internal/infra/postgres` can trigger `design/data-model.md`.
- New durable background work can trigger `rollout.md` and possibly a component/sequence expansion around lifecycle ownership.

## Blocked Handoff Examples
- The task changes dependency direction and no baseline has been loaded or captured in current design artifacts.
- The design proposes a new async worker hidden inside HTTP handlers despite the repository baseline preferring an explicit lifecycle owner.
- The design cannot identify whether behavior belongs in app, infra, domain, or bootstrap.
- A proposed package boundary conflicts with `docs/repo-architecture.md`; route back to design repair or specification if the approved scope must change.

## Exa Source Links
Exa MCP search and fetch were attempted before writing these examples, but the provider returned a 402 credits-limit error. Treat these fallback-verified links as calibration only; repo-local files remain authoritative.

- C4 model: https://c4model.com/
- arc42 documentation: https://docs.arc42.org/home/
- Google Cloud Architecture Framework: https://docs.cloud.google.com/architecture/framework
- Azure Well-Architected Framework: https://learn.microsoft.com/en-us/azure/well-architected/what-is-well-architected-framework

