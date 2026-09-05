# Route Labels

## Load When
Load when a change reads route identity, adds a signal labelled by route, or changes what an unmatched request is labelled.

## What Already Owns This
`routePathTemplateForRequest` in `internal/infra/http/middleware_access_log.go` is the single source of route identity for the access log label and the `http.route` attribute. A second extractor is how two signals start disagreeing about the same request.

Server span *names* are otelhttp's, taken from `r.Pattern` and not from this extractor, so an unmatched request produces the span name `GET /*` while its `http.route` is correctly absent. That divergence is intended; changing it means configuring otelhttp, not the extractor.

## Route Identity Exists Only After Routing
Read it after `next.ServeHTTP`. Before that, routing has not run and the pattern is whatever the outermost mount matched.

chi v5.3.1 sets `r.Pattern` to the accumulated template across mounts and subrouters — the full route, not the leaf — so it is a sound fallback when the chi route context is unavailable. What neither source gives you is a template when nothing matched: an unmatched or wrong-method request under the root mount reports `/` or `/*`, and a mounted non-chi handler reports its mount wildcard. `normalizeRoutePathTemplate` collapses those to empty, which is the signal that no route matched.

An empty template means `http.route` is omitted and the log carries the bounded constant `<unmatched>`. Health-probe log suppression matches on the routed template for the same reason: a request that merely looks like a probe path did not match one.

## Reject
- Substituting `r.URL.Path` when the template is empty: the caller then chooses the label, and the metric's cardinality is unbounded.
- Setting `http.route` to a fallback string: it is defined as the matched route, so a synthetic value makes route-based queries silently wrong instead of empty.

## Proof
Two concrete parameter values collapse to one label; two unknown paths collapse to one fallback. Assert the `http.route` attribute directly: a span whose name carries a route is not evidence that the attribute was set, and the unmatched case is exactly where the two diverge.
