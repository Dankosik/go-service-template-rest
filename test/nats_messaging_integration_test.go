//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/reqctx"
	containerapi "github.com/moby/moby/api/types/container"
	networkapi "github.com/moby/moby/api/types/network"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/baggage"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	natsImage         = "nats:2.14.3-alpine"
	sourceStream      = "EVENTS"
	deadLetterStream  = "EVENTS_DLQ"
	sourceSubject     = "events.test"
	deadLetterSubject = "dead.events.test"
)

type natsFixture struct {
	container testcontainers.Container
	url       string
	raw       *nats.Conn
	js        jetstream.JetStream
}

func newNATSFixture(t *testing.T) *natsFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve NATS test port: %v", err)
	}
	hostPort := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release NATS test port reservation: %v", err)
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        natsImage,
			ExposedPorts: []string{"4222/tcp"},
			Cmd:          []string{"-js", "-sd", "/data"},
			HostConfigModifier: func(hostConfig *containerapi.HostConfig) {
				hostConfig.PortBindings = networkapi.PortMap{
					networkapi.MustParsePort("4222/tcp"): {
						{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: fmt.Sprint(hostPort)},
					},
				}
			},
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
		t.Fatalf("connect fixture NATS client: %v", err)
	}
	t.Cleanup(raw.Close)
	js, err := jetstream.New(raw)
	if err != nil {
		t.Fatalf("create fixture JetStream client: %v", err)
	}
	if _, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:       sourceStream,
		Subjects:   []string{"events.>"},
		Storage:    jetstream.FileStorage,
		MaxMsgSize: natsjs.DefaultMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("create source stream: %v", err)
	}
	if _, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:       deadLetterStream,
		Subjects:   []string{"dead.>"},
		Storage:    jetstream.FileStorage,
		MaxMsgSize: 2 * natsjs.DefaultMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("create dead-letter stream: %v", err)
	}
	return &natsFixture{container: container, url: url, raw: raw, js: js}
}

func (f *natsFixture) client(t *testing.T, role natsjs.Role, configure ...func(*natsjs.Config)) *natsjs.Client {
	t.Helper()
	cfg := natsjs.DefaultConfig()
	cfg.URLs = []string{f.url}
	cfg.AllowPlaintext = true
	cfg.AllowUnauthenticated = true
	cfg.Stream = sourceStream
	for _, apply := range configure {
		apply(&cfg)
	}
	client, err := natsjs.Connect(t.Context(), cfg, role, natsjs.Observability{})
	if err != nil {
		t.Fatalf("connect messaging client: %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

func (f *natsFixture) worker(t *testing.T, handler natsjs.Handler, configure ...func(*natsjs.WorkerConfig)) (*natsjs.Client, *natsjs.Worker, <-chan error) {
	t.Helper()
	client := f.client(t, natsjs.RoleWorker)
	cfg := natsjs.DefaultWorkerConfig()
	cfg.Consumer = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	cfg.FilterSubject = sourceSubject
	cfg.DeadLetterSubject = deadLetterSubject
	for _, apply := range configure {
		apply(&cfg)
	}
	worker, err := client.NewWorker(t.Context(), cfg, handler)
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}
	runCtx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		errCh <- worker.Run(runCtx)
	}()
	t.Cleanup(func() {
		cancel()
		worker.ForceClose()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("worker did not stop")
		}
	})
	return client, worker, errCh
}

func testEvent(payload string) natsjs.Event {
	return natsjs.Event{
		Subject:       sourceSubject,
		MessageID:     natsjs.NewID(),
		PublicationID: natsjs.NewID(),
		Type:          "test.event",
		Schema:        "v1",
		OrderingKey:   "account-1",
		CreatedAt:     time.Now().UTC(),
		Payload:       []byte(payload),
	}
}

func TestNATSProducerOutcomesAndCapacity(t *testing.T) {
	f := newNATSFixture(t)
	client := f.client(t, natsjs.RoleProducer)
	event := testEvent("accepted")
	first, err := client.Producer().Publish(t.Context(), event)
	if err != nil {
		t.Fatalf("Publish(first) error = %v", err)
	}
	duplicate, err := client.Producer().Publish(t.Context(), event)
	if err != nil {
		t.Fatalf("Publish(duplicate) error = %v", err)
	}
	if !duplicate.Duplicate || duplicate.Sequence != first.Sequence {
		t.Fatalf("duplicate ack = %+v, first = %+v", duplicate, first)
	}

	rejected := testEvent("wrong stream")
	rejected.Subject = "other.test"
	if _, err := client.Producer().Publish(t.Context(), rejected); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Publish(stream mismatch) error = %v, want ErrRejected", err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.Producer().Publish(canceled, testEvent("canceled")); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Publish(pre-canceled) error = %v, want ErrRejected", err)
	}
}

func TestNATSServiceProducerOnlyProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process lifecycle is Unix-specific")
	}
	f := newNATSFixture(t)
	repositoryRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	serviceRoot := initializedMessagingServiceRoot(t, repositoryRoot)
	binary := filepath.Join(t.TempDir(), "service")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./cmd/service")
	build.Dir = serviceRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build initialized messaging service: %v\n%s", err, output)
	}

	start := func() (*exec.Cmd, <-chan error, *bytes.Buffer, string) {
		t.Helper()
		output := &bytes.Buffer{}
		httpAddress := freeMessagingAddress(t)
		process := exec.Command(binary)
		process.Dir = serviceRoot
		process.Env = append(cleanMessagingEnvironment(os.Environ()),
			"APP__APP__ENV=integration",
			"APP__HTTP__ADDR="+httpAddress,
			"APP__HTTP__READINESS_PROPAGATION_DELAY=0s",
			"APP__HEALTH__REFRESH_INTERVAL=100ms",
			"APP__HEALTH__FAILURE_THRESHOLD=1",
			"APP__OBSERVABILITY__METRICS__ADDR=",
			"APP__MESSAGING__ENABLED=true",
			"APP__MESSAGING__URLS="+f.url,
			"APP__MESSAGING__ALLOW_PLAINTEXT=true",
			"APP__MESSAGING__ALLOW_UNAUTHENTICATED=true",
			"APP__MESSAGING__STREAM="+sourceStream,
		)
		process.Stdout = output
		process.Stderr = output
		if err := process.Start(); err != nil {
			t.Fatalf("start initialized messaging service: %v", err)
		}
		waited := make(chan error, 1)
		finished := make(chan struct{})
		go func() {
			waited <- process.Wait()
			close(finished)
		}()
		t.Cleanup(func() {
			select {
			case <-finished:
			default:
				_ = process.Process.Kill()
				<-finished
			}
		})
		return process, waited, output, httpAddress
	}

	process, waited, output, httpAddress := start()
	waitHTTPStatus(t, httpAddress, http.StatusOK)
	stream, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup source stream: %v", err)
	}
	names := stream.ConsumerNames(t.Context())
	for name := range names.Name() {
		t.Fatalf("producer-only process created durable consumer %q", name)
	}
	if err := names.Err(); err != nil {
		t.Fatalf("list source consumers: %v", err)
	}
	if err := process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal initialized messaging service: %v", err)
	}
	if err := receive(t, waited, 10*time.Second, "producer-only service shutdown"); err != nil {
		t.Fatalf("producer-only service shutdown error = %v\n%s", err, output.String())
	}

	process, waited, output, httpAddress = start()
	waitHTTPStatus(t, httpAddress, http.StatusOK)
	stopTimeout := 10 * time.Second
	if err := f.container.Stop(t.Context(), &stopTimeout); err != nil {
		t.Fatalf("stop NATS under producer-only service: %v", err)
	}
	waitHTTPStatus(t, httpAddress, http.StatusServiceUnavailable)
	if err := receive(t, waited, 70*time.Second, "producer-only reconnect exhaustion"); err == nil {
		t.Fatalf("producer-only service exited zero after reconnect exhaustion\n%s", output.String())
	}
	if !strings.Contains(output.String(), "connection closed after reconnect exhaustion") {
		t.Fatalf("producer-only service exited without classified reconnect exhaustion\n%s", output.String())
	}
}

func TestNATSStartupAdmission(t *testing.T) {
	closedListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve unavailable endpoint: %v", err)
	}
	unavailableURL := "nats://" + closedListener.Addr().String()
	closedListener.Close()
	unavailable := natsjs.DefaultConfig()
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
	missing := natsjs.DefaultConfig()
	missing.URLs = []string{f.url}
	missing.AllowPlaintext = true
	missing.AllowUnauthenticated = true
	missing.Stream = "MISSING"
	if _, err := natsjs.Connect(t.Context(), missing, natsjs.RoleProducer, natsjs.Observability{}); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Connect(missing stream) error = %v, want ErrRejected", err)
	}

	client := f.client(t, natsjs.RoleWorker)
	workerCfg := natsjs.DefaultWorkerConfig()
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
		Name: sourceStream, Subjects: []string{"events.>"}, Storage: jetstream.FileStorage, MaxMsgSize: natsjs.DefaultMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("restore source stream: %v", err)
	}
	if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
		Name: deadLetterStream, Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: natsjs.DefaultMaxPayloadBytes,
	}); err != nil {
		t.Fatalf("undersize DLQ stream: %v", err)
	}
	if _, err := client.NewWorker(t.Context(), workerCfg, handler); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("NewWorker(undersized DLQ) error = %v, want ErrRejected", err)
	}
	if _, err := f.js.UpdateStream(t.Context(), jetstream.StreamConfig{
		Name: deadLetterStream, Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: 2 * natsjs.DefaultMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("restore DLQ stream: %v", err)
	}

	stream, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup source stream: %v", err)
	}
	incompatibleConfig := jetstream.ConsumerConfig{
		Name: "incompatible", Durable: "incompatible", DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy: jetstream.AckExplicitPolicy, AckWait: workerCfg.HandlerTimeout + 11*time.Second,
		MaxDeliver: -1, ReplayPolicy: jetstream.ReplayInstantPolicy, MaxWaiting: 2,
		MaxAckPending: workerCfg.MaxConcurrency, MaxRequestBatch: 1, MaxRequestExpires: 5 * time.Second,
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

func TestNATSConnectionLossAndReconnect(t *testing.T) {
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
	waitFor(t, 10*time.Second, func() bool { return !client.Ready() }, "client readiness to fail")
	if _, err := client.Producer().Publish(t.Context(), testEvent("during loss")); err == nil {
		t.Fatal("Publish(during loss) succeeded")
	}
	if err := f.container.Start(t.Context()); err != nil {
		t.Fatalf("restart NATS: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		fresh, err := nats.Connect(f.url, nats.Timeout(250*time.Millisecond), nats.NoReconnect())
		if err != nil {
			return false
		}
		fresh.Close()
		return true
	}, "restarted NATS listener")
	waitFor(t, 15*time.Second, func() bool {
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
		waitFor(t, 5*time.Second, func() bool { return !client.Ready() }, fmt.Sprintf("worker readiness to fail in cycle %d", cycle))
		select {
		case err := <-errCh:
			t.Fatalf("worker stopped during reconnect cycle %d: %v", cycle, err)
		default:
		}
		if err := f.container.Start(t.Context()); err != nil {
			t.Fatalf("restart NATS cycle %d: %v", cycle, err)
		}
		waitFor(t, 15*time.Second, func() bool {
			probeCtx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			return client.Check(probeCtx) == nil
		}, fmt.Sprintf("worker readiness to recover in cycle %d", cycle))
		if _, err := client.Producer().Publish(t.Context(), testEvent(fmt.Sprintf("worker reconnect %d", cycle))); err != nil {
			t.Fatalf("Publish(after worker reconnect cycle %d) error = %v", cycle, err)
		}
		receiveSignal(t, delivered, 10*time.Second, fmt.Sprintf("delivery after worker reconnect cycle %d", cycle))
	}
}

func TestNATSConnectionReconnectExhaustion(t *testing.T) {
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
	select {
	case err := <-errCh:
		if !errors.Is(err, natsjs.ErrTerminal) {
			t.Fatalf("Client.Run() error = %v, want ErrTerminal", err)
		}
	case <-time.After(70 * time.Second):
		t.Fatal("Client.Run() did not report reconnect exhaustion")
	}
}

func TestNATSConsumerSaturation(t *testing.T) {
	f := newNATSFixture(t)
	const maxDeliveryBytes = natsjs.DefaultMaxDeliveryBytes
	maxPayloadBytes := maxDeliveryBytes - natsjs.HeaderLimitBytes
	client := f.client(t, natsjs.RoleWorker, func(cfg *natsjs.Config) {
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
	workerCfg := natsjs.DefaultWorkerConfig()
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
		worker.ForceClose()
		<-done
	})
	first := receive(t, entered, 5*time.Second, "first handler")
	second := receive(t, entered, 5*time.Second, "second handler")
	if first != maxPayloadBytes || second != maxPayloadBytes {
		t.Fatalf("active handler payload bytes = %d,%d, want %d each", first, second, maxPayloadBytes)
	}
	select {
	case third := <-entered:
		t.Fatalf("third handler entered while saturated with %d bytes", third)
	default:
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
	_ = receive(t, entered, 5*time.Second, "third handler after capacity release")
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
	}, func(cfg *natsjs.WorkerConfig) { cfg.RetryDelays = []time.Duration{50 * time.Millisecond} })
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
	second := receive(t, deliveries, 5*time.Second, "redelivery")
	if first.message.MessageID() != second.message.MessageID() || first.message.PublicationID() != second.message.PublicationID() {
		t.Fatalf("identity changed across redelivery: first %q/%q second %q/%q", first.message.MessageID(), first.message.PublicationID(), second.message.MessageID(), second.message.PublicationID())
	}
	if second.message.Metadata().NumDelivered != 2 {
		t.Fatalf("redelivery NumDelivered = %d, want 2", second.message.Metadata().NumDelivered)
	}
	if elapsed := second.at.Sub(first.at); elapsed < 45*time.Millisecond {
		t.Fatalf("redelivery delay = %s, want at least configured 50ms minus scheduling tolerance", elapsed)
	}
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
		if got := receive(t, called, 5*time.Second, fmt.Sprintf("delivery %d", attempt)); got != attempt {
			t.Fatalf("delivery = %d, want %d", got, attempt)
		}
	}
	select {
	case err := <-errCh:
		if !errors.Is(err, natsjs.ErrTerminal) {
			t.Fatalf("worker error after unavailable DLQ = %v, want ErrTerminal", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not retain source after unavailable DLQ")
	}
	if _, err := f.js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name: deadLetterStream, Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage,
		MaxMsgSize: 2 * natsjs.DefaultMaxDeliveryBytes,
	}); err != nil {
		t.Fatalf("restore DLQ stream: %v", err)
	}

	secondClient := f.client(t, natsjs.RoleWorker)
	secondCfg := natsjs.DefaultWorkerConfig()
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
	waitFor(t, 15*time.Second, func() bool {
		stream, err := f.js.Stream(t.Context(), deadLetterStream)
		if err != nil {
			return false
		}
		_, err = stream.GetLastMsgForSubject(t.Context(), deadLetterSubject)
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
	if got := deadLetter.Header.Get("Original-Num-Delivered"); got != "6" {
		t.Fatalf("exhaustion DLQ original delivery = %q, want 6", got)
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
		if got := receive(t, attempts, 5*time.Second, fmt.Sprintf("crash-budget delivery %d", attempt)); got != attempt {
			t.Fatalf("crash-budget delivery = %d, want %d", got, attempt)
		}
	}
	worker.ForceClose()
	_ = receive(t, firstDone, 5*time.Second, "crashed worker stop")

	secondClient := f.client(t, natsjs.RoleWorker)
	cfg := natsjs.DefaultWorkerConfig()
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
	if got := receive(t, fifth, 15*time.Second, "fifth delivery after worker crash"); got != 5 {
		t.Fatalf("delivery after worker crash = %d, want final allowed attempt 5", got)
	}
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
	poison.Header.Set("Publication-Id", "poison-publication")
	poison.Data = rawPayload
	ack, err := f.js.PublishMsg(t.Context(), poison, jetstream.WithMsgID("poison-publication"))
	if err != nil {
		t.Fatalf("publish poison: %v", err)
	}
	var dlq *jetstream.RawStreamMsg
	waitFor(t, 5*time.Second, func() bool {
		stream, streamErr := f.js.Stream(t.Context(), deadLetterStream)
		if streamErr != nil {
			return false
		}
		dlq, streamErr = stream.GetLastMsgForSubject(t.Context(), deadLetterSubject)
		return streamErr == nil
	}, "poison dead-letter transfer")
	if handlerCalls.Load() != 0 {
		t.Fatalf("handler calls for malformed poison = %d, want 0", handlerCalls.Load())
	}
	if !slices.Equal(dlq.Data, rawPayload) {
		t.Fatalf("DLQ payload = %q, want %q", dlq.Data, rawPayload)
	}
	if got := dlq.Header.Get("Original-Stream-Sequence"); got != fmt.Sprint(ack.Sequence) {
		t.Fatalf("Original-Stream-Sequence = %q, want %d", got, ack.Sequence)
	}

	redrive := testEvent("redrive")
	redrive.MessageID = dlq.Header.Get("Message-Id")
	redrive.Subject = dlq.Header.Get("Original-Subject")
	client := f.client(t, natsjs.RoleProducer)
	result, err := client.Producer().Publish(t.Context(), redrive)
	if err != nil || result.Duplicate {
		t.Fatalf("redrive result = %+v, error = %v", result, err)
	}
	waitFor(t, 5*time.Second, func() bool { return handlerCalls.Load() == 1 }, "redriven poison delivery")
	stream, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup source stream for old-ID redrive: %v", err)
	}
	before, err := stream.Info(t.Context())
	if err != nil {
		t.Fatalf("read source state before old-ID redrive: %v", err)
	}
	oldIdentity := redrive
	oldIdentity.PublicationID = dlq.Header.Get("Original-Publication-Id")
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

func TestNATSOversizedSourceIsRetained(t *testing.T) {
	f := newNATSFixture(t)
	client, _, errCh := f.worker(t, func(context.Context, natsjs.Message) error {
		t.Fatal("oversized source reached handler")
		return nil
	}, func(cfg *natsjs.WorkerConfig) { cfg.Consumer = "oversized-worker" })
	payload := make([]byte, natsjs.DefaultMaxPayloadBytes+1)
	ack, err := f.js.Publish(t.Context(), sourceSubject, payload)
	if err != nil {
		t.Fatalf("publish oversized source: %v", err)
	}
	select {
	case err := <-errCh:
		if !errors.Is(err, natsjs.ErrTerminal) {
			t.Fatalf("worker error = %v, want ErrTerminal", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not reject oversized source")
	}
	if client.Ready() {
		t.Fatal("client remained ready after terminal oversized delivery")
	}
	consumer, err := f.js.Consumer(t.Context(), sourceStream, "oversized-worker")
	if err != nil {
		t.Fatalf("lookup oversized consumer: %v", err)
	}
	consumerInfo, err := consumer.Info(t.Context())
	if err != nil {
		t.Fatalf("read oversized consumer state: %v", err)
	}
	if consumerInfo.NumAckPending != 1 {
		t.Fatalf("oversized consumer ack pending = %d, want exact source retained", consumerInfo.NumAckPending)
	}
	source, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup oversized source stream: %v", err)
	}
	retained, err := source.GetMsg(t.Context(), ack.Sequence)
	if err != nil {
		t.Fatalf("read retained oversized source: %v", err)
	}
	if !slices.Equal(retained.Data, payload) {
		t.Fatalf("retained oversized source = %d bytes, want %d exact bytes", len(retained.Data), len(payload))
	}
	stream, err := f.js.Stream(t.Context(), deadLetterStream)
	if err != nil {
		t.Fatalf("lookup DLQ stream: %v", err)
	}
	info, err := stream.Info(t.Context())
	if err != nil {
		t.Fatalf("read DLQ stream: %v", err)
	}
	if info.State.Msgs != 0 {
		t.Fatalf("DLQ messages = %d, want 0", info.State.Msgs)
	}
}

func TestNATSOrderingKeyDoesNotSerialize(t *testing.T) {
	f := newNATSFixture(t)
	firstEntered := make(chan struct{})
	secondDone := make(chan struct{})
	releaseFirst := make(chan struct{})
	client, _, _ := f.worker(t, func(_ context.Context, msg natsjs.Message) error {
		switch string(msg.Payload()) {
		case "first":
			close(firstEntered)
			<-releaseFirst
		case "second":
			close(secondDone)
		}
		return nil
	}, func(cfg *natsjs.WorkerConfig) { cfg.MaxConcurrency = 2 })
	first := testEvent("first")
	second := testEvent("second")
	second.OrderingKey = first.OrderingKey
	if _, err := client.Producer().Publish(t.Context(), first); err != nil {
		t.Fatalf("publish first: %v", err)
	}
	if _, err := client.Producer().Publish(t.Context(), second); err != nil {
		t.Fatalf("publish second: %v", err)
	}
	receiveSignal(t, firstEntered, 5*time.Second, "first handler")
	receiveSignal(t, secondDone, 5*time.Second, "second handler completion")
	close(releaseFirst)
}

func TestNATSTraceCorrelation(t *testing.T) {
	const (
		payloadCanary = "TRACE_PAYLOAD_CANARY"
		typeCanary    = "TRACE_EVENT_TYPE_CANARY"
		schemaCanary  = "TRACE_SCHEMA_CANARY"
		baggageCanary = "TRACE_BAGGAGE_CANARY"
	)
	f := newNATSFixture(t)
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	observed := make(chan struct {
		requestID string
		traceID   string
		baggage   int
	}, 1)
	cfg := natsjs.DefaultConfig()
	cfg.URLs = []string{f.url}
	cfg.AllowPlaintext = true
	cfg.AllowUnauthenticated = true
	cfg.Stream = sourceStream
	client, err := natsjs.Connect(t.Context(), cfg, natsjs.RoleWorker, natsjs.Observability{Tracer: provider.Tracer("test")})
	if err != nil {
		t.Fatalf("connect messaging client: %v", err)
	}
	t.Cleanup(client.Close)
	workerCfg := natsjs.DefaultWorkerConfig()
	workerCfg.Consumer = "trace-worker"
	workerCfg.FilterSubject = sourceSubject
	workerCfg.DeadLetterSubject = deadLetterSubject
	worker, err := client.NewWorker(t.Context(), workerCfg, func(ctx context.Context, _ natsjs.Message) error {
		observed <- struct {
			requestID string
			traceID   string
			baggage   int
		}{reqctx.RequestID(ctx), traceID(ctx), baggage.FromContext(ctx).Len()}
		return nil
	})
	if err != nil {
		t.Fatalf("create trace worker: %v", err)
	}
	runCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { _ = worker.Run(runCtx) }()
	parentCtx, parent := provider.Tracer("test").Start(t.Context(), "parent")
	parentCtx = reqctx.ContextWithRequestID(parentCtx, "request-123")
	member, err := baggage.NewMember("untrusted", baggageCanary)
	if err != nil {
		t.Fatalf("create baggage canary: %v", err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatalf("create baggage: %v", err)
	}
	parentCtx = baggage.ContextWithBaggage(parentCtx, bag)
	event := testEvent(payloadCanary)
	event.Type = typeCanary
	event.Schema = schemaCanary
	if _, err := client.Producer().Publish(parentCtx, event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	got := receive(t, observed, 5*time.Second, "trace delivery")
	if got.requestID != "request-123" || got.traceID != parent.SpanContext().TraceID().String() || got.baggage != 0 {
		t.Fatalf("handler correlation = %+v, want request-123/%s and no baggage", got, parent.SpanContext().TraceID())
	}
	parent.End()
	waitFor(t, 5*time.Second, func() bool { return len(recorder.Ended()) >= 3 }, "producer, consumer, and parent spans")
	var spanData strings.Builder
	for _, span := range recorder.Ended() {
		fmt.Fprintln(&spanData, span.Name())
		for _, value := range span.Attributes() {
			fmt.Fprintln(&spanData, value.Key, value.Value.AsInterface())
		}
		for _, event := range span.Events() {
			fmt.Fprintln(&spanData, event.Name)
			for _, value := range event.Attributes {
				fmt.Fprintln(&spanData, value.Key, value.Value.AsInterface())
			}
		}
	}
	for _, forbidden := range []string{payloadCanary, typeCanary, schemaCanary, baggageCanary} {
		if strings.Contains(spanData.String(), forbidden) {
			t.Fatalf("messaging spans contain forbidden value %q: %s", forbidden, spanData.String())
		}
	}
}

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
	receiveSignal(t, entered, 5*time.Second, "blocked handler")
	shutdownCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if err := worker.Shutdown(shutdownCtx); err == nil {
		t.Fatal("Shutdown() error = nil, want forced shutdown")
	}

	redelivered := make(chan struct{}, 1)
	secondClient := f.client(t, natsjs.RoleWorker)
	cfg := natsjs.DefaultWorkerConfig()
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
	receiveSignal(t, redelivered, 15*time.Second, "redelivery after force close")
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
	receiveSignal(t, blockEntered, 5*time.Second, "blocking handler")
	if _, err := client.Producer().Publish(t.Context(), testEvent("panic")); err != nil {
		t.Fatalf("publish panic fixture: %v", err)
	}
	receiveSignal(t, panicEntered, 5*time.Second, "panicking handler")
	waitFor(t, 5*time.Second, func() bool { return !client.Ready() }, "terminal fail-closed readiness")
	if err := client.Check(t.Context()); err == nil || client.Ready() {
		t.Fatalf("client recovered after terminal handler failure: check error = %v, ready = %t", err, client.Ready())
	}
	if _, err := client.Producer().Publish(t.Context(), testEvent("worker publication")); !errors.Is(err, natsjs.ErrDraining) {
		t.Fatalf("worker Publish(after terminal failure) error = %v, want ErrDraining", err)
	}
	if _, err := producer.Producer().Publish(t.Context(), testEvent("external publication")); err != nil {
		t.Fatalf("publish external post-failure fixture: %v", err)
	}
	waitFor(t, 5*time.Second, func() bool {
		consumer, lookupErr := f.js.Consumer(t.Context(), sourceStream, "panic-worker")
		if lookupErr != nil {
			return false
		}
		info, infoErr := consumer.Info(t.Context())
		return infoErr == nil && info.NumPending >= 1
	}, "post-failure message to remain broker-pending")
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
	select {
	case err := <-errCh:
		if !errors.Is(err, natsjs.ErrTerminal) || strings.Contains(err.Error(), "feature panic canary") {
			t.Fatalf("worker panic supervision error = %v, want sanitized ErrTerminal", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handler panic did not reach worker supervision")
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
	client, worker, _ := f.worker(t, func(context.Context, natsjs.Message) error {
		close(entered)
		<-release
		return nil
	}, func(cfg *natsjs.WorkerConfig) { cfg.Consumer = "graceful-worker" })
	if _, err := client.Producer().Publish(t.Context(), testEvent("in flight")); err != nil {
		t.Fatalf("publish in-flight message: %v", err)
	}
	receiveSignal(t, entered, 5*time.Second, "in-flight handler")
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
	select {
	case err := <-shutdownErr:
		t.Fatalf("Shutdown() returned before handler completion: %v", err)
	default:
	}
	close(release)
	if err := receive(t, shutdownErr, 5*time.Second, "graceful shutdown"); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestNATSWorkerMainRejectsEmptyHandler(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for connection sentinel: %v", err)
	}
	accepted := make(chan struct{}, 1)
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			connection.Close()
			accepted <- struct{}{}
		}
	}()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repositoryRoot := filepath.Dir(workingDirectory)
	binary := filepath.Join(t.TempDir(), "worker")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./cmd/worker")
	build.Dir = repositoryRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build worker: %v\n%s", err, output)
	}
	run := exec.CommandContext(t.Context(), binary)
	run.Env = append(os.Environ(), "APP__MESSAGING__URLS=nats://"+listener.Addr().String())
	output, runErr := run.CombinedOutput()
	if runErr == nil || !strings.Contains(string(output), "worker feature handler is not registered") {
		t.Fatalf("worker result error = %v, output = %q", runErr, output)
	}
	listener.Close()
	<-acceptDone
	select {
	case <-accepted:
		t.Fatal("worker opened a broker connection before rejecting empty handler")
	default:
	}
}

func initializedMessagingServiceRoot(t *testing.T, repositoryRoot string) string {
	t.Helper()
	tree := exec.CommandContext(t.Context(), "git", "write-tree")
	tree.Dir = repositoryRoot
	treeOutput, err := tree.Output()
	if err != nil {
		t.Fatalf("write staged messaging tree: %v", err)
	}
	temporaryRoot := t.TempDir()
	archivePath := filepath.Join(temporaryRoot, "template.tar")
	archive := exec.CommandContext(t.Context(), "git", "archive", "--format=tar", "--output="+archivePath, strings.TrimSpace(string(treeOutput)))
	archive.Dir = repositoryRoot
	if output, err := archive.CombinedOutput(); err != nil {
		t.Fatalf("archive staged messaging tree: %v\n%s", err, output)
	}
	serviceRoot := filepath.Join(temporaryRoot, "service")
	if err := os.Mkdir(serviceRoot, 0o750); err != nil {
		t.Fatalf("create messaging service root: %v", err)
	}
	extract := exec.CommandContext(t.Context(), "tar", "-xf", archivePath, "-C", serviceRoot)
	if output, err := extract.CombinedOutput(); err != nil {
		t.Fatalf("extract staged messaging tree: %v\n%s", err, output)
	}
	initializeGit := exec.CommandContext(t.Context(), "git", "init", "-q")
	initializeGit.Dir = serviceRoot
	if output, err := initializeGit.CombinedOutput(); err != nil {
		t.Fatalf("initialize messaging fixture Git repository: %v\n%s", err, output)
	}
	commit := exec.CommandContext(t.Context(), "git", "-c", "user.name=messaging-process-integration", "-c", "user.email=messaging-process-integration@example.com", "commit", "-q", "--allow-empty", "-m", "template checkout")
	commit.Dir = serviceRoot
	if output, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("create messaging fixture HEAD: %v\n%s", err, output)
	}
	initialize := exec.CommandContext(t.Context(), "bash", "./scripts/init-module.sh", "github.com/acme/messaging-process-integration")
	initialize.Dir = serviceRoot
	initialize.Env = append(cleanMessagingEnvironment(os.Environ()),
		"CODEOWNER=@acme/platform",
		"DATABASE=none",
		"GRPC=none",
		"AUTHN=none",
		"OUTBOUND_HTTP=bounded",
		"MESSAGING=nats-jetstream",
		"REFERENCE_EXAMPLE=remove",
	)
	if output, err := initialize.CombinedOutput(); err != nil {
		t.Fatalf("initialize messaging process service: %v\n%s", err, output)
	}
	return serviceRoot
}

func freeMessagingAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve messaging process address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release messaging process address: %v", err)
	}
	return address
}

func cleanMessagingEnvironment(environment []string) []string {
	clean := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "APP__") || strings.HasPrefix(key, "OTEL_") {
			continue
		}
		switch key {
		case "CODEOWNER", "DATABASE", "GRPC", "AUTHN", "OUTBOUND_HTTP", "MESSAGING", "REFERENCE_EXAMPLE":
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}

func waitHTTPStatus(t *testing.T, address string, want int) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	waitFor(t, 10*time.Second, func() bool {
		response, err := client.Get("http://" + address + "/health/ready")
		if err != nil {
			return false
		}
		_ = response.Body.Close()
		return response.StatusCode == want
	}, fmt.Sprintf("HTTP readiness status %d", want))
}

func waitFor(t *testing.T, timeout time.Duration, predicate func() bool, description string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()
	ticker := time.NewTicker(20 * time.Millisecond)
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

func receive[T any](t *testing.T, ch <-chan T, timeout time.Duration, description string) T {
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

func receiveSignal(t *testing.T, ch <-chan struct{}, timeout time.Duration, description string) {
	t.Helper()
	_ = receive(t, ch, timeout, description)
}

func traceID(ctx context.Context) string {
	return trace.SpanContextFromContext(ctx).TraceID().String()
}
