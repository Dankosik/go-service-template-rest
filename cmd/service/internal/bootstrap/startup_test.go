package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"go.opentelemetry.io/otel"
)

func TestBootstrapNetworkPolicyStageRejectsPublicIngress(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "true")

	netPolicyResult := loadNetworkPolicy()
	if netPolicyResult.err != nil {
		t.Fatalf("loadNetworkPolicy() error = %v", netPolicyResult.err)
	}

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "metrics-exposure-policy")
	_, err := bootstrapNetworkPolicyStage(ctx, span, logger, netPolicyResult, config.Config{
		App:  config.AppConfig{Env: "prod"},
		HTTP: config.HTTPConfig{Addr: ":8080"},
	})
	span.End()
	if err == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want public ingress rejection")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !strings.Contains(err.Error(), "public ingress is unsupported") {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want public ingress detail", err)
	}

	if !strings.Contains(logBuffer.String(), `"dependency":"ingress_policy"`) {
		t.Fatalf("bootstrapNetworkPolicyStage() log = %q, want ingress policy dependency", logBuffer.String())
	}
}

func TestBootstrapNetworkPolicyStagePreservesConfigCause(t *testing.T) {
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_REASON", "temporary-diagnostic")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_SCOPE", "example.internal")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_EXPIRY", "not-rfc3339")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-egress-exception")

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "network-policy-stage")
	_, err := bootstrapNetworkPolicyStage(ctx, span, logger, loadNetworkPolicy(), config.Config{})
	span.End()
	if err == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want original parse detail", err)
	}
	logLine := logBuffer.String()
	if !strings.Contains(logLine, `"policy.class":"egress"`) {
		t.Fatalf("bootstrapNetworkPolicyStage() log = %q, want policy class", logLine)
	}
	if !strings.Contains(logLine, `"reason.class":"invalid_configuration"`) {
		t.Fatalf("bootstrapNetworkPolicyStage() log = %q, want reason class", logLine)
	}
}

func TestBootstrapNetworkPolicyStageRequiresExplicitIngressDeclarationForNonLocalWildcardBind(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "")

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "network-policy-stage")
	_, err := bootstrapNetworkPolicyStage(ctx, span, logger, loadNetworkPolicy(), config.Config{
		App:  config.AppConfig{Env: "prod"},
		HTTP: config.HTTPConfig{Addr: ":8080"},
	})
	span.End()
	if err == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !strings.Contains(err.Error(), envNetworkPublicIngressEnabled) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want missing ingress declaration detail", err)
	}
}
