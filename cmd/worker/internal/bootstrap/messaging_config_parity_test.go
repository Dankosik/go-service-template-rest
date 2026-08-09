package bootstrap

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/config/configtest"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

// internal/config restates the rules natsjs applies to a client configuration,
// because depguard forbids it from importing the adapter. This package is the
// only one that wires both, so it is the only one that can prove the two copies
// still describe the same configuration.
//
// The two are not required to be identical, and one direction is allowed:
// internal/config may reject what natsjs accepts, because it also canonicalizes
// and rejects duplicates. The direction that must never happen is the reverse —
// a value internal/config admits and natsjs then refuses moves what should be a
// config-load failure to connect time, where an operator reads it as a broker
// outage instead of a typo.
func TestMessagingConfigRulesMatchAdapter(t *testing.T) {
	for _, testCase := range configtest.MessagingCases() {
		t.Run(testCase.Name, func(t *testing.T) {
			cfg, loadErr := loadMessagingConfig(t, testCase.URLs, testCase.Stream, testCase.Plaintext)
			configRejected := cfg == nil
			if configRejected != testCase.ConfigRejects {
				t.Fatalf("config load rejected = %v, want %v (load error: %v)", configRejected, testCase.ConfigRejects, loadErr)
			}
			if configRejected {
				// internal/config already refused it, so the operator sees the
				// fault at load. Whether natsjs would also refuse it does not
				// change what anyone observes.
				return
			}
			if err := natsjs.ValidateConfig(messagingClientConfig(*cfg)); err != nil {
				t.Fatalf("internal/config accepted a configuration natsjs.ValidateConfig refuses: %v\n"+
					"This is the direction that must never diverge: the fault now surfaces at connect "+
					"instead of at config load. Fix internal/config/messaging.go to match, or relax natsjs.",
					err)
			}
		})
	}
}

// loadMessagingConfig runs one messaging setting set through the real load path
// and returns the validated config, or nil when validation rejected it. Only
// ErrValidate counts as a rejection; a parse or IO fault fails the test, so a
// broken fixture cannot read as the config side doing its job.
func loadMessagingConfig(t *testing.T, urls, stream string, plaintext bool) (*config.MessagingConfig, error) {
	t.Helper()
	// A concrete port: nothing is served here, but config validation rejects 0.
	setWorkerTestEnvironment(t, urls, "127.0.0.1:9464")
	t.Setenv("APP__MESSAGING__STREAM", stream)
	t.Setenv("APP__MESSAGING__ALLOW_PLAINTEXT", strconv.FormatBool(plaintext))

	cfg, _, err := config.LoadDetailedWithContext(t.Context(), config.LoadOptions{})
	if err == nil {
		return &cfg.Messaging, nil
	}
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("load messaging config: %v", err)
	}
	return nil, fmt.Errorf("load messaging config: %w", err)
}

// The adapter is also the authority on what a plaintext URL is. This pins the
// scheme sets themselves rather than only their accept/reject verdicts, because
// the two files list them independently.
func TestMessagingSchemeVocabularyMatchesAdapter(t *testing.T) {
	for _, scheme := range []string{"nats", "tls", "ws", "wss"} {
		cfg := natsjs.Config{
			URLs: []string{scheme + "://broker.example:4222"}, Stream: "EVENTS",
			AllowPlaintext: true, AllowUnauthenticated: true,
			MaxPayloadBytes: 1 << 10, MaxPendingPublishes: 1,
		}
		if err := natsjs.ValidateConfig(cfg); err != nil {
			t.Errorf("natsjs.ValidateConfig(%s) error = %v, want the scheme accepted", scheme, err)
		}
	}
	for _, scheme := range []string{"amqp", "http", "kafka", ""} {
		cfg := natsjs.Config{
			URLs: []string{scheme + "://broker.example:4222"}, Stream: "EVENTS",
			AllowPlaintext: true, AllowUnauthenticated: true,
			MaxPayloadBytes: 1 << 10, MaxPendingPublishes: 1,
		}
		if err := natsjs.ValidateConfig(cfg); !errors.Is(err, natsjs.ErrRejected) {
			t.Errorf("natsjs.ValidateConfig(%q) error = %v, want ErrRejected", scheme, err)
		}
	}
	// Both files also agree that only tls and wss are secure without an opt-in.
	for _, scheme := range []string{"nats", "ws"} {
		cfg := natsjs.Config{
			URLs: []string{scheme + "://broker.example:4222"}, Stream: "EVENTS",
			AllowUnauthenticated: true, MaxPayloadBytes: 1 << 10, MaxPendingPublishes: 1,
		}
		err := natsjs.ValidateConfig(cfg)
		if err == nil || !errors.Is(err, natsjs.ErrRejected) || !strings.Contains(err.Error(), "plaintext") {
			t.Errorf("natsjs.ValidateConfig(%s without opt-in) error = %v, want a plaintext rejection", scheme, err)
		}
	}
}
