package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
)

// newReadinessService builds the cached readiness verdict this process serves.
//
// The supervisor is both a probe and the owner of the refresher: a background
// task that failed is itself a readiness fact, and the refresher is registered
// as an ordinary supervised task so that a refresher which dies takes the same
// reported path as any other failed background work. health.Cached expiring a
// verdict nothing is refreshing is what covers the ways it can die without
// reporting.
//
// The probe budget is the aggregate readiness budget because it covers the
// complete serial probe set, including optional probes.
func newReadinessService(
	cfg config.Config,
	probes []health.Probe,
	supervisor *background.Supervisor,
) (*health.Service, error) {
	service, err := health.New(health.Policy{
		ProbeBudget:      cfg.HTTP.ReadinessTimeout,
		FailureThreshold: cfg.Health.FailureThreshold,
	}, append(probes, supervisor)...)
	if err != nil {
		return nil, fmt.Errorf("build readiness service: %w", err)
	}
	return service, nil
}

func superviseReadiness(
	cfg config.Config,
	log *slog.Logger,
	service *health.Service,
	supervisor *background.Supervisor,
	// profile:grpc:start
	grpcServer grpcRuntimeServer,
	// profile:grpc:end
) {
	supervisor.Go(background.Task{
		Name: "readiness_refresh",
		Run: func(ctx context.Context) error {
			return service.Watch(
				ctx,
				cfg.Health.RefreshInterval,
				func(err error) {
					logReadinessTransition(ctx, log, err)
					// profile:grpc:start
					if grpcServer != nil {
						grpcServer.SetServing(err == nil)
					}
					// profile:grpc:end
				},
			)
		},
	})
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
