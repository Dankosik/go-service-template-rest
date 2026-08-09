package bootstrap

import (
	"context"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
)

// newReadinessService builds the cached readiness verdict this process serves
// and registers the task that keeps it current.
//
// The supervisor is both a probe and the owner of the refresher: a background
// task that failed is itself a readiness fact, and the refresher is registered
// as an ordinary supervised task so that a refresher which dies takes the same
// reported path as any other failed background work. health.Cached expiring a
// verdict nothing is refreshing is what covers the ways it can die without
// reporting.
func newReadinessService(
	cfg config.Config,
	log *slog.Logger,
	probes []health.Probe,
	supervisor *background.Supervisor,
) *health.Service {
	service := health.New(append(probes, supervisor)...)
	supervisor.Go(background.Task{
		Name: "readiness_refresh",
		Run: func(ctx context.Context) error {
			return service.Watch(
				ctx,
				cfg.Health.RefreshInterval,
				readinessProbeBudget(cfg),
				cfg.Health.FailureThreshold,
				func(err error) {
					logReadinessTransition(ctx, log, err)
				},
			)
		},
	})
	return service
}

func logReadinessTransition(ctx context.Context, log *slog.Logger, err error) {
	if err != nil {
		log.WarnContext(
			ctx,
			"readiness_transition",
			"component", "readiness",
			"operation", "refresh",
			"outcome", "unhealthy",
			"err", err,
		)
		return
	}
	log.InfoContext(
		ctx,
		"readiness_transition",
		"component", "readiness",
		"operation", "refresh",
		"outcome", "healthy",
	)
}
