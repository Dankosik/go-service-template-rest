//go:build integration

package natsjs

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs/natsjstest"
	"github.com/example/go-service-template-rest/internal/waittest"
	dockerclient "github.com/moby/moby/client"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func newPackageNATSFixture(t *testing.T) *natsjstest.Server {
	t.Helper()
	return natsjstest.Start(t, natsjstest.WithFixedHostPort(), natsjstest.WithStreams(
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
	_ = pending
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

func TestNATSReadinessProbeRestoresStateAfterReconnect(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, testMaxPending)
	client.ready.Store(false) // The reconnect callback leaves readiness false until the cached probe succeeds.
	if err := client.Check(t.Context()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !client.Ready() {
		t.Fatal("successful readiness probe did not restore client readiness")
	}
}

func TestNATSNativeConsumeSurvivesBrokerRestart(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, testMaxPending)
	cfg := testWorkerConfig()
	cfg.Consumer = "restart-worker"
	cfg.FilterSubject = "events.test"
	cfg.DeadLetterSubject = "dead.events.test"
	received := make(chan string, 1)
	worker, err := client.NewWorker(t.Context(), cfg, func(_ context.Context, message Message) error {
		received <- string(message.Payload())
		return nil
	})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}
	runErr := make(chan error, 1)
	go func() { runErr <- worker.Run(t.Context()) }()

	stopTimeout := 10 * time.Second
	if err := f.Container.Stop(t.Context(), &stopTimeout); err != nil {
		t.Fatalf("stop NATS container: %v", err)
	}
	waittest.Until(t, 10*time.Second, func() bool { return !client.Ready() }, "worker disconnect")
	if err := f.Container.Start(t.Context()); err != nil {
		t.Fatalf("restart NATS container: %v", err)
	}
	waittest.Until(t, 10*time.Second, func() bool {
		fresh, connectErr := nats.Connect(f.URL, nats.Timeout(250*time.Millisecond), nats.NoReconnect())
		if connectErr != nil {
			return false
		}
		fresh.Close()
		return true
	}, "restarted NATS listener")
	waittest.Until(t, 20*time.Second, func() bool {
		select {
		case err := <-runErr:
			t.Fatalf("worker stopped during broker transition: %v", err)
		default:
		}
		probeCtx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		return client.Check(probeCtx) == nil
	}, "worker reconnect")
	if _, err := client.Producer().Publish(t.Context(), Event{
		Subject: "events.test", MessageID: rand.Text(), PublicationID: rand.Text(),
		Type: "restart.test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("after restart"),
	}); err != nil {
		t.Fatalf("Publish(after reconnect) error = %v", err)
	}
	if got := waittest.Receive(t, received, 10*time.Second, "post-restart delivery"); got != "after restart" {
		t.Fatalf("post-restart payload = %q", got)
	}
	worker.StartDrain()
	if err := waittest.Receive(t, runErr, 10*time.Second, "restart worker drain"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestNATSPublishDispatchCancellationAndNoRetry(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := packageClient(t, f, 2)

	before := client.nc.Stats().OutMsgs
	if _, err := client.Producer().Publish(t.Context(), Event{
		Subject: "unmatched.test", MessageID: rand.Text(), PublicationID: rand.Text(),
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
		Subject: "events.test", MessageID: rand.Text(), PublicationID: rand.Text(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("cancel after dispatch"),
	}
	publishCtx, cancelPublish := context.WithCancel(t.Context())
	canceledErr := make(chan error, 1)
	go func() {
		_, err := client.Producer().Publish(publishCtx, canceledEvent)
		canceledErr <- err
	}()
	deadlineEvent := Event{
		Subject: "events.test", MessageID: rand.Text(), PublicationID: rand.Text(),
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
	if _, err := docker.ContainerUnpause(t.Context(), f.Container.GetContainerID(), dockerclient.ContainerUnpauseOptions{}); err != nil {
		t.Fatalf("resume NATS: %v", err)
	}
	fresh := Event{
		Subject: "events.test", MessageID: rand.Text(), PublicationID: rand.Text(),
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
		if _, tracked := counts[stored.Header.Get(jetstream.MsgIDHeader)]; tracked {
			counts[stored.Header.Get(jetstream.MsgIDHeader)]++
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
		Subject: "events.test", MessageID: rand.Text(), PublicationID: rand.Text(),
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
		Subject: "events.test", MessageID: rand.Text(), PublicationID: rand.Text(),
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

func (c wrappingConsumer) Consume(handler jetstream.MessageHandler, opts ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, error) {
	return c.pullConsumer.Consume(func(msg jetstream.Msg) {
		handler(wrappingMsg{Msg: msg, failDoubleAck: c.failDoubleAck, doubleAcks: c.doubleAcks})
	}, opts...)
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
