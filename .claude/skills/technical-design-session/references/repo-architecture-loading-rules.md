# Repository Architecture Loading Rules

## Behavior Change Thesis
When loaded for a design pass with uncertain repository baseline needs, this file makes the model load or cite `docs/repo-architecture.md` before boundary decisions instead of relying on memory, inventing package ownership, or designing against generated/runtime surfaces as if they were authority.

## When To Load
Load when the design touches repository boundaries, runtime flow, ownership seams, new packages, dependency direction, generated contracts, async work, data flow, or bootstrap/app/infra edges.

## Decision Rubric
- Load `docs/repo-architecture.md` for fresh non-trivial design passes and for changes crossing transport, app orchestration, domain, persistence, config/bootstrap, generated contracts, async lifecycle, or dependency direction.
- Skip only for a narrow continuation where the current approved design bundle already captures the relevant repository baseline and the edit is local to one known seam.
- Record a skip rationale if you do not load it; "I remember the repo" is not a rationale.
- Use the architecture baseline to constrain ownership and dependency direction, not to rewrite approved scope from `spec.md`.
- If proposed design conflicts with the architecture baseline in a planning-critical way, block for design repair or route back to `specification` when the approved scope must change.

## Imitate
```markdown
Loaded `docs/repo-architecture.md`.
Reason: the design crosses HTTP transport, app behavior, persistence, async worker lifecycle, and generated OpenAPI bindings.
Design consequence: keep app orchestration transport-agnostic and put HTTP response mapping under infra/http.
```

Copy this shape: it names both the load trigger and the design consequence.

```markdown
Skipped architecture reload.
Reason: continuation-only edit to one ownership note; current approved `design/ownership-map.md` already captures the bootstrap/app/infra baseline for this seam.
```

Copy this shape: a skip is narrow and evidence-backed.

```markdown
Generated contract authority: `api/openapi/service.yaml`.
Design consequence: `design/contracts/` may capture planning context, but generated output is not hand-edited or treated as source of truth.
```

Copy this shape: it keeps design-only contract context below canonical sources.

## Reject
```markdown
I know this repo shape well enough, so I will design the package boundary without reloading the architecture baseline.
```

Failure: memory is not evidence for a cross-boundary design pass.

```markdown
The new app service can import HTTP helpers because the endpoint needs response formatting.
```

Failure: convenience crosses ownership boundaries that the architecture baseline may forbid.

```markdown
Mark design complete even though workflow control required the architecture baseline and it was not loaded or captured.
```

Failure: the planning handoff lacks repository-fit proof.

## Agent Traps
- Treating `docs/repo-architecture.md` as optional because `spec.md` is detailed.
- Letting generated code, not the canonical contract source, become the owner of API behavior.
- Hiding lifecycle ownership for background work inside HTTP handlers.
- Using the architecture baseline as a reason to expand scope rather than as a constraint on the approved spec.
