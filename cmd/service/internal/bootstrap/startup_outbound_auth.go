package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"github.com/example/go-service-template-rest/internal/infra/oauth2clientcredentials"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

type outboundAuthRuntime interface {
	Close(ctx context.Context) error
}

func initOutboundAuth( //nolint:ireturn // runtimeWiring needs this narrow lifecycle test seam.
	source config.OutboundAuthConfig,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) (outboundAuthRuntime, error) { //nolint:iface // runtimeWiring needs this narrow lifecycle test seam.
	runtimeConfig, err := mapOutboundAuthConfig(source)
	if err != nil {
		return nil, err
	}
	return oauth2clientcredentials.New(runtimeConfig, metrics.MeterProvider(), log) //nolint:wrapcheck // The credential owner supplies the closed startup error.
}

func mapOutboundAuthConfig(source config.OutboundAuthConfig) (oauth2clientcredentials.Config, error) {
	var targetClass httpclient.TargetClass
	switch source.TokenTargetClass {
	case "external_https":
		targetClass = httpclient.ExternalHTTPS
	case "private_https":
		targetClass = httpclient.PrivateHTTPS
	default:
		return oauth2clientcredentials.Config{}, errors.New("map outbound authentication config: token target class is invalid")
	}

	return oauth2clientcredentials.Config{
		DependencyName:         source.Dependency,
		ClientID:               source.ClientID,
		ClientSecret:           source.ClientSecret,
		ClientAuthentication:   source.ClientAuthentication,
		TokenEndpoint:          source.TokenEndpoint,
		TokenTargetClass:       targetClass,
		TokenPrivateHostSuffix: source.TokenPrivateHostSuffix,
		Scopes:                 strings.Fields(source.Scopes),
		Resource:               source.Resource,
		Audience:               source.Audience,
		ResourceAuthority:      source.ResourceAuthority,
		AcquisitionTimeout:     source.AcquisitionTimeout,
	}, nil
}
