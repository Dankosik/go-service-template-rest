//go:build integration

package integration_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestNATSForcedShutdownRedelivers(t *testing.T) {
	f := newNATSFixture(t)
	entered := make(chan struct{})
	client, worker, _ := f.worker(t, func(ctx context.Context, _ natsjs.Message) error {
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	}, func(cfg *natsjs.WorkerConfig) {
		cfg.Consumer = "forced-worker"
		cfg.HandlerTimeout = 100 * time.Millisecond
	})
	if _, err := client.Producer().Publish(t.Context(), testEvent("forced")); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	waittest.ReceiveSignal(t, entered, 5*time.Second, "blocked handler")
	shutdownCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if err := worker.Shutdown(shutdownCtx); err == nil {
		t.Fatal("Shutdown() error = nil, want forced shutdown")
	}

	redelivered := make(chan struct{}, 1)
	secondClient := f.client(t, natsjs.RoleWorker)
	cfg := testWorkerConfig()
	cfg.Consumer = "forced-worker"
	cfg.FilterSubject = sourceSubject
	cfg.DeadLetterSubject = deadLetterSubject
	cfg.HandlerTimeout = 100 * time.Millisecond
	secondWorker, err := secondClient.NewWorker(t.Context(), cfg, func(context.Context, natsjs.Message) error {
		redelivered <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("create second worker: %v", err)
	}
	secondRunCtx, secondCancel := context.WithCancel(t.Context())
	defer secondCancel()
	go func() { _ = secondWorker.Run(secondRunCtx) }()
	waittest.ReceiveSignal(t, redelivered, 15*time.Second, "redelivery after force close")
}

func TestNATSHandlerPanicIsSupervised(t *testing.T) {
	f := newNATSFixture(t)
	producer := f.client(t, natsjs.RoleProducer)
	blockEntered := make(chan struct{})
	panicEntered := make(chan struct{})
	startedAfterFailure := make(chan string, 1)
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	client, _, errCh := f.worker(t, func(ctx context.Context, message natsjs.Message) error {
		switch payload := string(message.Payload()); payload {
		case "block":
			close(blockEntered)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		case "panic":
			close(panicEntered)
			panic("feature panic canary")
		default:
			startedAfterFailure <- payload
			return nil
		}
	}, func(cfg *natsjs.WorkerConfig) {
		cfg.Consumer = "panic-worker"
		cfg.MaxConcurrency = 3
	})
	if _, err := client.Producer().Publish(t.Context(), testEvent("block")); err != nil {
		t.Fatalf("publish blocking fixture: %v", err)
	}
	waittest.ReceiveSignal(t, blockEntered, 5*time.Second, "blocking handler")
	if _, err := client.Producer().Publish(t.Context(), testEvent("panic")); err != nil {
		t.Fatalf("publish panic fixture: %v", err)
	}
	waittest.ReceiveSignal(t, panicEntered, 5*time.Second, "panicking handler")
	waittest.Until(t, 5*time.Second, func() bool { return !client.Ready() }, "terminal fail-closed readiness")
	if err := client.Check(t.Context()); err == nil || client.Ready() {
		t.Fatalf("client recovered after terminal handler failure: check error = %v, ready = %t", err, client.Ready())
	}
	if _, err := client.Producer().Publish(t.Context(), testEvent("worker publication")); !errors.Is(err, natsjs.ErrDraining) {
		t.Fatalf("worker Publish(after terminal failure) error = %v, want ErrDraining", err)
	}
	if _, err := producer.Producer().Publish(t.Context(), testEvent("external publication")); err != nil {
		t.Fatalf("publish external post-failure fixture: %v", err)
	}
	waittest.Until(t, 5*time.Second, func() bool {
		consumer, lookupErr := f.js.Consumer(t.Context(), sourceStream, "panic-worker")
		if lookupErr != nil {
			return false
		}
		info, infoErr := consumer.Info(t.Context())
		return infoErr == nil && (info.NumPending >= 1 || info.NumAckPending >= 1)
	}, "post-failure message to remain broker-owned")
	select {
	case payload := <-startedAfterFailure:
		t.Fatalf("handler started %q after terminal failure", payload)
	default:
	}
	select {
	case err := <-errCh:
		t.Fatalf("worker returned before admitted handler drained: %v", err)
	default:
	}
	close(release)
	err := waittest.Receive(t, errCh, 5*time.Second, "handler panic reaching worker supervision")
	if !errors.Is(err, natsjs.ErrTerminal) || strings.Contains(err.Error(), "feature panic canary") {
		t.Fatalf("worker panic supervision error = %v, want sanitized ErrTerminal", err)
	}
	if client.Ready() {
		t.Fatal("client remained ready after handler panic")
	}
	consumer, err := f.js.Consumer(t.Context(), sourceStream, "panic-worker")
	if err != nil {
		t.Fatalf("lookup panic consumer: %v", err)
	}
	info, err := consumer.Info(t.Context())
	if err != nil {
		t.Fatalf("read panic consumer: %v", err)
	}
	if info.NumAckPending != 1 {
		t.Fatalf("panic consumer ack pending = %d, want source retained", info.NumAckPending)
	}
}

func TestNATSGracefulDrain(t *testing.T) {
	f := newNATSFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	pendingHandled := make(chan struct{}, 1)
	client, worker, _ := f.worker(t, func(_ context.Context, msg natsjs.Message) error {
		if string(msg.Payload()) == "pending" {
			pendingHandled <- struct{}{}
			return nil
		}
		close(entered)
		<-release
		return nil
	}, func(cfg *natsjs.WorkerConfig) {
		cfg.Consumer = "graceful-worker"
		cfg.MaxConcurrency = 1
	})
	if _, err := client.Producer().Publish(t.Context(), testEvent("in flight")); err != nil {
		t.Fatalf("publish in-flight message: %v", err)
	}
	waittest.ReceiveSignal(t, entered, 5*time.Second, "in-flight handler")
	producer := f.client(t, natsjs.RoleProducer)
	if _, err := producer.Producer().Publish(t.Context(), testEvent("pending")); err != nil {
		t.Fatalf("publish pending message: %v", err)
	}
	worker.StartDrain()
	if client.Ready() {
		t.Fatal("client remained ready after StartDrain")
	}
	if err := client.Check(t.Context()); err == nil || client.Ready() {
		t.Fatalf("client readiness recovered during drain: check error = %v, ready = %t", err, client.Ready())
	}
	if _, err := client.Producer().Publish(t.Context(), testEvent("during drain")); !errors.Is(err, natsjs.ErrDraining) {
		t.Fatalf("Publish(during drain) error = %v, want ErrDraining", err)
	}
	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelShutdown()
	shutdownErr := make(chan error, 1)
	go func() { shutdownErr <- worker.Shutdown(shutdownCtx) }()
	close(release)
	if err := waittest.Receive(t, shutdownErr, 5*time.Second, "graceful shutdown"); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	select {
	case <-pendingHandled:
		t.Fatal("worker admitted pending message after drain started")
	default:
	}

	recoveryClient := f.client(t, natsjs.RoleWorker)
	recoveryCfg := testWorkerConfig()
	recoveryCfg.Consumer = "graceful-worker"
	recoveryCfg.FilterSubject = sourceSubject
	recoveryCfg.DeadLetterSubject = deadLetterSubject
	recoveryCfg.MaxConcurrency = 1
	recovered := make(chan struct{}, 1)
	recoveryWorker, err := recoveryClient.NewWorker(t.Context(), recoveryCfg, func(_ context.Context, msg natsjs.Message) error {
		if string(msg.Payload()) == "pending" {
			recovered <- struct{}{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("create recovery worker: %v", err)
	}
	recoveryCtx, recoveryCancel := context.WithCancel(t.Context())
	defer recoveryCancel()
	go func() { _ = recoveryWorker.Run(recoveryCtx) }()
	waittest.ReceiveSignal(t, recovered, 10*time.Second, "pending message redelivery after graceful drain")
}
