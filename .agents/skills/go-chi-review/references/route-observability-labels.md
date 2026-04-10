# Route Observability Labels

## When To Read
Read this when a diff touches metrics, traces, logs, span names, route labels, `http.route`, route-template extraction, unmatched-route fallback labels, or middleware that records route identity.

## Review Smell Patterns
- Metrics, spans, or logs label routes with `r.URL.Path`, wildcard captures, IDs, slugs, query strings, or raw unmatched paths.
- `http.route` is populated from the URI path when chi route templates are not available.
- Route labels are read before `next.ServeHTTP`, so nested routes or mounted subrouters produce incomplete templates.
- Traces use one route identity while metrics or logs use another.
- Unmatched routes fall back to raw user-controlled paths instead of one bounded label such as `unmatched`.

## Minimal Examples

```diff
 func routeMetrics(next http.Handler) http.Handler {
   return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
-    labels.Route = r.URL.Path
     next.ServeHTTP(w, r)
+    labels.Route = chi.RouteContext(r.Context()).RoutePattern()
+    if labels.Route == "" {
+      labels.Route = "unmatched"
+    }
     metrics.Record(labels)
   })
 }
```

Review finding shape: raw paths create high-cardinality telemetry. The smallest safe fix is route-template labels after routing plus a bounded unmatched fallback.

```diff
-span.SetName(r.Method + " " + r.URL.Path)
+if route := chi.RouteContext(r.Context()).RoutePattern(); route != "" {
+  span.SetName(r.Method + " " + route)
+  span.SetAttributes(attribute.String("http.route", route))
+}
```

Review finding shape: server span names should use method plus a low-cardinality route target when it is available; do not substitute a raw URL path for an unavailable route template.

## Validation
- Send `/users/1` and `/users/2` through the middleware and assert one recorded route label, such as `/users/{id}`.
- Send two unknown paths and assert both collapse to the same bounded fallback label.
- Assert metrics, trace span name, `http.route`, and structured logs use the same route-template source.
- Suggested validation command: `go test ./... -run 'Test.*RouteLabel|Test.*Observability|Test.*Telemetry|Test.*Trace|Test.*Metrics'`.

## Sources Gathered With Exa
- [OpenTelemetry HTTP semantic conventions](https://opentelemetry.io/docs/specs/semconv/http/)
- [OpenTelemetry HTTP span semantic conventions](https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/http/)
- [go-chi context.go source for RoutePattern timing](https://raw.githubusercontent.com/go-chi/chi/master/context.go)
- [go-chi mux.go source for request route pattern assignment](https://raw.githubusercontent.com/go-chi/chi/master/mux.go)

