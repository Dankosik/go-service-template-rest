package oidcjwt

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/time/rate"
)

type contextKey uint8

const refreshFailureKey contextKey = iota

type refreshFailure struct {
	failed atomic.Bool
}

// Verifier owns one issuer's parser, cached JWKS resolver, transport adapters,
// and refresh lifetime.
type Verifier struct {
	policy    Policy
	parser    *jwt.Parser
	keyFunc   func(context.Context) jwt.Keyfunc
	metrics   authnMetrics
	cancel    context.CancelFunc
	closeIdle func()
	closeOnce sync.Once
}

// New discovers the issuer's JWKS, installs the first key set synchronously,
// and starts the library-owned refresh loop.
func New(
	ctx context.Context,
	policy Policy,
	meterProvider metric.MeterProvider,
	log *slog.Logger,
) (*Verifier, error) {
	if log == nil {
		log = slog.Default()
	}
	jwksURI, err := discoverJWKSURI(ctx, policy, meterProvider)
	if err != nil {
		return nil, err
	}
	jwksClient, closeIdle, err := newJWKSClient(jwksURI, meterProvider)
	if err != nil {
		return nil, err
	}
	processCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	metrics := newAuthnMetrics(meterProvider)
	returnFirstError := false
	keys, err := keyfunc.NewDefaultOverrideCtx(processCtx, []string{jwksURI}, keyfunc.Override{
		Client:                    jwksClient,
		HTTPTimeout:               ProviderTimeout,
		NoErrorReturnFirstHTTPReq: &returnFirstError,
		RateLimitWaitMax:          time.Nanosecond,
		RefreshInterval:           RefreshInterval,
		RefreshUnknownKID:         rate.NewLimiter(rate.Every(RefreshCooldown), 1),
		RefreshErrorHandlerFunc: func(string) func(context.Context, error) {
			return func(refreshCtx context.Context, _ error) {
				if refreshCtx.Err() != nil {
					return
				}
				if observed, ok := refreshCtx.Value(refreshFailureKey).(*refreshFailure); ok {
					observed.failed.Store(true)
				}
				metrics.recordRefreshFailure(context.WithoutCancel(refreshCtx))
				log.WarnContext(refreshCtx, "authn_jwks_refresh_failed", "component", "authn")
			}
		},
		ValidationSkipAll: false,
	})
	if err != nil {
		cancel()
		closeIdle()
		return nil, failure(KindUnavailable)
	}
	signingKeys, err := keyfunc.New(keyfunc.Options{
		Ctx:          processCtx,
		Storage:      keys.Storage(),
		UseWhitelist: []jwkset.USE{"", jwkset.UseSig},
	})
	if err != nil {
		cancel()
		closeIdle()
		return nil, failure(KindUnavailable)
	}
	return newVerifier(policy, signingKeys.KeyfuncCtx, time.Now, metrics, cancel, closeIdle), nil
}

func newVerifier(
	policy Policy,
	keyFunc func(context.Context) jwt.Keyfunc,
	now func() time.Time,
	metrics authnMetrics,
	cancel context.CancelFunc,
	closeIdle func(),
) *Verifier {
	if now == nil {
		now = time.Now
	}
	if cancel == nil {
		cancel = func() {}
	}
	if closeIdle == nil {
		closeIdle = func() {}
	}
	return &Verifier{
		policy: policy,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{AllowedAlgorithm}),
			jwt.WithIssuer(policy.issuer),
			jwt.WithAudience(policy.audience),
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(ClockSkew),
			jwt.WithStrictDecoding(),
			jwt.WithJSONNumber(),
			jwt.WithTimeFunc(now),
		),
		keyFunc:   keyFunc,
		metrics:   metrics,
		cancel:    cancel,
		closeIdle: closeIdle,
	}
}

// Close stops library-owned refresh work and releases idle provider connections.
func (v *Verifier) Close() {
	v.closeOnce.Do(func() {
		v.cancel()
		v.closeIdle()
	})
}

type transport string

const (
	transportHTTP transport = "http"
	transportGRPC transport = "grpc"
)

func (v *Verifier) verifyCredential(ctx context.Context, values []string, carrier transport) (parsedToken, error) {
	compact, err := bearerToken(values)
	if err != nil {
		v.metrics.recordVerification(ctx, carrier, err)
		return parsedToken{}, err
	}
	return v.verifyToken(ctx, compact, carrier)
}

func (v *Verifier) recordRejection(ctx context.Context, carrier transport, err error) error {
	v.metrics.recordVerification(ctx, carrier, err)
	return err
}

func (v *Verifier) verifyToken(ctx context.Context, compact string, carrier transport) (parsed parsedToken, result error) {
	defer func() { v.metrics.recordVerification(ctx, carrier, result) }()
	if len(compact) > MaxTokenBytes {
		return parsedToken{}, failure(KindOversize)
	}
	refresh := new(refreshFailure)
	verifyCtx := context.WithValue(ctx, refreshFailureKey, refresh)
	claims := new(accessTokenClaims)
	token, err := v.parser.ParseWithClaims(compact, claims, func(token *jwt.Token) (any, error) {
		if v.policy.strictRFC9068() && !validAccessTokenType(token.Header["typ"]) {
			return nil, failure(KindInvalid)
		}
		if v.keyFunc == nil {
			return nil, failure(KindUnavailable)
		}
		return v.keyFunc(verifyCtx)(token)
	})
	if err != nil || token == nil || !token.Valid {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return parsedToken{}, fmt.Errorf("verify access token: %w", ctxErr)
		}
		if refresh.failed.Load() {
			return parsedToken{}, failure(KindUnavailable)
		}
		return parsedToken{}, failure(KindInvalid)
	}
	principal, err := principalFromClaims(claims, v.policy.strictRFC9068())
	if err != nil || claims.ExpiresAt == nil {
		return parsedToken{}, failure(KindInvalid)
	}
	return parsedToken{principal: principal, expiresAt: claims.ExpiresAt.Time}, nil
}

var _ http.RoundTripper = jwksRoundTripper{}
