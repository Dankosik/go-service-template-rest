# Reference Selector

State which runtime judgment the selected reference changes. Both branches use this selector: the branch decides what you return, the route node decides what you read.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| Middleware added, moved, or reordered; a new top-level path; generated-handler wiring; a second router beside the built one. | [route-tree-and-middleware.md](route-tree-and-middleware.md) | Compose onto the existing chain and the contract-owned route tree instead of building a parallel router or trusting the generated middleware slice to run first-to-first. |
| `NotFound`, `MethodNotAllowed`, `Allow`, `HEAD`, `OPTIONS`, preflight, or method probing with `Match`/`Find`. | [fallbacks-and-method-policy.md](fallbacks-and-method-policy.md) | Reuse the router's fallback owner and keep `Allow` derived from the live route tree with isolated probe contexts. |
| A log field, metric label, span name, or `http.route` derived from the request path. | [route-labels.md](route-labels.md) | Take route identity from the existing extractor after routing completes, and treat an empty template as "no route matched" rather than a reason to use the raw path. |
