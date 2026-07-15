# Reference Selector

| Pressure | Load | Required effect |
| --- | --- | --- |
| Root/subrouter shape, prefix ownership, generated/manual coexistence, conflicts, or wildcards | [router-topology-patterns.md](router-topology-patterns.md) | Choose one path owner and route-inventory proof. |
| Global/scoped middleware, exact order, context mutation, recovery, limits, logging, or generated wrappers | [middleware-layering-patterns.md](middleware-layering-patterns.md) | Specify scope and outer-to-inner behavior. |
| `NotFound`, `MethodNotAllowed`, `Allow`, `HEAD`, `OPTIONS`, preflight, fallback JSON, or CORS placement | [notfound-methodnotallowed-options-cors.md](notfound-methodnotallowed-options-cors.md) | Select explicit fallback/preflight mechanics and header proof. |
| Generated strict handlers, mount prefix, `BaseURL`, generated/manual ownership, or source authority | [openapi-oapi-codegen-integration.md](openapi-oapi-codegen-integration.md) | Prevent generated edits, double prefixes, and ownership collision. |
| Logs, metrics, traces, span names, `RoutePattern()`, `Match`/`Find`, or fallback identity | [route-template-observability.md](route-template-observability.md) | Use low-cardinality labels after route resolution. |
| Topology, fallback, order, preflight, route coverage, or labels need proof | [router-validation-test-patterns.md](router-validation-test-patterns.md) | Produce a risk-focused transport test matrix. |
