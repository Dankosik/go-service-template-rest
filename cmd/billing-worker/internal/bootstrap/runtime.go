package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Dankosik/billing-service/internal/app/billingauthority"
	"github.com/Dankosik/billing-service/internal/app/microleaseworker"
	"github.com/Dankosik/billing-service/internal/config"
	"github.com/Dankosik/billing-service/internal/infra/postgres"
	"github.com/Dankosik/billing-service/internal/infra/redpanda"
	"github.com/google/uuid"
)

const (
	workerDefaultOutboxBatch = int32(100)
	workerProducerIdentity   = "billing-service"
)

type runtimeCleanup func()

type workerRuntimeDependencies struct {
	probes             []microleaseworker.Probe
	terminalConsumer   redpanda.MessageConsumer
	checkpointConsumer redpanda.MessageConsumer
	closeConsumer      redpanda.MessageConsumer
	store              workerStore
	outboxProducer     redpanda.Producer
	closers            []redpanda.Closeable
}

type workerStore interface {
	redpanda.TerminalEventStore
	redpanda.CheckpointEventStore
	redpanda.CloseEventStore
	redpanda.OutboxStore
	RetryInbox(context.Context, int32) error
	ScanStaleReconciliation(context.Context, int32) error
	RenewAdmissionControl(context.Context, int32) error
}

type workerEventStore struct {
	authority         *billingauthority.Service
	authorityRepo     *postgres.BillingAuthorityRepository
	microleaseRepo    *postgres.MicroleaseRepository
	billingFactsTopic string
	scanBatchSize     int32
	now               func() time.Time
}

func buildWorkerRuntime(ctx context.Context, cfg config.Config) (*microleaseworker.Worker, runtimeCleanup, error) {
	pg, err := postgres.New(ctx, postgres.Options{
		DSN:                cfg.Postgres.DSN,
		ConnectTimeout:     cfg.Postgres.ConnectTimeout,
		HealthcheckTimeout: cfg.Postgres.HealthcheckTimeout,
		MaxOpenConns:       cfg.Postgres.MaxOpenConns,
		MaxIdleConns:       cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime:    cfg.Postgres.ConnMaxLifetime,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("open worker postgres: %w", err)
	}
	cleanup := runtimeCleanup(func() { pg.Close() })

	authorityRepo, err := postgres.NewBillingAuthorityRepository(pg)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("build authority repository: %w", err)
	}
	microleaseRepo, err := postgres.NewMicroleaseRepository(pg)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("build microlease repository: %w", err)
	}
	authoritySvc, err := billingauthority.New(authorityRepo)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("build authority service: %w", err)
	}

	brokers := splitBrokers(cfg.Redpanda.Brokers)
	terminalConsumer, err := redpanda.NewKafkaConsumer(redpanda.ClientConfig{
		Brokers:       brokers,
		Topic:         cfg.Redpanda.TerminalTopic,
		ConsumerGroup: cfg.Redpanda.ConsumerGroup,
	})
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("build terminal consumer: %w", err)
	}
	checkpointConsumer, err := redpanda.NewKafkaConsumer(redpanda.ClientConfig{
		Brokers:       brokers,
		Topic:         cfg.Redpanda.CheckpointTopic,
		ConsumerGroup: cfg.Redpanda.ConsumerGroup,
	})
	if err != nil {
		_ = terminalConsumer.Close()
		cleanup()
		return nil, nil, fmt.Errorf("build checkpoint consumer: %w", err)
	}
	closeConsumer, err := redpanda.NewKafkaConsumer(redpanda.ClientConfig{
		Brokers:       brokers,
		Topic:         cfg.Redpanda.CloseTopic,
		ConsumerGroup: cfg.Redpanda.ConsumerGroup,
	})
	if err != nil {
		_ = terminalConsumer.Close()
		_ = checkpointConsumer.Close()
		cleanup()
		return nil, nil, fmt.Errorf("build close consumer: %w", err)
	}
	producer, err := redpanda.NewKafkaProducer(brokers)
	if err != nil {
		_ = terminalConsumer.Close()
		_ = checkpointConsumer.Close()
		_ = closeConsumer.Close()
		cleanup()
		return nil, nil, fmt.Errorf("build outbox producer: %w", err)
	}

	store := &workerEventStore{
		authority:         authoritySvc,
		authorityRepo:     authorityRepo,
		microleaseRepo:    microleaseRepo,
		billingFactsTopic: cfg.Redpanda.BillingFactsTopic,
		scanBatchSize:     int32(cfg.Microlease.MaxReconciliationScanBatchSize), // #nosec G115 -- config validates the range.
		now:               func() time.Time { return time.Now().UTC() },
	}
	deps := workerRuntimeDependencies{
		probes: []microleaseworker.Probe{
			pg,
			redpanda.NewBrokerProbe(brokers, cfg.Redpanda.HealthcheckTimeout),
		},
		terminalConsumer:   terminalConsumer,
		checkpointConsumer: checkpointConsumer,
		closeConsumer:      closeConsumer,
		store:              store,
		outboxProducer:     producer,
		closers: []redpanda.Closeable{
			terminalConsumer,
			checkpointConsumer,
			closeConsumer,
			producer,
		},
	}
	worker, err := newWorkerRuntime(cfg, deps)
	if err != nil {
		for _, closer := range deps.closers {
			_ = closer.Close()
		}
		cleanup()
		return nil, nil, err
	}
	return worker, runtimeCleanup(func() {
		for _, closer := range deps.closers {
			_ = closer.Close()
		}
		cleanup()
	}), nil
}

func newWorkerRuntime(cfg config.Config, deps workerRuntimeDependencies) (*microleaseworker.Worker, error) {
	tasks, err := runtimeTasks(cfg, deps)
	if err != nil {
		return nil, err
	}
	worker, err := microleaseworker.New(workerConfig(cfg), deps.probes, tasks, nil)
	if err != nil {
		return nil, fmt.Errorf("create microlease worker: %w", err)
	}
	return worker, nil
}

func runtimeTasks(cfg config.Config, deps workerRuntimeDependencies) ([]microleaseworker.Task, error) {
	if deps.terminalConsumer == nil || deps.checkpointConsumer == nil || deps.closeConsumer == nil {
		return nil, fmt.Errorf("%w: redpanda consumers are required", microleaseworker.ErrInvalidWorker)
	}
	if deps.store == nil || deps.outboxProducer == nil {
		return nil, fmt.Errorf("%w: durable worker store and producer are required", microleaseworker.ErrInvalidWorker)
	}
	allowedProducers := []string{"gonka-proxy"}
	retryPolicy := redpanda.RetryPolicy{BaseDelay: 50 * time.Millisecond, MaxDelay: time.Second}
	store := deps.store
	batchSize := int32(cfg.Microlease.MaxReconciliationScanBatchSize) // #nosec G115 -- config validates the range.
	if batchSize <= 0 {
		batchSize = workerDefaultOutboxBatch
	}
	return []microleaseworker.Task{
		{
			Role:           microleaseworker.RoleTerminalConsumer,
			MaxConcurrency: 1,
			Run: func(ctx context.Context) error {
				return consumeOnce(ctx, deps.terminalConsumer, func(ctx context.Context, msg redpanda.Message) error {
					return redpanda.TerminalConsumer{
						Store:            store,
						Committer:        deps.terminalConsumer,
						AllowedProducers: allowedProducers,
						RetryPolicy:      retryPolicy,
					}.Handle(ctx, msg)
				})
			},
		},
		{
			Role:           microleaseworker.RoleCheckpointConsumer,
			MaxConcurrency: 1,
			Run: func(ctx context.Context) error {
				return consumeOnce(ctx, deps.checkpointConsumer, func(ctx context.Context, msg redpanda.Message) error {
					return redpanda.CheckpointConsumer{
						Store:            store,
						Committer:        deps.checkpointConsumer,
						AllowedProducers: allowedProducers,
						RetryPolicy:      retryPolicy,
					}.Handle(ctx, msg)
				})
			},
		},
		{
			Role:           microleaseworker.RoleCloseConsumer,
			MaxConcurrency: 1,
			Run: func(ctx context.Context) error {
				return consumeOnce(ctx, deps.closeConsumer, func(ctx context.Context, msg redpanda.Message) error {
					return redpanda.CloseConsumer{
						Store:            store,
						Committer:        deps.closeConsumer,
						AllowedProducers: allowedProducers,
						RetryPolicy:      retryPolicy,
					}.Handle(ctx, msg)
				})
			},
		},
		{
			Role:           microleaseworker.RoleInboxRetry,
			MaxConcurrency: 1,
			Run: func(ctx context.Context) error {
				return store.RetryInbox(ctx, batchSize)
			},
		},
		{
			Role:           microleaseworker.RoleOutboxRelay,
			MaxConcurrency: 1,
			Run: func(ctx context.Context) error {
				_, err := redpanda.OutboxRelay{
					Store:            store,
					Producer:         deps.outboxProducer,
					ProducerIdentity: workerProducerIdentity,
					RetryPolicy:      retryPolicy,
				}.RelayOnce(ctx, batchSize)
				if err != nil {
					return fmt.Errorf("relay billing outbox: %w", err)
				}
				return nil
			},
		},
		{
			Role:           microleaseworker.RoleStaleReconciliation,
			MaxConcurrency: 1,
			Run: func(ctx context.Context) error {
				return store.ScanStaleReconciliation(ctx, batchSize)
			},
		},
		{
			Role:           microleaseworker.RoleAdmissionControlRenew,
			MaxConcurrency: 1,
			Interval:       cfg.Microlease.AdmissionControlRenewalInterval,
			Run: func(ctx context.Context) error {
				return store.RenewAdmissionControl(ctx, batchSize)
			},
		},
	}, nil
}

func consumeOnce(ctx context.Context, consumer redpanda.MessageConsumer, handle func(context.Context, redpanda.Message) error) error {
	msg, err := consumer.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return fmt.Errorf("fetch worker message: %w", err)
	}
	return handle(ctx, msg)
}

func (s *workerEventStore) ApplyTerminalEvent(ctx context.Context, cmd redpanda.TerminalEventCommand) (redpanda.TerminalApplyResult, error) {
	child, err := s.authorityRepo.ReadTerminalChildDebitByAuthorization(ctx, cmd.MicroleaseID, cmd.DebitAuthorizationID)
	if err != nil {
		return "", fmt.Errorf("read terminal child debit: %w", err)
	}
	if child.UsageOperationID == "" {
		return redpanda.TerminalApplyResultConflict, nil
	}
	terminal := billingauthority.UsageTerminalCommand{
		AccountScopeKey:        child.AccountScopeKey,
		UsageOperationID:       child.UsageOperationID,
		IdempotencyKey:         cmd.EventID,
		RequestFingerprint:     cmd.EventFingerprint,
		MicroleaseID:           child.MicroleaseID,
		MicroleaseChildDebitID: child.MicroleaseChildDebitID,
		DebitAuthorizationID:   child.DebitAuthorizationID,
		TerminalOutcomeID:      deterministicUUID("terminal-outcome:" + cmd.EventID),
		TerminalFingerprint:    cmd.TerminalBasisFingerprint,
		ChargedUSDAtoms:        cmd.ChargedUSDAtoms,
		ReleasedUSDAtoms:       cmd.ReleasedUSDAtoms,
		WriteOffUSDAtoms:       cmd.WriteOffUSDAtoms,
		Pricing: billingauthority.PricingSnapshot{
			ID:          cmd.PricingSnapshotID,
			Fingerprint: cmd.PricingSnapshotFingerprint,
		},
		Metadata: cmd.SafeMetadata,
	}
	var snapshot billingauthority.UsageOperationSnapshot
	switch cmd.TerminalKind {
	case "write_off":
		snapshot, err = s.authority.WriteOffUsage(ctx, terminal)
	case "finalize", "abort_release":
		snapshot, err = s.authority.FinalizeUsage(ctx, terminal)
	default:
		return redpanda.TerminalApplyResultConflict, nil
	}
	if errors.Is(err, billingauthority.ErrConflict) || errors.Is(err, billingauthority.ErrRejected) || errors.Is(err, billingauthority.ErrInvalidRequest) {
		return redpanda.TerminalApplyResultConflict, nil
	}
	if err != nil {
		return "", fmt.Errorf("apply terminal authority command: %w", err)
	}
	if snapshot.ResultCode == "duplicate_stored_outcome" {
		return redpanda.TerminalApplyResultDuplicate, nil
	}
	return redpanda.TerminalApplyResultApplied, nil
}

func (s *workerEventStore) ApplyCheckpointEvent(ctx context.Context, cmd redpanda.CheckpointEventCommand) (redpanda.TerminalApplyResult, error) {
	lease, err := s.microleaseRepo.ReadMicrolease(ctx, cmd.MicroleaseID)
	if err != nil {
		return "", fmt.Errorf("read checkpoint microlease: %w", err)
	}
	if err := s.microleaseRepo.RecordCheckpoint(ctx, postgres.CheckpointCommand{
		InboxID:                    deterministicUUID("checkpoint-inbox:" + cmd.EventID),
		Topic:                      cmd.Topic,
		PartitionID:                cmd.PartitionID,
		OffsetValue:                cmd.OffsetValue,
		EventID:                    cmd.EventID,
		ProducerIdentity:           cmd.ProducerIdentity,
		EventFingerprint:           cmd.EventFingerprint,
		CheckpointID:               deterministicUUID("checkpoint:" + cmd.MicroleaseID + ":" + fmt.Sprint(cmd.CheckpointSequence)),
		MicroleaseID:               cmd.MicroleaseID,
		AccountID:                  lease.AccountID,
		AccountScopeKey:            cmd.AccountScopeKey,
		ProxyAllocatorOwnerID:      cmd.ProxyAllocatorOwnerID,
		MicroleaseGeneration:       cmd.MicroleaseGeneration,
		CheckpointSequence:         cmd.CheckpointSequence,
		CheckpointKind:             cmd.CheckpointKind,
		AllocatedChildHighWater:    cmd.AllocatedChildHighWater,
		AllocatedChildCount:        cmd.AllocatedChildCount,
		AllocatedChildCapUSDAtoms:  cmd.AllocatedChildCapSumUSDAtoms,
		TerminalSubmittedCount:     cmd.TerminalSubmittedCount,
		TerminalPublishedCount:     cmd.TerminalPublishedCount,
		TerminalAcceptedCount:      cmd.TerminalAcceptedCount,
		UnresolvedChildCount:       cmd.UnresolvedChildCount,
		UnresolvedChildCapUSDAtoms: cmd.UnresolvedChildCapSumUSDAtoms,
		LocalRemainingUSDAtoms:     cmd.LocalRemainingUSDAtoms,
		CheckpointFingerprint:      cmd.CheckpointFingerprint,
		CreatedAt:                  cmd.CreatedAt,
		AppliedAt:                  s.nowValue(),
		SafeMetadata:               cmd.SafeMetadata,
	}); err != nil {
		return "", fmt.Errorf("record checkpoint: %w", err)
	}
	return redpanda.TerminalApplyResultApplied, nil
}

func (s *workerEventStore) ApplyCloseEvent(ctx context.Context, cmd redpanda.CloseEventCommand) (redpanda.TerminalApplyResult, error) {
	lease, err := s.microleaseRepo.ReadMicrolease(ctx, cmd.MicroleaseID)
	if err != nil {
		return "", fmt.Errorf("read close microlease: %w", err)
	}
	unresolved := cmd.UnresolvedChildCapSumUSDAtoms
	released := max(lease.AvailableChildUSDAtoms-unresolved, 0)
	state := "closed"
	if unresolved > 0 {
		state = "reconcile_required"
	}
	if _, err := s.microleaseRepo.Close(ctx, postgres.CloseMicroleaseCommand{
		MicroleaseID:            cmd.MicroleaseID,
		AccountID:               lease.AccountID,
		IdempotencyRecordID:     deterministicUUID("close-idempotency:" + cmd.EventID),
		IdempotencyKey:          cmd.EventID,
		RequestFingerprint:      cmd.EventFingerprint,
		StoredOutcomeID:         deterministicUUID("close-outcome:" + cmd.EventID),
		LedgerEntryID:           deterministicUUID("close-ledger:" + cmd.EventID),
		OutboxID:                deterministicUUID("close-outbox:" + cmd.EventID),
		EventFingerprint:        cmd.EventFingerprint,
		ReleasedUSDAtoms:        released,
		UnresolvedReservedAtoms: unresolved,
		CloseState:              state,
		ClosedAt:                cmd.AllocatorClosedAt,
		Now:                     s.nowValue(),
		SafeMetadata:            cmd.SafeMetadata,
	}); err != nil {
		return "", fmt.Errorf("close microlease: %w", err)
	}
	return redpanda.TerminalApplyResultApplied, nil
}

func (s *workerEventStore) QuarantineEvent(ctx context.Context, record redpanda.QuarantineRecord) error {
	if err := s.microleaseRepo.RecordQuarantine(ctx, postgres.QuarantineCommand{
		InboxID:          deterministicUUID(fmt.Sprintf("quarantine:%s:%d:%d", record.Topic, record.PartitionID, record.OffsetValue)),
		Topic:            record.Topic,
		PartitionID:      record.PartitionID,
		OffsetValue:      record.OffsetValue,
		EventID:          record.EventID,
		ProducerIdentity: record.ProducerIdentity,
		BusinessIdentity: fmt.Sprintf("%s:%d:%d", record.Topic, record.PartitionID, record.OffsetValue),
		EventFingerprint: record.EventFingerprint,
		ReasonClass:      record.ReasonClass,
		QuarantinedAt:    record.QuarantinedAt,
		SafeMetadata:     record.SafeMetadata,
	}); err != nil {
		return fmt.Errorf("record quarantine: %w", err)
	}
	return nil
}

func (s *workerEventStore) ClaimOutbox(ctx context.Context, now time.Time, limit int32) ([]redpanda.OutboxRecord, error) {
	rows, err := s.microleaseRepo.ClaimOutbox(ctx, now, limit)
	if err != nil {
		return nil, fmt.Errorf("claim billing outbox: %w", err)
	}
	records := make([]redpanda.OutboxRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, redpanda.OutboxRecord{
			OutboxID:         uuid.UUID(row.OutboxID.Bytes).String(),
			Topic:            s.billingFactsTopic,
			Key:              []byte(row.AggregateID),
			EventType:        row.EventType,
			EventFingerprint: row.EventFingerprint,
			SafePayload:      row.SafePayload,
			Attempt:          int(row.AttemptCount),
		})
	}
	return records, nil
}

func (s *workerEventStore) MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error {
	if err := s.microleaseRepo.MarkOutboxPublished(ctx, outboxID, publishedAt); err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	return nil
}

func (s *workerEventStore) MarkOutboxRetry(ctx context.Context, outboxID string, nextAttempt time.Time, reason string) error {
	if err := s.microleaseRepo.MarkOutboxRetry(ctx, outboxID, nextAttempt, reason); err != nil {
		return fmt.Errorf("mark outbox retry: %w", err)
	}
	return nil
}

func (s *workerEventStore) RetryInbox(ctx context.Context, limit int32) error {
	_, err := s.microleaseRepo.RetryEligibleInbox(ctx, s.nowValue(), limit)
	if err != nil {
		return fmt.Errorf("retry inbox: %w", err)
	}
	return nil
}

func (s *workerEventStore) ScanStaleReconciliation(ctx context.Context, limit int32) error {
	_, err := s.microleaseRepo.ScanStaleMicroleases(ctx, s.nowValue(), limit)
	if err != nil {
		return fmt.Errorf("scan stale reconciliation: %w", err)
	}
	return nil
}

func (s *workerEventStore) RenewAdmissionControl(ctx context.Context, limit int32) error {
	stale, err := s.microleaseRepo.ScanStaleMicroleases(ctx, s.nowValue(), limit)
	if err != nil {
		return fmt.Errorf("scan stale microleases for admission control: %w", err)
	}
	state := "open"
	reason := "worker_ready"
	staleBucket := "none"
	if len(stale) > 0 {
		state = "fail_closed"
		reason = "stale_microlease_exposure"
		staleBucket = "critical"
	}
	now := s.nowValue()
	if err := s.microleaseRepo.UpsertAdmissionControl(ctx, postgres.AdmissionControlCommand{
		AdmissionControlID:          deterministicUUID("global-admission:usage_reserve"),
		ScopeKind:                   "global",
		ScopeKey:                    "all",
		UseClass:                    "usage_reserve",
		State:                       state,
		ReasonCode:                  reason,
		TerminalLagBucket:           "unknown",
		StaleAgeBucket:              staleBucket,
		ReconciliationBacklogBucket: "unknown",
		AuditedActorKind:            "worker",
		AuditedActorID:              workerProducerIdentity,
		ExpiresAt:                   now.Add(2 * time.Minute),
		RenewedAt:                   now,
		CreatedAt:                   now,
		SafeMetadata:                map[string]string{"worker_role": microleaseworker.RoleAdmissionControlRenew},
	}); err != nil {
		return fmt.Errorf("renew admission control: %w", err)
	}
	return nil
}

func (s *workerEventStore) nowValue() time.Time {
	if s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func deterministicUUID(seed string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seed)).String()
}

func splitBrokers(raw string) []string {
	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			brokers = append(brokers, trimmed)
		}
	}
	return brokers
}
