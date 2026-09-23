package oidcjwt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/time/rate"
)

var _ bearerauthn.Verifier = (*Verifier)(nil)

type contextKey uint8

const refreshFailureKey contextKey = iota

type refreshFailure struct {
	failed atomic.Bool
}

// Verifier owns one issuer's parser, cached JWKS resolver, and refresh lifetime.
type Verifier struct {
	policy    Policy
	parser    *jwt.Parser
	keyFunc   func(context.Context) jwt.Keyfunc
	cancel    context.CancelFunc
	closeIdle func()
	closeOnce sync.Once
}

// New uses ctx for issuer discovery, then loads the first key set synchronously
// with a process-owned context and starts the library-owned refresh loop. The
// key fetch uses the provider HTTP timeout; canceling ctx does not stop key
// loading or refresh. Close stops refresh after construction.
func New(
	ctx context.Context,
	policy Policy,
	meterProvider metric.MeterProvider,
	log *slog.Logger,
) (*Verifier, error) {
	if log == nil {
		log = slog.Default()
	}
	jwksURI, err := discoverJWKSURI(ctx, policy)
	if err != nil {
		return nil, err
	}
	jwksClient, closeIdle, err := newJWKSClient(jwksURI)
	if err != nil {
		return nil, err
	}
	processCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	metrics := newJWKSMetrics(meterProvider)
	ignoreFirstHTTPRequestError := false
	keys, err := keyfunc.NewDefaultOverrideCtx(processCtx, []string{jwksURI}, keyfunc.Override{
		Client:                    jwksClient,
		HTTPTimeout:               providerTimeout,
		NoErrorReturnFirstHTTPReq: &ignoreFirstHTTPRequestError,
		RateLimitWaitMax:          time.Nanosecond,
		RefreshInterval:           refreshInterval,
		RefreshUnknownKID:         rate.NewLimiter(rate.Every(refreshCooldown), 1),
		RefreshErrorHandlerFunc: func(string) func(context.Context, error) {
			return func(refreshCtx context.Context, _ error) {
				if !shouldReportRefreshFailure(processCtx, refreshCtx) {
					return
				}
				if observed, ok := refreshCtx.Value(refreshFailureKey).(*refreshFailure); ok {
					observed.failed.Store(true)
				}
				eventCtx := context.WithoutCancel(refreshCtx)
				metrics.recordRefreshFailure(eventCtx)
				log.WarnContext(eventCtx, "authn_jwks_refresh_failed", "component", "authn")
			}
		},
		ValidationSkipAll: false,
	})
	if err != nil {
		cancel()
		closeIdle()
		return nil, errors.New("OIDC startup failed at JWKS load")
	}
	signingKeys, err := keyfunc.New(keyfunc.Options{
		Ctx:          processCtx,
		Storage:      keys.Storage(),
		UseWhitelist: []jwkset.USE{"", jwkset.UseSig},
	})
	if err != nil {
		cancel()
		closeIdle()
		return nil, errors.New("OIDC startup failed at JWKS key selection")
	}
	return newVerifier(policy, signingKeys.KeyfuncCtx, time.Now, cancel, closeIdle), nil
}

func shouldReportRefreshFailure(processCtx, refreshCtx context.Context) bool {
	if processCtx.Err() != nil {
		return false
	}
	_, requestRefresh := refreshCtx.Value(refreshFailureKey).(*refreshFailure)
	return !requestRefresh || refreshCtx.Err() == nil
}

func newVerifier(
	policy Policy,
	keyFunc func(context.Context) jwt.Keyfunc,
	now func() time.Time,
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
			jwt.WithValidMethods([]string{allowedAlgorithm}),
			jwt.WithIssuer(policy.issuer),
			jwt.WithAudience(policy.audience),
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(bearerauthn.ClockSkew),
			jwt.WithStrictDecoding(),
			jwt.WithJSONNumber(),
			jwt.WithTimeFunc(now),
		),
		keyFunc:   keyFunc,
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

// Verify implements bearerauthn.Verifier for one already-parsed compact JWT.
func (v *Verifier) Verify(ctx context.Context, compact string) (bearerauthn.Result, error) {
	if len(compact) > bearerauthn.MaxTokenBytes {
		return bearerauthn.Result{}, bearerauthn.VerificationFailure(bearerauthn.KindOversize)
	}
	refresh := new(refreshFailure)
	verifyCtx := context.WithValue(ctx, refreshFailureKey, refresh)
	claims := new(accessTokenClaims)
	token, err := v.parser.ParseWithClaims(compact, claims, func(token *jwt.Token) (any, error) {
		if v.policy.strictRFC9068() && !validAccessTokenType(token.Header["typ"]) {
			return nil, bearerauthn.VerificationFailure(bearerauthn.KindInvalid)
		}
		if v.keyFunc == nil {
			return nil, bearerauthn.VerificationFailure(bearerauthn.KindUnavailable)
		}
		return v.keyFunc(verifyCtx)(token)
	})
	if err != nil || token == nil || !token.Valid {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return bearerauthn.Result{}, fmt.Errorf("verify access token: %w", ctxErr)
		}
		// A failed JWKS refresh makes verification unavailable even when the
		// token could otherwise be rejected as invalid.
		if refresh.failed.Load() {
			return bearerauthn.Result{}, bearerauthn.VerificationFailure(bearerauthn.KindUnavailable)
		}
		return bearerauthn.Result{}, bearerauthn.VerificationFailure(bearerauthn.KindInvalid)
	}
	principal, ok := principalFromClaims(claims, v.policy.strictRFC9068())
	if !ok || claims.ExpiresAt == nil {
		return bearerauthn.Result{}, bearerauthn.VerificationFailure(bearerauthn.KindInvalid)
	}
	return bearerauthn.Result{Principal: principal, ExpiresAt: claims.ExpiresAt.Time}, nil
}
