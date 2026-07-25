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

// resourceIdentity is what every exported span and metric is attributed to.
//
// service.instance.id is not optional here, and the OpenTelemetry Go SDK does not
// supply one: its detector is behind the experimental OTEL_GO_X_RESOURCE flag, and
// resource.New with WithAttributes runs no detectors at all. Without it, every
// replica pushes an identical resource — so a collector keying series by resource
// sees one logical stream fed by N monotonic counters with independent start
// times, and every replica restart reads as a counter reset that makes rate()
// wrong. On traces it makes "is this one instance or all of them" unanswerable,
// which is the first question in every latency incident.
//
// On the Prometheus scrape path the scraper supplies an `instance` label, which is
// why this was invisible until a service pushed over OTLP.
type resourceIdentity struct {
	serviceName    string
	serviceVersion string
	instanceID     string
	deploymentEnv  string
}

// instanceIDBytes is the entropy of the last-resort instance identifier. It only
// has to be unique among this service's live replicas.
const instanceIDBytes = 8

// ResolveInstanceID reports the instance identity to publish, and must be called
// once per process: a fallback identifier is generated, so resolving separately
// for traces and for metrics would attribute the two signals of one replica to two
// different instances.
//
// A configured value wins. Otherwise the hostname is used, which is the pod name
// on Kubernetes and the container id on most other platforms — the identifier an
// operator already has in hand. A random value is the last resort, because an
// empty instance id is the failure this exists to prevent.
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
// merges it under these attributes — sdktrace.WithResource and its metric
// equivalent call resource.Merge(resource.Environment(), r), and Merge is
// last-value-wins on r — so nothing a platform injects can override what this
// service configured, while k8s.pod.name, container.id, and the rest survive.
// The version this replaced unset the variable around provider construction,
// which protected nothing and discarded exactly those attributes.
func newResource(ctx context.Context, identity resourceIdentity) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", strings.TrimSpace(identity.serviceName)),
			attribute.String("service.version", strings.TrimSpace(identity.serviceVersion)),
			attribute.String("service.instance.id", ResolveInstanceID(identity.instanceID)),
			attribute.String("deployment.environment.name", strings.TrimSpace(identity.deploymentEnv)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}
	return res, nil
}
