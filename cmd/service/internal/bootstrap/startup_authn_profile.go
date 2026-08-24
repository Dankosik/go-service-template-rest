package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/example/go-service-template-rest/internal/infra/oidcjwt"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func initAuthn(
	ctx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) (*bearerauthn.Runtime, error) {
	policy, err := oidcjwt.NewPolicy(oidcjwt.PolicyInput{
		Issuer:       cfg.Authn.Issuer,
		Audience:     cfg.Authn.Audience,
		TokenProfile: cfg.Authn.TokenProfile,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize authentication policy: %w", err)
	}
	verifier, err := oidcjwt.New(ctx, policy, metrics.MeterProvider(), log)
	if err != nil {
		return nil, fmt.Errorf("initialize authentication trust: %w", err)
	}
	runtime, err := bearerauthn.New(verifier, metrics.MeterProvider())
	if err != nil {
		verifier.Close()
		return nil, fmt.Errorf("initialize authentication runtime: %w", err)
	}
	return runtime, nil
}
