//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/natsjs/natsjstest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
	"github.com/testcontainers/testcontainers-go"
)

const (
	sourceStream         = "EVENTS"
	deadLetterStream     = "EVENTS_DLQ"
	sourceSubject        = "events.test"
	deadLetterSubject    = "dead.events.test"
	testMaxPayloadBytes  = 256 << 10
	testMaxPending       = 64
	testMaxConcurrency   = 8
	testMaxDeliveryBytes = 1 << 20
	testOperationTimeout = 5 * time.Second
)

func testClientConfig() natsjs.Config {
	return natsjs.Config{
		MinStreamReplicas: 1, MinStreamRetention: 24 * time.Hour,
		MaxPayloadBytes: testMaxPayloadBytes, MaxPendingPublishes: testMaxPending,
	}
}

func testWorkerConfig() natsjs.WorkerConfig {
	return natsjs.WorkerConfig{
		MaxConcurrency:       testMaxConcurrency,
		MaxDeliveryBytes:     testMaxDeliveryBytes,
		HandlerTimeout:       30 * time.Second,
		RetryDelays:          []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute},
		DeadLetterRetryDelay: 30 * time.Second,
	}
}

type natsFixture struct {
	container testcontainers.Container
	url       string
	raw       *nats.Conn
	js        jetstream.JetStream
}

type authenticatedNATSFixture struct {
	url                    string
	credentialsFile        string
	invalidCredentialsFile string
}

type testNATSAccount struct {
	public      string
	claim       string
	credentials []byte
}

func newNATSFixture(t *testing.T) *natsFixture {
	t.Helper()
	// The fixed host port is required because reconnect cases stop and restart
	// this exact container while clients retain the admitted URL.
	server := natsjstest.Start(t, natsjstest.WithFixedHostPort(), natsjstest.WithStreams(
		jetstream.StreamConfig{
			Name:       sourceStream,
			Subjects:   []string{"events.>"},
			Storage:    jetstream.FileStorage,
			MaxMsgSize: testMaxDeliveryBytes,
		},
		jetstream.StreamConfig{
			Name:       deadLetterStream,
			Subjects:   []string{"dead.>"},
			Storage:    jetstream.FileStorage,
			MaxMsgSize: 2 * testMaxDeliveryBytes,
		},
	))
	return &natsFixture{container: server.Container, url: server.URL, raw: server.Conn, js: server.JS}
}

func newAuthenticatedNATSFixture(t *testing.T) *authenticatedNATSFixture {
	t.Helper()
	operatorKey, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatalf("create NATS test operator key: %v", err)
	}
	operatorPublic, err := operatorKey.PublicKey()
	if err != nil {
		t.Fatalf("read NATS test operator public key: %v", err)
	}
	systemAccount := newTestNATSAccount(t, operatorKey, false)
	account := newTestNATSAccount(t, operatorKey, true)
	invalidAccount := newTestNATSAccount(t, operatorKey, true)
	operatorClaims := jwt.NewOperatorClaims(operatorPublic)
	operatorClaims.SystemAccount = systemAccount.public
	operatorClaim, err := operatorClaims.Encode(operatorKey)
	if err != nil {
		t.Fatalf("encode NATS test operator claim: %v", err)
	}
	serverConfig := fmt.Sprintf(`operator: %s
resolver: MEMORY
resolver_preload: {
  %s: %s
  %s: %s
}
jetstream {
  store_dir: /data
}
`, operatorClaim, systemAccount.public, systemAccount.claim, account.public, account.claim)
	// The broker request is the shared one with its server configuration replaced,
	// so this fixture still runs the pinned image on the shared ready budget.
	request := natsjstest.Request()
	request.Cmd = []string{"-c", "/etc/nats/auth.conf"}
	request.Files = []testcontainers.ContainerFile{{
		Reader: strings.NewReader(serverConfig), ContainerFilePath: "/etc/nats/auth.conf", FileMode: 0o644,
	}}
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		// Started separately, because a configuration this fixture generated is
		// the likely cause of a failed start and the server log is the only place
		// that says which line it refused.
		Started: false,
	})
	if err != nil {
		t.Fatalf("create authenticated NATS container: %v", err)
	}
	natsjstest.Terminate(t, container)
	if err := container.Start(t.Context()); err != nil {
		logs, logsErr := container.Logs(t.Context())
		if logsErr != nil {
			t.Fatalf("start authenticated NATS container: %v; read logs: %v", err, logsErr)
		}
		output, readErr := io.ReadAll(logs)
		_ = logs.Close()
		if readErr != nil {
			t.Fatalf("start authenticated NATS container: %v; read log body: %v", err, readErr)
		}
		t.Fatalf("start authenticated NATS container: %v\n%s", err, output)
	}
	endpoint, err := container.Endpoint(t.Context(), "")
	if err != nil {
		t.Fatalf("resolve authenticated NATS endpoint: %v", err)
	}
	credentialsDirectory := t.TempDir()
	credentialsFile := filepath.Join(credentialsDirectory, "valid.creds")
	invalidCredentialsFile := filepath.Join(credentialsDirectory, "invalid.creds")
	if err := os.WriteFile(credentialsFile, account.credentials, 0o600); err != nil {
		t.Fatalf("write valid NATS credentials: %v", err)
	}
	if err := os.WriteFile(invalidCredentialsFile, invalidAccount.credentials, 0o600); err != nil {
		t.Fatalf("write invalid NATS credentials: %v", err)
	}
	url := "nats://" + endpoint
	client := natsjstest.Connect(t, url, nats.UserCredentials(credentialsFile))
	natsjstest.CreateStreams(t, client.JS, jetstream.StreamConfig{
		Name: sourceStream, Subjects: []string{"events.>"}, Storage: jetstream.FileStorage, MaxMsgSize: testMaxDeliveryBytes,
	})
	return &authenticatedNATSFixture{url: url, credentialsFile: credentialsFile, invalidCredentialsFile: invalidCredentialsFile}
}

func newTestNATSAccount(t *testing.T, operatorKey nkeys.KeyPair, enableJetStream bool) testNATSAccount {
	t.Helper()
	accountKey, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatalf("create NATS test account key: %v", err)
	}
	accountPublic, err := accountKey.PublicKey()
	if err != nil {
		t.Fatalf("read NATS test account public key: %v", err)
	}
	accountClaims := jwt.NewAccountClaims(accountPublic)
	if enableJetStream {
		accountClaims.Limits.JetStreamLimits = jwt.JetStreamLimits{
			MemoryStorage: jwt.NoLimit, DiskStorage: jwt.NoLimit, Streams: jwt.NoLimit, Consumer: jwt.NoLimit,
		}
	}
	accountClaim, err := accountClaims.Encode(operatorKey)
	if err != nil {
		t.Fatalf("encode NATS test account claim: %v", err)
	}
	userKey, err := nkeys.CreateUser()
	if err != nil {
		t.Fatalf("create NATS test user key: %v", err)
	}
	userPublic, err := userKey.PublicKey()
	if err != nil {
		t.Fatalf("read NATS test user public key: %v", err)
	}
	userClaim, err := jwt.NewUserClaims(userPublic).Encode(accountKey)
	if err != nil {
		t.Fatalf("encode NATS test user claim: %v", err)
	}
	userSeed, err := userKey.Seed()
	if err != nil {
		t.Fatalf("read NATS test user seed: %v", err)
	}
	credentials, err := jwt.FormatUserConfig(userClaim, userSeed)
	if err != nil {
		t.Fatalf("format NATS test credentials: %v", err)
	}
	return testNATSAccount{public: accountPublic, claim: accountClaim, credentials: credentials}
}

func (f *natsFixture) client(t *testing.T, role natsjs.Role, configure ...func(*natsjs.Config)) *natsjs.Client {
	t.Helper()
	cfg := testClientConfig()
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
	cfg := testWorkerConfig()
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
		stopWorker(worker)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("worker did not stop")
		}
	})
	return client, worker, errCh
}

func stopWorker(worker *natsjs.Worker) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = worker.Shutdown(ctx)
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

func waitConsumerSettled(t *testing.T, fixture *natsFixture, consumerName string) {
	t.Helper()
	waittest.Until(t, 5*time.Second, func() bool {
		consumer, err := fixture.js.Consumer(t.Context(), sourceStream, consumerName)
		if err != nil {
			return false
		}
		info, err := consumer.Info(t.Context())
		return err == nil && info.NumAckPending == 0 && info.NumPending == 0
	}, consumerName+" settlement")
}

func assertStreamMessages(t *testing.T, fixture *natsFixture, streamName string, want uint64) {
	t.Helper()
	stream, err := fixture.js.Stream(t.Context(), streamName)
	if err != nil {
		t.Fatalf("lookup stream %s: %v", streamName, err)
	}
	info, err := stream.Info(t.Context())
	if err != nil {
		t.Fatalf("read stream %s: %v", streamName, err)
	}
	if info.State.Msgs != want {
		t.Fatalf("stream %s messages = %d, want %d", streamName, info.State.Msgs, want)
	}
}
