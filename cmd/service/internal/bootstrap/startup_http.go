package bootstrap

import (
	"fmt"
	stdlog "log"
	"log/slog"
	"net/http"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/failure"

	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"

	// profile:authn-bearer:start
	"github.com/getkin/kin-openapi/openapi3filter"
	// profile:authn-bearer:end
)

// httpRuntimeBindings is what this service composes into its HTTP transport,
// and is the counterpart of grpcRuntimeBindings: the handlers behind the
// generated contract, and the authentication its declared security requirements
// resolve through. Everything else the router needs comes from configuration.
type httpRuntimeBindings struct {
	Handlers httpx.Handlers
	// profile:authn-bearer:start
	Authenticate          openapi3filter.AuthenticationFunc
	AuthenticateChallenge string
	// profile:authn-bearer:end
}

// newHTTPHandler builds the hardened router for this service's own OpenAPI
// contract. httpx.Harden owns the middleware chain and its order; this function
// owns only the crossing from loaded configuration into httpx.RouterConfig.
func newHTTPHandler(
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	domainErrors []failure.Mapper,
	bindings httpRuntimeBindings,
) (http.Handler, error) {
	handler, err := httpx.NewRouter(
		log,
		bindings.Handlers,
		metrics,
		httpx.RouterConfig{
			MaxBodyBytes:    cfg.HTTP.MaxBodyBytes,
			RequestTimeout:  cfg.HTTP.RequestTimeout,
			MaxInFlight:     cfg.HTTP.MaxInFlight,
			OTelServerName:  cfg.Observability.OTel.ServiceName,
			LogHealthProbes: cfg.HTTP.AccessLogHealthProbes,
			// The active profile's dependency failures, classified once here
			// rather than in every operation. A service appends its own mappers
			// to the local domainErrors slice in run.go; see failure.Mapper for
			// why the seam exists.
			DomainErrors: domainErrors,
			// profile:authn-bearer:start
			Authenticate:          bindings.Authenticate,
			AuthenticateChallenge: bindings.AuthenticateChallenge,
			// profile:authn-bearer:end
		},
	)
	if err != nil {
		return nil, fmt.Errorf("build http router: %w", err)
	}
	return handler, nil
}

// newHTTPServer applies the configured wire-level bounds to the API listener.
//
// errorLog routes net/http's own reporting through the service logger. Unset, a
// malformed request line or a TLS handshake failure goes to stderr as
// unstructured text, which a JSON-on-stdout pipeline drops. It is supplied
// rather than built here because the diagnostics listener publishes through the
// same one.
func newHTTPServer(cfg config.HTTPConfig, handler http.Handler, errorLog *stdlog.Logger) *http.Server {
	return &http.Server{
		Handler:           handler,
		ErrorLog:          errorLog,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}
