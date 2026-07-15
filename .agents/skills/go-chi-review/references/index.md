# Reference Selector

State which runtime judgment the selected reference changes.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| Late `Use`, construction order, `Route`/`Mount`, wildcard/subtree ownership, duplicate owner, nil mount. | [chi-router-registration-hazards.md](chi-router-registration-hazards.md) | Treat startup and subtree ownership as runtime defects instead of style or generic duplicates. |
| Middleware order/scope changes across `Use`, `With`, `Group`, `Route`, or `Mount`. | [middleware-order-and-scope.md](middleware-order-and-scope.md) | Prove exact coverage/order instead of assuming nested refactors preserve behavior. |
| `RouteContext`, `RoutePattern`, `Match`, `Find`, custom `Allow`/`OPTIONS`, or method probing. | [route-context-and-match-probing.md](route-context-and-match-probing.md) | Read route identity after resolution and probe with isolated contexts. |
| `NotFound`, `MethodNotAllowed`, `Allow`, `HEAD`, `OPTIONS`, CORS, preflight, or fallback wrappers. | [http-fallback-head-options-cors.md](http-fallback-head-options-cors.md) | Verify actual router capability and contract instead of inferred/hardcoded method support. |
| OpenAPI/generated chi handlers, generated/manual overlap, subtree wrappers, or generated no-touch files. | [generated-and-manual-route-drift.md](generated-and-manual-route-drift.md) | Preserve one source/route owner and policy parity instead of shadow routes or generated edits. |
| Metrics/traces/logs/span names, `http.route`, route extraction, or unmatched labels. | [route-observability-labels.md](route-observability-labels.md) | Use shared bounded route-template labels instead of raw or inconsistent identities. |
