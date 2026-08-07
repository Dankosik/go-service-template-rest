// Package oidcjwt authenticates strict OIDC JWT access tokens for the service
// transport adapters.
//
// A Verifier establishes trust once at startup, keeps it current in the
// background, and answers every transport this service exposes.
// [Verifier.ResolveHTTP] serves the OpenAPI security requirement.
// profile:grpc:start
// [Verifier.UnaryInterceptor] and [Verifier.StreamInterceptor] serve gRPC.
// profile:grpc:end
// What it publishes is deliberately minimal: a [reqctx.Principal] holding the
// opaque subject, and nothing else.
//
// # Extending it
//
// There is no plug-in point, and that is a decision: an authentication boundary
// with configurable claim rules turns a mistake into a configuration change
// instead of a code review. Extending the package means editing it:
//
//   - a claim requirement joins the checks in parseAccessTokenClaims, token.go.
//     A claim whose encoding needs policing first — several legal spellings, an
//     entry that may not repeat — also needs a type; audienceClaim and
//     numericDate are the worked examples. Accepting and rejecting cases go in
//     TestVerify_Claims, and the term is added to validClaims in oidcjwt_test.go
//     once rather than per test;
//   - propagating more than the subject means filling more of
//     [reqctx.Principal] where parseToken builds it. Scopes is an authorization
//     input, so anything put there becomes an access decision;
//   - a failure category is a Kind in errors.go. The exhaustive linter then
//     names every switch that must answer it, so none is forgotten silently.
//     Only the operator guide is unchecked by the compiler, and
//     TestDocumentedMetricReasonsMatchTheGuide closes that. A category an
//     adapter decides on its own, before any credential is read, is the one
//     that must still call recordRejection;
//   - a reason to fetch the key set is a refreshTrigger in refresh.go.
//     rateLimited must classify it, which is the decision of whether an attacker
//     can drive the fetch, and TestDocumentedTriggersMatchTheGuide holds it to
//     docs/authentication.md;
//   - a carrier is a value of [Transport], published verbatim as the
//     authn.transport attribute and therefore its own metric series, declared in
//     metrics.go beside recordVerification. The larger obligation is transport
//     trust, and nothing here will ask for it: trustedHTTPRequest in http.go
//     answers that per request, for HTTP;
//   - a configured trust value lands in this package and internal/config
//     together. The list is below.
//
// profile:grpc:start
// gRPC answers transport trust elsewhere and at a different time:
// validateGRPCAuthnTransport in internal/config, once at configuration load.
// grpcAuthenticationError says what that asymmetry costs to read.
// profile:grpc:end
//
// # Adding a configured trust value
//
// One added only here is never parsed; one added only in internal/config is
// never enforced:
//
//  1. a field on config.AuthnConfig with its koanf key;
//  2. its validation in validateAuthnConfig. internal/config cannot import this
//     package, so it restates the rules rather than calling them;
//     validProviderURL in provider.go owns why;
//  3. a field on [PolicyInput];
//  4. the value on [Policy], with the rule that admits it in [NewPolicy];
//  5. its population at the composition root, in
//     cmd/service/internal/bootstrap/startup_authn.go;
//  6. the key in env/.env.example;
//  7. the operator description in docs/authentication.md;
//  8. a refused case in TestPolicyRulesMatchConfigValidation.
//
// Tools cover all but two: exhaustruct fails lint at every call site that does
// not set the new field, and TestPolicyRulesMatchConfigValidation runs both
// owners over one corpus, so it stays red until internal/config refuses the
// value too. Item 2 is the expensive one to skip — without it a mistyped key
// fails at authn startup instead of at configuration load, which is the whole
// reason the validation is duplicated. Items 6 and 7 leave a build green.
//
// # A second issuer
//
// A Verifier owns one issuer's JWKS URI, key set, and refresh lifecycle, and
// parseToken matches the configured issuer while it decodes, so a dispatcher
// cannot read iss to pick a verifier without decoding the token twice. Two
// issuers means splitting parseToken into a decode step and a policy step, then
// giving each issuer its own Verifier and Run: a design change, not
// configuration.
//
// Telemetry is the half of that nothing will ask for. Every instrument is
// per-package: meterName is one constant, recordVerification names the transport
// and not the issuer, and registerKeyAgeGauge observes with no attributes. Two
// Verifiers therefore share one series per instrument — counters sum across
// issuers, and the age gauge reports whichever callback ran last — so one
// issuer's outage is indistinguishable from noise in the other's. Give the
// instruments an issuer attribute in the same change.
//
// These edits fork the template's copy, so a later template sync reports a
// conflict. That cost is intended: it makes a change to who may call this
// service visible in review.
//
// Authorization, roles, tenant policy, sessions, and user provisioning remain
// feature-owned. See docs/authentication.md for the operator contract and
// docs/repo-architecture.md for the repository's extension seams.
package oidcjwt
