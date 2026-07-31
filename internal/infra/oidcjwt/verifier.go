package oidcjwt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/metric"
)

const (
	TransportHTTP = "http"
	TransportGRPC = "grpc"
)

type lifecycleState uint8

const (
	lifecycleNew lifecycleState = iota
	lifecycleRunning
	lifecycleJoined
	lifecycleClosed
)

type refreshCall struct {
	done chan struct{}
	err  error
}

// Verifier owns one issuer's immutable keys, refresh lifecycle, and both
// transport adapters.
type Verifier struct {
	policy        Policy
	jwksURI       string
	client        providerClient
	now           func() time.Time
	newTimer      timerFactory
	log           *slog.Logger
	metrics       authnMetrics
	keys          atomic.Pointer[keySet]
	install       chan struct{}
	unregisterAge func()

	refreshMu    sync.Mutex
	active       *refreshCall
	cooldownTill time.Time

	baseCtx context.Context //nolint:containedctx // Verifier owns this lifecycle context and cancels it in Close.
	cancel  context.CancelFunc

	lifecycleMu sync.Mutex
	state       lifecycleState
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
		func(duration time.Duration) verifierTimer {
			return newRealVerifierTimer(duration)
		},
		meterProvider,
		log,
	)
}

func newVerifier(
	ctx context.Context,
	policy Policy,
	factory clientFactory,
	now func() time.Time,
	newTimer timerFactory,
	meterProvider metric.MeterProvider,
	log *slog.Logger,
) (*Verifier, error) {
	if now == nil {
		now = time.Now
	}
	if newTimer == nil {
		newTimer = func(duration time.Duration) verifierTimer {
			return newRealVerifierTimer(duration)
		}
	}
	if log == nil {
		log = slog.Default()
	}
	keys, jwksURI, client, err := bootstrapTrust(ctx, policy, factory, now, log)
	if err != nil {
		return nil, err
	}
	baseCtx, cancel := context.WithCancel(context.Background())
	metricWarning := newAuthnMetricWarning(log)
	verifier := &Verifier{
		policy:   policy,
		jwksURI:  jwksURI,
		client:   client,
		now:      now,
		newTimer: newTimer,
		log:      log,
		metrics:  newAuthnMetrics(meterProvider, metricWarning),
		install:  make(chan struct{}, 1),
		baseCtx:  baseCtx,
		cancel:   cancel,
		state:    lifecycleNew,
		runDone:  make(chan struct{}),
	}
	verifier.keys.Store(keys)
	verifier.unregisterAge = registerKeyAgeGauge(meterProvider, verifier.keys.Load, now, metricWarning)
	verifier.metrics.recordRefresh(ctx, "startup", nil)
	return verifier, nil
}

// Verify validates compact and trust policy and returns only the opaque subject.
func (v *Verifier) Verify(ctx context.Context, compact, transport string) (principal reqctx.Principal, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			principal = reqctx.Principal{}
			err = failure(KindUnavailable)
		}
		v.metrics.recordVerification(ctx, transport, err)
	}()

	parsed, err := parseToken(compact, v.policy, v.now())
	if err != nil {
		return reqctx.Principal{}, err
	}
	snapshot := v.keys.Load()
	if snapshot == nil {
		return reqctx.Principal{}, failure(KindUnavailable)
	}
	key := snapshot.keys[parsed.keyID]
	if key != nil {
		payload, verifyErr := parsed.signed.Verify(key)
		if verifyErr == nil && bytes.Equal(payload, parsed.payload) {
			if !v.keysCurrent(snapshot) {
				return reqctx.Principal{}, failure(KindUnavailable)
			}
			return parsed.principal, nil
		}
	}

	if err := v.refresh(ctx, "key_miss", true); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return reqctx.Principal{}, err
		}
		if v.currentKeys() == nil {
			return reqctx.Principal{}, failure(KindUnavailable)
		}
	}
	current := v.currentKeys()
	if current == nil {
		return reqctx.Principal{}, failure(KindUnavailable)
	}
	key = current.keys[parsed.keyID]
	if key == nil {
		return reqctx.Principal{}, failure(KindInvalid)
	}
	payload, err := parsed.signed.Verify(key)
	if err != nil || !bytes.Equal(payload, parsed.payload) {
		return reqctx.Principal{}, failure(KindInvalid)
	}
	return parsed.principal, nil
}

// CheckReady reports whether a completely validated key set is still current.
func (v *Verifier) CheckReady() error {
	if v == nil || v.currentKeys() == nil {
		return failure(KindUnavailable)
	}
	return nil
}

func (v *Verifier) currentKeys() *keySet {
	keys := v.keys.Load()
	if !v.keysCurrent(keys) {
		return nil
	}
	return keys
}

func (v *Verifier) keysCurrent(keys *keySet) bool {
	return keys != nil && v.now().Before(keys.fetchedAt.Add(MaxKeySetAge))
}

func (v *Verifier) beginRefresh(trigger string, miss bool) *refreshCall {
	now := v.now()
	v.refreshMu.Lock()
	if v.active != nil {
		call := v.active
		v.refreshMu.Unlock()
		return call
	}
	if miss && now.Before(v.cooldownTill) {
		v.refreshMu.Unlock()
		return nil
	}
	call := &refreshCall{done: make(chan struct{})}
	v.active = call
	if miss {
		v.cooldownTill = now.Add(RefreshCooldown)
	}
	v.refreshMu.Unlock()

	go func(refreshCtx context.Context) {
		call.err = v.fetchAndInstall(refreshCtx)
		v.metrics.recordRefresh(context.WithoutCancel(refreshCtx), trigger, call.err)
		v.refreshMu.Lock()
		if v.active == call {
			v.active = nil
		}
		close(call.done)
		v.refreshMu.Unlock()
	}(v.baseCtx)
	return call
}

func (v *Verifier) refresh(ctx context.Context, trigger string, miss bool) error {
	call := v.beginRefresh(trigger, miss)
	if call == nil {
		return nil
	}
	return waitRefresh(ctx, call)
}

func waitRefresh(ctx context.Context, call *refreshCall) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("wait for JWKS refresh: %w", ctx.Err())
	case <-call.done:
		return call.err
	}
}

func (v *Verifier) fetchAndInstall(ctx context.Context) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("JWKS refresh failed")
		}
	}()
	body, err := fetchDocument(ctx, v.client.request, v.jwksURI)
	if err != nil {
		return errors.New("JWKS refresh failed")
	}
	candidate, err := parseKeySet(body, v.now())
	if err != nil {
		return errors.New("JWKS refresh failed")
	}
	v.keys.Store(candidate)
	select {
	case v.install <- struct{}{}:
	default:
	}
	return nil
}

// Run owns scheduled refresh and exact trust-current readiness transitions.
func (v *Verifier) Run(ctx context.Context, onTrustCurrent func(bool)) error {
	v.lifecycleMu.Lock()
	if v.state != lifecycleNew {
		v.lifecycleMu.Unlock()
		return errors.New("OIDC verifier lifecycle is invalid")
	}
	v.state = lifecycleRunning
	v.lifecycleMu.Unlock()
	defer func() {
		v.cancel()
		v.waitForActiveRefresh()
		v.lifecycleMu.Lock()
		if v.state == lifecycleRunning {
			v.state = lifecycleJoined
		}
		close(v.runDone)
		v.lifecycleMu.Unlock()
	}()

	var (
		published   bool
		lastCurrent bool
	)
	publishCurrent := func() {
		current := v.CheckReady() == nil
		if onTrustCurrent != nil && (!published || current != lastCurrent) {
			onTrustCurrent(current)
		}
		published = true
		lastCurrent = current
	}
	publishCurrent()

	refreshDue := v.keys.Load().fetchedAt.Add(RefreshInterval)
	refreshTimer := v.newTimer(until(v.now(), refreshDue))
	staleTimer := v.newTimer(until(v.now(), v.keys.Load().fetchedAt.Add(MaxKeySetAge)))
	var (
		scheduled     *refreshCall
		scheduledDone <-chan struct{}
	)
	defer refreshTimer.Stop()
	defer staleTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			v.cancel()
			return fmt.Errorf("run OIDC verifier: %w", ctx.Err())
		case <-v.baseCtx.Done():
			return context.Canceled
		case <-v.install:
			keys := v.keys.Load()
			refreshTimer.Reset(until(v.now(), keys.fetchedAt.Add(RefreshInterval)))
			staleTimer.Reset(until(v.now(), keys.fetchedAt.Add(MaxKeySetAge)))
			publishCurrent()
		case <-refreshTimer.C():
			scheduled = v.beginRefresh("scheduled", false)
			scheduledDone = scheduled.done
		case <-scheduledDone:
			next := RefreshCooldown
			if scheduled.err == nil {
				next = RefreshInterval
			}
			scheduled = nil
			scheduledDone = nil
			refreshTimer.Reset(next)
			publishCurrent()
		case <-staleTimer.C():
			publishCurrent()
		}
	}
}

func until(now, future time.Time) time.Duration {
	if !future.After(now) {
		return 0
	}
	return future.Sub(now)
}

func (v *Verifier) waitForActiveRefresh() {
	v.refreshMu.Lock()
	call := v.active
	v.refreshMu.Unlock()
	if call != nil {
		<-call.done
	}
}

// Close cancels and joins owned work and releases the JWKS connection pool.
func (v *Verifier) Close() {
	if v == nil {
		return
	}
	v.lifecycleMu.Lock()
	state := v.state
	if state == lifecycleClosed {
		v.lifecycleMu.Unlock()
		return
	}
	if state == lifecycleNew {
		v.state = lifecycleClosed
	}
	done := v.runDone
	v.lifecycleMu.Unlock()

	v.cancel()
	if state == lifecycleRunning {
		<-done
	} else {
		v.waitForActiveRefresh()
	}
	v.closeOnce.Do(func() {
		v.client.close()
		if v.unregisterAge != nil {
			v.unregisterAge()
		}
	})
	v.lifecycleMu.Lock()
	v.state = lifecycleClosed
	v.lifecycleMu.Unlock()
}
