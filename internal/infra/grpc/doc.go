// Package grpcx owns the native gRPC server transport adapter.
//
// The package name does not match its directory on purpose: every file here also
// imports google.golang.org/grpc, so importers alias this one as grpcx rather
// than shadow the library everything in it is written against.
//
// [NewServer] returns a [Server] that already carries standard health,
// OpenTelemetry server instrumentation, finite transport bounds, and this
// repository's interceptor policy. A composition root supplies what it owns
// through [Options] and nothing else: this package never imports a feature
// package, and generated handlers never import this one.
//
// [Server] adapts grpc.Server to the process runtime's bounded lifecycle. Health
// is composed from separate inputs rather than assigned, so each caller records
// only its own: [Server.MarkServing] publishes the startup admission result, and
// [Server.StartDrain] is terminal, which is what stops a late input from making a
// draining server serving again. [Server.Shutdown] waits for active RPCs until
// its context expires, then forces the remainder.
//
// # Extending it
//
// Three seams cover the expected changes, all of them fields on [Options].
//
// To add a service, append a [RegisterService] to [Options.Services].
// Registration runs inside [NewServer], before Serve, because the telemetry
// filter closes over the resulting method set and then reads it from every RPC
// goroutine without synchronization — which is why there is no exported
// "register onto a running Server".
//
// To add authentication, authorization, or another cross-cutting policy, append
// to [Options.UnaryPolicy] and [Options.StreamPolicy]. Append rather than assign:
// a build profile may have filled them already, and an assignment compiles,
// passes every check, and silently drops what it replaced. A streaming policy
// that enriches the context declares its own grpc.ServerStream wrapper, because a
// policy package importing this one is the dependency direction [Options] exists
// to avoid. A policy that answers with a status returns a plain status.Error: the
// boundary above the policy slot trusts any status the error carries.
// profile:authn-oidc-jwt:start
//
// internal/infra/oidcjwt/grpc.go is the worked example.
// profile:authn-oidc-jwt:end
//
// To add a policy to this package's own chain, write one aroundRPC and add it to
// the list the builtinPolicies function in chain.go returns. That list is the
// only place the order below is decided, and both chains are built from it, so a
// policy cannot reach unary RPCs and miss streaming ones. It is a function rather than a
// literal inside [NewServer] because performance_test.go builds a subset of the
// same chain from it, and TestBenchmarkVariantsCoverEveryBuiltinPolicy fails
// until that file decides whether the new policy should be measured.
//
// To make a domain error reach the caller as anything other than INTERNAL,
// classify it through [Options.DomainErrors]. A handler returns its domain error,
// problem.Classify maps that to a problem.Code, and this package renders the gRPC
// code. The same mapper slice feeds the HTTP transport, which is what keeps one
// domain identity from answering 404 over HTTP and Internal over gRPC; there is
// deliberately no gRPC-only mapper seam. A raw status.Error returned from a
// handler carries no provenance this package trusts and is sanitized to INTERNAL.
//
// # Interceptor order
//
// [NewServer] builds one chain per RPC kind from a single ordered list, outermost
// first. grpc-go runs the first interceptor as the outer wrapper and the last as
// the innermost one around the handler:
//
//   - correlation — accepts or mints the request ID and publishes it in response
//     metadata, so everything below it and the handler agree on one identifier.
//   - access log — times the whole RPC, and sits outside both error boundaries
//     so it records the status the caller actually receives.
//   - recovery — turns a panic below it into INTERNAL, which is also what lets
//     the access log record the RPC instead of losing it with the goroutine.
//   - admission — holds the concurrency semaphore for the work below it, outside
//     the policy slot, so an RPC occupies a slot before any supplied policy runs.
//   - policy error boundary — sanitizes what the policy interceptors return.
//   - [Options.UnaryPolicy] and [Options.StreamPolicy] — supplied by the
//     composition root.
//   - handler error boundary — sanitizes what the handler returns.
//
// This package doc is the one prose owner of that order; docs/grpc.md describes
// what the chain guarantees a caller and points here for the positions.
//
// The two error boundaries are the same mechanism and differ only in how much of
// an error they already trust, which is what fixes their order: a policy
// interceptor is service-owned code that may choose its own status, and a
// handler's error is not. One consequence is worth knowing before writing a
// policy: the handler error boundary is innermost, so a policy observes an
// already-mapped status and never the handler's raw domain error.
//
// # One policy, both RPC kinds
//
// grpc-go types unary and streaming interceptors separately, so a policy written
// against those types exists twice and the two copies drift silently: adding one
// half compiles, lints, and passes any test that does not specifically drive
// streaming. A policy here is one aroundRPC instead, and asUnaryInterceptor and
// asStreamInterceptor adapt it to both types.
//
// Correlation is the exception and is still written twice, because the unary and
// streaming halves genuinely differ: one publishes response metadata through
// grpc.SetHeader, the other through the stream and must also replace the context
// the handler observes.
//
// See docs/grpc.md for the service author's contract — the proto and generation
// workflow, the full problem.Code to gRPC code table, and the bootstrap
// registration step — and docs/repo-architecture.md for where this sits among the
// repository's extension seams.
package grpcx
