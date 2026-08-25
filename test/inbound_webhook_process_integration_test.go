//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/waittest"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

const disclosureCanary = "provider-canary-text"

func TestInboundWebhookProcessRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	receiver := inboundReceiver(t, dsn)
	result, err := receiver.Receive(ctx, inboundDelivery("orders", inboundVectorID, inboundVectorBody, inboundVectorSignature))
	if err != nil || result != inboundwebhook.OutcomeAccepted {
		t.Fatalf("accept=%+v err=%v", result, err)
	}

	binary := filepath.Join(t.TempDir(), "jobs-worker")
	build := exec.CommandContext(ctx, "go", "build", "-tags", "inbound_webhook_test_worker", "-o", binary, "./cmd/jobs-worker")
	build.Dir = ".."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build jobs worker: %v\n%s", err, output)
	}

	marker := filepath.Join(t.TempDir(), "handled")
	endpoints := `{"endpoints":[{"endpoint_id":"orders","active_key_reference":"active"}]}`
	startWorker := func() *exec.Cmd {
		var output bytes.Buffer
		process := exec.CommandContext(ctx, binary)
		process.Stdout = &output
		process.Stderr = &output
		process.Env = append(os.Environ(),
			"APP__APP__ENV=local",
			"APP__POSTGRES__ENABLED=true",
			"APP__POSTGRES__DSN="+dsn,
			"APP__POSTGRES__MAX_OPEN_CONNS=4",
			"APP__JOBS__MAX_WORKERS=1",
			"APP__INBOUND_WEBHOOKS__ENDPOINTS="+endpoints,
			"APP__OBSERVABILITY__METRICS__ADDR="+waittest.FreeTCPAddr(t, "inbound diagnostics"),
			"INBOUND_WEBHOOK_TEST_MARKER="+marker,
		)
		if err := process.Start(); err != nil {
			t.Fatalf("start jobs worker: %v", err)
		}
		t.Cleanup(func() {
			if process.ProcessState == nil {
				_ = process.Process.Kill()
				_, _ = process.Process.Wait()
			}
		})
		return process
	}

	first := startWorker()
	waittest.Until(t, 30*time.Second, func(context.Context) bool {
		marked, err := os.ReadFile(marker)
		return err == nil && slices.Contains(strings.Fields(string(marked)), inboundVectorID)
	}, "first inbound worker handled receipt")
	_ = first.Process.Signal(syscall.SIGTERM)
	_, _ = first.Process.Wait()
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var outcome string
	if err := pool.QueryRow(ctx, `SELECT outcome FROM inbound_webhook_receipts WHERE delivery_id = $1`, inboundVectorID).Scan(&outcome); err != nil {
		t.Fatal(err)
	}
	if outcome != "handled" {
		t.Fatalf("first receipt outcome = %s", outcome)
	}
	startWorker()
	controlID := "msg_restart_control"
	controlBody := `{"hello":"control"}`
	webhook, err := standardwebhooks.NewWebhookRaw(inboundKey())
	if err != nil {
		t.Fatal(err)
	}
	controlSignature, err := webhook.Sign(controlID, time.Unix(1700000000, 0).UTC(), []byte(controlBody))
	if err != nil {
		t.Fatal(err)
	}
	control, err := receiver.Receive(ctx, inboundDelivery("orders", controlID, controlBody, controlSignature))
	if err != nil || control != inboundwebhook.OutcomeAccepted {
		t.Fatalf("control=%+v err=%v", control, err)
	}
	waittest.Until(t, 30*time.Second, func(context.Context) bool {
		marked, err := os.ReadFile(marker)
		return err == nil && slices.Contains(strings.Fields(string(marked)), controlID)
	}, "restarted inbound worker handled control receipt")
	marked, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	var originalCalls, controlCalls int
	for _, id := range strings.Fields(string(marked)) {
		switch id {
		case inboundVectorID:
			originalCalls++
		case controlID:
			controlCalls++
		}
	}
	if originalCalls != 1 || controlCalls != 1 {
		t.Fatalf("marker calls original=%d control=%d: %q", originalCalls, controlCalls, marked)
	}
}

func TestInboundWebhookDisclosureBoundary(t *testing.T) {
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	receiver := inboundReceiver(t, dsn)
	rejected, err := receiver.Receive(ctx, inboundDelivery("orders", inboundVectorID, `{"secret":"`+disclosureCanary+`"}`, "v1,bad"))
	if err != nil || rejected != inboundwebhook.OutcomeRejected {
		t.Fatalf("rejected=%+v err=%v", rejected, err)
	}

	acceptedBody := `{"hello":"` + disclosureCanary + `"}`
	webhook, err := standardwebhooks.NewWebhookRaw(inboundKey())
	if err != nil {
		t.Fatal(err)
	}
	signature, err := webhook.Sign("msg_disclosure", time.Unix(1700000000, 0).UTC(), []byte(acceptedBody))
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := receiver.Receive(ctx, inboundDelivery("orders", "msg_disclosure", acceptedBody, signature))
	if err != nil || accepted != inboundwebhook.OutcomeAccepted {
		t.Fatalf("accepted=%+v err=%v", accepted, err)
	}

	binary := filepath.Join(t.TempDir(), "jobs-worker")
	build := exec.CommandContext(ctx, "go", "build", "-tags", "inbound_webhook_test_worker", "-o", binary, "./cmd/jobs-worker")
	build.Dir = ".."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build jobs worker: %v\n%s", err, output)
	}
	logPath := filepath.Join(t.TempDir(), "worker.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logFile.Close() })
	process := exec.CommandContext(ctx, binary)
	process.Stdout = logFile
	process.Stderr = logFile
	process.Env = append(os.Environ(),
		"APP__APP__ENV=local",
		"APP__POSTGRES__ENABLED=true",
		"APP__POSTGRES__DSN="+dsn,
		"APP__POSTGRES__MAX_OPEN_CONNS=4",
		"APP__JOBS__MAX_WORKERS=1",
		`APP__INBOUND_WEBHOOKS__ENDPOINTS={"endpoints":[{"endpoint_id":"orders","active_key_reference":"active"}]}`,
		"APP__OBSERVABILITY__METRICS__ADDR="+waittest.FreeTCPAddr(t, "inbound disclosure"),
	)
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if process.ProcessState == nil {
			_ = process.Process.Kill()
			_, _ = process.Process.Wait()
		}
	})
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	waittest.Until(t, 30*time.Second, func(waitCtx context.Context) bool {
		var outcome string
		err := pool.QueryRow(waitCtx, `SELECT outcome FROM inbound_webhook_receipts WHERE delivery_id = 'msg_disclosure'`).Scan(&outcome)
		return err == nil && outcome == "handled"
	}, "disclosure receipt reached handled")
	_ = process.Process.Kill()
	_, _ = process.Process.Wait()
	_ = logFile.Close()
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	var errorsJSON []byte
	if err := pool.QueryRow(ctx, `SELECT coalesce(json_agg(errors)::text, '[]') FROM river_job`).Scan(&errorsJSON); err != nil {
		t.Fatal(err)
	}
	for _, sink := range []string{string(errorsJSON), string(logged)} {
		if strings.Contains(sink, disclosureCanary) {
			t.Fatalf("canary leaked: %q", sink)
		}
	}
}
