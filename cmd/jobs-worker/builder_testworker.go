//go:build jobs_test_worker

package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
)

// buildRegistry is compiled only for the black-box process carrier. It is the
// neutral F-JOBS fixture, never a production kind or an environment switch.
var buildRegistry bootstrap.RegistryBuilder = func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
	definitionInput := jobs.DefinitionInput[map[string]string]{
		Revision:        jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"},
		MaxPayloadBytes: 1024,
		Validate: func(args map[string]string) error {
			if len(args) == 0 {
				return errors.New("arguments are required")
			}
			return nil
		},
		Policy: jobs.Policy{
			Effect:             jobs.EffectPolicy{AmbiguousAction: jobs.AmbiguousEffectRetry},
			Retry:              jobs.RetryPolicy{MaxAttempts: 3, MaxElapsed: time.Hour, InitialBackoff: 20 * time.Millisecond, MaxBackoff: 20 * time.Millisecond, HintPolicy: jobs.RetryHintIgnore, Jitter: jobs.JitterNone, MaxRecoveryWave: 8},
			Recovery:           jobs.RecoveryPolicy{Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved},
			MaxAttemptDuration: 8 * time.Second, TerminationEnvelope: 8 * time.Second,
		},
	}
	definition, err := jobs.NewDefinition(definitionInput)
	if err != nil {
		return nil, nil, err
	}
	handler := func(ctx context.Context, input jobs.HandlerInput[map[string]string]) jobs.HandlerResult {
		switch os.Getenv("JOBS_WORKER_TEST_HANDLER") {
		case "noncooperative":
			select {}
		case "lease":
			writeJobsWorkerTestFile(os.Getenv("JOBS_WORKER_TEST_ENTERED_FILE"), "entered\n")
			<-ctx.Done()
			writeJobsWorkerTestFile(os.Getenv("JOBS_WORKER_TEST_CANCELLED_FILE"), "cancelled\n")
			select {}
		case "recovery":
			return runJobsWorkerRecoveryHandler(ctx, input)
		}
		<-ctx.Done()
		return jobs.HandlerResult{Outcome: jobs.OutcomeCancelled, Effect: jobs.EffectUnknown}
	}
	registry := new(jobs.Registry)
	if err := jobs.Register(registry, definition, handler); err != nil {
		return nil, nil, err
	}
	acceptanceInput := definitionInput
	acceptanceInput.Revision = jobs.Revision{Kind: "acceptance", ArgsVersion: "v1", PolicyVersion: "v1"}
	acceptanceInput.Policy.Effect.AmbiguousAction = jobs.AmbiguousEffectOutcomeUnknown
	acceptanceInput.Policy.Retry = jobs.RetryPolicy{MaxAttempts: 4, MaxElapsed: time.Hour, InitialBackoff: time.Second, MaxBackoff: time.Minute, HintPolicy: jobs.RetryHintPrefer, Jitter: jobs.JitterSHA256, JitterPermille: 100, MaxRecoveryWave: 8}
	acceptance, err := jobs.NewDefinition(acceptanceInput)
	if err != nil {
		return nil, nil, err
	}
	if err := jobs.Register(registry, acceptance, handler); err != nil {
		return nil, nil, err
	}
	return registry, func(context.Context) {
		writeJobsWorkerTestFile(os.Getenv("JOBS_WORKER_TEST_CLEANUP_FILE"), "cleaned\n")
	}, nil
}

func runJobsWorkerRecoveryHandler(ctx context.Context, input jobs.HandlerInput[map[string]string]) jobs.HandlerResult {
	writeJobsWorkerTestFile(os.Getenv("JOBS_WORKER_TEST_ENTERED_FILE"), "entered\n")
	switch os.Getenv("JOBS_WORKER_TEST_RESULT") {
	case "permanent":
		return jobs.HandlerResult{Outcome: jobs.OutcomePermanent, Effect: jobs.EffectNone}
	case "poison":
		return jobs.HandlerResult{Outcome: jobs.OutcomePoison, Effect: jobs.EffectNone}
	case "unknown":
		return jobs.HandlerResult{Outcome: jobs.OutcomeUnknown, Effect: jobs.EffectNone}
	case "retryable", "exhausted":
		return jobs.HandlerResult{Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectNone}
	}
	if os.Getenv("JOBS_WORKER_TEST_EFFECT_GATE") != "" && !waitJobsWorkerTestGate(ctx, os.Getenv("JOBS_WORKER_TEST_EFFECT_GATE")) {
		return jobs.HandlerResult{Outcome: jobs.OutcomeCancelled, Effect: jobs.EffectUnknown}
	}

	conn, err := pgx.Connect(ctx, os.Getenv("JOBS_WORKER_TEST_EFFECT_DSN"))
	if err != nil {
		return jobs.HandlerResult{Outcome: jobs.OutcomeUnknown, Effect: jobs.EffectUnknown}
	}
	defer conn.Close(context.Background())
	var logicalJobID string
	err = conn.QueryRow(ctx, `
INSERT INTO test_postgres_jobs_effect_ledger (effect_scope, effect_key, logical_job_id)
VALUES ($1, $2, $3)
ON CONFLICT (effect_scope, effect_key) DO UPDATE
SET logical_job_id = test_postgres_jobs_effect_ledger.logical_job_id
WHERE test_postgres_jobs_effect_ledger.logical_job_id = EXCLUDED.logical_job_id
RETURNING logical_job_id`, input.Identity.EffectScope, input.Identity.EffectKey, input.Identity.LogicalJobID).Scan(&logicalJobID)
	if err != nil || logicalJobID != string(input.Identity.LogicalJobID) {
		return jobs.HandlerResult{Outcome: jobs.OutcomeUnknown, Effect: jobs.EffectUnknown}
	}
	writeJobsWorkerTestFile(os.Getenv("JOBS_WORKER_TEST_EFFECT_FILE"), "committed\n")
	if os.Getenv("JOBS_WORKER_TEST_COMPLETE_GATE") != "" && !waitJobsWorkerTestGate(ctx, os.Getenv("JOBS_WORKER_TEST_COMPLETE_GATE")) {
		return jobs.HandlerResult{Outcome: jobs.OutcomeCancelled, Effect: jobs.EffectUnknown}
	}
	return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
}

func waitJobsWorkerTestGate(ctx context.Context, path string) bool {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}

func writeJobsWorkerTestFile(path, value string) {
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(value)
}
