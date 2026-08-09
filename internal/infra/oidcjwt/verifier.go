package oidcjwt

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Verifier owns one issuer's trusted keys, refresh lifecycle, and both transport
// adapters.
//
// Its state is split across four owners, and each guards itself: the fields
// below are immutable, [trustStore] owns the installed keys, [refreshAdmission]
// owns which fetches run, and the lifecycle group owns Run and Close. Mixing
// them is what a change here has to avoid.
type Verifier struct {
	policy        Policy
	jwksURI       string
	client        providerClient
	now           func() time.Time
	jitter        func(time.Duration) time.Duration
	log           *slog.Logger
	metrics       authnMetrics
	unregisterAge func()
	trust         *trustStore
	admission     *refreshAdmission

	// baseCtx is the Verifier's own lifetime expressed as a context, which is
	// what a fetch outliving the request that triggered it runs under. Close
	// cancels it, and that cancellation is the stop signal for Run and for any
	// refresh in flight.
	baseCtx context.Context //nolint:containedctx // Verifier owns this lifecycle context and cancels it in Close.
	cancel  context.CancelFunc

	// lifecycleMu guards the two questions Run and Close ask each other: has Run
	// started, and has this Verifier been retired. Between them they admit at
	// most one Run and one shutdown. runDone reports that Run has left, and
	// closeOnce both releases the owned client and gauge exactly once and holds a
	// second Close until the first has finished. lifecycle.go is the only file
	// that reads or writes any of it.
	lifecycleMu sync.Mutex
	runStarted  bool
	retired     bool
	runDone     chan struct{}
	closeOnce   sync.Once
}

// New establishes initial OIDC trust synchronously. It starts no goroutine.
func New(
	ctx context.Context,
	policy Policy,
	meterProvider metric.MeterProvider,
	log *slog.Logger,
) (*Verifier, error) {
	return newVerifier(
		ctx,
		policy,
		productionClientFactory(meterProvider),
		time.Now,
		meterProvider,
		log,
	)
}

func newVerifier(
	ctx context.Context,
	policy Policy,
	factory clientFactory,
	now func() time.Time,
	meterProvider metric.MeterProvider,
	log *slog.Logger,
) (*Verifier, error) {
	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.Default()
	}
	trust, err := bootstrapTrust(ctx, policy, factory, now, log)
	if err != nil {
		return nil, err
	}
	baseCtx, cancel := context.WithCancel(context.Background())
	reportDegraded := newDegradedWarning(log)
	store := newTrustStore(trust.keys)
	verifier := &Verifier{
		policy:        policy,
		jwksURI:       trust.jwksURI,
		client:        trust.client,
		now:           now,
		jitter:        refreshJitter,
		log:           log,
		metrics:       newAuthnMetrics(meterProvider, reportDegraded),
		unregisterAge: registerKeyAgeGauge(meterProvider, store.current, now, reportDegraded),
		trust:         store,
		baseCtx:       baseCtx,
		cancel:        cancel,
		runDone:       make(chan struct{}),
	}
	// admission refers to the verifier being built, so it cannot move into the
	// literal above.
	verifier.admission = &refreshAdmission{owner: verifier}
	verifier.metrics.recordRefresh(ctx, triggerStartup, nil)
	return verifier, nil
}

// transport names the adapter a verification arrived through. Keeping it
// private prevents package consumers from bypassing the adapter's credential
// stripping and transport-trust checks while choosing arbitrary metric labels.
type transport string

const (
	transportHTTP transport = "http"
	transportGRPC transport = "grpc"
)

// verifyToken returns the parsed token only after its signature and current
// trust have both been verified.
//
// Two error shapes leave here and an adapter has to answer both. Almost every
// failure is an [Error] carrying a [Kind], and the mandatory exhaustive linter
// holds every switch on one to the full set. The exception is callerAborted,
// which carries no Kind and so takes a switch on the Kind alone through its
// default arm; that predicate owns why, and an adapter tests it first.
// profile:grpc:start
// grpcAuthenticationError is the worked example.
// profile:grpc:end
// No linter names that obligation, because it is not a member of any enum.
//
// The results are named so the deferred recovery can replace a panic with a
// sanitized failure, and so one metric record covers every exit.
func (v *Verifier) verifyToken(
	ctx context.Context,
	compact string,
	transport transport,
) (verified parsedToken, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			verified = parsedToken{}
			err = failure(KindUnavailable)
			logRecoveredPanic(ctx, v.log, "verify", recovered)
		}
		v.metrics.recordVerification(ctx, transport, err)
	}()

	parsed, err := parseToken(compact, v.policy, v.now())
	if err != nil {
		return parsedToken{}, err
	}

	// Only a key miss is worth a refresh: it is what a provider rotation looks
	// like from here, and one coalesced, cooldown-limited fetch is the recovery. A
	// refresh the cooldown refused or the provider failed leaves the installed set
	// in place, so the decision below still answers from it. Staleness gets no
	// second attempt — this token already matched, so only a replacement the
	// scheduled cadence owns can clear the age.
	snapshot := v.trust.current()
	signed := snapshot.verifies(parsed)
	if !signed {
		refreshErr := v.refresh(ctx)
		if callerAborted(refreshErr) {
			return parsedToken{}, refreshErr
		}
		snapshot = v.trust.current()
		signed = snapshot.verifies(parsed)
	}

	// Age answers before the signature, and the categories differ because the
	// question does: against a current set, "no key signs this" is a complete
	// answer and the credential is invalid, while against a set we could not
	// replace the matching key may be in the one the refresh failed to fetch —
	// so the honest report is that trust is unavailable.
	if !v.keysCurrent(snapshot) {
		return parsedToken{}, failure(KindUnavailable)
	}
	if !signed {
		return parsedToken{}, failure(KindInvalid)
	}
	return parsed, nil
}

// verifyCredential extracts the bearer credential one transport carried and
// verifies it.
//
// Both adapters reach verifyToken through here, which is what keeps
// authn.verifications the complete count of what this boundary answered:
// verifyToken counts its own exits, but a header refused before it — missing,
// duplicated, or oversized — would otherwise go uncounted. Pairing the
// extraction with the count in one function means an adapter cannot take the
// first without the second.
func (v *Verifier) verifyCredential(
	ctx context.Context,
	values []string,
	transport transport,
) (parsedToken, error) {
	token, err := bearerToken(values)
	if err != nil {
		return parsedToken{}, v.recordRejection(ctx, transport, err)
	}
	return v.verifyToken(ctx, token, transport)
}

// recordRejection counts a refusal verifyToken never saw, and returns that same
// error. An adapter calls it directly for what only an adapter can know: that a
// request never arrived in a shape this boundary may authenticate at all.
//
// One refusal deliberately does not reach it, and naming that here is what keeps
// the count meaning something: errUnsupportedSecurityScheme is a requirement
// this boundary declined to answer rather than a credential it judged, so
// counting it would inflate the very series that reports how often callers are
// refused. Its declaration in http.go owns the rest of why.
func (v *Verifier) recordRejection(ctx context.Context, transport transport, err error) error {
	v.metrics.recordVerification(ctx, transport, err)
	return err
}

// CheckReady reports whether a completely validated key set is still current.
func (v *Verifier) CheckReady() error {
	if !v.keysCurrent(v.trust.current()) {
		return failure(KindUnavailable)
	}
	return nil
}

// keysCurrent is the single staleness policy every caller asks: a set past
// MaxKeySetAge is refused even though it is still installed and would still
// verify signatures. MaxKeySetAge owns why. The nil arm is unreachable through
// [trustStore] and answers fail-closed anyway, which is also what the deep lint
// gate's nil analysis requires of a dereference behind a pointer parameter.
func (v *Verifier) keysCurrent(keys *keySet) bool {
	return keys != nil && v.now().Before(keys.fetchedAt.Add(MaxKeySetAge))
}
