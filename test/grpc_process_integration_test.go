//go:build integration

// The gRPC transport as a shipped process: a generated service built from the
// template, started, dialed over a real port, and stopped with SIGTERM.
//
// Everything here is proven in-process somewhere else. What only this file can
// answer is whether the pieces hold once separated by a binary boundary and
// driven by environment variables: that the generated service turns APP__GRPC__*
// into an endpoint reporting SERVING and that SIGTERM ends the process through
// graceful shutdown.
//
// It is behind the integration tag because it compiles a service and needs a
// database, so it does not belong in the edit loop.

package integration_test

import (
	"bytes"
	"context"
	"fmt"
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
	"github.com/example/go-service-template-rest/internal/waittest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
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
	serviceRoot := initializedProcessServiceRoot(t, repositoryRoot)
	binaryPath := filepath.Join(t.TempDir(), "service")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binaryPath, "./cmd/service")
	build.Dir = serviceRoot
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("build service: %v\n%s", buildErr, output)
	}

	httpAddr := waittest.FreeTCPAddr(t, "service HTTP")
	grpcAddr := waittest.FreeTCPAddr(t, "service gRPC")
	postgresDSN := pgtest.DSN(t)
	var output bytes.Buffer
	process := exec.Command(binaryPath)
	process.Dir = serviceRoot
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
		grpcAddr,
		grpcclient.Options{
			TransportCredentials: insecure.NewCredentials(),
		},
	)
	if err != nil {
		t.Fatalf("build gRPC health client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	healthClient := healthgrpc.NewHealthClient(conn)
	var checkErr error
	waittest.UntilFunc(t, 10*time.Second, func(ctx context.Context) bool {
		response, err := healthClient.Check(
			ctx,
			&healthgrpc.HealthCheckRequest{},
		)
		checkErr = err
		if checkErr == nil && response.GetStatus() == healthgrpc.HealthCheckResponse_SERVING {
			return true
		}
		return false
	}, func() string { return fmt.Sprintf("gRPC health to become serving: %v\n%s", checkErr, output.String()) })
	readyCtx, cancelReady := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelReady()
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

func initializedProcessServiceRoot(t *testing.T, repositoryRoot string) string {
	t.Helper()

	temporaryRoot := t.TempDir()
	serviceRoot := filepath.Join(temporaryRoot, "service")
	copyCurrentRepository(t, repositoryRoot, serviceRoot)
	initializeGit := exec.CommandContext(t.Context(), "git", "init", "-q")
	initializeGit.Dir = serviceRoot
	if output, err := initializeGit.CombinedOutput(); err != nil {
		t.Fatalf("initialize fixture Git repository: %v\n%s", err, output)
	}
	commitGit := exec.CommandContext(
		t.Context(),
		"git",
		"-c",
		"user.email=grpc-process-integration@example.com",
		"-c",
		"user.name=grpc-process-integration",
		"commit",
		"-q",
		"--allow-empty",
		"-m",
		"template checkout",
	)
	commitGit.Dir = serviceRoot
	if output, err := commitGit.CombinedOutput(); err != nil {
		t.Fatalf("create fixture Git HEAD: %v\n%s", err, output)
	}

	initialize := exec.CommandContext(
		t.Context(),
		"bash",
		"./scripts/init-module.sh",
		"github.com/acme/grpc-process-integration",
	)
	initialize.Dir = serviceRoot
	initialize.Env = append(
		cleanServiceEnvironment(os.Environ()),
		"CODEOWNER=@acme/platform",
		"DATABASE=postgres",
		"GRPC=enabled",
		"AUTHN=none",
		"REFERENCE_EXAMPLE=remove",
	)
	if output, err := initialize.CombinedOutput(); err != nil {
		t.Fatalf("initialize process-test service: %v\n%s", err, output)
	}
	return serviceRoot
}

func cleanServiceEnvironment(environment []string) []string {
	clean := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "APP__") || strings.HasPrefix(key, "OTEL_") {
			continue
		}
		switch key {
		case "CODEOWNER", "DATABASE", "GRPC", "AUTHN", "REFERENCE_EXAMPLE":
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}
