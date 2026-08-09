package telemetry

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

// ResourceConfig is what every exported span and metric is attributed to.
//
// It is one type carried by both [TracingConfig] and [MetricsConfig] rather than
// five fields on each, because the two have to agree: the signals of one replica
// attributed to two identities cannot be correlated, and that is a mistake five
// separately spelled fields invite. [ResolveInstanceID] is resolved once per
// process for the same reason.
//
// service.instance.id must be supplied here: the OpenTelemetry Go SDK's detector
// is behind the experimental OTEL_GO_X_RESOURCE flag, and resource.New with
// WithAttributes runs no detectors at all. Without it every replica pushes an
// identical resource, so N monotonic counters collapse into one series and every
// replica restart reads as a counter reset.
//
// The Prometheus scrape path supplies its own `instance` label, which is why this
// only shows up once a service pushes over OTLP.
type ResourceConfig struct {
	ServiceName    string
	ServiceVersion string
	// ServiceCommit is the source revision the binary was built from, published
	// as vcs.revision. A version alone is often a branch or a tag that moves, so
	// it does not answer "which build is this" during a rollout; the revision
	// does.
	ServiceCommit string
	// ServiceInstanceID identifies this replica. Resolve it once per process
	// with ResolveInstanceID.
	ServiceInstanceID string
	DeploymentEnv     string
}

// instanceIDBytes is the entropy of the last-resort instance identifier. It only
// has to be unique among this service's live replicas.
const instanceIDBytes = 8

// ResolveInstanceID reports the instance identity to publish, and must be called
// once per process: a fallback identifier is generated, so resolving separately
// for traces and for metrics would attribute the two signals of one replica to two
// different instances.
//
// A configured value wins, then the hostname — the pod name on Kubernetes, the
// container id on most other platforms. A random value is the last resort,
// because an empty instance id is the failure this exists to prevent.
func ResolveInstanceID(configured string) string {
	if trimmed := strings.TrimSpace(configured); trimmed != "" {
		return trimmed
	}
	if hostname, err := os.Hostname(); err == nil {
		if trimmed := strings.TrimSpace(hostname); trimmed != "" {
			return trimmed
		}
	}
	return "instance-" + rand.Text()[:instanceIDBytes]
}

// newResource builds the resource for one signal.
//
// The ambient OTEL_RESOURCE_ATTRIBUTES is deliberately not suppressed. The SDK
// calls resource.Merge(resource.Environment(), r) and Merge is last-value-wins on
// r, so nothing a platform injects can override what this service configured
// while k8s.pod.name, container.id, and the rest survive.
func newResource(ctx context.Context, identity ResourceConfig) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", strings.TrimSpace(identity.ServiceName)),
			attribute.String("service.version", strings.TrimSpace(identity.ServiceVersion)),
			attribute.String("vcs.revision", strings.TrimSpace(identity.ServiceCommit)),
			attribute.String("service.instance.id", ResolveInstanceID(identity.ServiceInstanceID)),
			attribute.String("deployment.environment.name", strings.TrimSpace(identity.DeploymentEnv)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}
	return res, nil
}
