package bootstrap

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/config/configtest"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func TestNATSPublisherConfigParity(t *testing.T) {
	for _, test := range configtest.MessagingCases() {
		t.Run(test.Name, func(t *testing.T) {
			configtest.IsolateEnv(t)
			setOutboxBootstrapEnvironment(t, true)
			t.Setenv("APP__MESSAGING__ENABLED", "true")
			t.Setenv("APP__MESSAGING__URLS", test.URLs)
			t.Setenv("APP__MESSAGING__STREAM", test.Stream)
			t.Setenv("APP__MESSAGING__ALLOW_PLAINTEXT", strconv.FormatBool(test.Plaintext))
			t.Setenv("APP__MESSAGING__ALLOW_UNAUTHENTICATED", "true")
			cfg, _, err := config.LoadDetailedWithContext(t.Context(), config.LoadOptions{})
			if test.ConfigRejects {
				if !errors.Is(err, config.ErrValidate) {
					t.Fatalf("config load error = %v, want ErrValidate", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("config load error = %v", err)
			}
			if err := natsjs.ValidateConfig(runtimeopts.Messaging(cfg.Messaging)); err != nil {
				t.Fatalf("internal/config admitted a NATS publisher config the adapter rejects: %v", err)
			}
		})
	}

	t.Run("connect failure is returned without a runtime", func(t *testing.T) {
		cfg := config.Config{Messaging: config.MessagingConfig{
			Enabled: true, URLs: "nats://127.0.0.1:1", Stream: "EVENTS",
			AllowPlaintext: true, AllowUnauthenticated: true, MaxPayloadBytes: 1024, MaxPendingPublishes: 1,
		}}
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()
		runtime, err := BuildNATSPublisher(ctx, cfg, nil)
		if err == nil || runtime.publisher != nil || runtime.run != nil || runtime.ready != nil || runtime.shutdown != nil {
			t.Fatalf("BuildNATSPublisher() = %+v, %v; want zero runtime and connection error", runtime, err)
		}
	})
}
