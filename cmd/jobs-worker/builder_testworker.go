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
)

// buildRegistry is compiled only for the black-box process carrier. It is the
// neutral F-JOBS fixture, never a production kind or an environment switch.
var buildRegistry bootstrap.RegistryBuilder = func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
	newDefinition := func(revision jobs.Revision) (jobs.Definition[map[string]string], error) {
		return jobs.NewDefinition(jobs.DefinitionInput[map[string]string]{
			Revision:        revision,
			MaxPayloadBytes: 1024,
			Validate: func(args map[string]string) error {
				if len(args) == 0 {
					return errors.New("arguments are required")
				}
				return nil
			},
			Policy: jobs.Policy{
				Producer: jobs.ProducerPolicy{Scope: "feature-operation", RecognitionPeriod: time.Hour},
				Effect:   jobs.EffectPolicy{Authority: jobs.EffectConditionalWrite, DuplicateTolerance: "same key is harmless", LateResultPrecedence: "effect ledger wins", AmbiguousAction: jobs.AmbiguousEffectOutcomeUnknown, ReadbackAuthority: "effect ledger"},
				Retry:    jobs.RetryPolicy{MaxAttempts: 4, MaxElapsed: time.Hour, InitialBackoff: time.Second, MaxBackoff: time.Minute, HintPolicy: jobs.RetryHintPrefer, Jitter: jobs.JitterSHA256, JitterPermille: 100, MaxRecoveryWave: 8},
				Recovery: jobs.RecoveryPolicy{Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved},
				Schedule: jobs.ScheduleOneOff, MaxAttemptDuration: time.Minute, MaxAttemptCost: 1, MaxUsefulDuration: time.Hour, TerminationEnvelope: time.Minute,
				Data:     jobs.DataPolicy{Classification: "private", Redaction: "omit payload", Retention: "explicit deletion only", Deletion: "disabled", OperatorRoles: "none"},
				Operator: jobs.OperatorUnavailable, WorkClass: jobs.WorkClassNeutral,
			},
		})
	}
	definition, err := newDefinition(jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"})
	if err != nil {
		return nil, nil, err
	}
	handler := func(ctx context.Context, _ jobs.HandlerInput[map[string]string]) jobs.HandlerResult {
		switch os.Getenv("JOBS_WORKER_TEST_HANDLER") {
		case "noncooperative":
			select {}
		case "lease":
			writeJobsWorkerTestFile(os.Getenv("JOBS_WORKER_TEST_ENTERED_FILE"), "entered\n")
			<-ctx.Done()
			writeJobsWorkerTestFile(os.Getenv("JOBS_WORKER_TEST_CANCELLED_FILE"), "cancelled\n")
			select {}
		}
		<-ctx.Done()
		return jobs.HandlerResult{Outcome: jobs.OutcomeCancelled, Effect: jobs.EffectUnknown}
	}
	registry := jobs.NewRegistry()
	if err := jobs.Register(registry, definition, handler); err != nil {
		return nil, nil, err
	}
	acceptance, err := newDefinition(jobs.Revision{Kind: "acceptance", ArgsVersion: "v1", PolicyVersion: "v1"})
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
