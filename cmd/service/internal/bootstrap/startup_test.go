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

func TestBootstrapNetworkPolicyStageAllowsDeclaredPublicIngress(t *testing.T) {
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
	if err != nil {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want nil", err)
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
