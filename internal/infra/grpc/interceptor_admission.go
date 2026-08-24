package grpcx

import (
	"context"
	"fmt"
	"sync"

	"github.com/example/go-service-template-rest/internal/failure"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
)

const (
	serverMeterName           = "service.grpc.server"
	activeRPCsInstrument      = "rpc.server.active_requests"
	shedRPCsInstrument        = "rpc.server.shed_requests"
	healthShedRPCsInstrument  = "rpc.server.health.shed_requests"
	rpcInstrumentUnit         = "{rpc}"
	activeRPCsDescription     = "RPCs currently executing a handler, against the grpc.server.max_concurrent_rpcs limit."
	shedRPCsDescription       = "RPCs rejected before running a handler because the process RPC limit was reached."
	healthShedRPCsDescription = "Standard health RPCs rejected before running a handler because the health admission limit was reached."
	drainingFailureDetail     = "server is draining"
)

// admissionPolicy routes an RPC to the budget that owns it. The two budgets are
// separate because they answer to different owners: business concurrency is
// sized from what the service can serve, while the standard health service's is
// sized from how many peers may be connected.
//
// Over-matching the health prefix here costs an RPC the wrong budget, both of
// them finite — unlike the trust decision [isHealthMethod] warns against, which
// is why routing may share that definition and a public-method allowlist may
// not.
type admissionPolicy struct {
	business *admissionLimiter
	health   *admissionLimiter
	drain    *rpcDrain
}

// newAdmissionPolicy builds both budgets. Only business admissions contribute
// to active load: a standing health watch per connected peer would turn active
// into a peer count. Health refusals still use their dedicated signal because a
// failed watch makes a health-aware client stop selecting the backend.
func newAdmissionPolicy(businessLimit, healthLimit int, load loadRecorder) admissionPolicy {
	drain := newRPCDrain()
	return admissionPolicy{
		business: newAdmissionLimiter(businessLimit, load, drain),
		health:   newAdmissionLimiter(healthLimit, healthShedRecorder{loadRecorder: load}, nil),
		drain:    drain,
	}
}

func (p admissionPolicy) statsHandler() drainStatsHandler { return drainStatsHandler{drain: p.drain} }

type healthShedRecorder struct {
	loadRecorder
}

type loadRecorder interface {
	Admitted(ctx context.Context) func()
	Shed(ctx context.Context)
	HealthShed(ctx context.Context)
}

type serverLoad struct {
	active     metric.Int64UpDownCounter
	shed       metric.Int64Counter
	healthShed metric.Int64Counter
}

func newServerLoad(provider metric.MeterProvider) serverLoad {
	meter := provider.Meter(serverMeterName)
	active, err := meter.Int64UpDownCounter(
		activeRPCsInstrument,
		metric.WithDescription(activeRPCsDescription),
		metric.WithUnit(rpcInstrumentUnit),
	)
	if err != nil {
		otel.Handle(fmt.Errorf("create %s metric: %w", activeRPCsInstrument, err))
	}
	shed, err := meter.Int64Counter(
		shedRPCsInstrument,
		metric.WithDescription(shedRPCsDescription),
		metric.WithUnit(rpcInstrumentUnit),
	)
	if err != nil {
		otel.Handle(fmt.Errorf("create %s metric: %w", shedRPCsInstrument, err))
	}
	healthShed, err := meter.Int64Counter(
		healthShedRPCsInstrument,
		metric.WithDescription(healthShedRPCsDescription),
		metric.WithUnit(rpcInstrumentUnit),
	)
	if err != nil {
		otel.Handle(fmt.Errorf("create %s metric: %w", healthShedRPCsInstrument, err))
	}
	return serverLoad{active: active, shed: shed, healthShed: healthShed}
}

func (l serverLoad) Admitted(ctx context.Context) func() {
	if l.active == nil {
		return func() {}
	}
	l.active.Add(ctx, 1)
	return func() { l.active.Add(ctx, -1) }
}

func (l serverLoad) Shed(ctx context.Context) {
	if l.shed != nil {
		l.shed.Add(ctx, 1)
	}
}

func (l serverLoad) HealthShed(ctx context.Context) {
	if l.healthShed != nil {
		l.healthShed.Add(ctx, 1)
	}
}

func (healthShedRecorder) Admitted(context.Context) func() { return func() {} }

func (r healthShedRecorder) Shed(ctx context.Context) { r.HealthShed(ctx) }

// around holds one slot from the owning budget for the work below it. One policy
// value backs both chains, which is what makes each budget process-wide rather
// than per RPC kind.
func (p admissionPolicy) around(ctx context.Context, fullMethod string, call func(context.Context) error) error {
	switch {
	case isHealthCheck(fullMethod):
		// The one public probe holds no slot at all, so a saturated instance —
		// saturated in either budget — stays observable to its platform.
		return call(ctx)
	case isHealthMethod(fullMethod):
		return p.health.around(ctx, call)
	default:
		return p.business.around(ctx, call)
	}
}

type admissionLimiter struct {
	sem   *semaphore.Weighted
	load  loadRecorder
	drain *rpcDrain
}

func newAdmissionLimiter(limit int, load loadRecorder, drain *rpcDrain) *admissionLimiter {
	return &admissionLimiter{
		sem:   semaphore.NewWeighted(int64(limit)),
		load:  load,
		drain: drain,
	}
}

func (l *admissionLimiter) around(ctx context.Context, call func(context.Context) error) error {
	if l.drain != nil && l.drain.drainingNow() {
		return ownedStatus(codes.Unavailable, drainingFailureDetail)
	}
	if !l.sem.TryAcquire(1) {
		l.load.Shed(ctx)
		return ownedStatus(codes.ResourceExhausted, failure.AtCapacityDetail)
	}
	defer l.sem.Release(1)
	release := l.load.Admitted(ctx)
	defer release()
	return call(ctx)
}

type rpcDrain struct {
	mu       sync.Mutex
	active   int
	draining bool
	done     chan struct{}
}

func newRPCDrain() *rpcDrain { return &rpcDrain{done: make(chan struct{})} }

func (d *rpcDrain) track() (func(), bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.draining {
		return nil, false
	}
	d.active++
	return func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.active--
		if d.draining && d.active == 0 {
			close(d.done)
		}
	}, true
}

func (d *rpcDrain) drainingNow() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.draining
}

func (d *rpcDrain) start() <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.draining {
		d.draining = true
		if d.active == 0 {
			close(d.done)
		}
	}
	return d.done
}

type drainRPCContextKey struct{}

type drainStatsHandler struct {
	drain *rpcDrain
}

func (h drainStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	if isHealthMethod(info.FullMethodName) {
		return ctx
	}
	release, tracked := h.drain.track()
	if !tracked {
		return ctx
	}
	return context.WithValue(ctx, drainRPCContextKey{}, release)
}

func (drainStatsHandler) HandleRPC(ctx context.Context, event stats.RPCStats) {
	if _, ended := event.(*stats.End); !ended {
		return
	}
	if release, ok := ctx.Value(drainRPCContextKey{}).(func()); ok {
		release()
	}
}

func (drainStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (drainStatsHandler) HandleConn(context.Context, stats.ConnStats) {}
