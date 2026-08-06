// Resolver guard, selection: of the builders sanitizingResolverBuilders
// wraps, is the one grpc-go actually selects for a target always a wrapped one?
//
// Each case runs in a child process because the answer depends on process-global
// state — the resolver registry and resolver.GetDefaultScheme() — that a test
// cannot mutate without changing every other test in the binary. The child
// reports what it observed as a JSON record on stdout; see runResolverSelectionChild.
//
// resolver_internal_test.go covers the wrapper's own behavior, and
// resolver_live_test.go covers the guard over a real TLS connection.

package grpcclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

const (
	resolverSelectionScenarioEnv  = "GRPCCLIENT_RESOLVER_SELECTION_SCENARIO"
	resolverSelectionRecordPrefix = "resolver-selection-record:"
)

func TestResolverSelectionHarnessExposesUnwrappedAddressMetadata(t *testing.T) {
	unaryMetadata, _, serverTarget := startMetadataCaptureServer(t)
	builder := newSelectionResolverBuilder("selection-control")
	builder.address = strings.TrimPrefix(serverTarget, "passthrough:///")
	connection, err := grpc.NewClient(
		builder.scheme+":///service",
		grpc.WithResolvers(builder),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	callCtx, cancelCall := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelCall()
	if _, err := healthgrpc.NewHealthClient(connection).Check(
		callCtx,
		&healthgrpc.HealthCheckRequest{},
	); err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}
	if got := firstMetadataValue(<-unaryMetadata, "x-request-id"); got != "resolver-stale-request" {
		t.Fatalf("unwrapped resolver request ID = %q, want late metadata control", got)
	}
}

func TestResolverSelectionUsesNativeRulesInIsolatedProcesses(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		scenario     string
		wantScheme   string
		wantEndpoint string
		wantError    string
	}{
		{
			name:         "explicit registered scheme",
			scenario:     "explicit",
			wantScheme:   "selection-explicit",
			wantEndpoint: "service",
		},
		{
			name:         "bare target uses native dns default",
			scenario:     "bare",
			wantScheme:   "dns",
			wantEndpoint: "service.internal:8443",
		},
		{
			name:         "unregistered scheme falls back to native dns default",
			scenario:     "unregistered",
			wantScheme:   "dns",
			wantEndpoint: "selection-unregistered:///service",
		},
		{
			name:         "changed global default is respected",
			scenario:     "changed-default",
			wantScheme:   "selection-default",
			wantEndpoint: "service.internal:8443",
		},
		{
			name:      "missing effective builder fails construction",
			scenario:  "missing-effective-builder",
			wantError: `could not get resolver for default scheme: "selection-missing"`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			record := runResolverSelectionChild(t, testCase.scenario)
			if !strings.Contains(record.Error, testCase.wantError) {
				t.Fatalf("construction error = %q, want substring %q", record.Error, testCase.wantError)
			}
			if testCase.wantError != "" {
				return
			}
			if record.Scheme != testCase.wantScheme {
				t.Fatalf("resolver scheme = %q, want %q", record.Scheme, testCase.wantScheme)
			}
			if record.Endpoint != testCase.wantEndpoint {
				t.Fatalf("resolver endpoint = %q, want %q", record.Endpoint, testCase.wantEndpoint)
			}
			if record.WireRequestID != "" {
				t.Fatalf(
					"wire resolver request ID = %q, want removed by connection-local wrapper",
					record.WireRequestID,
				)
			}
		})
	}
}

func TestResolverSelectionChild(t *testing.T) {
	scenario := os.Getenv(resolverSelectionScenarioEnv)
	if scenario == "" {
		t.Skip("subprocess-only resolver selection fixture")
	}

	unaryMetadata, _, serverTarget := startMetadataCaptureServer(t)
	target, builder, missing := resolverSelectionFixture(t, scenario)
	if builder != nil {
		builder.address = strings.TrimPrefix(serverTarget, "passthrough:///")
		resolver.Register(builder)
	}

	connection, err := grpcclient.New(
		grpcclient.DefaultConfig(target),
		grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
	)
	if missing {
		writeResolverSelectionRecord(t, resolverSelectionRecord{Error: errorString(err)})
		if err == nil {
			_ = connection.Close()
			t.Fatal("grpcclient.New() error = nil, want missing effective builder rejected")
		}
		return
	}
	if err != nil {
		writeResolverSelectionRecord(t, resolverSelectionRecord{Error: err.Error()})
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	if builder == nil {
		_ = connection.Close()
		t.Fatal("resolverSelectionFixture() builder = nil for non-missing scenario")
	}
	defer func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
		}
	}()

	callCtx, cancelCall := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelCall()
	if _, err := healthgrpc.NewHealthClient(connection).Check(
		callCtx,
		&healthgrpc.HealthCheckRequest{},
	); err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}
	select {
	case selected := <-builder.called:
		incoming := <-unaryMetadata
		writeResolverSelectionRecord(t, resolverSelectionRecord{
			Scheme:        selected.URL.Scheme,
			Endpoint:      selected.Endpoint(),
			WireRequestID: firstMetadataValue(incoming, "x-request-id"),
		})
	case <-time.After(2 * time.Second):
		t.Fatal("selected resolver builder was not called")
	}
}

func resolverSelectionFixture(
	t *testing.T,
	scenario string,
) (string, *selectionResolverBuilder, bool) {
	t.Helper()

	switch scenario {
	case "explicit":
		return "selection-explicit:///service", newSelectionResolverBuilder("selection-explicit"), false
	case "bare":
		return "service.internal:8443", newSelectionResolverBuilder("dns"), false
	case "unregistered":
		return "selection-unregistered:///service", newSelectionResolverBuilder("dns"), false
	case "changed-default":
		resolver.SetDefaultScheme("selection-default")
		return "service.internal:8443", newSelectionResolverBuilder("selection-default"), false
	case "missing-effective-builder":
		resolver.SetDefaultScheme("selection-missing")
		return "service.internal:8443", nil, true
	default:
		t.Fatalf("unknown resolver selection scenario %q", scenario)
		return "", nil, false
	}
}

func runResolverSelectionChild(t *testing.T, scenario string) resolverSelectionRecord {
	t.Helper()

	output := runTestBinaryChild(
		t,
		"TestResolverSelectionChild",
		resolverSelectionScenarioEnv+"="+scenario,
	)

	for line := range strings.SplitSeq(string(output), "\n") {
		raw, ok := strings.CutPrefix(line, resolverSelectionRecordPrefix)
		if !ok {
			continue
		}
		var record resolverSelectionRecord
		if err := json.Unmarshal([]byte(raw), &record); err != nil {
			t.Fatalf("decode resolver selection child record %q: %v", raw, err)
		}
		return record
	}
	t.Fatalf("resolver selection child %q produced no record:\n%s", scenario, output)
	return resolverSelectionRecord{}
}

func writeResolverSelectionRecord(t *testing.T, record resolverSelectionRecord) {
	t.Helper()

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("encode resolver selection record: %v", err)
	}
	fmt.Printf("%s%s\n", resolverSelectionRecordPrefix, encoded)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type resolverSelectionRecord struct {
	Scheme        string `json:"scheme,omitempty"`
	Endpoint      string `json:"endpoint,omitempty"`
	WireRequestID string `json:"wire_request_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

type selectionResolverBuilder struct {
	scheme  string
	address string
	called  chan resolver.Target
}

func newSelectionResolverBuilder(scheme string) *selectionResolverBuilder {
	return &selectionResolverBuilder{
		scheme: scheme,
		called: make(chan resolver.Target, 1),
	}
}

func (b *selectionResolverBuilder) Build( //nolint:ireturn // Implements resolver.Builder for native selection proof.
	target resolver.Target,
	connection resolver.ClientConn,
	_ resolver.BuildOptions,
) (resolver.Resolver, error) {
	b.called <- target
	resolverMetadata := metadata.Pairs("x-request-id", "resolver-stale-request")
	if err := connection.UpdateState(resolver.State{
		Addresses: []resolver.Address{{
			Addr:     b.address,
			Metadata: &resolverMetadata,
		}},
	}); err != nil {
		return nil, fmt.Errorf("publish resolver selection state: %w", err)
	}
	return nopResolver{}, nil
}

func (b *selectionResolverBuilder) Scheme() string {
	return b.scheme
}

func firstMetadataValue(values metadata.MD, key string) string {
	entries := values.Get(key)
	if len(entries) == 0 {
		return ""
	}
	return entries[0]
}
