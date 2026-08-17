//go:build integration

package integration_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
)

func TestWebhookWorkerProcessLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("container signal lifecycle is Unix-specific")
	}
	image := strings.TrimSpace(os.Getenv("WEBHOOK_RUNTIME_IMAGE"))
	if image == "" {
		t.Fatal("WEBHOOK_RUNTIME_IMAGE is required")
	}
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	dsn := webhookDockerDSN(t, pool)
	secret := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	manifestJSON := fmt.Sprintf(`{"revision":1,"entries":[{"owner_scope":"owner-a","destination_id":"dest-a","key_reference":"key-a","secret":"whsec_%s"}]}`, secret)
	sequence := 0

	start := func() (string, string, <-chan error, *bytes.Buffer) {
		t.Helper()
		sequence++
		name := fmt.Sprintf("webhook-worker-test-%d-%d", os.Getpid(), sequence)
		address := waittest.FreeTCPAddr(t, "webhook worker diagnostics")
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			t.Fatal(err)
		}
		args := []string{
			"run", "--rm", "--name", name, "--add-host=host.docker.internal:host-gateway",
			"-p", "127.0.0.1:" + port + ":9090", "--entrypoint", "/webhook-worker",
		}
		for _, value := range []string{
			"APP__APP__ENV=integration",
			"APP__OBSERVABILITY__METRICS__ADDR=:9090",
			"APP__POSTGRES__ENABLED=true", "APP__POSTGRES__DSN=" + dsn,
			"APP__POSTGRES__MAX_OPEN_CONNS=4", "APP__POSTGRES__MIN_IDLE_CONNS=0",
			"APP__POSTGRES__CONNECT_TIMEOUT=1s", "APP__POSTGRES__HEALTHCHECK_TIMEOUT=1s",
			"APP__POSTGRES__ACQUIRE_TIMEOUT=100ms", "APP__POSTGRES__STATEMENT_TIMEOUT=500ms",
			"APP__HTTP__REQUEST_TIMEOUT=2s", "APP__HTTP__GRACE_PERIOD=25s",
			"APP__HTTP__SHUTDOWN_TIMEOUT=3s", "APP__HTTP__WRITE_TIMEOUT=2s",
			"APP__HTTP__READINESS_TIMEOUT=1s", "APP__HTTP__READINESS_PROPAGATION_DELAY=0s",
			"APP__WEBHOOKS__ENABLED=true", "APP__WEBHOOKS__CAPACITY_REVISION=1",
			"APP__WEBHOOKS__GLOBAL_CONCURRENCY=2", "APP__WEBHOOKS__CLAIM_SCAN_PAGE=4",
			"APP__WEBHOOKS__POLL_INTERVAL=50ms", "APP__WEBHOOKS__OBSERVATION_INTERVAL=100ms",
			"APP__WEBHOOKS__STORE_OPERATION_TIMEOUT=200ms", "APP__WEBHOOKS__ATTEMPT_TIMEOUT=5s",
			"APP__WEBHOOKS__RESPONSE_HEADER_TIMEOUT=2s", "APP__WEBHOOKS__RESPONSE_HEADER_BYTES=4096",
			"APP__WEBHOOKS__RESPONSE_BODY_BYTES=4096", "APP__WEBHOOKS__DRAIN_TIMEOUT=10s",
			"APP__WEBHOOKS__MAINTENANCE_INTERVAL=100ms", "APP__WEBHOOKS__MAINTENANCE_BATCH=10",
			"APP__WEBHOOKS__STATIC_SECRETS=" + manifestJSON,
		} {
			args = append(args, "-e", value)
		}
		args = append(args, image)
		output := &bytes.Buffer{}
		process := exec.Command("docker", args...)
		process.Stdout, process.Stderr = output, output
		if err := process.Start(); err != nil {
			t.Fatalf("start webhook worker image: %v", err)
		}
		waited := make(chan error, 1)
		go func() { waited <- process.Wait() }()
		t.Cleanup(func() {
			if t.Failed() {
				t.Logf("webhook worker output:\n%s", output.String())
			}
			_ = exec.Command("docker", "rm", "-f", name).Run()
			select {
			case <-waited:
			default:
			}
		})
		return name, address, waited, output
	}

	stop := func(name string, waited <-chan error, output *bytes.Buffer, wantSuccess bool) {
		t.Helper()
		command := exec.CommandContext(t.Context(), "docker", "stop", "--time=25", name)
		if stopOutput, err := command.CombinedOutput(); err != nil {
			t.Fatalf("stop webhook worker: %v\n%s\n%s", err, stopOutput, output.String())
		}
		select {
		case err := <-waited:
			if (err == nil) != wantSuccess {
				t.Fatalf("webhook worker exit = %v, want success=%t\n%s", err, wantSuccess, output.String())
			}
		case <-time.After(30 * time.Second):
			t.Fatalf("webhook worker did not exit\n%s", output.String())
		}
	}

	name, address, waited, output := start()
	waitWebhookStatus(t, address, http.StatusOK)
	lock, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Exec(ctx, `LOCK TABLE webhook_clock IN ACCESS EXCLUSIVE MODE`); err != nil {
		_ = lock.Rollback(ctx)
		t.Fatal(err)
	}
	waitWebhookStatus(t, address, http.StatusServiceUnavailable)
	if err := lock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	waitWebhookStatus(t, address, http.StatusOK)
	stop(name, waited, output, true)

	name, address, waited, output = start()
	waitWebhookStatus(t, address, http.StatusOK)
	if killOutput, err := exec.CommandContext(t.Context(), "docker", "kill", "--signal=KILL", name).CombinedOutput(); err != nil {
		t.Fatalf("kill webhook worker: %v\n%s", err, killOutput)
	}
	select {
	case err := <-waited:
		if err == nil {
			t.Fatalf("SIGKILL worker exited successfully\n%s", output.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("killed webhook worker did not exit\n%s", output.String())
	}

	prepared := webhookPrepared(t, "process-recovery")
	acceptance := prepared.Acceptance
	acceptance.Destinations = []postgreswebhook.DestinationSnapshot{prepared.Destinations[0].DestinationSnapshot}
	acceptance.Destinations[0].Policy.BackoffBase = time.Minute
	acceptance.Destinations[0].Policy.BackoffCap = time.Minute
	prepared, err = postgreswebhook.PrepareAcceptance(acceptance)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "crashed-worker", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim(process recovery) = %+v, %v", claim, err)
	}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", KeyReferences: []string{"key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}

	name, address, waited, output = start()
	waitWebhookStatus(t, address, http.StatusOK)
	var deliveries, attempts int
	if err := pool.PGX().QueryRow(ctx, `SELECT (SELECT count(*) FROM webhook_deliveries), (SELECT count(*) FROM webhook_attempts)`).Scan(&deliveries, &attempts); err != nil {
		t.Fatal(err)
	}
	if deliveries != 1 || attempts != 1 {
		t.Fatalf("readiness probes changed durable work: deliveries=%d attempts=%d", deliveries, attempts)
	}
	var state, summary string
	var leased int
	var retryDue, retryDelay bool
	if err := pool.PGX().QueryRow(ctx, `SELECT d.state, d.cumulative_summary,
        (SELECT count(*) FROM webhook_capacity_slots WHERE attempt_id IS NOT NULL),
        d.next_due_at > clock_timestamp(),
        a.retry_delay_ns > 0
      FROM webhook_deliveries d
      JOIN webhook_attempts a ON a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id
      WHERE d.delivery_id = $1`, claim.Attempt.Identity.DeliveryID).Scan(&state, &summary, &leased, &retryDue, &retryDelay); err != nil {
		t.Fatal(err)
	}
	if state != "scheduled" || summary != "outcome_unknown" || leased != 0 || !retryDue || !retryDelay {
		t.Fatalf("restarted recovery = %s/%s leased=%d retry_due=%t retry_delay=%t", state, summary, leased, retryDue, retryDelay)
	}
	stop(name, waited, output, true)
}

func waitWebhookStatus(t *testing.T, address string, status int) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	waittest.Until(t, 20*time.Second, func() bool {
		response, err := client.Get("http://" + address + "/health/ready")
		if err != nil {
			return false
		}
		_ = response.Body.Close()
		return response.StatusCode == status
	}, "webhook worker readiness")
}

func webhookDockerDSN(t *testing.T, pool *postgres.Pool) string {
	t.Helper()
	config := pool.PGX().Config().ConnConfig
	return (&url.URL{
		Scheme: "postgres", User: url.UserPassword(config.User, config.Password),
		Host: net.JoinHostPort("host.docker.internal", strconv.Itoa(int(config.Port))), Path: "/" + config.Database,
		RawQuery: url.Values{"sslmode": []string{"disable"}}.Encode(),
	}).String()
}
