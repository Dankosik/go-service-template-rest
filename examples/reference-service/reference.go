// Package referenceservice is the composition root of the reference example: it
// wires one feature onto its generated contract and through the shared hardened
// middleware chain.
//
// It deliberately owns no process lifecycle. The version this replaced carried
// its own main, listener, signal handling, and shutdown — a hundred lines that
// duplicated cmd/service/internal/bootstrap and got it worse, in the file a
// reader opens first to learn how to compose a feature.
//
// What is worth copying is below: a feature package that knows nothing about
// HTTP, an httpapi package that maps it onto the generated contract, one
// classification table for its errors, an authentication seam that yields a
// principal, and httpx.Harden supplying everything else. NewHandler is exercised
// end to end over httptest, so the composition is proved rather than merely
// compiled.
package referenceservice

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/article/memory"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/httpapi"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// The transport bounds this example serves under. The real service reads
// equivalents from typed config; they are constants here so the example stays
// readable, but every one of them is set on purpose rather than left at a zero
// value.
const (
	// RequestTimeout is the per-request handler budget.
	RequestTimeout = 8 * time.Second
	maxBodyBytes   = 1 << 20
	maxInFlight    = 256

	// rateLimitPerSecond and rateLimitBurst bound one caller rather than the
	// whole instance. MaxInFlight stops the process from being overloaded; it
	// cannot stop one caller from taking every slot, which is what this covers.
	rateLimitPerSecond = 20
	rateLimitBurst     = 40

	// writeTokenSubject is who the demonstration credential stands for. A real
	// service resolves this from the credential rather than pinning it, but the
	// value has to be something a log line can attribute an action to.
	writeTokenSubject = "reference-writer"

	// authenticateChallenge names an HTTP authentication scheme, which is not the
	// same vocabulary as the contract's securityScheme key.
	authenticateChallenge = "Bearer"
)

// Options is what the caller of this example supplies.
type Options struct {
	// WriteToken is the demonstration credential for protected operations. It is
	// required rather than defaulted, so the example cannot run with a guessable
	// token baked into source. It is not an authentication design.
	WriteToken string
	// Seed is the content the in-memory repository starts with.
	Seed []article.Article
}

// NewHandler builds the reference service's HTTP handler.
//
// Composition happens here, not in the feature package: httpapi maps the feature
// onto its contract, and this root supplies the transport policy and wraps the
// result in the shared middleware chain.
func NewHandler(log *slog.Logger, opts Options) (http.Handler, error) {
	if log == nil {
		return nil, errors.New("reference service: logger is required")
	}
	if strings.TrimSpace(opts.WriteToken) == "" {
		return nil, errors.New("reference service: write token is required")
	}

	repository, err := memory.New(opts.Seed)
	if err != nil {
		return nil, fmt.Errorf("build article repository: %w", err)
	}
	// The repository is both the store and the unit of work. A PostgreSQL adapter
	// passes postgres.Pool.InTx behind the same article.Atomically port; nothing
	// in the feature package changes.
	articles, err := article.NewService(repository, repository)
	if err != nil {
		return nil, fmt.Errorf("build article service: %w", err)
	}

	// The credential lives only in this root. httpx.Authenticated turns the
	// identity it proves into a reqctx.Principal the handler can authorize
	// against, which is what keeps the scope check out of the credential-parsing
	// code and the credential out of the feature package.
	apiHandler, err := httpapi.NewAPIHandler(articles, httpapi.Options{
		Authenticate:  httpx.Authenticated(resolveWriter(opts.WriteToken)),
		RejectRequest: httpx.RejectRequest(log, authenticateChallenge),
		// One classification table for the whole feature. Handlers return their
		// use case's error and this decides what the client sees, so adding an
		// operation does not mean copying a switch — which is how the local
		// status table this replaced drifted and answered a 409 with the
		// internal-error type.
		RejectResponse: httpx.RejectResponse(log, article.ClassifyError),
	})
	if err != nil {
		return nil, fmt.Errorf("build reference api handler: %w", err)
	}

	rateLimit, err := httpx.NewKeyedRateLimiter(rateLimitPerSecond, rateLimitBurst, 0)
	if err != nil {
		return nil, fmt.Errorf("build reference rate limiter: %w", err)
	}

	// The chain instruments every request, so it needs a registry to record into.
	// This example exposes no scrape endpoint of its own.
	handler, err := httpx.Harden(log, telemetry.New(), httpx.HardenConfig{
		MaxBodyBytes:   maxBodyBytes,
		RequestTimeout: RequestTimeout,
		MaxInFlight:    maxInFlight,
		OTelServerName: "reference-service",
		RateLimit:      rateLimit,
		// Keyed on the credential rather than the resolved principal: the limiter
		// runs ahead of the generated validator, which is what resolves identity,
		// and limiting after schema validation means an over-budget caller still
		// got the expensive half of their request done for them. The key is
		// hashed, so the credential never becomes a map key.
		RateLimitKey: httpx.HeaderRateLimitKey("Authorization"),
	}, apiHandler)
	if err != nil {
		return nil, fmt.Errorf("build reference router: %w", err)
	}
	return handler, nil
}

// resolveWriter accepts exactly the configured demonstration credential and
// reports the caller it stands for.
//
// Two properties are worth copying and the rest is not. The comparison is
// constant time, so a wrong token cannot be recovered by timing. And what it
// returns is an identity with a scope rather than a bare nil: the operation's
// permission check then lives with the operation, in httpapi, instead of being
// implied by whichever credential happened to be accepted here. A real service
// replaces the token with its own identity design and keeps that shape.
func resolveWriter(expected string) httpx.PrincipalResolver {
	return func(_ context.Context, input *openapi3filter.AuthenticationInput) (reqctx.Principal, error) {
		if input == nil || input.SecurityScheme == nil {
			return reqctx.Principal{}, errors.New("missing security scheme")
		}
		if !strings.EqualFold(input.SecurityScheme.Type, "http") ||
			!strings.EqualFold(input.SecurityScheme.Scheme, "bearer") {
			return reqctx.Principal{}, fmt.Errorf("unsupported security scheme %q", input.SecuritySchemeName)
		}

		header := input.RequestValidationInput.Request.Header.Get("Authorization")
		presented, ok := strings.CutPrefix(header, "Bearer ")
		if !ok {
			return reqctx.Principal{}, errors.New("bearer credential is missing")
		}
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(presented)), []byte(expected)) != 1 {
			return reqctx.Principal{}, errors.New("bearer credential is invalid")
		}

		return reqctx.Principal{
			Subject: writeTokenSubject,
			Scopes:  []string{httpapi.ArticleWriteScope},
		}, nil
	}
}
