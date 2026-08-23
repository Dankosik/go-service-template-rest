//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestNATSConsumerSaturation(t *testing.T) {
	f := newNATSFixture(t)
	const maxDeliveryBytes = testMaxDeliveryBytes
	maxPayloadBytes := maxDeliveryBytes - natsjs.HeaderLimitBytes
	client := f.client(t, func(cfg *natsjs.Config) {
		cfg.MaxPayloadBytes = maxPayloadBytes
	})
	stream, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup saturation source stream: %v", err)
	}
	wireSizes := make([]int, 0, 3)
	for index := 0; index < 3; index++ {
		event := testEvent(strings.Repeat("x", maxPayloadBytes))
		result, err := client.Producer().Publish(t.Context(), event)
		if err != nil {
			t.Fatalf("publish near-limit event %d: %v", index+1, err)
		}
		raw, err := stream.GetMsg(t.Context(), result.Sequence)
		if err != nil {
			t.Fatalf("read near-limit event %d: %v", index+1, err)
		}
		size := (&nats.Msg{Subject: raw.Subject, Header: raw.Header, Data: raw.Data}).Size()
		if size < maxPayloadBytes || size > maxDeliveryBytes {
			t.Fatalf("near-limit event %d wire size = %d, want [%d,%d]", index+1, size, maxPayloadBytes, maxDeliveryBytes)
		}
		wireSizes = append(wireSizes, size)
	}
	if wireSizes[0]+wireSizes[1] > natsjs.ResidentDeliveryLimit {
		t.Fatalf("two active delivery wire sizes = %d, exceed resident limit %d", wireSizes[0]+wireSizes[1], natsjs.ResidentDeliveryLimit)
	}

	entered := make(chan int, 3)
	release := make(chan struct{})
	workerCfg := testWorkerConfig()
	workerCfg.Consumer = "saturation-worker"
	workerCfg.FilterSubject = sourceSubject
	workerCfg.DeadLetterSubject = deadLetterSubject
	workerCfg.MaxConcurrency = 2
	workerCfg.MaxDeliveryBytes = maxDeliveryBytes
	worker, err := client.NewWorker(t.Context(), workerCfg, func(ctx context.Context, msg natsjs.Message) error {
		entered <- len(msg.Payload())
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil
	})
	if err != nil {
		t.Fatalf("create saturation worker: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = worker.Run(runCtx)
	}()
	t.Cleanup(func() {
		cancelRun()
		stopWorker(worker)
		<-done
	})
	first := waittest.Receive(t, entered, 5*time.Second, "first handler")
	second := waittest.Receive(t, entered, 5*time.Second, "second handler")
	if first != maxPayloadBytes || second != maxPayloadBytes {
		t.Fatalf("active handler payload bytes = %d,%d, want %d each", first, second, maxPayloadBytes)
	}
	consumer, err := f.js.Consumer(t.Context(), sourceStream, "saturation-worker")
	if err != nil {
		t.Fatalf("lookup saturated consumer: %v", err)
	}
	info, err := consumer.Info(t.Context())
	if err != nil {
		t.Fatalf("read saturated consumer: %v", err)
	}
	if info.NumAckPending != 2 || info.NumPending < 1 || info.NumWaiting > 2 {
		t.Fatalf("saturated consumer state = ack_pending:%d pending:%d waiting:%d, want 2, >=1, <=2", info.NumAckPending, info.NumPending, info.NumWaiting)
	}
	close(release)
	_ = waittest.Receive(t, entered, 5*time.Second, "third handler after capacity release")
}

func TestNATSHandlerAckAndRedelivery(t *testing.T) {
	f := newNATSFixture(t)
	type observedDelivery struct {
		message natsjs.Message
		at      time.Time
	}
	deliveries := make(chan observedDelivery, 2)
	var calls atomic.Int32
	client, _, errCh := f.worker(t, func(_ context.Context, msg natsjs.Message) error {
		deliveries <- observedDelivery{message: msg, at: time.Now()}
		if calls.Add(1) == 1 {
			return errors.New("retry")
		}
		return nil
	}, func(cfg *natsjs.WorkerConfig) {
		cfg.Consumer = "ack-redelivery-worker"
		cfg.RetryDelays = []time.Duration{50 * time.Millisecond}
	})
	event := testEvent("retry")
	if _, err := client.Producer().Publish(t.Context(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	var first observedDelivery
	select {
	case first = <-deliveries:
	case err := <-errCh:
		t.Fatalf("worker stopped before first delivery: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for first delivery")
	}
	second := waittest.Receive(t, deliveries, 5*time.Second, "redelivery")
	if first.message.MessageID() != second.message.MessageID() || first.message.PublicationID() != second.message.PublicationID() {
		t.Fatalf("identity changed across redelivery: first %q/%q second %q/%q", first.message.MessageID(), first.message.PublicationID(), second.message.MessageID(), second.message.PublicationID())
	}
	if second.message.Metadata().NumDelivered != 2 {
		t.Fatalf("redelivery NumDelivered = %d, want 2", second.message.Metadata().NumDelivered)
	}
	if elapsed := second.at.Sub(first.at); elapsed < 45*time.Millisecond {
		t.Fatalf("redelivery delay = %s, want at least configured 50ms minus scheduling tolerance", elapsed)
	}
	waitConsumerSettled(t, f, "ack-redelivery-worker")
	assertStreamMessages(t, f, deadLetterStream, 0)
}

func TestNATSRetryExhaustionAndCrashBudget(t *testing.T) {
	f := newNATSFixture(t)
	retryDelays := []time.Duration{20 * time.Millisecond, 20 * time.Millisecond, 20 * time.Millisecond, 20 * time.Millisecond}
	called := make(chan uint64, 5)
	client, _, errCh := f.worker(t, func(_ context.Context, msg natsjs.Message) error {
		called <- msg.Metadata().NumDelivered
		if msg.Metadata().NumDelivered == 5 {
			if err := f.js.DeleteStream(t.Context(), deadLetterStream); err != nil {
				t.Errorf("delete DLQ before final handoff: %v", err)
			}
		}
		return errors.New("retry")
	}, func(cfg *natsjs.WorkerConfig) {
		cfg.Consumer = "exhaustion-worker"
		cfg.HandlerTimeout = 50 * time.Millisecond
		cfg.RetryDelays = retryDelays
	})
	event := testEvent("exhaust")
	if _, err := client.Producer().Publish(t.Context(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	for attempt := uint64(1); attempt <= 5; attempt++ {
		if got := waittest.Receive(t, called, 5*time.Second, fmt.Sprintf("delivery %d", attempt)); got != attempt {
			t.Fatalf("delivery = %d, want %d", got, attempt)
		}
	}
	if err := waittest.Receive(t, errCh, 5*time.Second, "worker retaining source after unavailable DLQ"); !errors.Is(err, natsjs.ErrTerminal) {
		t.Fatalf("worker error after unavailable DLQ = %v, want ErrTerminal", err)
	}
	if _, err := f.js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name: deadLetterStream, Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage,
		MaxMsgSize: 2 * testMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("restore DLQ stream: %v", err)
	}

	secondClient := f.client(t)
	secondCfg := testWorkerConfig()
	secondCfg.Consumer = "exhaustion-worker"
	secondCfg.FilterSubject = sourceSubject
	secondCfg.DeadLetterSubject = deadLetterSubject
	secondCfg.HandlerTimeout = 50 * time.Millisecond
	secondCfg.RetryDelays = retryDelays
	unexpectedHandler := make(chan uint64, 1)
	secondWorker, err := secondClient.NewWorker(t.Context(), secondCfg, func(_ context.Context, msg natsjs.Message) error {
		unexpectedHandler <- msg.Metadata().NumDelivered
		return nil
	})
	if err != nil {
		t.Fatalf("create recovery worker: %v", err)
	}
	secondRunCtx, secondCancel := context.WithCancel(t.Context())
	defer secondCancel()
	go func() { _ = secondWorker.Run(secondRunCtx) }()
	waittest.Until(t, 15*time.Second, func(ctx context.Context) bool {
		stream, err := f.js.Stream(ctx, deadLetterStream)
		if err != nil {
			return false
		}
		_, err = stream.GetLastMsgForSubject(ctx, deadLetterSubject)
		return err == nil
	}, "delivery beyond budget to dead-letter transfer")
	select {
	case delivery := <-unexpectedHandler:
		t.Fatalf("handler invoked beyond finite budget at delivery %d", delivery)
	default:
	}
	dlq, err := f.js.Stream(t.Context(), deadLetterStream)
	if err != nil {
		t.Fatalf("lookup exhaustion DLQ: %v", err)
	}
	deadLetter, err := dlq.GetLastMsgForSubject(t.Context(), deadLetterSubject)
	if err != nil {
		t.Fatalf("read exhaustion DLQ: %v", err)
	}
	if got := deadLetter.Header.Get("Dead-Letter-Reason"); got != "exhausted" {
		t.Fatalf("exhaustion DLQ reason = %q", got)
	}
}

func TestNATSRetryCrashConsumesAttemptBudget(t *testing.T) {
	f := newNATSFixture(t)
	retryDelays := []time.Duration{20 * time.Millisecond, 20 * time.Millisecond, 20 * time.Millisecond, 20 * time.Millisecond}
	attempts := make(chan uint64, 4)
	client, worker, firstDone := f.worker(t, func(ctx context.Context, msg natsjs.Message) error {
		attempts <- msg.Metadata().NumDelivered
		if msg.Metadata().NumDelivered < 4 {
			return errors.New("retry")
		}
		<-ctx.Done()
		return ctx.Err()
	}, func(cfg *natsjs.WorkerConfig) {
		cfg.Consumer = "crash-budget-worker"
		cfg.HandlerTimeout = 50 * time.Millisecond
		cfg.RetryDelays = retryDelays
	})
	if _, err := client.Producer().Publish(t.Context(), testEvent("crash budget")); err != nil {
		t.Fatalf("publish crash-budget event: %v", err)
	}
	for attempt := uint64(1); attempt <= 4; attempt++ {
		if got := waittest.Receive(t, attempts, 5*time.Second, fmt.Sprintf("crash-budget delivery %d", attempt)); got != attempt {
			t.Fatalf("crash-budget delivery = %d, want %d", got, attempt)
		}
	}
	stopWorker(worker)
	_ = waittest.Receive(t, firstDone, 5*time.Second, "crashed worker stop")

	secondClient := f.client(t)
	cfg := testWorkerConfig()
	cfg.Consumer = "crash-budget-worker"
	cfg.FilterSubject = sourceSubject
	cfg.DeadLetterSubject = deadLetterSubject
	cfg.HandlerTimeout = 50 * time.Millisecond
	cfg.RetryDelays = retryDelays
	fifth := make(chan uint64, 1)
	secondWorker, err := secondClient.NewWorker(t.Context(), cfg, func(_ context.Context, msg natsjs.Message) error {
		fifth <- msg.Metadata().NumDelivered
		return nil
	})
	if err != nil {
		t.Fatalf("create crash recovery worker: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(t.Context())
	defer cancelRun()
	go func() { _ = secondWorker.Run(runCtx) }()
	if got := waittest.Receive(t, fifth, 15*time.Second, "fifth delivery after worker crash"); got != 5 {
		t.Fatalf("delivery after worker crash = %d, want final allowed attempt 5", got)
	}
	waitConsumerSettled(t, f, "crash-budget-worker")
	assertStreamMessages(t, f, deadLetterStream, 0)
}

func TestNATSPoisonDLQAndRedrive(t *testing.T) {
	f := newNATSFixture(t)
	var handlerCalls atomic.Int32
	_, _, _ = f.worker(t, func(context.Context, natsjs.Message) error {
		handlerCalls.Add(1)
		return nil
	})
	rawPayload := []byte("POISON_PAYLOAD")
	poison := nats.NewMsg(sourceSubject)
	poison.Header.Set("Message-Id", "poison-message")
	poison.Header.Set(jetstream.MsgIDHeader, "poison-publication")
	poison.Data = rawPayload
	ack, err := f.js.PublishMsg(t.Context(), poison, jetstream.WithMsgID("poison-publication"))
	if err != nil {
		t.Fatalf("publish poison: %v", err)
	}
	var dlq *jetstream.RawStreamMsg
	waittest.Until(t, 5*time.Second, func(ctx context.Context) bool {
		stream, streamErr := f.js.Stream(ctx, deadLetterStream)
		if streamErr != nil {
			return false
		}
		dlq, streamErr = stream.GetLastMsgForSubject(ctx, deadLetterSubject)
		return streamErr == nil
	}, "poison dead-letter transfer")
	if handlerCalls.Load() != 0 {
		t.Fatalf("handler calls for malformed poison = %d, want 0", handlerCalls.Load())
	}
	if !slices.Equal(dlq.Data, rawPayload) {
		t.Fatalf("DLQ payload = %q, want %q", dlq.Data, rawPayload)
	}

	redrive := testEvent("redrive")
	redrive.MessageID = dlq.Header.Get("Message-Id")
	redrive.Subject = dlq.Header.Get("Original-Subject")
	client := f.client(t)
	result, err := client.Producer().Publish(t.Context(), redrive)
	if err != nil || result.Duplicate {
		t.Fatalf("redrive result = %+v, error = %v", result, err)
	}
	waittest.Until(t, 5*time.Second, func(context.Context) bool { return handlerCalls.Load() == 1 }, "redriven poison delivery")
	stream, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup source stream for old-ID redrive: %v", err)
	}
	before, err := stream.Info(t.Context())
	if err != nil {
		t.Fatalf("read source state before old-ID redrive: %v", err)
	}
	oldIdentity := redrive
	oldIdentity.PublicationID = "poison-publication"
	duplicate, err := client.Producer().Publish(t.Context(), oldIdentity)
	if err != nil || !duplicate.Duplicate || duplicate.Sequence != ack.Sequence {
		t.Fatalf("old-ID redrive result = %+v, error = %v, want original duplicate sequence %d", duplicate, err, ack.Sequence)
	}
	after, err := stream.Info(t.Context())
	if err != nil {
		t.Fatalf("read source state after old-ID redrive: %v", err)
	}
	if after.State.Msgs != before.State.Msgs {
		t.Fatalf("old-ID redrive changed source message count from %d to %d", before.State.Msgs, after.State.Msgs)
	}
}
