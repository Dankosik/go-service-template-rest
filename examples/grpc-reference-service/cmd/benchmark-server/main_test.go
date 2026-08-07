// The benchmark server as the harness that drives it sees it: does it start,
// publish the address it bound, serve an RPC, and stop on SIGTERM?
//
// It checks first that settingsFromDefaults carried the canonical gRPC defaults
// through unchanged. A benchmark run against a server with its own quietly
// diverged bounds produces numbers that do not describe this template, and
// nothing else in the run would say so.

package main

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	referencev1 "github.com/example/go-service-template-rest/examples/grpc-reference-service/internal/gen/proto/reference/v1"
	"github.com/example/go-service-template-rest/internal/config"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	grpcclient "github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc/credentials/insecure"
)

func TestBenchmarkServerProcessLifecycle(t *testing.T) {
	defaults := config.DefaultGRPCServerConfig()
	settings, err := settingsFromDefaults(defaults)
	if err != nil {
		t.Fatalf("settingsFromDefaults() error = %v", err)
	}
	if settings.maxConnections != defaults.MaxConnections {
		t.Fatalf("max connections = %d, want canonical %d", settings.maxConnections, defaults.MaxConnections)
	}
	assertTransportMirrorsDefaults(t, defaults, settings.transport)

	binary := filepath.Join(t.TempDir(), "grpc-benchmark-server")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build benchmark server: %v\n%s", err, output)
	}

	var stderr bytes.Buffer
	command := exec.CommandContext(t.Context(), binary)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- command.Wait()
	}()
	processExited := false
	t.Cleanup(func() {
		if processExited {
			return
		}
		_ = command.Process.Kill()
		select {
		case <-waitDone:
		case <-time.After(2 * time.Second):
		}
	})

	readyDone := make(chan struct {
		line string
		err  error
	}, 1)
	go func() {
		line, err := bufio.NewReader(stdout).ReadString('\n')
		readyDone <- struct {
			line string
			err  error
		}{line: line, err: err}
	}()

	var readyLine string
	select {
	case result := <-readyDone:
		if result.err != nil {
			t.Fatalf("read ready line: %v; stderr=%s", result.err, stderr.String())
		}
		readyLine = strings.TrimSpace(result.line)
	case err := <-waitDone:
		processExited = true
		t.Fatalf("benchmark server exited before readiness: %v; stderr=%s", err, stderr.String())
	case <-time.After(5 * time.Second):
		t.Fatal("benchmark server did not publish readiness within 5s")
	}

	if !strings.HasPrefix(readyLine, readyPrefix) {
		t.Fatalf("ready line = %q, want prefix %q", readyLine, readyPrefix)
	}
	address := strings.TrimPrefix(readyLine, readyPrefix)
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("ready address %q: %v", address, err)
	}
	if host != "127.0.0.1" || port == "" {
		t.Fatalf("ready address = %q, want 127.0.0.1:<allocated-port>", address)
	}

	connection, err := grpcclient.New(
		grpcclient.DefaultConfig(address),
		grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
	)
	if err != nil {
		t.Fatalf("build generated client connection: %v", err)
	}
	defer func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
		}
	}()

	rpcCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	response, err := referencev1.NewEchoServiceClient(connection).Unary(
		rpcCtx,
		referencev1.UnaryRequest_builder{Value: new("benchmark-process")}.Build(),
	)
	if err != nil {
		t.Fatalf("Unary() error = %v; stderr=%s", err, stderr.String())
	}
	if got := response.GetValue(); got != "benchmark-process" {
		t.Fatalf("Unary() value = %q, want benchmark-process", got)
	}

	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Signal(SIGTERM) error = %v", err)
	}
	select {
	case err := <-waitDone:
		processExited = true
		if err != nil {
			t.Fatalf("benchmark server exit = %v; stderr=%s", err, stderr.String())
		}
	case <-time.After(shutdownTimeout + 2*time.Second):
		t.Fatal("benchmark server did not exit within its shutdown budget")
	}

	probe, err := (&net.Dialer{Timeout: 250 * time.Millisecond}).DialContext(t.Context(), "tcp4", address)
	if err == nil {
		_ = probe.Close()
		t.Fatalf("benchmark listener %s remained reachable after process exit", address)
	}
	if command.ProcessState == nil || !command.ProcessState.Exited() {
		t.Fatalf("benchmark process did not report an exited state: %v", command.ProcessState)
	}

	t.Logf("benchmark server lifecycle closed on %s", address)
}

// assertTransportMirrorsDefaults holds every [grpcx.Config] bound to the
// identically named field of the canonical gRPC defaults.
//
// It walks the target type rather than naming the fields, because a listed
// conjunction goes on passing the day a tenth bound is added — and the number a
// benchmark reports for a server with one bound quietly left at its zero value
// describes no template. Asking from the target side is also what catches a
// field added to grpcx.Config with no source to fill it.
//
// This mapping is written out twice more, in cmd/service/internal/bootstrap and
// in internal/infra/grpc's parity oracle, because neither is importable from
// here; each carries this same question as its own test.
func assertTransportMirrorsDefaults(t *testing.T, defaults config.GRPCServerConfig, mapped grpcx.Config) {
	t.Helper()

	source := reflect.ValueOf(defaults)
	target := reflect.ValueOf(mapped)
	for index := range target.NumField() {
		name := target.Type().Field(index).Name
		origin := source.FieldByName(name)
		if !origin.IsValid() {
			t.Errorf("grpcx.Config.%s has no config.GRPCServerConfig field to carry", name)
			continue
		}
		if !origin.Equal(target.Field(index)) {
			t.Errorf(
				"transport bound %s = %v, want canonical default %v",
				name,
				target.Field(index),
				origin,
			)
		}
	}
}
