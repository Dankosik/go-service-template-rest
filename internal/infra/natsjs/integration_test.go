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
		{Name: "EVENTS", Subjects: []string{"events.>"}, Storage: jetstream.FileStorage, MaxMsgSize: DefaultMaxDeliveryBytes},
		{Name: "EVENTS_DLQ", Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: 2 * DefaultMaxDeliveryBytes},
	} {
		if _, err := js.CreateStream(ctx, cfg); err != nil {
			t.Fatalf("create stream %s: %v", cfg.Name, err)
		}
	}
	return &packageNATSFixture{container: container, url: url, js: js}
}

func (f *packageNATSFixture) client(t *testing.T, pending int) *Client {
	t.Helper()
	cfg := DefaultConfig()
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
	client := f.client(t, 1)

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
	if _, err := docker.ContainerPause(t.Context(), f.container.GetContainerID(), dockerclient.ContainerPauseOptions{}); err != nil {
		t.Fatalf("pause NATS: %v", err)
	}
	t.Cleanup(func() {
		_, _ = docker.ContainerUnpause(context.Background(), f.container.GetContainerID(), dockerclient.ContainerUnpauseOptions{})
	})
	dispatchedBefore := client.nc.Stats().OutMsgs
	publishCtx, cancelPublish := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Producer().Publish(publishCtx, Event{
			Subject: "events.test", MessageID: NewID(), PublicationID: NewID(),
			Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("cancel after dispatch"),
		})
		errCh <- err
	}()
	waitPackage(t, 3*time.Second, func() bool { return client.nc.Stats().OutMsgs > dispatchedBefore }, "publish dispatch")
	if _, err := client.Producer().Publish(t.Context(), Event{
		Subject: "events.test", MessageID: NewID(), PublicationID: NewID(),
		Type: "test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("over capacity"),
	}); !errors.Is(err, ErrCapacity) {
		t.Fatalf("Publish(over capacity) error = %v, want ErrCapacity", err)
	}
	cancelPublish()
	if err := <-errCh; !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("Publish(canceled after dispatch) error = %v, want ErrAmbiguous", err)
	}
	if _, err := docker.ContainerUnpause(t.Context(), f.container.GetContainerID(), dockerclient.ContainerUnpauseOptions{}); err != nil {
		t.Fatalf("resume NATS: %v", err)
	}
}

func TestNATSHandlerAckAmbiguityRedelivers(t *testing.T) {
	f := newPackageNATSFixture(t)
	client := f.client(t, DefaultMaxPending)
	cfg := DefaultWorkerConfig()
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
	client := f.client(t, DefaultMaxPending)
	cfg := DefaultWorkerConfig()
	cfg.Consumer = "dlq-ack-ambiguity"
	cfg.FilterSubject = "events.test"
	cfg.DeadLetterSubject = "dead.events.test"
	cfg.HandlerTimeout = 50 * time.Millisecond
	cfg.DeadLetterRetryDelay = 50 * time.Millisecond
	deliveries := make(chan uint64, 2)
	worker, err := client.NewWorker(t.Context(), cfg, func(_ context.Context, msg Message) error {
		deliveries <- msg.Metadata().NumDelivered
		return Permanent(errors.New("poison"))
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
}

func (c wrappingConsumer) Fetch(batch int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	result, err := c.pullConsumer.Fetch(batch, opts...)
	if err != nil {
		return nil, err
	}
	messages := make(chan jetstream.Msg, batch)
	for msg := range result.Messages() {
		messages <- wrappingMsg{Msg: msg, failDoubleAck: c.failDoubleAck}
	}
	close(messages)
	return staticBatch{messages: messages, err: result.Error()}, nil
}

type wrappingMsg struct {
	jetstream.Msg
	failDoubleAck bool
}

func (m wrappingMsg) DoubleAck(ctx context.Context) error {
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
