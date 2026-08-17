package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/config/configtest"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"github.com/example/go-service-template-rest/internal/infra/oauth2clientcredentials"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/google/go-cmp/cmp"
)

// TestOutboundAuthConfigParity proves that every configuration value admitted
// by the source loader reaches the runtime validator unchanged or canonically
// mapped. Config may refuse extra source spellings, but it must not construct a
// runtime config that the credential owner rejects.
func TestOutboundAuthConfigParity(t *testing.T) {
	for _, testCase := range outboundAuthParityCases() {
		t.Run(testCase.name, func(t *testing.T) {
			raw := outboundAuthParitySource(t, testCase.env)
			runtimeConfig, runtimeErr := mapOutboundAuthConfig(raw)
			if runtimeErr == nil {
				runtime, err := oauth2clientcredentials.New(runtimeConfig, telemetry.New().MeterProvider(), slog.Default())
				if err != nil {
					runtimeErr = err
				} else if err := runtime.Close(context.Background()); err != nil {
					t.Fatalf("runtime.Close() error = %v", err)
				}
			}
			if (runtimeErr == nil) != testCase.runtimeAcceptable {
				t.Fatalf("runtime validation error = %v, want acceptable = %t", runtimeErr, testCase.runtimeAcceptable)
			}

			source, configErr := loadOutboundAuthParityConfig(t, testCase.env)
			if (configErr == nil) != testCase.configAcceptable {
				t.Fatalf("config.LoadDetailed() error = %v, want acceptable = %t", configErr, testCase.configAcceptable)
			}
			if configErr == nil && runtimeErr != nil {
				t.Fatalf("config admitted a value the runtime rejected: %v", runtimeErr)
			}
			if configErr != nil {
				return
			}

			got, err := mapOutboundAuthConfig(source)
			if err != nil {
				t.Fatalf("mapOutboundAuthConfig() error = %v", err)
			}
			if diff := cmp.Diff(testCase.want, got); diff != "" {
				t.Fatalf("mapOutboundAuthConfig() mismatch (-want +got):\n%s", diff)
			}
			if got.ClientID != source.ClientID || got.ClientSecret != source.ClientSecret {
				t.Fatal("mapOutboundAuthConfig() changed exact client credential bytes")
			}

			runtime, err := oauth2clientcredentials.New(got, telemetry.New().MeterProvider(), slog.Default())
			if err != nil {
				t.Fatalf("runtime config rejected source-admitted value: %v", err)
			}
			t.Cleanup(func() {
				if err := runtime.Close(context.Background()); err != nil {
					t.Errorf("runtime.Close() error = %v", err)
				}
			})
		})
	}
}

type outboundAuthParityCase struct {
	name              string
	env               map[string]string
	configAcceptable  bool
	runtimeAcceptable bool
	want              oauth2clientcredentials.Config
}

func outboundAuthParityCases() []outboundAuthParityCase {
	canonical := outboundAuthParityEnv()
	validRuntime := oauth2clientcredentials.Config{
		DependencyName:       "payments",
		ClientID:             " client:id+ ",
		ClientSecret:         " secret:p@ss+ ",
		ClientAuthentication: "client_secret_basic",
		TokenEndpoint:        "https://auth.example.com/oauth/token",
		TokenTargetClass:     httpclient.ExternalHTTPS,
		Scopes:               []string{"payments.read", "payments.write"},
		Resource:             "https://payments.example.com",
		ResourceAuthority:    "https://payments.example.com",
		AcquisitionTimeout:   2 * time.Second,
	}
	private := maps.Clone(canonical)
	private["TOKEN_ENDPOINT"] = "HTTPS://AUTH.SERVICE.INTERNAL/oauth/token"
	private["TOKEN_TARGET_CLASS"] = "private_https"
	private["TOKEN_PRIVATE_HOST_SUFFIX"] = ".SERVICE.INTERNAL."
	private["SCOPES"] = "payments.write payments.read"
	privateRuntime := validRuntime
	privateRuntime.TokenEndpoint = "https://auth.service.internal/oauth/token"
	privateRuntime.TokenTargetClass = httpclient.PrivateHTTPS
	privateRuntime.TokenPrivateHostSuffix = "service.internal"

	audience := maps.Clone(canonical)
	audience["RESOURCE"] = ""
	audience["AUDIENCE"] = "payments-api"
	audienceRuntime := validRuntime
	audienceRuntime.Resource = ""
	audienceRuntime.Audience = "payments-api"

	cases := []outboundAuthParityCase{
		{name: "canonical external", env: canonical, configAcceptable: true, runtimeAcceptable: true, want: validRuntime},
		{name: "canonical private", env: private, configAcceptable: true, runtimeAcceptable: true, want: privateRuntime},
		{name: "audience binding", env: audience, configAcceptable: true, runtimeAcceptable: true, want: audienceRuntime},
	}
	for _, invalid := range []struct {
		name  string
		field string
		value string
	}{
		{name: "missing dependency", field: "DEPENDENCY", value: ""},
		{name: "invalid dependency", field: "DEPENDENCY", value: "Payments"},
		{name: "long dependency", field: "DEPENDENCY", value: "a" + strings.Repeat("b", 64)},
		{name: "missing client ID", field: "CLIENT_ID", value: ""},
		{name: "long client ID", field: "CLIENT_ID", value: strings.Repeat("i", 513)},
		{name: "client ID control", field: "CLIENT_ID", value: "client\u0085id"},
		{name: "missing client secret", field: "CLIENT_SECRET", value: ""},
		{name: "blank client secret", field: "CLIENT_SECRET", value: "   "},
		{name: "long client secret", field: "CLIENT_SECRET", value: strings.Repeat("s", 4097)},
		{name: "client secret control", field: "CLIENT_SECRET", value: "secret\u0085value"},
		{name: "unsupported client authentication", field: "CLIENT_AUTHENTICATION", value: "client_secret_post"},
		{name: "non-exact client authentication", field: "CLIENT_AUTHENTICATION", value: " client_secret_basic "},
		{name: "relative token endpoint", field: "TOKEN_ENDPOINT", value: "/oauth/token"},
		{name: "plaintext token endpoint", field: "TOKEN_ENDPOINT", value: "http://auth.example.com/oauth/token"},
		{name: "token endpoint user info", field: "TOKEN_ENDPOINT", value: "https://user@auth.example.com/oauth/token"},
		{name: "token endpoint query", field: "TOKEN_ENDPOINT", value: "https://auth.example.com/oauth/token?tenant=one"},
		{name: "token endpoint fragment", field: "TOKEN_ENDPOINT", value: "https://auth.example.com/oauth/token#fragment"},
		{name: "long token endpoint", field: "TOKEN_ENDPOINT", value: "https://auth.example.com/" + strings.Repeat("a", 2048)},
		{name: "unknown target class", field: "TOKEN_TARGET_CLASS", value: "private_http"},
		{name: "external private suffix", field: "TOKEN_PRIVATE_HOST_SUFFIX", value: "internal"},
		{name: "duplicate scope", field: "SCOPES", value: "payments.read payments.read"}, //nolint:dupword // The input intentionally repeats this scope.
		{name: "invalid scope", field: "SCOPES", value: "payments\\read"},
		{name: "Unicode scope separator", field: "SCOPES", value: "payments.read\u00a0payments.write"},
		{name: "resource and audience", field: "AUDIENCE", value: "payments-api"},
		{name: "relative resource", field: "RESOURCE", value: "/payments"},
		{name: "resource fragment", field: "RESOURCE", value: "https://payments.example.com#fragment"},
		{name: "long resource", field: "RESOURCE", value: "urn:" + strings.Repeat("r", 2048)},
		{name: "long audience", field: "AUDIENCE", value: strings.Repeat("a", 2049)},
		{name: "audience control", field: "AUDIENCE", value: "payments\u0085api"},
		{name: "plaintext resource authority", field: "RESOURCE_AUTHORITY", value: "http://payments.example.com"},
		{name: "resource authority path", field: "RESOURCE_AUTHORITY", value: "https://payments.example.com/api"},
		{name: "short acquisition timeout", field: "ACQUISITION_TIMEOUT", value: "99ms"},
		{name: "long acquisition timeout", field: "ACQUISITION_TIMEOUT", value: "31s"},
	} {
		env := maps.Clone(canonical)
		env[invalid.field] = invalid.value
		if invalid.field == "AUDIENCE" && invalid.name != "resource and audience" {
			env["RESOURCE"] = ""
		}
		cases = append(cases, outboundAuthParityCase{
			name:              invalid.name,
			env:               env,
			runtimeAcceptable: invalid.name == "Unicode scope separator",
		})
	}
	return cases
}

func outboundAuthParityEnv() map[string]string {
	return map[string]string{
		"DEPENDENCY":                "payments",
		"CLIENT_ID":                 " client:id+ ",
		"CLIENT_SECRET":             " secret:p@ss+ ",
		"CLIENT_AUTHENTICATION":     "client_secret_basic",
		"TOKEN_ENDPOINT":            "https://auth.example.com/oauth/token",
		"TOKEN_TARGET_CLASS":        "external_https",
		"TOKEN_PRIVATE_HOST_SUFFIX": "",
		"SCOPES":                    "payments.read payments.write",
		"RESOURCE":                  "https://payments.example.com",
		"AUDIENCE":                  "",
		"RESOURCE_AUTHORITY":        "https://payments.example.com",
		"ACQUISITION_TIMEOUT":       "2s",
	}
}

func outboundAuthParitySource(t *testing.T, values map[string]string) config.OutboundAuthConfig {
	t.Helper()
	duration, err := time.ParseDuration(values["ACQUISITION_TIMEOUT"])
	if err != nil {
		t.Fatalf("time.ParseDuration(%q) error = %v", values["ACQUISITION_TIMEOUT"], err)
	}
	return config.OutboundAuthConfig{
		Dependency:             values["DEPENDENCY"],
		ClientID:               values["CLIENT_ID"],
		ClientSecret:           values["CLIENT_SECRET"],
		ClientAuthentication:   values["CLIENT_AUTHENTICATION"],
		TokenEndpoint:          values["TOKEN_ENDPOINT"],
		TokenTargetClass:       values["TOKEN_TARGET_CLASS"],
		TokenPrivateHostSuffix: values["TOKEN_PRIVATE_HOST_SUFFIX"],
		Scopes:                 values["SCOPES"],
		Resource:               values["RESOURCE"],
		Audience:               values["AUDIENCE"],
		ResourceAuthority:      values["RESOURCE_AUTHORITY"],
		AcquisitionTimeout:     duration,
	}
}

func loadOutboundAuthParityConfig(t *testing.T, values map[string]string) (config.OutboundAuthConfig, error) {
	t.Helper()
	configtest.IsolateEnv(t)
	// profile:object-storage:start
	for key, value := range map[string]string{
		"APP__OBJECT_STORAGE__PROVIDER":                   "amazon_s3",
		"APP__OBJECT_STORAGE__ENDPOINT":                   "https://s3.us-east-1.amazonaws.com",
		"APP__OBJECT_STORAGE__REGION":                     "us-east-1",
		"APP__OBJECT_STORAGE__BUCKET":                     "examplebucket",
		"APP__OBJECT_STORAGE__ACCESS_KEY_ID":              "test-access-key",
		"APP__OBJECT_STORAGE__SECRET_ACCESS_KEY":          "test-secret-key",
		"APP__OBJECT_STORAGE__SESSION_TOKEN":              "test-session-token",
		"APP__OBJECT_STORAGE__EXPECTED_BUCKET_OWNER":      "123456789012",
		"APP__OBJECT_STORAGE__MAX_OBJECT_BYTES":           "10485760",
		"APP__OBJECT_STORAGE__MULTIPART_CHUNK_BYTES":      "5242880",
		"APP__OBJECT_STORAGE__MAX_ACTIVE_OPERATIONS":      "2",
		"APP__OBJECT_STORAGE__MAX_OPERATION_DURATION":     "1s",
		"APP__OBJECT_STORAGE__MAX_PRESIGN_LIFETIME":       "1m",
		"APP__OBJECT_STORAGE__MAX_RESPONSE_HEADER_BYTES":  "1024",
		"APP__OBJECT_STORAGE__MAX_CONTROL_RESPONSE_BYTES": "1024",
		"APP__OBJECT_STORAGE__MAX_WORKING_MEMORY_BYTES":   "62149760",
	} {
		t.Setenv(key, value)
	}
	// profile:object-storage:end
	for key, value := range values {
		t.Setenv("APP__OUTBOUND_AUTH__"+key, value)
	}
	cfg, _, err := config.LoadDetailed(config.LoadOptions{})
	return cfg.OutboundAuth, err //nolint:wrapcheck // The parity oracle compares the loader's direct result.
}

func TestOutboundAuthStartupIsLocalOnly(t *testing.T) {
	source := validOutboundAuthConfig()
	source.TokenEndpoint = "https://provider.invalid/oauth/token"

	runtime, err := initOutboundAuth(source, telemetry.New(), shutdownTestLogger())
	if err != nil {
		t.Fatalf("initOutboundAuth() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := runtime.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestOutboundAuthStartupAndCloseErrorsAreSanitized(t *testing.T) {
	const canary = "forbidden-auth-canary"
	source := validOutboundAuthConfig()
	source.ClientSecret = canary + "\n"
	_, safeErr := initOutboundAuth(source, telemetry.New(), shutdownTestLogger())
	if safeErr == nil {
		t.Fatal("initOutboundAuth() error = nil, want invalid configuration")
	}
	if class, ok := oauth2clientcredentials.FailureClassOf(safeErr); !ok || class != oauth2clientcredentials.FailureInvalidConfiguration {
		t.Fatalf("startup failure class = %q, %v, want invalid_configuration", class, ok)
	}
	if safeErr.Error() != "outbound authentication configuration is invalid" || strings.Contains(safeErr.Error(), canary) {
		t.Fatalf("startup error = %q, want only the closed safe text", safeErr)
	}

	resetShutdownConfigEnv(t)
	runtime := &recordingOutboundAuthRuntime{closeErr: safeErr}
	wiring := outboundAuthTestWiring(runtime)
	wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error { return nil }

	err := runWithRuntime(nil, wiring)
	if class, ok := oauth2clientcredentials.FailureClassOf(err); !ok || class != oauth2clientcredentials.FailureInvalidConfiguration {
		t.Fatalf("close failure class = %q, %v, want invalid_configuration", class, ok)
	}
	if err.Error() != safeErr.Error() || strings.Contains(err.Error(), canary) {
		t.Fatalf("close error = %q, want only the closed safe text", err)
	}
	if runtime.closeCalls != 1 {
		t.Fatalf("Close calls = %d, want 1", runtime.closeCalls)
	}
}

func TestOutboundAuthOutageDoesNotChangeHealth(t *testing.T) {
	resetShutdownConfigEnv(t)
	runtime := &recordingOutboundAuthRuntime{}
	wiring := outboundAuthTestWiring(runtime)
	initCalls := 0
	wiring.initOutboundAuth = func(
		config.OutboundAuthConfig,
		*telemetry.Metrics,
		*slog.Logger,
	) (outboundAuthRuntime, error) {
		initCalls++
		return runtime, nil
	}
	stopServing := errors.New("stop serving")
	wiring.serve = func(_ context.Context, _ context.Context, args serveRuntimeArgs) error {
		if initCalls != 1 {
			t.Fatalf("outbound auth initializations = %d, want 1", initCalls)
		}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		if err := args.readinessCheck(ctx); err != nil { //nolint:contextcheck // The test invokes readiness with its test-owned cancellation context.
			t.Fatalf("initial readiness check error = %v", err)
		}
		args.admission.MarkReady()
		for range 3 {
			if err := args.healthSvc.Cached(); err != nil {
				t.Fatalf("cached readiness error = %v", err)
			}
		}
		if runtime.closeCalls != 0 {
			t.Fatalf("health reads called outbound auth Close %d times", runtime.closeCalls)
		}
		return stopServing
	}

	if err := runWithRuntime(nil, wiring); !errors.Is(err, stopServing) {
		t.Fatalf("runWithRuntime() error = %v, want %v", err, stopServing)
	}
	if runtime.closeCalls != 1 {
		t.Fatalf("Close calls after shutdown = %d, want 1", runtime.closeCalls)
	}
}

func TestOutboundAuthInvalidConfigFailsBeforeServing(t *testing.T) {
	resetShutdownConfigEnv(t)
	t.Setenv("APP__OUTBOUND_AUTH__CLIENT_AUTHENTICATION", "client_secret_post")

	mutated := false
	wiring := testRuntimeWiring()
	wiring.lifecycle = func(runtimeLifecycleStage) { mutated = true }
	wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
		mutated = true
		return runtimeDependencies{}, nil
	}
	wiring.initOutboundAuth = func(
		config.OutboundAuthConfig,
		*telemetry.Metrics,
		*slog.Logger,
	) (outboundAuthRuntime, error) {
		mutated = true
		return &recordingOutboundAuthRuntime{}, nil
	}
	wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error {
		mutated = true
		return nil
	}

	err := runWithRuntime(nil, wiring)
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("runWithRuntime() error = %v, want config.ErrValidate", err)
	}
	if mutated {
		t.Fatal("invalid outbound auth config reached runtime or listener mutation")
	}
}

func outboundAuthTestWiring(runtime outboundAuthRuntime) runtimeWiring {
	wiring := testRuntimeWiring()
	wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
		return runtimeDependencies{}, nil
	}
	wiring.initOutboundAuth = func(
		config.OutboundAuthConfig,
		*telemetry.Metrics,
		*slog.Logger,
	) (outboundAuthRuntime, error) {
		return runtime, nil
	}
	// profile:authn-oidc-jwt:start
	wiring.initAuthn = func(context.Context, config.Config, *telemetry.Metrics, *slog.Logger) (authnRuntime, error) {
		return fakeAuthnRuntime{}, nil
	}
	// profile:authn-oidc-jwt:end
	return wiring
}

type recordingOutboundAuthRuntime struct {
	closeCalls   int
	closeContext context.Context //nolint:containedctx // The lifecycle test asserts the shutdown deadline passed to Close.
	closeErr     error
	onClose      func()
}

func (r *recordingOutboundAuthRuntime) Close(ctx context.Context) error {
	r.closeCalls++
	r.closeContext = ctx
	if r.onClose != nil {
		r.onClose()
	}
	return r.closeErr
}

func validOutboundAuthConfig() config.OutboundAuthConfig {
	return config.OutboundAuthConfig{
		Dependency:           "payments",
		ClientID:             "test-client",
		ClientSecret:         "test-secret",
		ClientAuthentication: "client_secret_basic",
		TokenEndpoint:        "https://auth.example.com/oauth/token",
		TokenTargetClass:     "external_https",
		Scopes:               "payments.read",
		ResourceAuthority:    "https://payments.example.com",
		AcquisitionTimeout:   time.Second,
	}
}
