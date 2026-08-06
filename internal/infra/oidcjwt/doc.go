// Package oidcjwt authenticates strict OIDC JWT access tokens for the service
// transport adapters.
//
// A Verifier establishes trust once at startup, keeps it current in the
// background, and answers every transport this service exposes:
// [Verifier.ResolveHTTP] serves the OpenAPI security requirement.
// profile:grpc:start
// [Verifier.UnaryInterceptor] and [Verifier.StreamInterceptor] serve gRPC.
// profile:grpc:end
// What it publishes is deliberately minimal: a [reqctx.Principal] holding the
// opaque subject, and nothing else.
//
// # Extending it
//
// The package has no plug-in point, and that is a decision rather than an
// omission: an authentication boundary with configurable claim rules is one
// where a mistake is a configuration change instead of a code review. There is
// nothing to register, so extending it means editing this package:
//
//   - an additional claim requirement belongs in parseAccessTokenClaims in
//     token.go, beside the checks it joins. A claim whose own encoding has to be
//     policed first — more than one legal spelling, an entry that may not repeat
//     — needs a type of its own too; audienceClaim and numericDate in that file
//     are the two worked examples and say where the line falls. TestVerify_Claims
//     is where the accepting and rejecting cases go, and a row there can only
//     vary what tokenClaims carries: the term is added to that fixture and to
//     validClaims in oidcjwt_test.go once, rather than once per test that builds
//     a payload;
//   - propagating more than the subject — scopes, tenant, client id — means
//     filling more of [reqctx.Principal] where parseToken builds it. Read
//     [reqctx.Principal] first: Scopes is an authorization input, so anything
//     put there becomes an access decision;
//   - a new failure category is a Kind in errors.go. The mandatory exhaustive
//     linter then names every switch that must answer it: the message table
//     there, verificationReason, and the problem mapping in internal/infra/http.
//
// profile:grpc:start
//
//	The gRPC adapter's grpcAuthenticationError is a fourth.
//
// profile:grpc:end
//
//	  A default arm does not excuse one of them, so none
//	  can be forgotten silently. The label the new category joins reaches
//	  operators through docs/authentication.md, and
//	  TestDocumentedMetricReasonsMatchTheGuide fails until it is published there.
//	  Counting takes care of itself for a category [Verifier.Verify] returns and
//	  for one the shared verifyCredential rejects; a category an adapter decides
//	  on its own, before any credential is read, is the case that still has to
//	  call recordRejection;
//	- a new reason to fetch the key set is a refreshTrigger in refresh.go, and it
//	  lands in three places at once. The mandatory exhaustive linter names the
//	  first: rateLimited beside it has to classify the trigger, which is the
//	  decision of whether an attacker can drive that fetch. The value then
//	  reaches operators as a fourth authn.refresh.trigger series rather than a
//	  label, so docs/authentication.md has to publish it and
//	  TestDocumentedTriggersMatchTheGuide fails until it does. The third is who
//	  admits the fetch: refresh.go reaches begin from a key miss and waits, and
//	  Run reaches it from the cadence and selects, so a trigger that fits neither
//	  route needs one of its own;
//	- another carrier is another value of [Transport], and because each value is
//	  published verbatim as the authn.transport attribute it is also another
//	  metric series rather than a label, which is why it is declared in
//	  metrics.go beside recordVerification. The
//	  larger obligation is the transport-trust decision, and nothing here will
//	  ask for it: this package authenticates a credential, and whether the
//	  connection carrying it may be believed at all is a separate question.
//	  trustedHTTPRequest in http.go answers it per request, for HTTP;
//
// profile:grpc:start
//
//	validateGRPCAuthnTransport in internal/config answers it for gRPC once at
//	configuration load, so the two present carriers answer it in different
//	places and at different times and there is no shared owner to inherit an
//	answer from; grpcAuthenticationError says what that asymmetry costs to read;
//
// profile:grpc:end
//   - a second configured trust value is not one edit, and it lands in this
//     package and internal/config together. The list is below.
//
// # Adding a configured trust value
//
// Each of these carries part of the value. One added only in this package is
// never parsed; one added only in internal/config is never enforced:
//
//  1. a field on config.AuthnConfig with its koanf key;
//  2. its validation in that package's validateAuthnConfig. internal/config
//     cannot import this package, so it restates these rules rather than calling
//     them; validProviderURL in provider.go owns why;
//  3. a field on [PolicyInput];
//  4. the value on [Policy], with the rule that admits it in [NewPolicy];
//  5. the population of that field at the composition root, in
//     cmd/service/internal/bootstrap/startup_authn.go;
//  6. the key in env/.env.example;
//  7. the operator description in docs/authentication.md;
//  8. a refused case in TestPolicyRulesMatchConfigValidation that varies the new
//     value.
//
// Item 3 is the premise, since it is the edit that starts this, and two tools run
// from it. The exhaustruct entry for [PolicyInput] fails lint at every production
// call site that does not set the new field, which is item 5, and item 1 follows
// because the composition root needs somewhere to read the value from.
// TestPolicyRulesMatchConfigValidation then fails until its corpus holds a
// refused case for the new field, which is item 8; that case forces the rule in
// [NewPolicy] that refuses it, which is half of item 4; and because the test runs
// both owners over one corpus it stays red until internal/config refuses the
// value too, which is item 2.
//
// Item 2 is the expensive one to skip, and the reason it is worth holding a test
// to: without it internal/config accepts a value this package will refuse, so a
// deployment that mistypes the new key fails at authn startup instead of at
// configuration load — which is the entire reason that duplicated validation
// exists.
//
// Nothing asks for the rest. Item 4's other half is the value carried onto
// [Policy], which [NewPolicy] can validate and then drop, and items 6 and 7 leave
// a build green with no key in env/.env.example and no operator description in
// docs/authentication.md.
//
// # A second issuer
//
// A second issuer is the one extension this shape does not absorb. A Verifier
// owns one issuer's JWKS URI, key set, and refresh lifecycle, and parseToken
// matches the configured issuer while it decodes, so a dispatcher cannot read
// iss to pick between verifiers without decoding the token twice. Accepting two
// issuers means first splitting parseToken into a decode step and a policy
// step, then giving each issuer its own Verifier and Run; treat it as a design
// change to this package, not as configuration.
//
// Telemetry is the half of that change nothing will ask for. Every instrument
// here is per-package rather than per-issuer: meterName is one constant,
// recordVerification names the transport and not the issuer, and the callback
// registerKeyAgeGauge installs observes with no attributes at all. Two Verifiers
// therefore write into one series per instrument rather than two: the counters
// sum across issuers, and the age gauge reports whichever callback ran last. One
// issuer's outage is then indistinguishable from noise in the other's, so the
// signals that would show a partial failure are the ones hiding it. Give the
// instruments an issuer attribute in the same change.
//
// Those edits are a fork of the template's copy of this package, so a later
// template sync will report them as a conflict. That cost is intended: it is
// what makes a change to who may call this service visible in review.
//
// Authorization, roles, tenant policy, sessions, and user provisioning remain
// feature-owned work. See docs/authentication.md for the operator-facing
// contract and docs/repo-architecture.md for where this sits among the
// repository's extension seams.
package oidcjwt
