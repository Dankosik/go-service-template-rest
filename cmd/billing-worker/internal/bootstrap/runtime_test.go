package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	eventsv1 "github.com/Dankosik/billing-service/internal/api/events/v1"
	"github.com/Dankosik/billing-service/internal/app/microleaseworker"
	"github.com/Dankosik/billing-service/internal/config"
	"github.com/Dankosik/billing-service/internal/infra/redpanda"
)

type fakeRuntimeConsumer struct {
	msg       redpanda.Message
	err       error
	fetched   bool
	committed bool
}

func (c *fakeRuntimeConsumer) FetchMessage(context.Context) (redpanda.Message, error) {
	c.fetched = true
	if c.err != nil {
		return redpanda.Message{}, c.err
	}
	return c.msg, nil
}

func (c *fakeRuntimeConsumer) CommitOffset(context.Context, redpanda.Message) error {
	c.committed = true
	return nil
}

type fakeRuntimeStore struct {
	mu        sync.Mutex
	calls     map[string]int
	limits    map[string]int32
	published string
}

func (s *fakeRuntimeStore) ApplyTerminalEvent(context.Context, redpanda.TerminalEventCommand) (redpanda.TerminalApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[microleaseworker.RoleTerminalConsumer]++
	return redpanda.TerminalApplyResultApplied, nil
}

func (s *fakeRuntimeStore) ApplyCheckpointEvent(context.Context, redpanda.CheckpointEventCommand) (redpanda.TerminalApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[microleaseworker.RoleCheckpointConsumer]++
	return redpanda.TerminalApplyResultApplied, nil
}

func (s *fakeRuntimeStore) ApplyCloseEvent(context.Context, redpanda.CloseEventCommand) (redpanda.TerminalApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[microleaseworker.RoleCloseConsumer]++
	return redpanda.TerminalApplyResultApplied, nil
}

func (s *fakeRuntimeStore) QuarantineEvent(context.Context, redpanda.QuarantineRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls["quarantine"]++
	return nil
}

func (s *fakeRuntimeStore) ClaimOutbox(context.Context, time.Time, int32) ([]redpanda.OutboxRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[microleaseworker.RoleOutboxRelay]++
	payload := []byte(`{"result":"ok"}`)
	return []redpanda.OutboxRecord{{
		OutboxID:         "outbox-1",
		Topic:            "billing.microlease.facts.v1",
		Key:              []byte("aggregate-1"),
		EventType:        "MicroleaseIssued",
		EventFingerprint: redpanda.FingerprintOutboxPayload(payload),
		SafePayload:      payload,
	}}, nil
}

func (s *fakeRuntimeStore) MarkOutboxPublished(_ context.Context, outboxID string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.published = outboxID
	return nil
}

func (s *fakeRuntimeStore) MarkOutboxRetry(context.Context, string, time.Time, string) error {
	return nil
}

func (s *fakeRuntimeStore) RetryInbox(context.Context, int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[microleaseworker.RoleInboxRetry]++
	return nil
}

func (s *fakeRuntimeStore) ScanStaleReconciliation(_ context.Context, limit int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[microleaseworker.RoleStaleReconciliation]++
	if s.limits != nil {
		s.limits[microleaseworker.RoleStaleReconciliation] = limit
	}
	return nil
}

func (s *fakeRuntimeStore) RenewAdmissionControl(_ context.Context, limit int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[microleaseworker.RoleAdmissionControlRenew]++
	if s.limits != nil {
		s.limits[microleaseworker.RoleAdmissionControlRenew] = limit
	}
	return nil
}

type fakeRuntimeProducer struct {
	produced int
}

func (p *fakeRuntimeProducer) Produce(context.Context, redpanda.ProduceMessage) error {
	p.produced++
	return nil
}

func TestRuntimeTasksRejectMissingConcreteDependencies(t *testing.T) {
	t.Parallel()

	if _, err := runtimeTasks(config.Config{}, workerRuntimeDependencies{}); !errors.Is(err, microleaseworker.ErrInvalidWorker) {
		t.Fatalf("runtimeTasks() error = %v, want ErrInvalidWorker", err)
	}
}

func TestRuntimeTasksWireEveryConcreteRole(t *testing.T) {
	t.Parallel()

	store := &fakeRuntimeStore{calls: map[string]int{}}
	producer := &fakeRuntimeProducer{}
	terminal := &fakeRuntimeConsumer{msg: terminalRuntimeMessage(t)}
	checkpoint := &fakeRuntimeConsumer{msg: checkpointRuntimeMessage(t)}
	closeConsumer := &fakeRuntimeConsumer{msg: closeRuntimeMessage(t)}
	tasks, err := runtimeTasks(config.Config{
		Microlease: config.MicroleaseConfig{
			MaxReconciliationScanBatchSize:  10,
			AdmissionControlRenewalInterval: time.Hour,
		},
	}, workerRuntimeDependencies{
		terminalConsumer:   terminal,
		checkpointConsumer: checkpoint,
		closeConsumer:      closeConsumer,
		store:              store,
		outboxProducer:     producer,
	})
	if err != nil {
		t.Fatalf("runtimeTasks() error = %v", err)
	}
	if len(tasks) != 7 {
		t.Fatalf("tasks = %d, want 7", len(tasks))
	}
	for _, task := range tasks {
		if err := task.Run(context.Background()); err != nil {
			t.Fatalf("task %s Run() error = %v", task.Role, err)
		}
	}
	for _, role := range []string{
		microleaseworker.RoleTerminalConsumer,
		microleaseworker.RoleCheckpointConsumer,
		microleaseworker.RoleCloseConsumer,
		microleaseworker.RoleInboxRetry,
		microleaseworker.RoleOutboxRelay,
		microleaseworker.RoleStaleReconciliation,
		microleaseworker.RoleAdmissionControlRenew,
	} {
		if store.calls[role] == 0 {
			t.Fatalf("role %s did not execute concrete store path; calls = %+v", role, store.calls)
		}
	}
	if !terminal.committed || !checkpoint.committed || !closeConsumer.committed {
		t.Fatalf("consumer commits = terminal:%v checkpoint:%v close:%v, want all true", terminal.committed, checkpoint.committed, closeConsumer.committed)
	}
	if producer.produced != 1 || store.published != "outbox-1" {
		t.Fatalf("outbox relay produced=%d published=%q, want one published outbox", producer.produced, store.published)
	}
}

func TestRuntimeTasksUseDefaultBatchWhenConfigOmitsScanSize(t *testing.T) {
	t.Parallel()

	store := &fakeRuntimeStore{calls: map[string]int{}, limits: map[string]int32{}}
	tasks, err := runtimeTasks(config.Config{}, workerRuntimeDependencies{
		terminalConsumer:   &fakeRuntimeConsumer{msg: terminalRuntimeMessage(t)},
		checkpointConsumer: &fakeRuntimeConsumer{msg: checkpointRuntimeMessage(t)},
		closeConsumer:      &fakeRuntimeConsumer{msg: closeRuntimeMessage(t)},
		store:              store,
		outboxProducer:     &fakeRuntimeProducer{},
	})
	if err != nil {
		t.Fatalf("runtimeTasks() error = %v", err)
	}

	for _, task := range tasks {
		switch task.Role {
		case microleaseworker.RoleStaleReconciliation, microleaseworker.RoleAdmissionControlRenew:
			if err := task.Run(context.Background()); err != nil {
				t.Fatalf("task %s Run() error = %v", task.Role, err)
			}
		}
	}
	if got := store.limits[microleaseworker.RoleStaleReconciliation]; got != workerDefaultOutboxBatch {
		t.Fatalf("stale reconciliation limit = %d, want default %d", got, workerDefaultOutboxBatch)
	}
	if got := store.limits[microleaseworker.RoleAdmissionControlRenew]; got != workerDefaultOutboxBatch {
		t.Fatalf("admission renewal limit = %d, want default %d", got, workerDefaultOutboxBatch)
	}
}

func TestNewWorkerRuntimeBuildsRunnableConcreteTaskGraph(t *testing.T) {
	t.Parallel()

	store := &fakeRuntimeStore{calls: map[string]int{}, limits: map[string]int32{}}
	producer := &fakeRuntimeProducer{}
	worker, err := newWorkerRuntime(config.Config{
		HTTP: config.HTTPConfig{
			ShutdownTimeout: time.Second,
		},
		Redpanda: config.RedpandaConfig{
			HealthcheckTimeout: time.Second,
		},
		Microlease: config.MicroleaseConfig{
			AdmissionControlRenewalInterval: time.Hour,
			MaxReconciliationScanBatchSize:  7,
		},
	}, workerRuntimeDependencies{
		terminalConsumer:   &fakeRuntimeConsumer{msg: terminalRuntimeMessage(t)},
		checkpointConsumer: &fakeRuntimeConsumer{msg: checkpointRuntimeMessage(t)},
		closeConsumer:      &fakeRuntimeConsumer{msg: closeRuntimeMessage(t)},
		store:              store,
		outboxProducer:     producer,
	})
	if err != nil {
		t.Fatalf("newWorkerRuntime() error = %v", err)
	}
	if err := worker.Ready(context.Background()); !errors.Is(err, microleaseworker.ErrNotReady) {
		t.Fatalf("Ready(before Run) error = %v, want ErrNotReady", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := worker.Run(runCtx); err != nil {
		t.Fatalf("Run(canceled) error = %v, want nil shutdown", err)
	}

	for _, role := range []string{
		microleaseworker.RoleTerminalConsumer,
		microleaseworker.RoleCheckpointConsumer,
		microleaseworker.RoleCloseConsumer,
		microleaseworker.RoleInboxRetry,
		microleaseworker.RoleOutboxRelay,
		microleaseworker.RoleStaleReconciliation,
		microleaseworker.RoleAdmissionControlRenew,
	} {
		if store.calls[role] == 0 {
			t.Fatalf("role %s did not run through worker graph; calls = %+v", role, store.calls)
		}
	}
	if producer.produced != 1 || store.published != "outbox-1" {
		t.Fatalf("outbox relay produced=%d published=%q, want one published event", producer.produced, store.published)
	}
	if got := store.limits[microleaseworker.RoleStaleReconciliation]; got != 7 {
		t.Fatalf("stale reconciliation limit = %d, want configured batch 7", got)
	}
	if got := store.limits[microleaseworker.RoleAdmissionControlRenew]; got != 7 {
		t.Fatalf("admission renewal limit = %d, want configured batch 7", got)
	}
}

func TestConsumeOnceCancellationLeavesWorkReplayable(t *testing.T) {
	t.Parallel()

	consumer := &fakeRuntimeConsumer{err: context.Canceled}
	called := false
	err := consumeOnce(context.Background(), consumer, func(context.Context, redpanda.Message) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("consumeOnce(context canceled) error = %v", err)
	}
	if !consumer.fetched {
		t.Fatal("FetchMessage was not called")
	}
	if called {
		t.Fatal("handler called after canceled fetch; want no handler call")
	}

	consumer = &fakeRuntimeConsumer{err: errors.New("broker unavailable")}
	err = consumeOnce(context.Background(), consumer, func(context.Context, redpanda.Message) error {
		t.Fatal("handler called after failed fetch")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "fetch worker message") {
		t.Fatalf("consumeOnce(fetch error) error = %v, want wrapped fetch error", err)
	}
}

func TestParseLoadOptionsAndOverlayFlag(t *testing.T) {
	t.Parallel()

	var overlays overlayPathsFlag
	if err := overlays.Set("  overlay-a.yaml  "); err != nil {
		t.Fatalf("overlay Set() error = %v", err)
	}
	if err := overlays.Set(" "); err == nil {
		t.Fatal("overlay Set(empty) error = nil, want error")
	}
	if got := overlays.String(); got != "overlay-a.yaml" {
		t.Fatalf("overlay String() = %q, want trimmed path", got)
	}

	opts, err := parseLoadOptions([]string{"-config", " config.yaml ", "-config-overlay", "overlay-a.yaml", "-config-overlay", "overlay-b.yaml", "-config-strict"})
	if err != nil {
		t.Fatalf("parseLoadOptions() error = %v", err)
	}
	if opts.ConfigPath != "config.yaml" || !opts.Strict || len(opts.ConfigOverlays) != 2 || opts.ConfigOverlays[1] != "overlay-b.yaml" {
		t.Fatalf("load options = %+v, want config, strict, two overlays", opts)
	}
	if opts.LoadBudget != 10*time.Second || opts.ValidateBudget != 2*time.Second {
		t.Fatalf("budgets = %s/%s, want fixed worker budgets", opts.LoadBudget, opts.ValidateBudget)
	}

	if _, err := parseLoadOptions([]string{"-config-overlay", ""}); err == nil {
		t.Fatal("parseLoadOptions(empty overlay) error = nil, want error")
	}
}

func TestWorkerConfigAndBrokerParsing(t *testing.T) {
	t.Parallel()

	cfg := config.Config{}
	cfg.Redpanda.HealthcheckTimeout = 3 * time.Second
	cfg.Microlease.AdmissionControlRenewalInterval = 4 * time.Second
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	workerCfg := workerConfig(cfg)
	if workerCfg.ReadinessTimeout != 3*time.Second || workerCfg.DefaultInterval != 4*time.Second || workerCfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("workerConfig() = %+v, want configured timeouts", workerCfg)
	}

	if got := splitBrokers(" redpanda:9092, ,localhost:19092 "); strings.Join(got, ",") != "redpanda:9092,localhost:19092" {
		t.Fatalf("splitBrokers() = %v, want trimmed non-empty brokers", got)
	}
}

func TestWorkerEventStoreFailsClosedWithoutConcreteRepositories(t *testing.T) {
	t.Parallel()

	store := &workerEventStore{billingFactsTopic: "billing.microlease.facts.v1"}
	if _, err := store.ApplyTerminalEvent(context.Background(), redpanda.TerminalEventCommand{}); err == nil || !strings.Contains(err.Error(), "read terminal child debit") {
		t.Fatalf("ApplyTerminalEvent(nil repos) error = %v, want read child debit failure", err)
	}
	if _, err := store.ApplyCheckpointEvent(context.Background(), redpanda.CheckpointEventCommand{}); err == nil || !strings.Contains(err.Error(), "read checkpoint microlease") {
		t.Fatalf("ApplyCheckpointEvent(nil repos) error = %v, want read microlease failure", err)
	}
	if _, err := store.ApplyCloseEvent(context.Background(), redpanda.CloseEventCommand{}); err == nil || !strings.Contains(err.Error(), "read close microlease") {
		t.Fatalf("ApplyCloseEvent(nil repos) error = %v, want read microlease failure", err)
	}
	if err := store.QuarantineEvent(context.Background(), redpanda.QuarantineRecord{}); err == nil || !strings.Contains(err.Error(), "record quarantine") {
		t.Fatalf("QuarantineEvent(nil repos) error = %v, want quarantine failure", err)
	}
	if _, err := store.ClaimOutbox(context.Background(), time.Now(), 1); err == nil || !strings.Contains(err.Error(), "claim billing outbox") {
		t.Fatalf("ClaimOutbox(nil repos) error = %v, want claim failure", err)
	}
	if err := store.MarkOutboxPublished(context.Background(), "outbox", time.Now()); err == nil || !strings.Contains(err.Error(), "mark outbox published") {
		t.Fatalf("MarkOutboxPublished(nil repos) error = %v, want publish mark failure", err)
	}
	if err := store.MarkOutboxRetry(context.Background(), "outbox", time.Now(), "publish_failed"); err == nil || !strings.Contains(err.Error(), "mark outbox retry") {
		t.Fatalf("MarkOutboxRetry(nil repos) error = %v, want retry mark failure", err)
	}
	if err := store.RetryInbox(context.Background(), 1); err == nil || !strings.Contains(err.Error(), "retry inbox") {
		t.Fatalf("RetryInbox(nil repos) error = %v, want retry failure", err)
	}
	if err := store.ScanStaleReconciliation(context.Background(), 1); err == nil || !strings.Contains(err.Error(), "scan stale reconciliation") {
		t.Fatalf("ScanStaleReconciliation(nil repos) error = %v, want scan failure", err)
	}
	if err := store.RenewAdmissionControl(context.Background(), 1); err == nil || !strings.Contains(err.Error(), "scan stale microleases for admission control") {
		t.Fatalf("RenewAdmissionControl(nil repos) error = %v, want scan failure", err)
	}

	fixed := time.Date(2026, 6, 2, 1, 0, 0, 0, time.FixedZone("test", 3*60*60))
	store.now = func() time.Time { return fixed }
	if got := store.nowValue(); !got.Equal(fixed.UTC()) {
		t.Fatalf("nowValue() = %v, want UTC-normalized %v", got, fixed.UTC())
	}
}

func TestBuildWorkerRuntimeFailsClosedWhenPostgresCannotOpen(t *testing.T) {
	t.Parallel()

	worker, cleanup, err := buildWorkerRuntime(context.Background(), config.Config{})
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatalf("buildWorkerRuntime() worker=%v error=nil, want Postgres open error", worker)
	}
	if worker != nil || cleanup != nil {
		t.Fatalf("buildWorkerRuntime() worker=%v cleanup=%v, want no runtime on open failure", worker, cleanup)
	}
	if !strings.Contains(err.Error(), "open worker postgres") {
		t.Fatalf("buildWorkerRuntime() error = %v, want open worker postgres context", err)
	}
}

func terminalRuntimeMessage(t *testing.T) redpanda.Message {
	t.Helper()
	now := time.Date(2026, 6, 2, 1, 0, 0, 0, time.UTC)
	event := eventsv1.MicroleaseChildTerminalSubmitted{
		Envelope: eventsv1.MicroleaseEventEnvelope{
			EventID:           "event-terminal-runtime",
			EventType:         "MicroleaseChildTerminalSubmitted",
			ContractVersion:   "v1",
			SchemaVersion:     "billing.events.v1",
			ProducerIdentity:  "gonka-proxy",
			OccurredAtEpochMS: now.UnixMilli(),
			ProducedAtEpochMS: now.UnixMilli(),
		},
		Identity: eventsv1.MicroleaseIdentity{
			MicroleaseID:          "11111111-1111-1111-1111-111111111111",
			AccountScopeKey:       "acct_runtime",
			ProxyAllocatorOwnerID: "proxy-owner",
			MicroleaseGeneration:  1,
		},
		DebitAuthorizationID:     "debit-runtime",
		ChildSequence:            1,
		ChildCapUSDAtoms:         100,
		TerminalKind:             "finalize",
		ChargedUSDAtoms:          40,
		ReleasedUSDAtoms:         60,
		RequestBasisFingerprint:  "request-basis",
		TerminalBasisFingerprint: "terminal-basis",
		Pricing: eventsv1.PricingSnapshotBasis{
			PricingSnapshotID:   "pricing-1",
			SnapshotFingerprint: "pricing-fingerprint",
			PolicyVersion:       "pricing-policy",
		},
		TerminalDeadlineEpochMS: now.Add(time.Minute).UnixMilli(),
		ObservedTerminalEpochMS: now.UnixMilli(),
	}
	fingerprint, err := redpanda.FingerprintTerminalSubmitted(event)
	event.Envelope.EventFingerprint = mustRuntimeFingerprint(t, fingerprint, err)
	return runtimeMessage(t, "billing.microlease.terminal.v1", event)
}

func checkpointRuntimeMessage(t *testing.T) redpanda.Message {
	t.Helper()
	now := time.Date(2026, 6, 2, 1, 0, 0, 0, time.UTC)
	event := checkpointRuntimeEvent(now)
	fingerprint, err := redpanda.FingerprintCheckpointReported(event)
	event.Envelope.EventFingerprint = mustRuntimeFingerprint(t, fingerprint, err)
	return runtimeMessage(t, "billing.microlease.checkpoint.v1", event)
}

func closeRuntimeMessage(t *testing.T) redpanda.Message {
	t.Helper()
	now := time.Date(2026, 6, 2, 1, 0, 0, 0, time.UTC)
	checkpoint := checkpointRuntimeEvent(now)
	checkpoint.Envelope.EventID = "event-close-runtime"
	checkpoint.Envelope.EventType = "MicroleaseCloseReported"
	event := eventsv1.MicroleaseCloseReported{
		Checkpoint:                 checkpoint,
		CloseReason:                "normal_close",
		AllocatorClosedEpochMS:     now.UnixMilli(),
		FinalLocalStateFingerprint: "final-local-state",
	}
	fingerprint, err := redpanda.FingerprintCloseReported(event)
	event.Checkpoint.Envelope.EventFingerprint = mustRuntimeFingerprint(t, fingerprint, err)
	return runtimeMessage(t, "billing.microlease.close.v1", event)
}

func checkpointRuntimeEvent(now time.Time) eventsv1.MicroleaseCheckpointReported {
	return eventsv1.MicroleaseCheckpointReported{
		Envelope: eventsv1.MicroleaseEventEnvelope{
			EventID:           "event-checkpoint-runtime",
			EventType:         "MicroleaseCheckpointReported",
			ContractVersion:   "v1",
			SchemaVersion:     "billing.events.v1",
			ProducerIdentity:  "gonka-proxy",
			OccurredAtEpochMS: now.UnixMilli(),
			ProducedAtEpochMS: now.UnixMilli(),
		},
		Identity: eventsv1.MicroleaseIdentity{
			MicroleaseID:          "11111111-1111-1111-1111-111111111111",
			AccountScopeKey:       "acct_runtime",
			ProxyAllocatorOwnerID: "proxy-owner",
			MicroleaseGeneration:  1,
		},
		CheckpointSequence:            1,
		CheckpointKind:                "periodic",
		AllocatedChildHighWater:       1,
		AllocatedChildCount:           1,
		AllocatedChildCapSumUSDAtoms:  100,
		TerminalSubmittedCount:        1,
		TerminalPublishedCount:        1,
		TerminalAcceptedCount:         1,
		UnresolvedChildCount:          0,
		UnresolvedChildCapSumUSDAtoms: 0,
		LocalRemainingUSDAtoms:        0,
		CheckpointFingerprint:         "checkpoint-fingerprint",
	}
}

func runtimeMessage(t *testing.T, topic string, event any) redpanda.Message {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal runtime event: %v", err)
	}
	return redpanda.Message{
		Topic:     topic,
		Partition: 0,
		Offset:    1,
		Key:       []byte("runtime"),
		Value:     data,
	}
}

func mustRuntimeFingerprint(t *testing.T, fingerprint string, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("fingerprint runtime event: %v", err)
	}
	return fingerprint
}
