//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/nats-io/nats.go/jetstream"
)

func TestNATSStartupAdmission(t *testing.T) {
	closedListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve unavailable endpoint: %v", err)
	}
	unavailableURL := "nats://" + closedListener.Addr().String()
	_ = closedListener.Close()
	unavailable := testClientConfig()
	unavailable.URLs = []string{unavailableURL}
	unavailable.AllowPlaintext = true
	unavailable.AllowUnauthenticated = true
	unavailable.Stream = sourceStream
	startupCtx, cancelStartup := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancelStartup()
	if _, err := natsjs.Connect(startupCtx, unavailable, natsjs.RoleProducer, natsjs.Observability{}); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Connect(unavailable) error = %v, want ErrRejected", err)
	}

	f := newNATSFixture(t)
	invalidCredentials := testClientConfig()
	invalidCredentials.URLs = []string{f.url}
	invalidCredentials.AllowPlaintext = true
	invalidCredentials.CredentialsFile = filepath.Join(t.TempDir(), "missing.creds")
	invalidCredentials.Stream = sourceStream
	if _, err := natsjs.Connect(t.Context(), invalidCredentials, natsjs.RoleProducer, natsjs.Observability{}); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Connect(unusable credentials) error = %v, want ErrRejected", err)
	}

	missing := testClientConfig()
	missing.URLs = []string{f.url}
	missing.AllowPlaintext = true
	missing.AllowUnauthenticated = true
	missing.Stream = "MISSING"
	if _, err := natsjs.Connect(t.Context(), missing, natsjs.RoleProducer, natsjs.Observability{}); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Connect(missing stream) error = %v, want ErrRejected", err)
	}

	client := f.client(t, natsjs.RoleWorker)
	cfg := testWorkerConfig()
	cfg.Consumer = "startup-admission"
	cfg.FilterSubject = sourceSubject
	cfg.DeadLetterSubject = "missing.dead.letter"
	if _, err := client.NewWorker(t.Context(), cfg, func(context.Context, natsjs.Message) error { return nil }); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("NewWorker(missing DLQ) error = %v, want ErrRejected", err)
	}

	cfg.DeadLetterSubject = deadLetterSubject
	stream, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup source stream: %v", err)
	}
	if _, err := stream.CreateConsumer(t.Context(), jetstream.ConsumerConfig{
		Name: cfg.Consumer, Durable: cfg.Consumer, AckPolicy: jetstream.AckExplicitPolicy,
		MaxAckPending: 1, FilterSubject: cfg.FilterSubject,
	}); err != nil {
		t.Fatalf("create old consumer: %v", err)
	}
	if _, err := client.NewWorker(t.Context(), cfg, func(context.Context, natsjs.Message) error { return nil }); err != nil {
		t.Fatalf("NewWorker(reconcile consumer) error = %v", err)
	}
	consumer, err := stream.Consumer(t.Context(), cfg.Consumer)
	if err != nil {
		t.Fatalf("lookup reconciled consumer: %v", err)
	}
	info, err := consumer.Info(t.Context())
	if err != nil {
		t.Fatalf("read reconciled consumer: %v", err)
	}
	if info.Config.MaxAckPending != cfg.MaxConcurrency || info.Config.MaxDeliver != -1 {
		t.Fatalf("reconciled consumer = pending %d, max deliver %d", info.Config.MaxAckPending, info.Config.MaxDeliver)
	}
}

func TestNATSAuthenticatedStartupAdmission(t *testing.T) {
	f := newAuthenticatedNATSFixture(t)
	valid := testClientConfig()
	valid.URLs = []string{f.url}
	valid.AllowPlaintext = true
	valid.CredentialsFile = f.credentialsFile
	valid.Stream = sourceStream
	client, err := natsjs.Connect(t.Context(), valid, natsjs.RoleProducer, natsjs.Observability{})
	if err != nil {
		t.Fatalf("Connect(valid broker credentials) error = %v", err)
	}
	t.Cleanup(client.Close)
	if !client.Ready() {
		t.Fatal("authenticated client was not ready")
	}

	invalid := valid
	invalid.CredentialsFile = f.invalidCredentialsFile
	if _, err := natsjs.Connect(t.Context(), invalid, natsjs.RoleProducer, natsjs.Observability{}); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Connect(invalid broker credentials) error = %v, want ErrRejected", err)
	}
}
