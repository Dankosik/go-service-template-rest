//go:build integration

package natsjs

import (
	"context"
	"errors"
	"testing"
	"time"

	dockerclient "github.com/moby/moby/client"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type packageNATSFixture struct {
	container testcontainers.Container
	url       string
	js        jetstream.JetStream
}

func newPackageNATSFixture(t *testing.T) *packageNATSFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:2.14.3-alpine",
			ExposedPorts: []string{"4222/tcp"},
			Cmd:          []string{"-js", "-sd", "/data"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("4222/tcp"),
				wait.ForLog("Server is ready"),
			).WithDeadline(time.Minute),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start NATS container: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := container.Terminate(cleanupCtx); err != nil {
			t.Errorf("terminate NATS container: %v", err)
		}
	})
	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("resolve NATS endpoint: %v", err)
	}
	url := "nats://" + endpoint
	raw, err := nats.Connect(url, nats.Timeout(5*time.Second))
	if err != nil {
		t.Fatalf("connect fixture client: %v", err)
	}
	t.Cleanup(raw.Close)
	js, err := jetstream.New(raw)
	if err != nil {
		t.Fatalf("create fixture JetStream client: %v", err)
	}
	for _, cfg := range []jetstream.StreamConfig{
		{Name: "EVENTS", Subjects: []string{"events.>"}, Storage: jetstream.FileStorage, MaxMsgSize: testMaxDeliveryBytes},
		{Name: "EVENTS_DLQ", Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: 2 * testMaxDeliveryBytes},
	} {
		if _, err := js.CreateStream(ctx, cfg); err != nil {
			t.Fatalf("create stream %s: %v", cfg.Name, err)
		}
	}
	return &packageNATSFixture{container: container, url: url, js: js}
}

func (f *packageNATSFixture) client(t *testing.T, pending int) *Client {
	t.Helper()
	cfg := testConfig()
	cfg.URLs = []string{f.url}
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

func TestNATSPublishDispatchCancellationAndNoRetry(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := f.client(t, 2)

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
	source, err := f.js.Stream(t.Context(), "EVENTS")
	if err != nil {
		t.Fatalf("lookup source stream: %v", err)
	}
	beforeState, err := source.Info(t.Context())
	if err != nil {
		t.Fatalf("read source state before ambiguous publishes: %v", err)
	}
	if _, err := docker.ContainerPause(t.Context(), f.container.GetContainerID(), dockerclient.ContainerPauseOptions{}); err != nil {
		t.Fatalf("pause NATS: %v", err)
	}
	t.Cleanup(func() {
		_, _ = docker.ContainerUnpause(context.Background(), f.container.GetContainerID(), dockerclient.ContainerUnpauseOptions{})
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
	waitPackage(t, 3*time.Second, func() bool { return client.nc.Stats().OutMsgs >= dispatchedBefore+2 }, "publish dispatches")
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
	if _, err := docker.ContainerUnpause(t.Context(), f.container.GetContainerID(), dockerclient.ContainerUnpauseOptions{}); err != nil {
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
	client := f.client(t, testMaxPending)
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
	if first := receivePackage(t, deliveries, 5*time.Second, "first delivery"); first != 1 {
		t.Fatalf("first NumDelivered = %d, want 1", first)
	}
	if second := receivePackage(t, deliveries, 15*time.Second, "duplicate delivery"); second != 2 {
		t.Fatalf("second NumDelivered = %d, want 2", second)
	}
}

func TestNATSDLQSourceAckAmbiguityDeduplicates(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := f.client(t, testMaxPending)
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
	if first := receivePackage(t, deliveries, 5*time.Second, "first DLQ delivery"); first != 1 {
		t.Fatalf("first NumDelivered = %d, want 1", first)
	}
	if second := receivePackage(t, deliveries, 15*time.Second, "DLQ source-ack redelivery"); second != 2 {
		t.Fatalf("second NumDelivered = %d, want 2", second)
	}
	receivePackage(t, doubleAcks, 5*time.Second, "first DLQ source ACK attempt")
	receivePackage(t, doubleAcks, 5*time.Second, "second DLQ source ACK attempt")
	dlq, err := f.js.Stream(t.Context(), "EVENTS_DLQ")
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

func waitPackage(t *testing.T, timeout time.Duration, predicate func() bool, description string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if predicate() {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s", description)
		}
	}
}

func receivePackage[T any](t *testing.T, ch <-chan T, timeout time.Duration, description string) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(timeout):
		var zero T
		t.Fatalf("timed out waiting for %s", description)
		return zero
	}
}
