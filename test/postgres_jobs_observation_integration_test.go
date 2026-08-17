//go:build integration

package integration_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestPostgresJobsObservation(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newPostgresJobsFixture(t)
	states := []jobs.State{
		jobs.StateReady, jobs.StateScheduled, jobs.StateRetryWait, jobs.StateRunning,
		jobs.StateCancelRequested, jobs.StateSucceeded, jobs.StateCancelled,
		jobs.StateExhausted, jobs.StatePermanent, jobs.StatePoison, jobs.StateOutcomeUnknown,
	}
	for index, state := range states {
		prepared := stageDuePostgresJob(ctx, t, pool, store, "observe-"+string(state))
		terminal := state == jobs.StateSucceeded || state == jobs.StateCancelled || state == jobs.StateExhausted ||
			state == jobs.StatePermanent || state == jobs.StatePoison || state == jobs.StateOutcomeUnknown
		running := state == jobs.StateRunning || state == jobs.StateCancelRequested
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs
			SET state = $2,
				available_at = clock_timestamp() - $3::integer * interval '1 second',
				current_worker_id = CASE WHEN $4::boolean THEN 'worker-observe' ELSE NULL END,
				lease_expires_at = CASE WHEN $4::boolean THEN clock_timestamp() + interval '1 minute' ELSE NULL END,
				terminal_at = CASE WHEN $5::boolean THEN clock_timestamp() ELSE NULL END
			WHERE logical_job_id = $1`, string(prepared.Identity().LogicalJobID), string(state), index+1, running, terminal); err != nil {
			t.Fatalf("seed observation state %q: %v", state, err)
		}
	}
	unknown := stageDuePostgresJob(ctx, t, pool, store, "observe-unknown-revision")
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET kind = 'unknown' WHERE logical_job_id = $1`, string(unknown.Identity().LogicalJobID)); err != nil {
		t.Fatalf("seed unknown required revision: %v", err)
	}
	terminalUnknown := stageDuePostgresJob(ctx, t, pool, store, "observe-terminal-unknown-revision")
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET kind = 'terminal-unknown', state = 'permanent', terminal_at = clock_timestamp() WHERE logical_job_id = $1`, string(terminalUnknown.Identity().LogicalJobID)); err != nil {
		t.Fatalf("seed terminal historical revision: %v", err)
	}

	session := acquirePostgresJobsSession(ctx, t, store)
	defer session.Release(ctx)
	registry := []jobs.Revision{{Kind: "acceptance", ArgsVersion: "v1", PolicyVersion: "v1"}}
	observation, err := session.Observe(ctx, registry)
	if err != nil {
		t.Fatalf("Observe(): %v", err)
	}
	if observation.Compatible || observation.ObservedAt.IsZero() {
		t.Fatalf("observation compatibility/freshness = %t/%s, want false and writer instant", observation.Compatible, observation.ObservedAt)
	}
	wantRevisions := []jobs.Revision{
		{Kind: "acceptance", ArgsVersion: "v1", PolicyVersion: "v1"},
		{Kind: "unknown", ArgsVersion: "v1", PolicyVersion: "v1"},
	}
	if !slices.Equal(observation.RequiredRevisions, wantRevisions) {
		t.Fatalf("required revisions = %+v, want %+v", observation.RequiredRevisions, wantRevisions)
	}

	type stateRow struct {
		state  string
		count  int64
		oldest time.Time
	}
	rows, err := pool.PGX().Query(ctx, `SELECT state, count(*), min(available_at) FROM postgres_jobs GROUP BY state ORDER BY state`)
	if err != nil {
		t.Fatalf("read independent state inventory: %v", err)
	}
	defer rows.Close()
	var independent []stateRow
	for rows.Next() {
		var row stateRow
		if err := rows.Scan(&row.state, &row.count, &row.oldest); err != nil {
			t.Fatalf("scan independent state inventory: %v", err)
		}
		independent = append(independent, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate independent state inventory: %v", err)
	}
	if len(observation.States) != len(independent) {
		t.Fatalf("state observations = %+v, want %d independent rows", observation.States, len(independent))
	}
	for index, got := range observation.States {
		want := independent[index]
		if string(got.State) != want.state || got.Count != uint64(want.count) || !got.OldestAvailableAt.Equal(want.oldest) {
			t.Fatalf("state observation[%d] = %+v, want %+v", index, got, want)
		}
	}

	var terminated bool
	if err := pool.PGX().QueryRow(ctx, `SELECT pg_terminate_backend($1)`, session.BackendPID()).Scan(&terminated); err != nil || !terminated {
		t.Fatalf("terminate observation Session = %t, %v", terminated, err)
	}
	failed, err := session.Observe(ctx, registry)
	if !errors.Is(err, postgresjobs.ErrSessionTerminal) || !failed.ObservedAt.IsZero() {
		t.Fatalf("Observe(after database loss) = %+v, %v; want no fresh replacement and ErrSessionTerminal", failed, err)
	}
}
