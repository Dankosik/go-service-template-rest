//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGRPCProcessLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process lifecycle is Unix-specific")
	}

	repositoryRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	binaryPath := filepath.Join(t.TempDir(), "service")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binaryPath, "./cmd/service")
	build.Dir = repositoryRoot
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("build service: %v\n%s", buildErr, output)
	}

	httpAddr := freeTCPAddress(t)
	grpcAddr := freeTCPAddress(t)
	postgresDSN := pgtest.DSN(t)
	var output bytes.Buffer
	process := exec.Command(binaryPath)
	process.Dir = repositoryRoot
	process.Env = append(
		cleanServiceEnvironment(os.Environ()),
		"APP__APP__ENV=integration",
		"APP__HTTP__ADDR="+httpAddr,
		"APP__HTTP__READINESS_PROPAGATION_DELAY=0s",
		"APP__OBSERVABILITY__METRICS__ADDR=",
		"APP__POSTGRES__ENABLED=true",
		"APP__POSTGRES__DSN="+postgresDSN,
		"APP__GRPC__SERVER__ENABLED=true",
		"APP__GRPC__SERVER__ADDR="+grpcAddr,
		"APP__GRPC__SERVER__TRANSPORT_SECURITY=plaintext",
		"APP__GRPC__SERVER__ALLOW_PLAINTEXT=true",
	)
	process.Stdout = &output
	process.Stderr = &output
	if err := process.Start(); err != nil {
		t.Fatalf("start service: %v", err)
	}

	waited := make(chan error, 1)
	processFinished := make(chan struct{})
	go func() {
		waited <- process.Wait()
		close(processFinished)
	}()
	t.Cleanup(func() {
		select {
		case <-processFinished:
			return
		default:
		}
		_ = process.Process.Kill()
		<-processFinished
	})

	conn, err := grpcclient.New(
		grpcclient.DefaultConfig(grpcAddr),
		grpcclient.Options{
			TransportCredentials: insecure.NewCredentials(),
			Propagation:          grpcclient.PropagationTrustedService,
		},
	)
	if err != nil {
		t.Fatalf("build gRPC health client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	readyCtx, cancelReady := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelReady()
	healthClient := healthgrpc.NewHealthClient(conn)
	retry := time.NewTicker(20 * time.Millisecond)
	defer retry.Stop()
	const requestID = "grpc_process_integration_123"
	readyCallCtx := reqctx.ContextWithRequestID(readyCtx, requestID)
	var responseHeader metadata.MD
	for {
		response, checkErr := healthClient.Check(
			readyCallCtx,
			&healthgrpc.HealthCheckRequest{},
			grpc.Header(&responseHeader),
		)
		if checkErr == nil && response.GetStatus() == healthgrpc.HealthCheckResponse_SERVING {
			break
		}
		select {
		case <-readyCtx.Done():
			_ = process.Process.Kill()
			<-waited
			t.Fatalf("gRPC health did not become serving: %v\n%s", checkErr, output.String())
		case <-retry.C:
		}
	}
	if got := responseHeader.Get("x-request-id"); len(got) != 1 || got[0] != requestID {
		t.Fatalf("gRPC health response request IDs = %v, want %q", got, requestID)
	}
	_, err = healthClient.Check(readyCtx, &healthgrpc.HealthCheckRequest{
		Service: "service.not.registered",
	})
	if code := status.Code(err); code != codes.NotFound {
		t.Fatalf("unknown gRPC health service code = %s, want %s", code, codes.NotFound)
	}

	httpClient := &http.Client{Timeout: time.Second}
	response, err := httpClient.Get("http://" + httpAddr + "/health/ready")
	if err != nil {
		t.Fatalf("HTTP readiness request: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("HTTP readiness status = %d, want 200", response.StatusCode)
	}

	if err := process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal service: %v", err)
	}
	select {
	case err := <-waited:
		if err != nil {
			t.Fatalf("service shutdown: %v\n%s", err, output.String())
		}
	case <-time.After(10 * time.Second):
		_ = process.Process.Kill()
		<-waited
		t.Fatalf("service did not finish graceful shutdown\n%s", output.String())
	}
}

func freeTCPAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve TCP address: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release TCP address %s: %v", addr, err)
	}
	return addr
}

func cleanServiceEnvironment(environment []string) []string {
	clean := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "APP__") || strings.HasPrefix(key, "OTEL_") {
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}
