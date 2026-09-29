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
}

// newAdmissionPolicy builds both budgets. Only business admissions contribute
// to active load: a standing health watch per connected peer would turn active
// into a peer count. Health refusals still use their dedicated signal because a
// failed watch makes a health-aware client stop selecting the backend.
//
// drain refuses business RPCs once Shutdown starts; health never consults it.
func newAdmissionPolicy(businessLimit, healthLimit int, load serverLoad, drain *rpcDrain) admissionPolicy {
	return admissionPolicy{
		business: &admissionLimiter{
			sem:    semaphore.NewWeighted(int64(businessLimit)),
			shed:   load.shed,
			active: load.active,
			drain:  drain,
		},
		health: &admissionLimiter{
			sem:  semaphore.NewWeighted(int64(healthLimit)),
			shed: load.healthShed,
		},
	}
}

// serverLoad holds the admission instruments. An instrument that failed to
// build is nil and simply goes unrecorded.
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

// admissionLimiter is one budget. shed counts its refusals; active, when set,
// counts its admitted work as load, and a nil active leaves it out of that
// signal. A nil drain never refuses for draining.
type admissionLimiter struct {
	sem    *semaphore.Weighted
	shed   metric.Int64Counter
	active metric.Int64UpDownCounter
	drain  *rpcDrain
}

func (l *admissionLimiter) around(ctx context.Context, call func(context.Context) error) error {
	if l.drain != nil && l.drain.drainingNow() {
		return ownedStatus(codes.Unavailable, drainingFailureDetail)
	}
	if !l.sem.TryAcquire(1) {
		if l.shed != nil {
			l.shed.Add(ctx, 1)
		}
		return ownedStatus(codes.ResourceExhausted, failure.AtCapacityDetail)
	}
	defer l.sem.Release(1)
	if l.active != nil {
		l.active.Add(ctx, 1)
		defer l.active.Add(ctx, -1)
	}
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
