//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestNATSStartupAdmission(t *testing.T) {
	t.Parallel()
	closedListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve unavailable endpoint: %v", err)
	}
	unavailableURL := "nats://" + closedListener.Addr().String()
	closedListener.Close()
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
	unsafeStreams := map[string]jetstream.StreamConfig{
		"memory storage": {
			Name: "MEMORY", Subjects: []string{"memory.>"}, Storage: jetstream.MemoryStorage,
		},
		"short retention": {
			Name: "SHORT", Subjects: []string{"short.>"}, Storage: jetstream.FileStorage, MaxAge: time.Hour,
		},
		"evicting capacity": {
			Name: "EVICT", Subjects: []string{"evict.>"}, Storage: jetstream.FileStorage, MaxMsgs: 1,
		},
	}
	for name, streamConfig := range unsafeStreams {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := f.js.CreateStream(t.Context(), streamConfig); err != nil {
				t.Fatalf("create unsafe stream: %v", err)
			}
			cfg := testClientConfig()
			cfg.URLs = []string{f.url}
			cfg.AllowPlaintext = true
			cfg.AllowUnauthenticated = true
			cfg.Stream = streamConfig.Name
			if _, err := natsjs.Connect(t.Context(), cfg, natsjs.RoleProducer, natsjs.Observability{}); !errors.Is(err, natsjs.ErrRejected) {
				t.Fatalf("Connect(%s) error = %v, want ErrRejected", name, err)
			}
		})
	}

	client := f.client(t, natsjs.RoleWorker)
	workerCfg := testWorkerConfig()
	workerCfg.Consumer = "startup-admission"
	workerCfg.FilterSubject = sourceSubject
	workerCfg.DeadLetterSubject = deadLetterSubject
	var handlerCalls atomic.Int32
	handler := func(context.Context, natsjs.Message) error {
		handlerCalls.Add(1)
		return nil
	}

	for name, maxMessageSize := range map[string]int32{
		"unbounded source": 0,
		"oversized source": int32(workerCfg.MaxDeliveryBytes + 1),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
				Name: sourceStream, Subjects: []string{"events.>"}, Storage: jetstream.FileStorage, MaxMsgSize: maxMessageSize,
			}); err != nil {
				t.Fatalf("update source stream: %v", err)
			}
			if _, err := client.NewWorker(t.Context(), workerCfg, handler); !errors.Is(err, natsjs.ErrRejected) {
				t.Fatalf("NewWorker() error = %v, want ErrRejected", err)
			}
		})
	}
	if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
		Name: sourceStream, Subjects: []string{"events.>"}, Storage: jetstream.FileStorage, MaxMsgSize: testMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("restore source stream: %v", err)
	}
	if err := f.js.DeleteStream(t.Context(), deadLetterStream); err != nil {
		t.Fatalf("delete DLQ stream: %v", err)
	}
	if _, err := client.NewWorker(t.Context(), workerCfg, handler); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("NewWorker(missing DLQ) error = %v, want ErrRejected", err)
	}
	if _, err := f.js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name: deadLetterStream, Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: 2 * testMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("restore DLQ stream: %v", err)
	}
	if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
		Name: deadLetterStream, Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: testMaxPayloadBytes,
	}); err != nil {
		t.Fatalf("undersize DLQ stream: %v", err)
	}
	if _, err := client.NewWorker(t.Context(), workerCfg, handler); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("NewWorker(undersized DLQ) error = %v, want ErrRejected", err)
	}
	if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
		Name: deadLetterStream, Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: 2 * testMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("restore DLQ stream: %v", err)
	}

	stream, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup source stream: %v", err)
	}
	incompatibleConfig := jetstream.ConsumerConfig{
		Name: "incompatible", Durable: "incompatible", DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy: jetstream.AckExplicitPolicy, AckWait: workerCfg.HandlerTimeout + 2*testOperationTimeout + time.Second,
		MaxDeliver: -1, ReplayPolicy: jetstream.ReplayInstantPolicy, MaxWaiting: 2,
		MaxAckPending: workerCfg.MaxConcurrency, MaxRequestBatch: 1, MaxRequestExpires: testOperationTimeout,
		MaxRequestMaxBytes: workerCfg.MaxDeliveryBytes, FilterSubject: workerCfg.FilterSubject, HeadersOnly: true,
	}
	consumer, err := stream.CreateConsumer(t.Context(), incompatibleConfig)
	if err != nil {
		t.Fatalf("create incompatible consumer: %v", err)
	}
	before, err := consumer.Info(t.Context())
	if err != nil {
		t.Fatalf("read incompatible consumer before admission: %v", err)
	}
	workerCfg.Consumer = "incompatible"
	if _, err := client.NewWorker(t.Context(), workerCfg, handler); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("NewWorker(incompatible consumer) error = %v, want ErrRejected", err)
	}
	after, err := consumer.Info(t.Context())
	if err != nil {
		t.Fatalf("read incompatible consumer after admission: %v", err)
	}
	if !reflect.DeepEqual(before.Config, after.Config) {
		t.Fatalf("incompatible consumer mutated: before %#v after %#v", before.Config, after.Config)
	}
	if handlerCalls.Load() != 0 {
		t.Fatalf("handler calls during failed admission = %d, want 0", handlerCalls.Load())
	}
}

func TestNATSReadinessRejectsStreamContractDrift(t *testing.T) {
	t.Parallel()
	f := newNATSFixture(t)
	client := f.client(t, natsjs.RoleWorker)
	workerCfg := testWorkerConfig()
	workerCfg.Consumer = "contract-drift"
	workerCfg.FilterSubject = sourceSubject
	workerCfg.DeadLetterSubject = deadLetterSubject
	if _, err := client.NewWorker(t.Context(), workerCfg, func(context.Context, natsjs.Message) error { return nil }); err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
		Name: sourceStream, Subjects: []string{"events.>"}, Storage: jetstream.FileStorage,
		MaxAge: time.Hour, MaxMsgSize: testMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("shorten source retention: %v", err)
	}
	if err := client.Check(t.Context()); !errors.Is(err, natsjs.ErrRejected) || client.Ready() {
		t.Fatalf("Check(source drift) error/ready = %v/%t, want ErrRejected/false", err, client.Ready())
	}

	if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
		Name: sourceStream, Subjects: []string{"events.>"}, Storage: jetstream.FileStorage,
		MaxMsgSize: testMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("restore source retention: %v", err)
	}
	if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
		Name: deadLetterStream, Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage,
		MaxAge: time.Hour, MaxMsgSize: 2 * testMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("shorten DLQ retention: %v", err)
	}
	if err := client.Check(t.Context()); !errors.Is(err, natsjs.ErrRejected) || client.Ready() {
		t.Fatalf("Check(DLQ drift) error/ready = %v/%t, want ErrRejected/false", err, client.Ready())
	}
}

func TestNATSAuthenticatedStartupAdmission(t *testing.T) {
	t.Parallel()
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
	if !client.Ready() {
		t.Fatal("authenticated client was not ready after topology admission")
	}
	client.Close()

	invalid := valid
	invalid.CredentialsFile = f.invalidCredentialsFile
	if client, err := natsjs.Connect(t.Context(), invalid, natsjs.RoleProducer, natsjs.Observability{}); client != nil || !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Connect(credentials for unknown account) = %#v, %v, want ErrRejected before readiness", client, err)
	}
}

func TestNATSConnectionLossAndReconnect(t *testing.T) {
	t.Parallel()
	f := newNATSFixture(t)
	client := f.client(t, natsjs.RoleProducer)
	runCtx, cancelRun := context.WithCancel(t.Context())
	defer cancelRun()
	errCh := make(chan error, 1)
	go func() { errCh <- client.Run(runCtx) }()
	stopTimeout := 10 * time.Second
	if err := f.container.Stop(t.Context(), &stopTimeout); err != nil {
		t.Fatalf("stop NATS: %v", err)
	}
	waittest.Until(t, 10*time.Second, func() bool { return !client.Ready() }, "client readiness to fail")
	if _, err := client.Producer().Publish(t.Context(), testEvent("during loss")); err == nil {
		t.Fatal("Publish(during loss) succeeded")
	}
	if err := f.container.Start(t.Context()); err != nil {
		t.Fatalf("restart NATS: %v", err)
	}
	waittest.Until(t, 10*time.Second, func() bool {
		fresh, err := nats.Connect(f.url, nats.Timeout(250*time.Millisecond), nats.NoReconnect())
		if err != nil {
			return false
		}
		fresh.Close()
		return true
	}, "restarted NATS listener")
	waittest.Until(t, 15*time.Second, func() bool {
		probeCtx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		return client.Check(probeCtx) == nil
	}, "client readiness to recover")
	if _, err := client.Producer().Publish(t.Context(), testEvent("after reconnect")); err != nil {
		t.Fatalf("Publish(after reconnect) error = %v", err)
	}
	cancelRun()
	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("Client.Run() error = %v, want context.Canceled", err)
	}
}

func TestNATSWorkerConnectionLossAndReconnect(t *testing.T) {
	t.Parallel()
	f := newNATSFixture(t)
	delivered := make(chan struct{}, 1)
	client, _, errCh := f.worker(t, func(context.Context, natsjs.Message) error {
		delivered <- struct{}{}
		return nil
	})
	for cycle := 1; cycle <= 2; cycle++ {
		stopTimeout := 10 * time.Second
		if err := f.container.Stop(t.Context(), &stopTimeout); err != nil {
			t.Fatalf("stop NATS cycle %d: %v", cycle, err)
		}
		waittest.Until(t, 5*time.Second, func() bool { return !client.Ready() }, fmt.Sprintf("worker readiness to fail in cycle %d", cycle))
		select {
		case err := <-errCh:
			t.Fatalf("worker stopped during reconnect cycle %d: %v", cycle, err)
		default:
		}
		if err := f.container.Start(t.Context()); err != nil {
			t.Fatalf("restart NATS cycle %d: %v", cycle, err)
		}
		waittest.Until(t, 15*time.Second, func() bool {
			probeCtx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			return client.Check(probeCtx) == nil
		}, fmt.Sprintf("worker readiness to recover in cycle %d", cycle))
		if _, err := client.Producer().Publish(t.Context(), testEvent(fmt.Sprintf("worker reconnect %d", cycle))); err != nil {
			t.Fatalf("Publish(after worker reconnect cycle %d) error = %v", cycle, err)
		}
		waittest.ReceiveSignal(t, delivered, 10*time.Second, fmt.Sprintf("delivery after worker reconnect cycle %d", cycle))
	}
}

func TestNATSConnectionReconnectExhaustion(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("reconnect exhaustion exercises the fixed 60 second budget")
	}
	f := newNATSFixture(t)
	client := f.client(t, natsjs.RoleProducer)
	errCh := make(chan error, 1)
	go func() { errCh <- client.Run(t.Context()) }()
	stopTimeout := 10 * time.Second
	if err := f.container.Stop(t.Context(), &stopTimeout); err != nil {
		t.Fatalf("stop NATS: %v", err)
	}
	err := waittest.Receive(t, errCh, 70*time.Second, "Client.Run reconnect exhaustion")
	if !errors.Is(err, natsjs.ErrTerminal) {
		t.Fatalf("Client.Run() error = %v, want ErrTerminal", err)
	}
}
