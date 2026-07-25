package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/article/memory"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/httpapi"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// The bounds this example serves under. The real service reads equivalents from
// typed config; they are constants here so the example stays one readable binary,
// but every one of them is set on purpose rather than left at a zero value.
const (
	listenAddress   = ":8080"
	shutdownTimeout = 5 * time.Second

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 5 * time.Second
	// requestTimeout stays below writeTimeout so the budget expires while the
	// connection can still carry the 504 that reports it.
	requestTimeout = 8 * time.Second
	writeTimeout   = 10 * time.Second
	idleTimeout    = 60 * time.Second
	maxHeaderBytes = 16 << 10
	maxBodyBytes   = 1 << 20
	maxInFlight    = 256

	// writeTokenEnv supplies the demonstration credential for protected
	// operations. It is not an authentication design; see
	// docs/first-production-feature.md before building a real one.
	writeTokenEnv = "REFERENCE_WRITE_TOKEN"

	// writeTokenSubject is who the demonstration credential stands for. A real
	// service resolves this from the credential rather than pinning it, but the
	// value has to be something a log line can attribute an action to.
	writeTokenSubject = "reference-writer"

	// authenticateChallenge names an HTTP authentication scheme, which is not the
	// same vocabulary as the contract's securityScheme key.
	authenticateChallenge = "Bearer"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	// logctx is what makes a handler's own records joinable to the access log line
	// and the trace for the same request. A logger built from a bare JSONHandler
	// looks identical and silently carries none of that correlation.
	log := slog.New(logctx.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, log); err != nil {
		log.Error("reference_service_stopped", "err", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, log *slog.Logger) error {
	repository, err := memory.New([]article.Article{{
		Slug:      "clear-owners",
		Title:     "Keep responsibilities with their owner",
		Summary:   "A transport maps HTTP, a use case owns behavior, and an adapter owns storage details.",
		Published: true,
	}})
	if err != nil {
		return fmt.Errorf("build article repository: %w", err)
	}
	articles, err := article.NewService(repository)
	if err != nil {
		return fmt.Errorf("build article service: %w", err)
	}
	// The demonstration credential is required rather than defaulted, so the
	// example cannot start with a guessable write token baked into source.
	writeToken := os.Getenv(writeTokenEnv)
	if strings.TrimSpace(writeToken) == "" {
		return fmt.Errorf("%s is required to run the reference example", writeTokenEnv)
	}

	// Composition happens here, not in the feature package: httpapi maps the
	// feature onto its contract, and this root supplies the transport policy and
	// wraps the result in the shared middleware chain.
	//
	// The credential lives only in this root. httpx.Authenticated turns the
	// identity it proves into a reqctx.Principal the handler can authorize
	// against, which is what keeps the scope check out of the credential-parsing
	// code and the credential out of the feature package.
	apiHandler, err := httpapi.NewAPIHandler(articles, httpapi.Options{
		Authenticate:   httpx.Authenticated(resolveWriter(writeToken)),
		RejectRequest:  httpx.RejectRequest(log, authenticateChallenge),
		RejectResponse: httpx.RejectResponse(),
	})
	if err != nil {
		return fmt.Errorf("build reference api handler: %w", err)
	}

	// The chain instruments every request, so it needs a registry to record into.
	// This example exposes no scrape endpoint of its own.
	handler, err := httpx.Harden(log, telemetry.New(), httpx.RouterConfig{
		MaxBodyBytes:   maxBodyBytes,
		RequestTimeout: requestTimeout,
		MaxInFlight:    maxInFlight,
		OTelServerName: "reference-service",
	}, apiHandler)
	if err != nil {
		return fmt.Errorf("build reference router: %w", err)
	}

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("listen for reference HTTP: %w", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	log.InfoContext(ctx, "reference_service_started", "http.addr", listener.Addr().String())
	return serve(ctx, log, listener, handler)
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

func serve(ctx context.Context, log *slog.Logger, listener net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler: handler,
		// Without ErrorLog, net/http reports malformed request lines and any panic
		// that escapes the chain as unstructured text on stderr, beside this
		// service's JSON on stdout.
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve reference HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		closeErr := server.Close()
		serveErr := <-serverErr
		var stopErr error
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			stopErr = fmt.Errorf("stop reference HTTP: %w", serveErr)
		}
		return errors.Join(
			fmt.Errorf("shutdown reference HTTP: %w", err),
			closeErr,
			stopErr,
		)
	}
	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("stop reference HTTP: %w", err)
	}
	return nil
}
