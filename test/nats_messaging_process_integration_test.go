//go:build integration

package integration_test

import (
	"bytes"
	"fmt"
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

	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestNATSServiceProducerOnlyProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process lifecycle is Unix-specific")
	}
	f := newNATSFixture(t)
	repositoryRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	serviceRoot := initializedMessagingServiceRoot(t, repositoryRoot)
	binary := filepath.Join(t.TempDir(), "service")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./cmd/service")
	build.Dir = serviceRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build initialized messaging service: %v\n%s", err, output)
	}

	start := func() (*exec.Cmd, <-chan error, *bytes.Buffer, string) {
		t.Helper()
		output := &bytes.Buffer{}
		httpAddress := waittest.FreeTCPAddr(t, "messaging process")
		process := exec.Command(binary)
		process.Dir = serviceRoot
		process.Env = append(cleanMessagingEnvironment(os.Environ()),
			"APP__APP__ENV=integration",
			"APP__HTTP__ADDR="+httpAddress,
			"APP__HTTP__READINESS_PROPAGATION_DELAY=0s",
			"APP__HEALTH__REFRESH_INTERVAL=1s",
			// A high cached-health threshold proves that messaging's immediate
			// readiness gate, rather than a completed background probe, returns 503.
			"APP__HEALTH__FAILURE_THRESHOLD=100",
			"APP__OBSERVABILITY__METRICS__ADDR=",
			"APP__MESSAGING__URLS="+f.url,
			"APP__MESSAGING__ALLOW_PLAINTEXT=true",
			"APP__MESSAGING__ALLOW_UNAUTHENTICATED=true",
			"APP__MESSAGING__STREAM="+sourceStream,
		)
		process.Stdout = output
		process.Stderr = output
		if err := process.Start(); err != nil {
			t.Fatalf("start initialized messaging service: %v", err)
		}
		waited := make(chan error, 1)
		finished := make(chan struct{})
		go func() {
			waited <- process.Wait()
			close(finished)
		}()
		t.Cleanup(func() {
			select {
			case <-finished:
			default:
				_ = process.Process.Kill()
				<-finished
			}
		})
		return process, waited, output, httpAddress
	}

	process, waited, output, httpAddress := start()
	waitHTTPStatus(t, httpAddress, http.StatusOK)
	stream, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup source stream: %v", err)
	}
	names := stream.ConsumerNames(t.Context())
	for name := range names.Name() {
		t.Fatalf("producer-only process created durable consumer %q", name)
	}
	if err := names.Err(); err != nil {
		t.Fatalf("list source consumers: %v", err)
	}
	if err := process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal initialized messaging service: %v", err)
	}
	if err := waittest.Receive(t, waited, 10*time.Second, "producer-only service shutdown"); err != nil {
		t.Fatalf("producer-only service shutdown error = %v\n%s", err, output.String())
	}

	process, waited, output, httpAddress = start()
	waitHTTPStatus(t, httpAddress, http.StatusOK)
	stopTimeout := 10 * time.Second
	if err := f.container.Stop(t.Context(), &stopTimeout); err != nil {
		t.Fatalf("stop NATS under producer-only service: %v", err)
	}
	waitHTTPStatus(t, httpAddress, http.StatusServiceUnavailable)
	if err := waittest.Receive(t, waited, 70*time.Second, "producer-only reconnect exhaustion"); err == nil {
		t.Fatalf("producer-only service exited zero after reconnect exhaustion\n%s", output.String())
	}
	if !strings.Contains(output.String(), "connection closed after reconnect exhaustion") {
		t.Fatalf("producer-only service exited without classified reconnect exhaustion\n%s", output.String())
	}
}

func TestNATSWorkerMainRejectsEmptyHandler(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for connection sentinel: %v", err)
	}
	accepted := make(chan struct{}, 1)
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = connection.Close()
			accepted <- struct{}{}
		}
	}()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repositoryRoot := filepath.Dir(workingDirectory)
	binary := filepath.Join(t.TempDir(), "worker")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./cmd/worker")
	build.Dir = repositoryRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build worker: %v\n%s", err, output)
	}
	run := exec.CommandContext(t.Context(), binary)
	run.Env = append(cleanMessagingEnvironment(os.Environ()),
		"APP__APP__ENV=integration",
		// profile:authn-oidc-jwt:start
		// Trust configuration has no default, and config rejects both an unknown
		// key and a missing required one, so the worker reaches its own handler
		// assertion below only when these match the compiled-in profile exactly.
		"APP__AUTHN__ISSUER=https://issuer.example.com",
		"APP__AUTHN__AUDIENCE=https://api.example.com",
		// profile:authn-oidc-jwt:end
		"APP__MESSAGING__URLS=nats://"+listener.Addr().String(),
		"APP__MESSAGING__ALLOW_PLAINTEXT=true",
		"APP__MESSAGING__ALLOW_UNAUTHENTICATED=true",
		"APP__MESSAGING__STREAM=EVENTS",
		"APP__MESSAGING__WORKER__CONSUMER=unregistered-worker",
		"APP__MESSAGING__WORKER__FILTER_SUBJECT=events.test",
		"APP__MESSAGING__WORKER__DEAD_LETTER_SUBJECT=dead.events.test",
		"APP__OBSERVABILITY__METRICS__ADDR=127.0.0.1:19090",
	)
	output, runErr := run.CombinedOutput()
	if runErr == nil || !strings.Contains(string(output), "worker feature handler builder is not registered") {
		t.Fatalf("worker result error = %v, output = %q", runErr, output)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close connection sentinel: %v", err)
	}
	<-acceptDone
	select {
	case <-accepted:
		t.Fatal("worker opened a broker connection before rejecting empty handler")
	default:
	}
}

func initializedMessagingServiceRoot(t *testing.T, repositoryRoot string) string {
	t.Helper()
	temporaryRoot := t.TempDir()
	serviceRoot := filepath.Join(temporaryRoot, "service")
	copyCurrentRepository(t, repositoryRoot, serviceRoot)
	initializeGit := exec.CommandContext(t.Context(), "git", "init", "-q")
	initializeGit.Dir = serviceRoot
	if output, err := initializeGit.CombinedOutput(); err != nil {
		t.Fatalf("initialize messaging fixture Git repository: %v\n%s", err, output)
	}
	stage := exec.CommandContext(t.Context(), "git", "add", "-A")
	stage.Dir = serviceRoot
	if output, err := stage.CombinedOutput(); err != nil {
		t.Fatalf("stage messaging fixture repository: %v\n%s", err, output)
	}
	commit := exec.CommandContext(t.Context(), "git", "-c", "user.name=messaging-process-integration", "-c", "user.email=messaging-process-integration@example.com", "commit", "-q", "--allow-empty", "-m", "template checkout")
	commit.Dir = serviceRoot
	if output, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("create messaging fixture HEAD: %v\n%s", err, output)
	}
	initialize := exec.CommandContext(t.Context(), "bash", "./scripts/init-module.sh", "github.com/acme/messaging-process-integration")
	initialize.Dir = serviceRoot
	initialize.Env = append(cleanMessagingEnvironment(os.Environ()),
		"CODEOWNER=@acme/platform",
		"DATABASE=none",
		"GRPC=none",
		"AUTHN=none",
		"OUTBOUND_HTTP=bounded",
		"MESSAGING=nats-jetstream",
		"REFERENCE_EXAMPLE=remove",
	)
	if output, err := initialize.CombinedOutput(); err != nil {
		t.Fatalf("initialize messaging process service: %v\n%s", err, output)
	}
	return serviceRoot
}

func cleanMessagingEnvironment(environment []string) []string {
	clean := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "APP__") || strings.HasPrefix(key, "OTEL_") {
			continue
		}
		switch key {
		case "CODEOWNER", "DATABASE", "GRPC", "AUTHN", "OUTBOUND_HTTP", "MESSAGING", "REFERENCE_EXAMPLE":
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}

func waitHTTPStatus(t *testing.T, address string, want int) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	waittest.Until(t, 10*time.Second, func() bool {
		response, err := client.Get("http://" + address + "/health/ready")
		if err != nil {
			return false
		}
		_ = response.Body.Close()
		return response.StatusCode == want
	}, fmt.Sprintf("HTTP readiness status %d", want))
}
