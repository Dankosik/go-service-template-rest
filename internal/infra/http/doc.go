// Package httpx owns the HTTP transport edge for the service.
//
// The package name does not match its directory on purpose. Declaring it httpx
// rather than http is what lets a caller import both this package and net/http
// in one file without aliasing either.
//
// [NewRouter] mounts the generated contract and wraps it in this repository's
// middleware chain and fallback policy. [Harden] is that same chain without any
// generated type, so a service whose OpenAPI contract lives in its own package
// inherits the chain instead of rebuilding it by hand and silently getting none
// of it. [Harden]'s doc comment is the one prose owner of the chain order, and
// chain_order_test.go fails when that comment and the argument order it
// describes disagree — so the order is stated once, not here as well.
//
// # Extending it
//
// To add an operation, implement it and assign it to [Handlers.API]. That field
// is the composition seam: a service adds operations without editing any file in
// this package. See docs/repo-architecture.md for where it sits among the
// repository's extension seams.
//
// To make a domain error reach the caller as anything other than 500, classify
// it through [RouterConfig.DomainErrors]. The same mapper slice feeds the gRPC
// transport, which is what keeps one domain identity from answering 404 here and
// Internal there; there is deliberately no HTTP-only mapper seam.
//
// To charge callers against a budget, set [HardenConfig.RateLimit] and
// [HardenConfig.RateLimitKey] together. Their field comments own why neither has
// a default that a template could honestly guess.
//
// profile:http-idempotency-postgres:start
// To opt an authenticated business operation into idempotency, pass one complete
// [IdempotencyOperation] through [RouterConfig.IdempotencyOperations]. The
// router checks its OpenAPI declaration before serving; no registration leaves
// the health-only template inert.
// profile:http-idempotency-postgres:end
//
// # Where things live
//
// Assembly, hardening, and fallback are separate owners. router.go mounts this
// repository's generated contract; harden.go builds the reusable chain;
// router_policy.go answers a request the mounted contract did not match — 404,
// 405 with Allow, and CORS preflight — because each changes for its own reasons.
//
// Each middleware owns the file named for it, and the file states why that
// middleware exists rather than what it does. Two of them split further: the
// bounded token-bucket map behind RateLimit is ratelimit_keyed.go, which holds
// no net/http import at all, and middleware_guards.go carries the two guards too
// small to have earned a file each.
//
// The error path splits by which half of an exchange failed, not by status.
// request_errors.go answers a request the generated validator rejected;
// domain_errors.go answers an operation that returned an error instead of a
// typed response. Only the request half can name fields, which is why
// problem_violations.go sits beside it. Both render through problem.go, and the
// status, title, and type URI each code maps to belong to internal/problem,
// where a transport cannot reach them.
//
// Three leaves are shared rather than owned by their first caller, and each says
// so in its own header: route_template.go names the route a request matched, for
// the access log, the panic recorder, and the rejection callbacks alike;
// health_probe_routes.go recognizes the platform probe paths that the access log
// and the admission middlewares each treat differently; response_commit.go
// reports whether a response has already been committed, which the panic and
// timeout middlewares both have to know before writing.
//
// handlers.go owns the [Handlers] seam and the startup check that a contract
// which grew past the platform probes cannot reach a nil [Handlers.API].
// health_handlers.go is the two probe bodies, kept apart because they shadow the
// embedded interface. identity.go resolves the authenticated principal a secured
// operation runs as.
package httpx
