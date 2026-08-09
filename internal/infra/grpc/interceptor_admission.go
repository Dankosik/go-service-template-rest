package grpcx

import (
	"context"

	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
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

// newAdmissionPolicy builds both budgets. Only the business one reports to load:
// active and shed are the capacity signal an operator sizes MaxConcurrentRPCs
// from, and a standing health watch per connected peer would turn active into a
// count of peers instead. A health budget under pressure is a hostile-peer
// signal rather than a capacity one, and AccessLogHealthChecks is the switch
// that surfaces it.
func newAdmissionPolicy(businessLimit, healthLimit int, load LoadRecorder) admissionPolicy {
	return admissionPolicy{
		business: newAdmissionLimiter(businessLimit, load),
		health:   newAdmissionLimiter(healthLimit, noopLoadRecorder{}),
	}
}

// around holds one slot from the owning budget for the work below it. One policy
// value backs both chains, which is what makes each budget process-wide rather
// than per RPC kind. Since NewServer builds the two policy lists separately,
// that sharing is its composition rather than a structural guarantee:
// TestOneAdmissionPolicyServesBothInterceptorTypes proves this method is shareable, and
// TestAdmissionBudgetIsProcessWide proves the server actually shares it.
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
	sem  *semaphore.Weighted
	load LoadRecorder
}

func newAdmissionLimiter(limit int, load LoadRecorder) *admissionLimiter {
	return &admissionLimiter{
		sem:  semaphore.NewWeighted(int64(limit)),
		load: load,
	}
}

func (l *admissionLimiter) around(ctx context.Context, call func(context.Context) error) error {
	if !l.sem.TryAcquire(1) {
		l.load.Shed(ctx)
		return ownedStatus(codes.ResourceExhausted, "server is at capacity")
	}
	defer l.sem.Release(1)
	release := l.load.Admitted(ctx)
	defer release()
	return call(ctx)
}
