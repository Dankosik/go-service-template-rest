//go:build integration

package integration_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPostgresHTTPIdempotencyRecordedFingerprintVersion(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-recorded-version", 4)
	v1 := httpIDAttempt(t, "v1", `{"name":"saved"}`)
	v2 := httpIDAttempt(t, "v2", `{"name":"current"}`)
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{
		"v1": v1.Fingerprint,
		"v2": v2.Fingerprint,
	})
	reservation := mustHTTPIDReserve(t, fixture, v1, resolver)
	mustHTTPIDComplete(t, fixture, reservation, resolver, httpIDResult(`{"id":"one"}`))
	assertHTTPIDDuplicateRisk(t, fixture, v1, false, 2*time.Hour)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE postgres_http_idempotency
		SET duplicate_risk_nanos = NULL
		WHERE identity_token = $1`, v1.Identity[:]); err == nil {
		t.Fatal("finite duplicate risk accepted a missing duration")
	} else if postgresError, ok := errors.AsType[*pgconn.PgError](err); !ok || postgresError.Code != pgerrcode.CheckViolation {
		t.Fatalf("finite duplicate risk constraint error = %v, want check violation", err)
	}
	if _, err := fixture.store.MaterializeEpoch(fixture.ctx, v1); err != nil {
		t.Fatalf("MaterializeEpoch(): %v", err)
	}

	_, decision, err := fixture.store.Reserve(fixture.ctx, fixture.contract, v2, resolver)
	if err != nil {
		t.Fatalf("Reserve() under recorded v1: %v", err)
	}
	if decision.Outcome != httpidempotency.OutcomeReplay || decision.Result == nil || string(decision.Result.Payload) != `{"id":"one"}` {
		t.Fatalf("recorded v1 decision = %+v, want replay of the retained result", decision)
	}

	_, decision, err = fixture.store.Reserve(fixture.ctx, fixture.contract, v2, httpIDResolver(map[string]httpidempotency.Fingerprint{
		"v2": v2.Fingerprint,
	}))
	if err != nil {
		t.Fatalf("Reserve() with removed v1 resolver: %v", err)
	}
	if decision.Outcome != httpidempotency.OutcomeUnavailable {
		t.Fatalf("removed v1 resolver outcome = %v, want unavailable", decision.Outcome)
	}

	permanent := fixture
	permanent.contract.DuplicateRisk = httpidempotency.DuplicateRiskPolicy{Permanent: true}
	permanentAttempt := httpIDAttemptWithKey(t, "v2", `{"name":"permanent"}`, "key-permanent")
	permanentResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": permanentAttempt.Fingerprint})
	permanentReservation := mustHTTPIDReserve(t, permanent, permanentAttempt, permanentResolver)
	mustHTTPIDComplete(t, permanent, permanentReservation, permanentResolver, httpIDResult(`{"id":"permanent"}`))
	assertHTTPIDDuplicateRisk(t, permanent, permanentAttempt, true, 0)
}

func TestPostgresHTTPIdempotencyFirstPublicationAndPoolHeadroom(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-publisher-a", 2)
	otherPool := newHTTPIDPool(t, fixture.ctx, httpIDDSN(t, fixture.dsn, "idempotency-authority-b"), 1)
	gatePool := newHTTPIDPool(t, fixture.ctx, httpIDDSN(t, fixture.dsn, "idempotency-gate"), 1)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE FUNCTION test_http_idempotency_publication_gate() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			IF current_setting('application_name', true) = 'idempotency-publisher-a' THEN
				PERFORM pg_advisory_xact_lock(48001);
			END IF;
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER test_http_idempotency_publication_gate
		BEFORE INSERT ON postgres_http_idempotency
		FOR EACH ROW EXECUTE FUNCTION test_http_idempotency_publication_gate();`); err != nil {
		t.Fatalf("create publication gate: %v", err)
	}

	lock, err := gatePool.Begin(fixture.ctx)
	if err != nil {
		t.Fatalf("begin advisory gate: %v", err)
	}
	t.Cleanup(func() { _ = lock.Rollback(context.Background()) })
	if _, err := lock.Exec(fixture.ctx, "SELECT pg_advisory_xact_lock(48001)"); err != nil {
		t.Fatalf("lock publication gate: %v", err)
	}

	attempt := httpIDAttempt(t, "v2", `{"name":"first"}`)
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	type reserveResult struct {
		decision httpidempotency.Decision
		err      error
	}
	results := make(chan reserveResult, 2)
	reserve := func() {
		_, decision, err := fixture.store.Reserve(fixture.ctx, fixture.contract, attempt, resolver)
		results <- reserveResult{decision: decision, err: err}
	}
	go reserve()
	waittest.Until(t, 5*time.Second, func() bool {
		return publicationBackendCount(t, fixture.ctx, otherPool, "idempotency-publisher-a") == 1
	}, "one reservation publisher to block on the application-name gate")
	go reserve()

	if count := publicationBackendCount(t, fixture.ctx, otherPool, "idempotency-publisher-a"); count != 1 {
		t.Fatalf("publication backend count = %d, want 1", count)
	}
	if _, err := otherPool.Exec(fixture.ctx, "SELECT 1"); err != nil {
		t.Fatalf("authority B writer headroom: %v", err)
	}
	if err := lock.Rollback(fixture.ctx); err != nil {
		t.Fatalf("release publication gate: %v", err)
	}

	execute := 0
	inProgress := 0
	for range 2 {
		result := waittest.Receive(t, results, 5*time.Second, "reservation publication result")
		if result.err != nil {
			t.Fatalf("Reserve(): %v", result.err)
		}
		switch result.decision.Outcome {
		case httpidempotency.OutcomeExecute:
			execute++
		case httpidempotency.OutcomeInProgress:
			inProgress++
		default:
			t.Fatalf("Reserve() outcome = %v, want execute or in-progress", result.decision.Outcome)
		}
	}
	if execute != 1 || inProgress != 1 {
		t.Fatalf("publication outcomes execute/in-progress = %d/%d, want 1/1", execute, inProgress)
	}
	var rows int
	if err := otherPool.QueryRow(fixture.ctx, "SELECT count(*) FROM postgres_http_idempotency").Scan(&rows); err != nil {
		t.Fatalf("count reservations: %v", err)
	}
	if rows != 1 {
		t.Fatalf("reservation rows = %d, want 1", rows)
	}
}

func TestPostgresHTTPIdempotencyClassificationBudget(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-classification", 2)
	gatePool := newHTTPIDPool(t, fixture.ctx, httpIDDSN(t, fixture.dsn, "idempotency-classification-gate"), 1)
	lock, err := gatePool.Begin(fixture.ctx)
	if err != nil {
		t.Fatalf("begin writer lock: %v", err)
	}
	t.Cleanup(func() { _ = lock.Rollback(context.Background()) })
	if _, err := lock.Exec(fixture.ctx, "LOCK TABLE postgres_http_idempotency IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatalf("take writer lock: %v", err)
	}

	contract := fixture.contract
	contract.InProgressWait = 100 * time.Millisecond
	attempt := httpIDAttempt(t, "v2", `{"name":"blocked"}`)
	started := time.Now()
	_, decision, err := fixture.store.Reserve(fixture.ctx, contract, attempt, httpIDResolver(map[string]httpidempotency.Fingerprint{
		"v2": attempt.Fingerprint,
	}))
	if err != nil {
		t.Fatalf("Reserve(): %v", err)
	}
	if decision.Outcome != httpidempotency.OutcomeUnavailable {
		t.Fatalf("blocked writer outcome = %v, want unavailable", decision.Outcome)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("writer classification took %s, exceeded its one in-progress budget", elapsed)
	}
	if err := lock.Rollback(fixture.ctx); err != nil {
		t.Fatalf("release writer lock: %v", err)
	}
	var rows int
	if err := fixture.pool.QueryRow(fixture.ctx, "SELECT count(*) FROM postgres_http_idempotency").Scan(&rows); err != nil {
		t.Fatalf("count blocked reservations: %v", err)
	}
	if rows != 0 {
		t.Fatalf("blocked writer created %d reservations, want none", rows)
	}
}

func TestPostgresHTTPIdempotencyOwnerRecovery(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-owner-recovery", 6)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE TABLE test_http_idempotency_feature (
			id text PRIMARY KEY
		)`); err != nil {
		t.Fatalf("create owner-recovery feature probe: %v", err)
	}
	attempt := httpIDAttempt(t, "v2", `{"name":"recover"}`)
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	first := mustHTTPIDReserve(t, fixture, attempt, resolver)
	if first.Recovery != httpidempotency.ReservationRecoveryNone {
		t.Fatalf("first reservation recovery = %v, want none", first.Recovery)
	}
	firstGeneration := httpIDGeneration(t, fixture, attempt)
	if err := fixture.store.Release(fixture.ctx, first); err != nil {
		t.Fatalf("Release() after definite rollback: %v", err)
	}
	if rows := httpIDRows(t, fixture); rows != 0 {
		t.Fatalf("rows after definite rollback = %d, want 0", rows)
	}

	_ = mustHTTPIDReserve(t, fixture, attempt, resolver)
	abandonedGeneration := httpIDGeneration(t, fixture, attempt)
	if abandonedGeneration <= firstGeneration {
		t.Fatalf("new reservation generation = %d, want above %d", abandonedGeneration, firstGeneration)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE postgres_http_idempotency
		SET recover_after = clock_timestamp() - interval '1 microsecond'
		WHERE identity_token = $1`, attempt.Identity[:]); err != nil {
		t.Fatalf("make reservation recoverable: %v", err)
	}

	firstSuccessor, decision, err := fixture.store.Reserve(fixture.ctx, fixture.contract, attempt, resolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("first successor Reserve() = (%v, %v), want execute", decision.Outcome, err)
	}
	if firstSuccessor.Recovery != httpidempotency.ReservationRecoveryDue {
		t.Fatalf("first successor recovery = %v, want due", firstSuccessor.Recovery)
	}
	secondSuccessor, decision, err := fixture.store.Reserve(fixture.ctx, fixture.contract, attempt, resolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("second successor Reserve() = (%v, %v), want execute candidate", decision.Outcome, err)
	}
	if secondSuccessor.Recovery != httpidempotency.ReservationRecoveryDue {
		t.Fatalf("second successor recovery = %v, want due", secondSuccessor.Recovery)
	}

	acquired := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseFirst := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseFirst)
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- postgres.InTx(fixture.ctx, fixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
			_, decision, err := fixture.store.Acquire(fixture.ctx, tx, fixture.contract, firstSuccessor, resolver)
			if err != nil {
				return err
			}
			if decision.Outcome != httpidempotency.OutcomeExecute {
				return fmt.Errorf("first successor acquire outcome %v", decision.Outcome)
			}
			close(acquired)
			<-release
			return nil
		})
	}()
	waittest.ReceiveSignal(t, acquired, 5*time.Second, "first successor to lock the recovered reservation")
	if err := postgres.InTx(fixture.ctx, fixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, decision, err := fixture.store.Acquire(fixture.ctx, tx, fixture.contract, secondSuccessor, resolver)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeInProgress {
			return fmt.Errorf("second successor acquire outcome %v", decision.Outcome)
		}
		return nil
	}); err != nil {
		t.Fatalf("second successor caller transaction: %v", err)
	}
	releaseFirst()
	if err := waittest.Receive(t, firstDone, 5*time.Second, "first successor transaction"); err != nil {
		t.Fatalf("first successor caller transaction: %v", err)
	}
	if generation := httpIDGeneration(t, fixture, attempt); generation <= abandonedGeneration {
		t.Fatalf("recovered generation = %d, want above %d", generation, abandonedGeneration)
	}

	childRead, childWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("create owner helper IPC pipe: %v", err)
	}
	t.Cleanup(func() { _ = childRead.Close() })
	childDSN := httpIDDSN(t, fixture.dsn, "idempotency-owner-helper")
	command := exec.Command(os.Args[0], "-test.run=^TestPostgresHTTPIdempotencyHelperProcess$")
	command.Env = append(os.Environ(),
		"HTTP_IDEMPOTENCY_HELPER=1",
		"HTTP_IDEMPOTENCY_HELPER_MODE=owner",
		"HTTP_IDEMPOTENCY_DSN="+childDSN,
		"PGTEST_POSTGRES_DSN="+childDSN,
	)
	command.ExtraFiles = []*os.File{childWrite}
	if err := command.Start(); err != nil {
		t.Fatalf("start owner helper process: %v", err)
	}
	if err := childWrite.Close(); err != nil {
		t.Fatalf("close owner helper write end: %v", err)
	}
	line := waitHTTPIDHelperLine(t, childRead, "owner helper transaction lock")
	var backendPID int
	if _, err := fmt.Sscanf(line, "owner:%d\n", &backendPID); err != nil || backendPID <= 0 {
		t.Fatalf("owner helper IPC = %q, parse PID: %v", line, err)
	}
	if !httpIDBackendExists(t, fixture, backendPID) {
		t.Fatalf("owner helper backend %d is not present before SIGKILL", backendPID)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatalf("kill owner helper process: %v", err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("owner helper unexpectedly exited cleanly after SIGKILL")
	}
	waittest.Until(t, 5*time.Second, func() bool {
		return !httpIDBackendExists(t, fixture, backendPID)
	}, "killed owner backend and row lock to disappear")
	var ownerEffects int
	if err := fixture.pool.QueryRow(fixture.ctx, "SELECT count(*) FROM test_http_idempotency_feature WHERE id = 'owner'").Scan(&ownerEffects); err != nil {
		t.Fatalf("count killed owner feature effects: %v", err)
	}
	if ownerEffects != 0 {
		t.Fatalf("killed owner committed %d feature effects, want 0", ownerEffects)
	}

	ownerAttempt := httpIDAttemptWithKey(t, "v2", `{"name":"owner"}`, "key-owner")
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE postgres_http_idempotency
		SET recover_after = clock_timestamp() - interval '1 microsecond'
		WHERE identity_token = $1`, ownerAttempt.Identity[:]); err != nil {
		t.Fatalf("make killed owner reservation recoverable: %v", err)
	}
	ownerResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": ownerAttempt.Fingerprint})
	ownerSuccessor := mustHTTPIDReserve(t, fixture, ownerAttempt, ownerResolver)
	if err := postgres.InTx(fixture.ctx, fixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, decision, err := fixture.store.Acquire(fixture.ctx, tx, fixture.contract, ownerSuccessor, ownerResolver)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeExecute {
			return fmt.Errorf("killed owner successor acquire outcome %v", decision.Outcome)
		}
		return nil
	}); err != nil {
		t.Fatalf("killed owner successor transaction: %v", err)
	}
}

func TestPostgresHTTPIdempotencyCallerTransactionAtomicity(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-caller-transaction", 4)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE TABLE test_http_idempotency_feature (
			id text PRIMARY KEY
		)`); err != nil {
		t.Fatalf("create feature probe: %v", err)
	}
	attempt := httpIDAttempt(t, "v2", `{"name":"atomic"}`)
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	reservation := mustHTTPIDReserve(t, fixture, attempt, resolver)
	if err := postgres.InTx(fixture.ctx, fixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		acquired, decision, err := fixture.store.Acquire(fixture.ctx, tx, fixture.contract, reservation, resolver)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeExecute {
			return fmt.Errorf("rollback acquire outcome %v", decision.Outcome)
		}
		if _, err := tx.Exec(fixture.ctx, "INSERT INTO test_http_idempotency_feature (id) VALUES ('rolled-back')"); err != nil {
			return err
		}
		_ = acquired
		return errors.New("feature failed before completion")
	}); err == nil {
		t.Fatal("caller transaction unexpectedly committed")
	}
	assertHTTPIDFeatureAndOutboxCounts(t, fixture, "rolled-back", 0, 0)
	if phase := httpIDPhase(t, fixture, attempt); phase != "reserved" {
		t.Fatalf("rolled-back reservation phase = %q, want reserved", phase)
	}

	outbox, err := postgresoutbox.NewStore(fixture.pool, nil)
	if err != nil {
		t.Fatalf("postgresoutbox.NewStore(): %v", err)
	}
	if err := postgres.InTx(fixture.ctx, fixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		acquired, decision, err := fixture.store.Acquire(fixture.ctx, tx, fixture.contract, reservation, resolver)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeExecute {
			return fmt.Errorf("commit acquire outcome %v", decision.Outcome)
		}
		if _, err := tx.Exec(fixture.ctx, "INSERT INTO test_http_idempotency_feature (id) VALUES ('committed')"); err != nil {
			return err
		}
		if err := outbox.Append(fixture.ctx, tx, postgresoutbox.Event{
			ID:          "idempotency-atomic-event",
			Type:        "test.created",
			Source:      "idempotency-test",
			Destination: "events",
			Schema:      "v1",
			OccurredAt:  time.Now().UTC(),
			Payload:     []byte(`{"id":"committed"}`),
			Metadata:    []byte(`{"test":true}`),
		}); err != nil {
			return err
		}
		return fixture.store.Complete(fixture.ctx, tx, fixture.contract, acquired, httpIDResult(`{"id":"committed"}`))
	}); err != nil {
		t.Fatalf("caller transaction commit: %v", err)
	}
	assertHTTPIDFeatureAndOutboxCounts(t, fixture, "committed", 1, 1)
	if phase := httpIDPhase(t, fixture, attempt); phase != "completed" {
		t.Fatalf("committed reservation phase = %q, want completed", phase)
	}
}

func TestPostgresHTTPIdempotencyReplayResultBoundAndPostCommitDeath(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-replay-bound", 6)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE TABLE test_http_idempotency_feature (
			id text PRIMARY KEY
		)`); err != nil {
		t.Fatalf("create feature probe: %v", err)
	}

	exactAttempt := httpIDAttemptWithKey(t, "v2", `{"name":"exact"}`, "key-exact")
	exactResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": exactAttempt.Fingerprint})
	exactResult := httpIDResult(`{"id":"exact"}`)
	encoded, err := httpidempotency.EncodeResult(fixture.contract, exactResult)
	if err != nil {
		t.Fatalf("encode exact result: %v", err)
	}
	exactFixture := fixture
	exactFixture.contract.ResultMaxBytes = len(encoded)
	exactReservation := mustHTTPIDReserve(t, exactFixture, exactAttempt, exactResolver)
	mustHTTPIDComplete(t, exactFixture, exactReservation, exactResolver, exactResult)
	if _, err := exactFixture.store.MaterializeEpoch(exactFixture.ctx, exactAttempt); err != nil {
		t.Fatalf("materialize exact result epoch: %v", err)
	}

	overflowAttempt := httpIDAttemptWithKey(t, "v2", `{"name":"overflow"}`, "key-overflow")
	overflowResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": overflowAttempt.Fingerprint})
	overflowFixture := fixture
	overflowFixture.contract.ResultMaxBytes = len(encoded) - 1
	overflowReservation := mustHTTPIDReserve(t, overflowFixture, overflowAttempt, overflowResolver)
	err = postgres.InTx(overflowFixture.ctx, overflowFixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		acquired, decision, err := overflowFixture.store.Acquire(overflowFixture.ctx, tx, overflowFixture.contract, overflowReservation, overflowResolver)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeExecute {
			return fmt.Errorf("overflow acquire outcome %v", decision.Outcome)
		}
		return overflowFixture.store.Complete(overflowFixture.ctx, tx, overflowFixture.contract, acquired, exactResult)
	})
	if !errors.Is(err, postgresidempotency.ErrResultTooLarge) {
		t.Fatalf("overflow completion error = %v, want result-too-large", err)
	}
	if phase := httpIDPhase(t, overflowFixture, overflowAttempt); phase != "reserved" {
		t.Fatalf("overflow reservation phase = %q, want reserved", phase)
	}
	if err := overflowFixture.store.Release(overflowFixture.ctx, overflowReservation); err != nil {
		t.Fatalf("release overflow reservation: %v", err)
	}
	if _, decision, err := overflowFixture.store.Reserve(overflowFixture.ctx, overflowFixture.contract, overflowAttempt, overflowResolver); err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("reuse after overflow = (%v, %v), want execute", decision.Outcome, err)
	}

	const epochGate = 48002
	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE FUNCTION test_http_idempotency_epoch_gate() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			IF current_setting('application_name', true) = 'idempotency-helper'
				AND OLD.committed_at IS NULL AND NEW.committed_at IS NOT NULL THEN
				PERFORM pg_advisory_xact_lock(48002);
			END IF;
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER test_http_idempotency_epoch_gate
		BEFORE UPDATE OF committed_at ON postgres_http_idempotency
		FOR EACH ROW EXECUTE FUNCTION test_http_idempotency_epoch_gate();`); err != nil {
		t.Fatalf("create epoch gate: %v", err)
	}
	gatePool := newHTTPIDPool(t, fixture.ctx, httpIDDSN(t, fixture.dsn, "idempotency-epoch-gate"), 1)
	gate, err := gatePool.Begin(fixture.ctx)
	if err != nil {
		t.Fatalf("begin epoch gate: %v", err)
	}
	t.Cleanup(func() { _ = gate.Rollback(context.Background()) })
	if _, err := gate.Exec(fixture.ctx, "SELECT pg_advisory_xact_lock(48002)"); err != nil {
		t.Fatalf("lock epoch gate: %v", err)
	}

	childRead, childWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("create helper IPC pipe: %v", err)
	}
	t.Cleanup(func() { _ = childRead.Close() })
	childDSN := httpIDDSN(t, fixture.dsn, "idempotency-helper")
	command := exec.Command(os.Args[0], "-test.run=^TestPostgresHTTPIdempotencyHelperProcess$")
	command.Env = append(os.Environ(),
		"HTTP_IDEMPOTENCY_HELPER=1",
		"HTTP_IDEMPOTENCY_DSN="+childDSN,
		"PGTEST_POSTGRES_DSN="+childDSN,
	)
	command.ExtraFiles = []*os.File{childWrite}
	if err := command.Start(); err != nil {
		t.Fatalf("start helper process: %v", err)
	}
	if err := childWrite.Close(); err != nil {
		t.Fatalf("close parent helper write end: %v", err)
	}
	if value := waitHTTPIDHelperLine(t, childRead, "helper commit"); value != "committed\n" {
		t.Fatalf("helper IPC = %q, want committed", value)
	}
	waittest.Until(t, 5*time.Second, func() bool {
		return publicationBackendCount(t, fixture.ctx, fixture.pool, "idempotency-helper") == 1
	}, "helper epoch materializer to block on its application-name gate")

	helperAttempt := httpIDAttemptWithKey(t, "v2", `{"name":"helper"}`, "key-helper")
	var expectedEpoch time.Time
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT pg_xact_commit_timestamp(xmin)
		FROM postgres_http_idempotency
		WHERE identity_token = $1`, helperAttempt.Identity[:]).Scan(&expectedEpoch); err != nil {
		t.Fatalf("read helper commit epoch before materialization: %v", err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatalf("kill helper process: %v", err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("helper process unexpectedly exited cleanly after SIGKILL")
	}
	if err := gate.Rollback(fixture.ctx); err != nil {
		t.Fatalf("release epoch gate: %v", err)
	}
	_, decision, err := fixture.store.Reserve(fixture.ctx, fixture.contract, helperAttempt, httpIDResolver(map[string]httpidempotency.Fingerprint{
		"v2": helperAttempt.Fingerprint,
	}))
	if err != nil || decision.Outcome != httpidempotency.OutcomeReplay {
		t.Fatalf("fresh replay after helper death = (%v, %v), want replay", decision.Outcome, err)
	}
	var committedAt time.Time
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT committed_at FROM postgres_http_idempotency WHERE identity_token = $1`, helperAttempt.Identity[:]).Scan(&committedAt); err != nil {
		t.Fatalf("read materialized epoch: %v", err)
	}
	if !committedAt.Equal(expectedEpoch) {
		t.Fatalf("materialized epoch = %s, want original commit epoch %s", committedAt, expectedEpoch)
	}
	assertHTTPIDFeatureAndOutboxCounts(t, fixture, "helper", 1, 1)
}

func TestPostgresHTTPIdempotencyCommitReconciliation(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-reconcile", 6)

	completed := httpIDAttemptWithKey(t, "v2", `{"name":"completed"}`, "key-completed")
	completedResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": completed.Fingerprint})
	completedReservation := mustHTTPIDReserve(t, fixture, completed, completedResolver)
	mustHTTPIDComplete(t, fixture, completedReservation, completedResolver, httpIDResult(`{"id":"completed"}`))
	_, decision, err := fixture.store.Reconcile(fixture.ctx, fixture.contract, completed, completedResolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeReplay {
		t.Fatalf("completed reconciliation = (%v, %v), want replay", decision.Outcome, err)
	}

	lockable := httpIDAttemptWithKey(t, "v2", `{"name":"lockable"}`, "key-lockable")
	lockableResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": lockable.Fingerprint})
	_ = mustHTTPIDReserve(t, fixture, lockable, lockableResolver)
	lockableReservation, decision, err := fixture.store.Reconcile(fixture.ctx, fixture.contract, lockable, lockableResolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("lockable reconciliation = (%v, %v), want execute", decision.Outcome, err)
	}
	if lockableReservation.Recovery != httpidempotency.ReservationRecoveryReconciled {
		t.Fatalf("lockable reconciliation recovery = %v, want reconciled", lockableReservation.Recovery)
	}
	if err := postgres.InTx(fixture.ctx, fixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, decision, err := fixture.store.Acquire(fixture.ctx, tx, fixture.contract, lockableReservation, lockableResolver)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeExecute {
			return fmt.Errorf("lockable successor acquire outcome %v", decision.Outcome)
		}
		return nil
	}); err != nil {
		t.Fatalf("lockable successor transaction: %v", err)
	}

	locked := httpIDAttemptWithKey(t, "v2", `{"name":"locked"}`, "key-locked")
	lockedResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": locked.Fingerprint})
	lockedReservation := mustHTTPIDReserve(t, fixture, locked, lockedResolver)
	acquired := make(chan struct{})
	release := make(chan struct{})
	lockedDone := make(chan error, 1)
	go func() {
		lockedDone <- postgres.InTx(fixture.ctx, fixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
			_, decision, err := fixture.store.Acquire(fixture.ctx, tx, fixture.contract, lockedReservation, lockedResolver)
			if err != nil {
				return err
			}
			if decision.Outcome != httpidempotency.OutcomeExecute {
				return fmt.Errorf("locked acquire outcome %v", decision.Outcome)
			}
			close(acquired)
			<-release
			return errors.New("force caller rollback")
		})
	}()
	waittest.ReceiveSignal(t, acquired, 5*time.Second, "locked caller transaction")
	_, decision, err = fixture.store.Reconcile(fixture.ctx, fixture.contract, locked, lockedResolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeUnknown {
		t.Fatalf("locked reconciliation = (%v, %v), want unknown", decision.Outcome, err)
	}
	close(release)
	if err := waittest.Receive(t, lockedDone, 5*time.Second, "locked caller rollback"); err == nil {
		t.Fatal("locked caller transaction unexpectedly committed")
	}

	absent := httpIDAttemptWithKey(t, "v2", `{"name":"absent"}`, "key-absent")
	absentResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": absent.Fingerprint})
	type reconcileResult struct {
		decision httpidempotency.Decision
		err      error
	}
	start := make(chan struct{})
	results := make(chan reconcileResult, 2)
	for range 2 {
		go func() {
			<-start
			_, decision, err := fixture.store.Reconcile(fixture.ctx, fixture.contract, absent, absentResolver)
			results <- reconcileResult{decision: decision, err: err}
		}()
	}
	close(start)
	execute := 0
	inProgress := 0
	for range 2 {
		result := waittest.Receive(t, results, 5*time.Second, "absence reconciliation")
		if result.err != nil {
			t.Fatalf("absence Reconcile(): %v", result.err)
		}
		switch result.decision.Outcome {
		case httpidempotency.OutcomeExecute:
			execute++
		case httpidempotency.OutcomeInProgress:
			inProgress++
		default:
			t.Fatalf("absence reconciliation outcome = %v, want execute or in-progress", result.decision.Outcome)
		}
	}
	if execute != 1 || inProgress != 1 {
		t.Fatalf("absence reconciliation execute/in-progress = %d/%d, want 1/1", execute, inProgress)
	}

	conflictingStored := httpIDAttemptWithKey(t, "v2", `{"name":"stored"}`, "key-conflict")
	conflictingResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": conflictingStored.Fingerprint})
	conflictingReservation := mustHTTPIDReserve(t, fixture, conflictingStored, conflictingResolver)
	mustHTTPIDComplete(t, fixture, conflictingReservation, conflictingResolver, httpIDResult(`{"id":"stored"}`))
	conflictingCurrent := httpIDAttemptWithKey(t, "v2", `{"name":"different"}`, "key-conflict")
	_, decision, err = fixture.store.Reconcile(fixture.ctx, fixture.contract, conflictingCurrent, httpIDResolver(map[string]httpidempotency.Fingerprint{
		"v2": conflictingCurrent.Fingerprint,
	}))
	if err != nil || decision.Outcome != httpidempotency.OutcomeIntegrityConflict {
		t.Fatalf("conflicting reconciliation = (%v, %v), want integrity conflict", decision.Outcome, err)
	}

	readOnlyDSN := httpIDDSNParam(t, httpIDDSN(t, fixture.dsn, "idempotency-read-only"), "default_transaction_read_only", "on")
	readOnlyPool := newHTTPIDPool(t, fixture.ctx, readOnlyDSN, 1)
	readOnlyStore, err := postgresidempotency.NewStore(readOnlyPool, httpIDStoreOptions())
	if err != nil {
		t.Fatalf("new read-only Store: %v", err)
	}
	readOnlyAttempt := httpIDAttemptWithKey(t, "v2", `{"name":"readonly"}`, "key-read-only")
	_, decision, err = readOnlyStore.Reconcile(fixture.ctx, fixture.contract, readOnlyAttempt, httpIDResolver(map[string]httpidempotency.Fingerprint{
		"v2": readOnlyAttempt.Fingerprint,
	}))
	if err != nil || decision.Outcome != httpidempotency.OutcomeUnknown {
		t.Fatalf("read-only reconciliation = (%v, %v), want unknown", decision.Outcome, err)
	}
}

func TestPostgresHTTPIdempotencyCommitEpoch(t *testing.T) {
	fixture, container := newRestartableHTTPIDFixture(t)
	attempt := httpIDAttemptWithKey(t, "v2", `{"name":"epoch"}`, "key-epoch")
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	reservation := mustHTTPIDReserve(t, fixture, attempt, resolver)
	mustHTTPIDComplete(t, fixture, reservation, resolver, httpIDResult(`{"id":"epoch"}`))

	var expected time.Time
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT pg_xact_commit_timestamp(xmin)
		FROM postgres_http_idempotency
		WHERE identity_token = $1`, attempt.Identity[:]).Scan(&expected); err != nil {
		t.Fatalf("read original commit timestamp: %v", err)
	}
	materialized, err := fixture.store.MaterializeEpoch(fixture.ctx, attempt)
	if err != nil {
		t.Fatalf("MaterializeEpoch(): %v", err)
	}
	if !materialized.Equal(expected) {
		t.Fatalf("materialized epoch = %s, want %s", materialized, expected)
	}
	var stored time.Time
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT committed_at FROM postgres_http_idempotency WHERE identity_token = $1`, attempt.Identity[:]).Scan(&stored); err != nil {
		t.Fatalf("read stored commit epoch: %v", err)
	}
	if !stored.Equal(expected) {
		t.Fatalf("stored epoch = %s, want %s", stored, expected)
	}

	lost := httpIDAttemptWithKey(t, "v2", `{"name":"lost"}`, "key-epoch-lost")
	lostResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": lost.Fingerprint})
	lostReservation := mustHTTPIDReserve(t, fixture, lost, lostResolver)
	mustHTTPIDComplete(t, fixture, lostReservation, lostResolver, httpIDResult(`{"id":"lost"}`))
	fixture.pool.Close()
	fixture.dsn = httpIDDSN(t, setRestartableCommitTimestamps(t, fixture.ctx, container, fixture.dsn, "off"), "idempotency-commit-epoch")
	fixture.pool = newHTTPIDPool(t, fixture.ctx, fixture.dsn, 4)
	fixture.store, err = postgresidempotency.NewStore(fixture.pool, httpIDStoreOptions())
	if err != nil {
		t.Fatalf("new Store after epoch loss restart: %v", err)
	}
	if _, err := fixture.store.MaterializeEpoch(fixture.ctx, lost); !errors.Is(err, postgresidempotency.ErrEpochLost) {
		t.Fatalf("lost epoch materialization error = %v, want epoch lost", err)
	}
}

func TestPostgresHTTPIdempotencyRetentionAndCleanupRaces(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-retention", 4)
	subMicrosecondContract := fixture.contract
	subMicrosecondContract.ReplayTTL = time.Nanosecond
	subMicrosecondContract.DuplicateRisk = httpidempotency.DuplicateRiskPolicy{Duration: time.Nanosecond}
	subMicrosecond := seedHTTPIDCompleted(t, fixture, subMicrosecondContract, "retention-sub-microsecond")
	if _, err := fixture.store.MaterializeEpoch(fixture.ctx, subMicrosecond); err != nil {
		t.Fatalf("materialize sub-microsecond epoch: %v", err)
	}
	assertHTTPIDCeilingHorizon(t, fixture, subMicrosecond)

	active := seedHTTPIDCompleted(t, fixture, fixture.contract, "retention-active")
	if _, err := fixture.store.MaterializeEpoch(fixture.ctx, active); err != nil {
		t.Fatalf("materialize active epoch: %v", err)
	}
	unknown := seedHTTPIDCompleted(t, fixture, fixture.contract, "retention-unknown")
	expired := seedHTTPIDCompleted(t, fixture, fixture.contract, "retention-expired")
	guard := seedHTTPIDCompleted(t, fixture, fixture.contract, "retention-guard")
	permanentContract := fixture.contract
	permanentContract.DuplicateRisk = httpidempotency.DuplicateRiskPolicy{Permanent: true}
	permanent := seedHTTPIDCompleted(t, fixture, permanentContract, "retention-permanent")
	setHTTPIDCommittedAgo(t, fixture, expired, 3*time.Hour)
	setHTTPIDCommittedAgo(t, fixture, guard, 90*time.Minute)
	setHTTPIDCommittedAgo(t, fixture, permanent, 3*time.Hour)

	options := httpIDStoreOptions()
	options.CleanupBatchSize = 1
	bounded, err := postgresidempotency.NewStore(fixture.pool, options)
	if err != nil {
		t.Fatalf("new bounded maintenance Store: %v", err)
	}
	before := httpIDMaintenanceBacklog(t, fixture)
	if before < 5 {
		t.Fatalf("maintenance backlog = %d, want materialize/expiry/guard work", before)
	}
	for before > 0 {
		if err := bounded.Maintain(fixture.ctx); err != nil {
			t.Fatalf("bounded Maintain(): %v", err)
		}
		after := httpIDMaintenanceBacklog(t, fixture)
		if after != before-1 {
			t.Fatalf("one-item maintenance changed backlog %d -> %d", before, after)
		}
		before = after
	}
	assertHTTPIDStoredState(t, fixture, active, true, true)
	assertHTTPIDStoredState(t, fixture, unknown, true, true)
	assertHTTPIDStoredState(t, fixture, permanent, true, false)
	assertHTTPIDStoredState(t, fixture, guard, true, false)
	assertHTTPIDStoredState(t, fixture, expired, false, false)

	locked := seedHTTPIDCompleted(t, fixture, fixture.contract, "retention-request-first")
	setHTTPIDCommittedAgo(t, fixture, locked, 3*time.Hour)
	lockTx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatalf("begin request-first row lock: %v", err)
	}
	if _, err := lockTx.Exec(fixture.ctx, `
		SELECT 1 FROM postgres_http_idempotency
		WHERE identity_token = $1
		FOR UPDATE`, locked.Identity[:]); err != nil {
		t.Fatalf("lock request-first row: %v", err)
	}
	maintained := make(chan error, 1)
	go func() { maintained <- bounded.Maintain(fixture.ctx) }()
	select {
	case err := <-maintained:
		if err != nil {
			t.Fatalf("maintenance while request held row: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("maintenance did not skip the request-locked row")
	}
	assertHTTPIDStoredState(t, fixture, locked, true, true)
	if err := lockTx.Rollback(fixture.ctx); err != nil {
		t.Fatalf("release request-first row lock: %v", err)
	}
	if err := bounded.Maintain(fixture.ctx); err != nil {
		t.Fatalf("expire unlocked request-first row: %v", err)
	}
	if err := bounded.Maintain(fixture.ctx); err != nil {
		t.Fatalf("delete unlocked request-first row: %v", err)
	}
	assertHTTPIDStoredState(t, fixture, locked, false, false)

	cleanupFirst := seedHTTPIDCompleted(t, fixture, fixture.contract, "retention-cleanup-first")
	setHTTPIDCommittedAgo(t, fixture, cleanupFirst, 3*time.Hour)
	if err := bounded.Maintain(fixture.ctx); err != nil {
		t.Fatalf("cleanup-first expiry: %v", err)
	}
	if err := bounded.Maintain(fixture.ctx); err != nil {
		t.Fatalf("cleanup-first guard deletion: %v", err)
	}
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": cleanupFirst.Fingerprint})
	if _, decision, err := bounded.Reserve(fixture.ctx, fixture.contract, cleanupFirst, resolver); err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("request after cleanup = (%v, %v), want execute", decision.Outcome, err)
	}
}

func TestPostgresHTTPIdempotencyMaintenanceCapacity(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-maintenance-capacity", 4)
	retained := seedHTTPIDCompleted(t, fixture, fixture.contract, "capacity-retained")
	if _, err := fixture.store.MaterializeEpoch(fixture.ctx, retained); err != nil {
		t.Fatalf("materialize retained row: %v", err)
	}
	var relationBytes int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT pg_total_relation_size('postgres_http_idempotency'::regclass)`).Scan(&relationBytes); err != nil {
		t.Fatalf("read relation size: %v", err)
	}
	options := httpIDStoreOptions()
	options.AdmissionHeadroomBytes = 1024
	options.MaxRelationBytes = relationBytes + options.AdmissionHeadroomBytes
	capacityStore, err := postgresidempotency.NewStore(fixture.pool, options)
	if err != nil {
		t.Fatalf("new capacity Store: %v", err)
	}
	if err := capacityStore.Maintain(fixture.ctx); err != nil {
		t.Fatalf("capacity Maintain(): %v", err)
	}
	if err := capacityStore.Check(fixture.ctx); !errors.Is(err, postgresidempotency.ErrUnavailable) {
		t.Fatalf("capacity readiness = %v, want unavailable", err)
	}
	retainedResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": retained.Fingerprint})
	if _, decision, err := capacityStore.Reserve(fixture.ctx, fixture.contract, retained, retainedResolver); err != nil || decision.Outcome != httpidempotency.OutcomeReplay {
		t.Fatalf("retained read under capacity closure = (%v, %v), want replay", decision.Outcome, err)
	}
	first := httpIDAttemptWithKey(t, "v2", `{"name":"capacity-first"}`, "capacity-first")
	firstResolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": first.Fingerprint})
	if _, decision, err := capacityStore.Reserve(fixture.ctx, fixture.contract, first, firstResolver); err != nil || decision.Outcome != httpidempotency.OutcomeUnavailable {
		t.Fatalf("new first execution under capacity closure = (%v, %v), want unavailable", decision.Outcome, err)
	}
	if rows := httpIDRows(t, fixture); rows != 1 {
		t.Fatalf("capacity closure rows = %d, want retained row only", rows)
	}

	gate, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatalf("begin maintenance relation gate: %v", err)
	}
	if _, err := gate.Exec(fixture.ctx, "LOCK TABLE postgres_http_idempotency IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatalf("lock maintenance relation: %v", err)
	}
	firstCycle := make(chan error, 1)
	go func() { firstCycle <- fixture.store.Maintain(fixture.ctx) }()
	waittest.Until(t, 5*time.Second, func() bool {
		var blocked int
		err := fixture.pool.QueryRow(fixture.ctx, `
			SELECT count(*)
			FROM pg_stat_activity
			WHERE datname = current_database()
			  AND application_name = 'idempotency-maintenance-capacity'
			  AND wait_event_type = 'Lock'
			  AND query LIKE '%WITH candidates AS%'`).Scan(&blocked)
		return err == nil && blocked == 1
	}, "one maintenance cycle to block on the relation")
	if err := fixture.store.Maintain(fixture.ctx); !errors.Is(err, postgresidempotency.ErrUnavailable) {
		t.Fatalf("overlapping Maintain() error = %v, want unavailable", err)
	}
	if err := gate.Rollback(fixture.ctx); err != nil {
		t.Fatalf("release maintenance relation gate: %v", err)
	}
	if err := <-firstCycle; err != nil {
		t.Fatalf("first maintenance cycle after gate release: %v", err)
	}
}

func TestPostgresHTTPIdempotencyTelemetry(t *testing.T) {
	reader := telemetrytest.InstallManualReader(t)
	fixture := newHTTPIDFixture(t, "idempotency-telemetry", 4)
	attempt := httpIDAttemptWithKey(t, "v2", `{"name":"telemetry"}`, "key-telemetry")
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	reservation := mustHTTPIDReserve(t, fixture, attempt, resolver)
	mustHTTPIDComplete(t, fixture, reservation, resolver, httpIDResult(`{"id":"telemetry"}`))
	if _, err := fixture.store.MaterializeEpoch(fixture.ctx, attempt); err != nil {
		t.Fatalf("MaterializeEpoch(): %v", err)
	}
	fixture.store.ObserveTerminal(fixture.ctx, httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil)
	if _, decision, err := fixture.store.Reserve(fixture.ctx, fixture.contract, attempt, resolver); err != nil || decision.Outcome != httpidempotency.OutcomeReplay {
		t.Fatalf("telemetry replay = (%v, %v), want replay", decision.Outcome, err)
	} else {
		fixture.store.ObserveTerminal(fixture.ctx, decision, nil)
	}
	if err := fixture.store.Maintain(fixture.ctx); err != nil {
		t.Fatalf("telemetry Maintain(): %v", err)
	}

	var wantRows, wantResultBytes, wantRelationBytes int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*), coalesce(sum(octet_length(result)), 0),
		       pg_total_relation_size('postgres_http_idempotency'::regclass)
		FROM postgres_http_idempotency`).Scan(&wantRows, &wantResultBytes, &wantRelationBytes); err != nil {
		t.Fatalf("read telemetry SQL oracle: %v", err)
	}
	wantGauge := map[string]int64{
		"http.idempotency.rows":           wantRows,
		"http.idempotency.relation.bytes": wantRelationBytes,
		"http.idempotency.result.bytes":   wantResultBytes,
	}
	found := make(map[string]bool, len(wantGauge))
	terminalCount := int64(0)
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if want, ok := wantGauge[measured.Name]; ok {
			got := telemetrytest.SinglePoint(t, measured.Name, telemetrytest.Int64Gauge(t, measured).DataPoints)
			if got != want {
				t.Fatalf("%s = %d, want SQL value %d", measured.Name, got, want)
			}
			found[measured.Name] = true
		}
		if measured.Name == "http.idempotency.requests" {
			for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
				terminalCount += point.Value
			}
		}
	})
	for name := range wantGauge {
		if !found[name] {
			t.Errorf("metric %s was not collected", name)
		}
	}
	if terminalCount != 2 {
		t.Fatalf("terminal request outcomes = %d, want one for execute and one for replay", terminalCount)
	}
	telemetrytest.AssertNoAttributeContains(t, reader, "key-telemetry", "telemetry")
}

func seedHTTPIDCompleted(
	t *testing.T,
	fixture httpIDFixture,
	contract httpidempotency.Contract,
	key string,
) httpidempotency.Attempt {
	t.Helper()
	seed := fixture
	seed.contract = contract
	attempt := httpIDAttemptWithKey(t, "v2", `{"name":"`+key+`"}`, key)
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	reservation := mustHTTPIDReserve(t, seed, attempt, resolver)
	mustHTTPIDComplete(t, seed, reservation, resolver, httpIDResult(`{"id":"`+key+`"}`))
	return attempt
}

func setHTTPIDCommittedAgo(t *testing.T, fixture httpIDFixture, attempt httpidempotency.Attempt, ago time.Duration) {
	t.Helper()
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE postgres_http_idempotency
		SET committed_at = clock_timestamp() - $2::bigint * interval '1 microsecond'
		WHERE identity_token = $1`, attempt.Identity[:], ago.Microseconds()); err != nil {
		t.Fatalf("set writer-clock commit horizon: %v", err)
	}
}

func httpIDMaintenanceBacklog(t *testing.T, fixture httpIDFixture) int {
	t.Helper()
	var backlog int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT
			count(*) FILTER (WHERE phase = 'completed' AND committed_at IS NULL)
			+ count(*) FILTER (
				WHERE phase = 'completed'
				  AND committed_at IS NOT NULL
				  AND octet_length(result) > 0
			  AND committed_at + ((replay_nanos - 1) / 1000 + 1) * interval '1 microsecond' <= clock_timestamp()
			)
			+ count(*) FILTER (
				WHERE phase = 'completed'
				  AND committed_at IS NOT NULL
				  AND duplicate_risk_permanent = false
			  AND committed_at + ((duplicate_risk_nanos - 1) / 1000 + 1) * interval '1 microsecond' <= clock_timestamp()
			)
		FROM postgres_http_idempotency`).Scan(&backlog); err != nil {
		t.Fatalf("read maintenance backlog: %v", err)
	}
	return backlog
}

func assertHTTPIDCeilingHorizon(t *testing.T, fixture httpIDFixture, attempt httpidempotency.Attempt) {
	t.Helper()
	var committedAt, replayExpiry, guardExpiry time.Time
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT
			committed_at,
			committed_at + ((replay_nanos - 1) / 1000 + 1) * interval '1 microsecond',
			committed_at + ((duplicate_risk_nanos - 1) / 1000 + 1) * interval '1 microsecond'
		FROM postgres_http_idempotency
		WHERE identity_token = $1`, attempt.Identity[:]).Scan(&committedAt, &replayExpiry, &guardExpiry); err != nil {
		t.Fatalf("read ceiling-normalized horizons: %v", err)
	}
	want := committedAt.Add(time.Microsecond)
	if !replayExpiry.Equal(want) || !guardExpiry.Equal(want) {
		t.Fatalf("sub-microsecond horizons = (%s, %s), want both %s", replayExpiry, guardExpiry, want)
	}
}

func assertHTTPIDStoredState(
	t *testing.T,
	fixture httpIDFixture,
	attempt httpidempotency.Attempt,
	wantExists, wantResult bool,
) {
	t.Helper()
	var rows int
	var hasResult bool
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*), coalesce(bool_or(octet_length(result) > 0), false)
		FROM postgres_http_idempotency
		WHERE identity_token = $1`, attempt.Identity[:]).Scan(&rows, &hasResult); err != nil {
		t.Fatalf("read retained state: %v", err)
	}
	if (rows == 1) != wantExists || hasResult != wantResult {
		t.Fatalf("stored state = (exists %t, result %t), want (%t, %t)", rows == 1, hasResult, wantExists, wantResult)
	}
}

func TestPostgresHTTPIdempotencyWriterAuthority(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-writer-authority", 4)
	readOnlyPool := newHTTPIDPool(t, fixture.ctx, httpIDDSNParam(
		t,
		httpIDDSN(t, fixture.dsn, "idempotency-read-only-authority"),
		"default_transaction_read_only",
		"on",
	), 1)
	readOnlyStore, err := postgresidempotency.NewStore(readOnlyPool, httpIDStoreOptions())
	if err != nil {
		t.Fatalf("new read-only Store: %v", err)
	}
	attempt := httpIDAttemptWithKey(t, "v2", `{"name":"read-only"}`, "key-read-only-authority")
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	if _, decision, err := readOnlyStore.Reserve(fixture.ctx, fixture.contract, attempt, resolver); err != nil || decision.Outcome != httpidempotency.OutcomeUnavailable {
		t.Fatalf("read-only reserve = (%v, %v), want unavailable", decision.Outcome, err)
	}
	if rows := httpIDRows(t, fixture); rows != 0 {
		t.Fatalf("read-only authority created %d rows, want 0", rows)
	}
	if _, decision, err := readOnlyStore.Reconcile(fixture.ctx, fixture.contract, attempt, resolver); err != nil || decision.Outcome != httpidempotency.OutcomeUnknown {
		t.Fatalf("read-only reconcile = (%v, %v), want unknown", decision.Outcome, err)
	}
}

func TestPostgresHTTPIdempotencyHelperProcess(t *testing.T) {
	if os.Getenv("HTTP_IDEMPOTENCY_HELPER") != "1" {
		return
	}
	if os.Getenv("HTTP_IDEMPOTENCY_HELPER_MODE") == "owner" {
		runHTTPIDOwnerHelper(t)
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	pool := newHTTPIDPool(t, ctx, os.Getenv("HTTP_IDEMPOTENCY_DSN"), 2)
	store, err := postgresidempotency.NewStore(pool, httpIDStoreOptions())
	if err != nil {
		t.Fatalf("postgresidempotency.NewStore(): %v", err)
	}
	if err := store.Maintain(ctx); err != nil {
		t.Fatalf("initial idempotency maintenance: %v", err)
	}
	contract := httpIDContract()
	attempt := httpIDAttemptWithKey(t, "v2", `{"name":"helper"}`, "key-helper")
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	reservation, decision, err := store.Reserve(ctx, contract, attempt, resolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("helper Reserve() = (%v, %v), want execute", decision.Outcome, err)
	}
	outbox, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("postgresoutbox.NewStore(): %v", err)
	}
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		acquired, decision, err := store.Acquire(ctx, tx, contract, reservation, resolver)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeExecute {
			return fmt.Errorf("helper acquire outcome %v", decision.Outcome)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_feature (id) VALUES ('helper')"); err != nil {
			return err
		}
		if err := outbox.Append(ctx, tx, postgresoutbox.Event{
			ID:          "idempotency-atomic-event",
			Type:        "test.created",
			Source:      "idempotency-test",
			Destination: "events",
			Schema:      "v1",
			OccurredAt:  time.Now().UTC(),
			Payload:     []byte(`{"id":"helper"}`),
			Metadata:    []byte(`{"test":true}`),
		}); err != nil {
			return err
		}
		return store.Complete(ctx, tx, contract, acquired, httpIDResult(`{"id":"helper"}`))
	}); err != nil {
		t.Fatalf("helper transaction: %v", err)
	}
	pipe := os.NewFile(uintptr(3), "idempotency-helper-ipc")
	if pipe == nil {
		t.Fatal("open helper IPC")
	}
	defer pipe.Close()
	if _, err := fmt.Fprintln(pipe, "committed"); err != nil {
		t.Fatalf("write helper IPC: %v", err)
	}
	if _, err := store.MaterializeEpoch(ctx, attempt); err != nil {
		t.Fatalf("helper materialize epoch: %v", err)
	}
}

func runHTTPIDOwnerHelper(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	pool := newHTTPIDPool(t, ctx, os.Getenv("HTTP_IDEMPOTENCY_DSN"), 2)
	store, err := postgresidempotency.NewStore(pool, httpIDStoreOptions())
	if err != nil {
		t.Fatalf("postgresidempotency.NewStore(): %v", err)
	}
	if err := store.Maintain(ctx); err != nil {
		t.Fatalf("initial idempotency maintenance: %v", err)
	}
	contract := httpIDContract()
	attempt := httpIDAttemptWithKey(t, "v2", `{"name":"owner"}`, "key-owner")
	resolver := httpIDResolver(map[string]httpidempotency.Fingerprint{"v2": attempt.Fingerprint})
	reservation, decision, err := store.Reserve(ctx, contract, attempt, resolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("owner helper Reserve() = (%v, %v), want execute", decision.Outcome, err)
	}
	pipe := os.NewFile(uintptr(3), "idempotency-owner-ipc")
	if pipe == nil {
		t.Fatal("open owner helper IPC")
	}
	defer pipe.Close()
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, decision, err := store.Acquire(ctx, tx, contract, reservation, resolver)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeExecute {
			return fmt.Errorf("owner helper acquire outcome %v", decision.Outcome)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_feature (id) VALUES ('owner')"); err != nil {
			return err
		}
		var backendPID int
		if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&backendPID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(pipe, "owner:%d\n", backendPID); err != nil {
			return err
		}
		select {}
	}); err != nil {
		t.Fatalf("owner helper transaction: %v", err)
	}
}

func waitHTTPIDHelperLine(t *testing.T, reader *os.File, description string) string {
	t.Helper()
	values := make(chan string, 1)
	errors := make(chan error, 1)
	go func() {
		value, err := bufio.NewReader(reader).ReadString('\n')
		if err != nil {
			errors <- err
			return
		}
		values <- value
	}()
	select {
	case value := <-values:
		return value
	case err := <-errors:
		t.Fatalf("read %s IPC: %v", description, err)
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s IPC", description)
	}
	return ""
}

func httpIDBackendExists(t *testing.T, fixture httpIDFixture, backendPID int) bool {
	t.Helper()
	var exists bool
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = $1)`, backendPID).Scan(&exists); err != nil {
		t.Fatalf("inspect backend %d: %v", backendPID, err)
	}
	return exists
}

func httpIDGeneration(t *testing.T, fixture httpIDFixture, attempt httpidempotency.Attempt) int64 {
	t.Helper()
	var generation int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT generation FROM postgres_http_idempotency WHERE identity_token = $1`, attempt.Identity[:]).Scan(&generation); err != nil {
		t.Fatalf("read reservation generation: %v", err)
	}
	return generation
}

func httpIDRows(t *testing.T, fixture httpIDFixture) int {
	t.Helper()
	var rows int
	if err := fixture.pool.QueryRow(fixture.ctx, "SELECT count(*) FROM postgres_http_idempotency").Scan(&rows); err != nil {
		t.Fatalf("count idempotency rows: %v", err)
	}
	return rows
}

func httpIDPhase(t *testing.T, fixture httpIDFixture, attempt httpidempotency.Attempt) string {
	t.Helper()
	var phase string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT phase FROM postgres_http_idempotency WHERE identity_token = $1`, attempt.Identity[:]).Scan(&phase); err != nil {
		t.Fatalf("read idempotency phase: %v", err)
	}
	return phase
}

func assertHTTPIDDuplicateRisk(
	t *testing.T,
	fixture httpIDFixture,
	attempt httpidempotency.Attempt,
	wantPermanent bool,
	wantDuration time.Duration,
) {
	t.Helper()
	var duration pgtype.Int8
	var permanent bool
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT duplicate_risk_nanos, duplicate_risk_permanent
		FROM postgres_http_idempotency
		WHERE identity_token = $1`, attempt.Identity[:]).Scan(&duration, &permanent); err != nil {
		t.Fatalf("read duplicate-risk policy: %v", err)
	}
	if permanent != wantPermanent {
		t.Fatalf("duplicate-risk permanent = %t, want %t", permanent, wantPermanent)
	}
	if wantPermanent {
		if duration.Valid {
			t.Fatalf("permanent duplicate risk retained %d nanos", duration.Int64)
		}
		return
	}
	if !duration.Valid || time.Duration(duration.Int64) != wantDuration {
		t.Fatalf("duplicate-risk duration = %d/%t, want %s", duration.Int64, duration.Valid, wantDuration)
	}
}

func assertHTTPIDFeatureAndOutboxCounts(t *testing.T, fixture httpIDFixture, id string, wantFeature, wantOutbox int) {
	t.Helper()
	var feature, outbox int
	if err := fixture.pool.QueryRow(fixture.ctx, "SELECT count(*) FROM test_http_idempotency_feature WHERE id = $1", id).Scan(&feature); err != nil {
		t.Fatalf("count feature rows: %v", err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, "SELECT count(*) FROM outbox_events WHERE id = $1", "idempotency-atomic-event").Scan(&outbox); err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if feature != wantFeature || outbox != wantOutbox {
		t.Fatalf("feature/outbox counts = %d/%d, want %d/%d", feature, outbox, wantFeature, wantOutbox)
	}
}

func publicationBackendCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, applicationName string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM pg_stat_activity
		WHERE application_name = $1
		  AND pid <> pg_backend_pid()
		  AND backend_type = 'client backend'`, applicationName).Scan(&count); err != nil {
		t.Fatalf("count publication backends: %v", err)
	}
	return count
}
