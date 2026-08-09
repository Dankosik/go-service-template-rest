//go:build integration

package natsjs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs/natsjstest"
	"github.com/example/go-service-template-rest/internal/waittest"
	dockerclient "github.com/moby/moby/client"
	"github.com/nats-io/nats.go/jetstream"
)

func newPackageNATSFixture(t *testing.T) *natsjstest.Server {
	t.Helper()
	return natsjstest.Start(t, natsjstest.WithStreams(
		jetstream.StreamConfig{
			Name: "EVENTS", Subjects: []string{"events.>"},
			Storage: jetstream.FileStorage, MaxMsgSize: testMaxDeliveryBytes,
		},
		jetstream.StreamConfig{
			Name: "EVENTS_DLQ", Subjects: []string{"dead.>"},
			Storage: jetstream.FileStorage, MaxMsgSize: 2 * testMaxDeliveryBytes,
		},
	))
}

func packageClient(t *testing.T, f *natsjstest.Server, pending int) *Client {
	t.Helper()
	cfg := testConfig()
	cfg.URLs = []string{f.URL}
	cfg.AllowPlaintext = true
	cfg.AllowUnauthenticated = true
	cfg.Stream = "EVENTS"
	cfg.MaxPendingPublishes = pending
	client, err := Connect(t.Context(), cfg, RoleWorker, Observability{})
	if err != nil {
		t.Fatalf("connect messaging client: %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

func TestNATSWorkerRegistrationIsSingleton(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, testMaxPending)
	cfg := testWorkerConfig()
	cfg.Consumer = "singleton-first"
	cfg.FilterSubject = "events.test"
	cfg.DeadLetterSubject = "dead.events.test"
	if _, err := client.NewWorker(t.Context(), cfg, func(context.Context, Message) error { return nil }); err != nil {
		t.Fatalf("NewWorker(first) error = %v", err)
	}
	second := cfg
	second.Consumer = "singleton-second"
	if worker, err := client.NewWorker(t.Context(), second, func(context.Context, Message) error { return nil }); worker != nil || !errors.Is(err, ErrRejected) {
		t.Fatalf("NewWorker(second) = %#v, %v, want ErrRejected", worker, err)
	}
	if _, err := f.JS.Consumer(t.Context(), "EVENTS", second.Consumer); !errors.Is(err, jetstream.ErrConsumerNotFound) {
		t.Fatalf("second consumer lookup error = %v, want no broker mutation", err)
	}

	concurrentClient := packageClient(t, f, testMaxPending)
	concurrent := cfg
	concurrent.Consumer = "singleton-concurrent"
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := concurrentClient.NewWorker(t.Context(), concurrent, func(context.Context, Message) error { return nil })
			results <- err
		}()
	}
	close(start)
	var accepted, rejected int
	for range 2 {
		switch err := <-results; {
		case err == nil:
			accepted++
		case errors.Is(err, ErrRejected):
			rejected++
		default:
			t.Fatalf("concurrent NewWorker() error = %v", err)
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("concurrent NewWorker() accepted = %d, rejected = %d, want 1 each", accepted, rejected)
	}
}

func TestNATSConnectedTopologyErrorDoesNotEnterReconnect(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, testMaxPending)
	client.ready.Store(false)
	if client.waitForReconnect(t.Context(), jetstream.ErrConsumerDeleted) {
		t.Fatal("connected topology error entered reconnect recovery")
	}
}

func TestNATSReconnectProbeStopsWithRunContext(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, testMaxPending)
	probeEntered := make(chan struct{}, 1)
	client.js = blockingStreamLookup{JetStream: client.js, entered: probeEntered}

	runCtx, cancelRun := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	client.events <- eventReconnect
	go func() { runErr <- client.Run(runCtx) }()

	waittest.Receive(t, probeEntered, 5*time.Second, "reconnect probe")
	cancelRun()
	if err := waittest.Receive(t, runErr, time.Second, "messaging client cancellation"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}

func TestNATSPublishDispatchCancellationAndNoRetry(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, 2)

	before := client.nc.Stats().OutMsgs
	if _, err := client.Producer().Publish(t.Context(), Event{
		Subject: "unmatched.test", MessageID: NewID(), PublicationID: NewID(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("one attempt"),
	}); !errors.Is(err, ErrRejected) {
		t.Fatalf("Publish(no responder) error = %v, want ErrRejected", err)
	}
	if delta := client.nc.Stats().OutMsgs - before; delta != 1 {
		t.Fatalf("no-responder wire attempts = %d, want 1", delta)
	}

	docker, err := dockerclient.New(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("create Docker client: %v", err)
	}
	t.Cleanup(func() { _ = docker.Close() })
	source, err := f.JS.Stream(t.Context(), "EVENTS")
	if err != nil {
		t.Fatalf("lookup source stream: %v", err)
	}
	beforeState, err := source.Info(t.Context())
	if err != nil {
		t.Fatalf("read source state before ambiguous publishes: %v", err)
	}
	if _, err := docker.ContainerPause(t.Context(), f.Container.GetContainerID(), dockerclient.ContainerPauseOptions{}); err != nil {
		t.Fatalf("pause NATS: %v", err)
	}
	t.Cleanup(func() {
		_, _ = docker.ContainerUnpause(context.Background(), f.Container.GetContainerID(), dockerclient.ContainerUnpauseOptions{})
	})
	dispatchedBefore := client.nc.Stats().OutMsgs
	canceledEvent := Event{
		Subject: "events.test", MessageID: NewID(), PublicationID: NewID(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("cancel after dispatch"),
	}
	publishCtx, cancelPublish := context.WithCancel(t.Context())
	canceledErr := make(chan error, 1)
	go func() {
		_, err := client.Producer().Publish(publishCtx, canceledEvent)
		canceledErr <- err
	}()
	deadlineEvent := Event{
		Subject: "events.test", MessageID: NewID(), PublicationID: NewID(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("deadline after dispatch"),
	}
	deadlineCtx, cancelDeadline := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelDeadline()
	deadlineErr := make(chan error, 1)
	go func() {
		_, err := client.Producer().Publish(deadlineCtx, deadlineEvent)
		deadlineErr <- err
	}()
	waittest.Until(t, 3*time.Second, func() bool { return client.nc.Stats().OutMsgs >= dispatchedBefore+2 }, "publish dispatches")
	if _, err := client.Producer().Publish(t.Context(), Event{
		Subject: "events.test", MessageID: NewID(), PublicationID: NewID(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("over capacity"),
	}); !errors.Is(err, ErrCapacity) {
		t.Fatalf("Publish(over capacity) error = %v, want ErrCapacity", err)
	}
	cancelPublish()
	if err := <-canceledErr; !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("Publish(canceled after dispatch) error = %v, want ErrAmbiguous", err)
	}
	if err := <-deadlineErr; !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("Publish(deadline after dispatch) error = %v, want ErrAmbiguous", err)
	}
	if delta := client.nc.Stats().OutMsgs - dispatchedBefore; delta != 2 {
		t.Fatalf("ambiguous publish wire attempts = %d, want exactly one per call", delta)
	}
	if got := len(client.Producer().tokens); got != 0 {
		t.Fatalf("producer capacity tokens after ambiguous completion = %d, want 0", got)
	}
	if _, err := docker.ContainerUnpause(t.Context(), f.Container.GetContainerID(), dockerclient.ContainerUnpauseOptions{}); err != nil {
		t.Fatalf("resume NATS: %v", err)
	}
	fresh := Event{
		Subject: "events.test", MessageID: NewID(), PublicationID: NewID(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("capacity released"),
	}
	accepted, err := client.Producer().Publish(t.Context(), fresh)
	if err != nil {
		t.Fatalf("Publish(after ambiguous completion) error = %v", err)
	}
	counts := map[string]int{canceledEvent.PublicationID: 0, deadlineEvent.PublicationID: 0}
	for sequence := beforeState.State.LastSeq + 1; sequence <= accepted.Sequence; sequence++ {
		stored, getErr := source.GetMsg(t.Context(), sequence)
		if getErr != nil {
			t.Fatalf("read source sequence %d: %v", sequence, getErr)
		}
		if _, tracked := counts[stored.Header.Get(headerPublicationID)]; tracked {
			counts[stored.Header.Get(headerPublicationID)]++
		}
	}
	for publicationID, count := range counts {
		if count > 1 {
			t.Fatalf("ambiguous publication %q stored %d times, want at most once", publicationID, count)
		}
	}
}

func TestNATSHandlerAckAmbiguityRedelivers(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, testMaxPending)
	cfg := testWorkerConfig()
	cfg.Consumer = "ack-ambiguity"
	cfg.FilterSubject = "events.test"
	cfg.DeadLetterSubject = "dead.events.test"
	cfg.HandlerTimeout = 50 * time.Millisecond
	deliveries := make(chan uint64, 2)
	worker, err := client.NewWorker(t.Context(), cfg, func(_ context.Context, msg Message) error {
		deliveries <- msg.Metadata().NumDelivered
		return nil
	})
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}
	worker.consumer = wrappingConsumer{pullConsumer: worker.consumer, failDoubleAck: true}
	runCtx, cancelRun := context.WithCancel(t.Context())
	defer cancelRun()
	go func() { _ = worker.Run(runCtx) }()
	if _, err := client.Producer().Publish(t.Context(), Event{
		Subject: "events.test", MessageID: NewID(), PublicationID: NewID(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("ack ambiguity"),
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if first := waittest.Receive(t, deliveries, 5*time.Second, "first delivery"); first != 1 {
		t.Fatalf("first NumDelivered = %d, want 1", first)
	}
	if second := waittest.Receive(t, deliveries, 15*time.Second, "duplicate delivery"); second != 2 {
		t.Fatalf("second NumDelivered = %d, want 2", second)
	}
}

func TestNATSDLQSourceAckAmbiguityDeduplicates(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, testMaxPending)
	cfg := testWorkerConfig()
	cfg.Consumer = "dlq-ack-ambiguity"
	cfg.FilterSubject = "events.test"
	cfg.DeadLetterSubject = "dead.events.test"
	cfg.HandlerTimeout = 50 * time.Millisecond
	cfg.DeadLetterRetryDelay = 50 * time.Millisecond
	deliveries := make(chan uint64, 2)
	doubleAcks := make(chan struct{}, 2)
	worker, err := client.NewWorker(t.Context(), cfg, func(_ context.Context, msg Message) error {
		deliveries <- msg.Metadata().NumDelivered
		return Permanent(errors.New("poison"))
	})
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}
	worker.consumer = wrappingConsumer{pullConsumer: worker.consumer, failDoubleAck: true, doubleAcks: doubleAcks}
	runCtx, cancelRun := context.WithCancel(t.Context())
	defer cancelRun()
	go func() { _ = worker.Run(runCtx) }()
	if _, err := client.Producer().Publish(t.Context(), Event{
		Subject: "events.test", MessageID: NewID(), PublicationID: NewID(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("dlq ack ambiguity"),
	}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if first := waittest.Receive(t, deliveries, 5*time.Second, "first DLQ delivery"); first != 1 {
		t.Fatalf("first NumDelivered = %d, want 1", first)
	}
	if second := waittest.Receive(t, deliveries, 15*time.Second, "DLQ source-ack redelivery"); second != 2 {
		t.Fatalf("second NumDelivered = %d, want 2", second)
	}
	waittest.Receive(t, doubleAcks, 5*time.Second, "first DLQ source ACK attempt")
	waittest.Receive(t, doubleAcks, 5*time.Second, "second DLQ source ACK attempt")
	dlq, err := f.JS.Stream(t.Context(), "EVENTS_DLQ")
	if err != nil {
		t.Fatalf("lookup DLQ stream: %v", err)
	}
	info, err := dlq.Info(t.Context())
	if err != nil {
		t.Fatalf("read DLQ stream: %v", err)
	}
	if info.State.Msgs != 1 {
		t.Fatalf("DLQ messages after ambiguous source ACK = %d, want one deduplicated transfer", info.State.Msgs)
	}
}

type wrappingConsumer struct {
	pullConsumer
	failDoubleAck bool
	doubleAcks    chan<- struct{}
}

type blockingStreamLookup struct {
	jetstream.JetStream
	entered chan<- struct{}
}

func (s blockingStreamLookup) Stream(ctx context.Context, _ string) (jetstream.Stream, error) {
	s.entered <- struct{}{}
	<-ctx.Done()
	return nil, ctx.Err()
}

func (c wrappingConsumer) Fetch(batch int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	result, err := c.pullConsumer.Fetch(batch, opts...)
	if err != nil {
		return nil, err
	}
	messages := make(chan jetstream.Msg, batch)
	for msg := range result.Messages() {
		messages <- wrappingMsg{Msg: msg, failDoubleAck: c.failDoubleAck, doubleAcks: c.doubleAcks}
	}
	close(messages)
	return staticBatch{messages: messages, err: result.Error()}, nil
}

type wrappingMsg struct {
	jetstream.Msg
	failDoubleAck bool
	doubleAcks    chan<- struct{}
}

func (m wrappingMsg) DoubleAck(ctx context.Context) error {
	if m.doubleAcks != nil {
		m.doubleAcks <- struct{}{}
	}
	if m.failDoubleAck {
		return context.DeadlineExceeded
	}
	return m.Msg.DoubleAck(ctx)
}

type staticBatch struct {
	messages <-chan jetstream.Msg
	err      error
}

func (b staticBatch) Messages() <-chan jetstream.Msg { return b.messages }
func (b staticBatch) Error() error                   { return b.err }
