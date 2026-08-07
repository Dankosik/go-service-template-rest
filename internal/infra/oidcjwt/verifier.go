package oidcjwt

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/metric"
)

// Verifier owns one issuer's immutable keys, refresh lifecycle, and both
// transport adapters.
//
// Each field group below names the regime that guards it, and refresh admission
// is one more that [refreshAdmission] owns outright. Mixing them is what a
// change here has to avoid.
type Verifier struct {
	// Fixed for the Verifier's whole life; safe to read without synchronization.
	policy        Policy
	jwksURI       string
	client        providerClient
	now           func() time.Time
	log           *slog.Logger
	metrics       authnMetrics
	unregisterAge func()
	admission     *refreshAdmission

	// keys is replaced wholesale, never mutated, so every verification reads one
	// consistent set without blocking a refresh. It is non-nil from newVerifier
	// onward and nothing ever stores nil over it, which is why Run reads fetchedAt
	// straight off it. installed carries one non-blocking nudge per successful
	// replacement to whatever Run is scheduling.
	keys      atomic.Pointer[keySet]
	installed chan struct{}

	// Fixed for the Verifier's whole life as well, and grouped apart because it
	// is the lifetime rather than a value: baseCtx is the Verifier's own lifetime
	// expressed as a context. Close cancels it, and that cancellation is the stop
	// signal for Run and for any refresh in flight. refreshAdmission.begin hands
	// it to each fetch it launches and says what a fetch needs from it.
	baseCtx context.Context //nolint:containedctx // Verifier owns this lifecycle context and cancels it in Close.
	cancel  context.CancelFunc

	// lifecycleMu guards the two questions Run and Close ask each other: has Run
	// started, and has this Verifier been retired. Between them they admit at
	// most one Run and one shutdown. runDone reports that Run has left, and
	// closeOnce both releases the owned client and gauge exactly once and holds a
	// second Close until the first has finished. lifecycle.go owns this group; it
	// is the only file that reads or writes any of it.
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
	verifier := &Verifier{
		policy:    policy,
		jwksURI:   trust.jwksURI,
		client:    trust.client,
		now:       now,
		log:       log,
		metrics:   newAuthnMetrics(meterProvider, reportDegraded),
		installed: make(chan struct{}, 1),
		baseCtx:   baseCtx,
		cancel:    cancel,
		runDone:   make(chan struct{}),
	}
	verifier.keys.Store(trust.keys)
	// admission and unregisterAge both refer to the verifier being built, so
	// neither can move into the literal above.
	verifier.admission = &refreshAdmission{owner: verifier}
	verifier.unregisterAge = registerKeyAgeGauge(meterProvider, verifier.keys.Load, now, reportDegraded)
	verifier.metrics.recordRefresh(ctx, triggerStartup, nil)
	return verifier, nil
}

// Transport names the carrier a verification arrived on. It is a named type
// rather than a plain string so the accepted set is closed by the compiler, and
// each value is published verbatim as the authn.transport metric attribute, so a
// third is a third metric series rather than a label. Adding one is an extension
// the package documentation owns.
type Transport string

const (
	TransportHTTP Transport = "http"
	TransportGRPC Transport = "grpc"
)

// Verify checks the compact token against trust policy and returns only the
// opaque subject.
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
// sanitized failure and so one metric record covers every exit.
func (v *Verifier) Verify(
	ctx context.Context,
	compact string,
	transport Transport,
) (principal reqctx.Principal, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			principal = reqctx.Principal{}
			err = failure(KindUnavailable)
			// The category is the honest answer to the caller and says nothing to
			// whoever has to fix it; logRecoveredPanic owns that split.
			logRecoveredPanic(ctx, v.log, "verify", recovered)
		}
		v.metrics.recordVerification(ctx, transport, err)
	}()

	parsed, err := parseToken(compact, v.policy, v.now())
	if err != nil {
		return reqctx.Principal{}, err
	}

	// Two facts decide every answer below: whether an installed key signs this
	// token, and whether the set holding it is still current.
	//
	// Only a key miss is worth a refresh. It is what a provider rotation looks
	// like from here, and one coalesced, cooldown-limited fetch is the recovery;
	// a refresh the cooldown refused or the provider failed leaves the installed
	// set in place, so the decision below still answers from it. Staleness gets
	// no second attempt: this token already matched, so only a replacement the
	// scheduled cadence owns can clear the age.
	snapshot := v.keys.Load()
	signed := snapshot.verifies(parsed)
	if !signed {
		refreshErr := v.refresh(ctx)
		if callerAborted(refreshErr) {
			return reqctx.Principal{}, refreshErr
		}
		snapshot = v.keys.Load()
		signed = snapshot.verifies(parsed)
	}

	// Age answers before the signature, and the categories differ because the
	// question does: against a current set, "no key signs this" is a complete
	// answer and the credential is invalid, while against a set we could not
	// replace the matching key may be in the one the refresh failed to fetch —
	// so the honest report is that trust is unavailable.
	if !v.keysCurrent(snapshot) {
		return reqctx.Principal{}, failure(KindUnavailable)
	}
	if !signed {
		return reqctx.Principal{}, failure(KindInvalid)
	}
	return parsed.principal, nil
}

// verifyCredential extracts the bearer credential one transport carried and
// verifies it.
//
// Both adapters reach [Verifier.Verify] through here, and that is what keeps
// authn.verifications the complete count of what this boundary answered. Verify
// counts every one of its own exits from a single deferred record, so a
// credential rejected before it — a missing, duplicated, or oversized header —
// would otherwise go uncounted. Routing the extraction and the count through one
// function means an adapter cannot forget the second.
func (v *Verifier) verifyCredential(
	ctx context.Context,
	values []string,
	transport Transport,
) (reqctx.Principal, error) {
	token, err := bearerToken(values)
	if err != nil {
		return reqctx.Principal{}, v.recordRejection(ctx, transport, err)
	}
	return v.Verify(ctx, token, transport)
}

// recordRejection counts a refusal [Verifier.Verify] never saw, and returns that
// same error.
//
// There are exactly two, and both reach it. One is verifyCredential's own, just
// above: bearerToken refused the header, so Verify — and the deferred record that
// counts every one of its exits — never ran. The other belongs to an adapter
// alone, which is the only party that can know a request never arrived in a shape
// this boundary may authenticate; the HTTP adapter is today's only such caller,
// for a request it cannot reach and for a peer the deployment does not trust.
func (v *Verifier) recordRejection(ctx context.Context, transport Transport, err error) error {
	v.metrics.recordVerification(ctx, transport, err)
	return err
}

// CheckReady reports whether a completely validated key set is still current.
func (v *Verifier) CheckReady() error {
	if !v.keysCurrent(v.keys.Load()) {
		return failure(KindUnavailable)
	}
	return nil
}

// keysCurrent is the single staleness policy every caller asks: a set past
// MaxKeySetAge is refused even though it is still installed and would still
// verify signatures. MaxKeySetAge owns why. A nil set is not current either, so
// callers need no separate guard.
func (v *Verifier) keysCurrent(keys *keySet) bool {
	return keys != nil && v.now().Before(keys.fetchedAt.Add(MaxKeySetAge))
}
