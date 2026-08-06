// The two owners of this transport's bounds, held to one answer.
//
// [validateConfig] and internal/config's validateGRPCConfig both bound the values
// a gRPC server is built from, and cannot share code: the depguard rule
// config_no_runtime_owners stops internal/config from importing this package, so
// it restates the rules and a rule changed on one side alone breaks nothing
// anyone runs. The check lives here because the import only works in this
// direction.
//
// The two owners are not equivalent, and the file is split along that line:
// internal/config is the tighter one on capacity, so what it accepts must build a
// server, while the two access-log rules are duplicated term for term and must
// agree in both directions. Each test compares only the values its corpus varies,
// so a bound added to Config and to config.GRPCServerConfig without a case here
// is one they go on reporting as agreed.

package grpcx

import (
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/config/configtest"
)

// serverConfigFromRuntime maps a loaded runtime configuration the way
// cmd/service/internal/bootstrap/startup_grpc.go does.
//
// It is restated rather than called because that mapping is grpcServerConfig in
// cmd/service/internal/bootstrap, which is internal to another binary and cannot
// be imported from here at all.
//
// So this is an independent oracle, not a second guard on production: a rename
// already stops the production mapping compiling on its own, and a bound this
// copy drops is invisible to the service. That copy has its own target-side
// proof, TestGRPCServerConfigFillsEveryTransportBound in that package. What this
// one buys is the corpus below being able to ask what NewServer does with a
// configuration the loader accepted, which needs some crossing in this package.
func serverConfigFromRuntime(server config.GRPCServerConfig) Config {
	return Config{
		MaxConcurrentRPCs:          server.MaxConcurrentRPCs,
		MaxConcurrentStreams:       server.MaxConcurrentStreams,
		MaxHeaderListBytes:         server.MaxHeaderListBytes,
		MaxReceiveMessageBytes:     server.MaxReceiveMessageBytes,
		MaxSendMessageBytes:        server.MaxSendMessageBytes,
		LogHealthChecks:            server.AccessLogHealthChecks,
		AccessLogSuccessSampleRate: server.AccessLogSuccessSampleRate,
		AccessLogSlowThreshold:     server.AccessLogSlowThreshold,
		TelemetryHealthChecks:      server.TelemetryHealthChecks,
	}
}

// TestConfigAcceptedBoundsAlwaysBuildAServer proves the containment that keeps a
// deployment from passing configuration load and then failing at server
// construction.
//
// internal/config is the tighter owner on every capacity bound, so the corpus
// sits on its maxima: those are the values where loosening one of its ranges
// without loosening this package's would first be observed, and they are the
// ones a hand-check is least likely to try.
func TestConfigAcceptedBoundsAlwaysBuildAServer(t *testing.T) {
	for _, testCase := range []struct {
		name string
		env  map[string]string
	}{
		{name: "defaults"},
		{
			name: "capacity minima",
			env: map[string]string{
				"MAX_CONNECTIONS":           "1",
				"MAX_CONCURRENT_RPCS":       "1",
				"MAX_CONCURRENT_STREAMS":    "1",
				"MAX_HEADER_LIST_BYTES":     "1",
				"MAX_RECEIVE_MESSAGE_BYTES": "1",
				"MAX_SEND_MESSAGE_BYTES":    "1",
			},
		},
		{
			name: "capacity maxima",
			env: map[string]string{
				"MAX_CONNECTIONS":           "1000000",
				"MAX_CONCURRENT_RPCS":       "100000",
				"MAX_CONCURRENT_STREAMS":    "100000",
				"MAX_HEADER_LIST_BYTES":     strconv.Itoa(math.MaxInt32),
				"MAX_RECEIVE_MESSAGE_BYTES": strconv.Itoa(math.MaxInt32),
				"MAX_SEND_MESSAGE_BYTES":    strconv.Itoa(math.MaxInt32),
			},
		},
		{
			name: "observability extremes",
			env: map[string]string{
				"ACCESS_LOG_SUCCESS_SAMPLE_RATE": "0",
				"ACCESS_LOG_SLOW_THRESHOLD":      "24h",
				"ACCESS_LOG_HEALTH_CHECKS":       "true",
				"TELEMETRY_HEALTH_CHECKS":        "true",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			loaded, err := loadGRPCServerConfig(t, testCase.env)
			if err != nil {
				t.Fatalf("configuration load rejected a case this corpus claims is valid: %v", err)
			}
			if err := validateConfig(serverConfigFromRuntime(loaded)); err != nil {
				t.Fatalf(
					"configuration load accepted %+v but NewServer would refuse it: %v",
					loaded,
					err,
				)
			}
		})
	}
}

// TestAccessLogRulesMatchConfigValidation holds the two rules both owners spell
// out to a single answer over one corpus.
//
// These are the rules the comments on [validateConfig] and validateGRPCConfig
// name, and the only ones where the two ranges are meant to be identical rather
// than nested — so unlike the capacity bounds above, disagreement in either
// direction is the defect.
func TestAccessLogRulesMatchConfigValidation(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		rate       string
		threshold  string
		acceptable bool
	}{
		{name: "rate off", rate: "0", acceptable: true},
		{name: "rate full", rate: "1", acceptable: true},
		{name: "rate partial", rate: "0.5", acceptable: true},
		{name: "rate below range", rate: "-0.1"},
		{name: "rate above range", rate: "1.1"},
		{name: "rate not a number", rate: "NaN"},
		{name: "rate infinite", rate: "+Inf"},

		{name: "threshold disabled", rate: "1", threshold: "0s", acceptable: true},
		{name: "threshold set", rate: "1", threshold: "250ms", acceptable: true},
		{name: "threshold negative", rate: "1", threshold: "-1ns"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			env := map[string]string{"ACCESS_LOG_SUCCESS_SAMPLE_RATE": testCase.rate}
			if testCase.threshold != "" {
				env["ACCESS_LOG_SLOW_THRESHOLD"] = testCase.threshold
			}
			_, configErr := loadGRPCServerConfig(t, env)

			// Built from the corpus rather than from the loaded configuration,
			// because a rejected load returns nothing to map — and the whole
			// question is what each owner does with the same intended value.
			intended := testServerConfig()
			intended.AccessLogSuccessSampleRate = parseFloat(t, testCase.rate)
			if testCase.threshold != "" {
				intended.AccessLogSlowThreshold = parseDuration(t, testCase.threshold)
			}
			serverErr := validateConfig(intended)

			if (serverErr == nil) != testCase.acceptable {
				t.Fatalf("validateConfig() error = %v, want acceptable = %v", serverErr, testCase.acceptable)
			}
			// A configuration load can refuse a value while decoding rather than
			// while validating, so the two owners' messages need not match. That
			// it refuses at all is the property: internal/config must never hand
			// a value onward that this package would then reject.
			if (configErr == nil) != (serverErr == nil) {
				t.Fatalf(
					"access-log rules disagree: configuration load error = %v, validateConfig() error = %v",
					configErr,
					serverErr,
				)
			}
		})
	}
}

// TestServerConfigMappingFillsEveryTransportBound closes the gap the two corpora
// above cannot close themselves: each compares only the values it varies, so a
// bound added to both owners without a case there goes on being reported as
// agreed.
//
// It asks the question from the target side, which is what makes it survive a
// field nobody remembered to name here. Every [Config] field is a bound filled
// from configuration — transport credentials are on [Options], because the
// composition root builds them rather than copying a bound — so one still at its
// zero value after mapping a corpus that sets every knob means either
// serverConfigFromRuntime dropped it or this corpus never set its source, and
// both need the same look.
//
// The subject is this file's oracle, which the two corpora above run every case
// through. The production mapping is a different owner with its own copy of this
// question; serverConfigFromRuntime names it.
func TestServerConfigMappingFillsEveryTransportBound(t *testing.T) {
	loaded, err := loadGRPCServerConfig(t, map[string]string{
		"MAX_CONNECTIONS":                "1000",
		"MAX_CONCURRENT_RPCS":            "8",
		"MAX_CONCURRENT_STREAMS":         "16",
		"MAX_HEADER_LIST_BYTES":          "8192",
		"MAX_RECEIVE_MESSAGE_BYTES":      "65536",
		"MAX_SEND_MESSAGE_BYTES":         "65536",
		"ACCESS_LOG_HEALTH_CHECKS":       "true",
		"ACCESS_LOG_SUCCESS_SAMPLE_RATE": "0.5",
		"ACCESS_LOG_SLOW_THRESHOLD":      "250ms",
		"TELEMETRY_HEALTH_CHECKS":        "true",
	})
	if err != nil {
		t.Fatalf("configuration load rejected the mapping corpus: %v", err)
	}

	mapped := reflect.ValueOf(serverConfigFromRuntime(loaded))
	for index := range mapped.NumField() {
		if mapped.Field(index).IsZero() {
			t.Errorf(
				"Config.%s is zero after mapping: either serverConfigFromRuntime "+
					"does not set it, or this test's corpus does not set its source",
				mapped.Type().Field(index).Name,
			)
		}
	}
}

// loadGRPCServerConfig runs one corpus entry through the real configuration
// loader and returns the gRPC section it produced.
//
// Keys are the grpc.server suffix; the server stays disabled because the bounds
// under test are validated either way, which keeps every case free of the
// address and transport-security values an enabled server additionally demands.
func loadGRPCServerConfig(t *testing.T, env map[string]string) (config.GRPCServerConfig, error) {
	t.Helper()
	configtest.IsolateEnv(t)
	for key, value := range env {
		t.Setenv("APP__GRPC__SERVER__"+key, value)
	}

	cfg, _, err := config.LoadDetailed(config.LoadOptions{})
	if err != nil {
		return config.GRPCServerConfig{}, err //nolint:wrapcheck // The corpus compares this owner's verdict, not its wording.
	}
	return cfg.GRPC.Server, nil
}

func parseFloat(t *testing.T, value string) float64 {
	t.Helper()
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		t.Fatalf("strconv.ParseFloat(%q) error = %v", value, err)
	}
	return parsed
}

func parseDuration(t *testing.T, value string) time.Duration {
	t.Helper()
	parsed, err := time.ParseDuration(value)
	if err != nil {
		t.Fatalf("time.ParseDuration(%q) error = %v", value, err)
	}
	return parsed
}
