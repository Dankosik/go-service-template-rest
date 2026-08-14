//go:build integration

package integration_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
)

var postgresJobsSchemaTables = []string{
	"postgres_job_actions",
	"postgres_job_attempts",
	"postgres_job_claim_scopes",
	"postgres_jobs",
}

type postgresJobsSchemaMutation struct {
	name      string
	authority string
	sql       string
}

func TestPostgresJobsSchema(t *testing.T) {
	t.Run("canonical read-only authority", func(t *testing.T) {
		ctx, _, store := newPostgresJobsFixture(t)
		if err := store.CheckSchema(ctx); err != nil {
			t.Fatalf("CheckSchema() error = %v", err)
		}
	})

	for _, state := range []struct {
		name string
		sql  string
	}{
		{name: "paused", sql: "UPDATE postgres_job_claim_scopes SET paused = true, scope_generation = 1 WHERE work_class = 'neutral'"},
		{name: "resumed", sql: "UPDATE postgres_job_claim_scopes SET scope_generation = 2 WHERE work_class = 'neutral'"},
	} {
		t.Run("accepts mutable neutral scope "+state.name, func(t *testing.T) {
			ctx, pool, store := newPostgresJobsFixture(t)
			if _, err := pool.PGX().Exec(ctx, state.sql); err != nil {
				t.Fatalf("change neutral scope state: %v", err)
			}
			if err := store.CheckSchema(ctx); err != nil {
				t.Fatalf("CheckSchema() error = %v", err)
			}
		})
	}

	for _, mutation := range postgresJobsSchemaMutations(t) {
		t.Run("rejects "+mutation.name, func(t *testing.T) {
			ctx, pool, store := newPostgresJobsFixture(t)
			if _, err := pool.PGX().Exec(ctx, mutation.sql); err != nil {
				t.Fatalf("apply schema mutation: %v", err)
			}
			if err := store.CheckSchema(ctx); !errors.Is(err, postgresjobs.ErrSchemaIncompatible) || !strings.Contains(err.Error(), mutation.authority) {
				t.Fatalf("CheckSchema() error = %v, want ErrSchemaIncompatible", err)
			}
		})
	}

	t.Run("Go and database vocabulary are bijective", func(t *testing.T) {
		ctx, pool, _ := newPostgresJobsFixture(t)
		states := []jobs.State{
			jobs.StateReady, jobs.StateScheduled, jobs.StateRetryWait, jobs.StateRunning,
			jobs.StateCancelRequested, jobs.StateSucceeded, jobs.StateCancelled,
			jobs.StateExhausted, jobs.StatePermanent, jobs.StatePoison, jobs.StateOutcomeUnknown,
		}
		for index, state := range states {
			id := fmt.Sprintf("state-%d", index)
			owned := state == jobs.StateRunning || state == jobs.StateCancelRequested
			terminal := state == jobs.StateSucceeded || state == jobs.StateCancelled || state == jobs.StateExhausted ||
				state == jobs.StatePermanent || state == jobs.StatePoison || state == jobs.StateOutcomeUnknown
			if _, err := pool.PGX().Exec(ctx, `
				INSERT INTO postgres_jobs (
					logical_job_id, producer_scope, producer_key, occurrence_scope, occurrence_id,
					effect_scope, effect_key, intent_fingerprint, kind, args_version, policy_version,
					payload, work_class, state, available_at, current_worker_id, lease_expires_at, terminal_at
				) VALUES ($1, 'producer', $1, 'occurrence', $1, 'effect', $1, decode(repeat('00', 32), 'hex'),
					'email', 'v1', 'v1', '{}'::bytea, 'neutral', $2, clock_timestamp(),
					CASE WHEN $3 THEN 'worker' END, CASE WHEN $3 THEN clock_timestamp() + interval '1 minute' END,
					CASE WHEN $4 THEN clock_timestamp() END)`, id, state, owned, terminal); err != nil {
				t.Fatalf("insert state %q: %v", state, err)
			}
		}
		if _, err := pool.PGX().Exec(ctx, `
			INSERT INTO postgres_jobs (
				logical_job_id, producer_scope, producer_key, occurrence_scope, occurrence_id,
				effect_scope, effect_key, intent_fingerprint, kind, args_version, policy_version,
				payload, work_class, state, available_at
			) VALUES ('unknown-state', 'producer', 'unknown-state', 'occurrence', 'unknown-state',
				'effect', 'unknown-state', decode(repeat('00', 32), 'hex'), 'email', 'v1', 'v1',
				'{}'::bytea, 'neutral', 'future', clock_timestamp())`); err == nil {
			t.Fatal("database accepted unknown job state")
		}

		if _, err := pool.PGX().Exec(ctx, `
			INSERT INTO postgres_jobs (
				logical_job_id, producer_scope, producer_key, occurrence_scope, occurrence_id,
				effect_scope, effect_key, intent_fingerprint, kind, args_version, policy_version,
				payload, work_class, state, available_at, current_worker_id, lease_expires_at
			) VALUES ('attempt-job', 'producer', 'attempt-job', 'occurrence', 'attempt-job',
				'effect', 'attempt-job', decode(repeat('00', 32), 'hex'), 'email', 'v1', 'v1',
				'{}'::bytea, 'neutral', 'running', clock_timestamp(), 'worker', clock_timestamp() + interval '1 minute')`); err != nil {
			t.Fatalf("insert attempt parent: %v", err)
		}
		outcomes := []jobs.OutcomeClass{
			jobs.OutcomeSuccess, jobs.OutcomeRetryable, jobs.OutcomePermanent, jobs.OutcomePoison,
			jobs.OutcomeTimeout, jobs.OutcomeCancelled, jobs.OutcomePanic, jobs.OutcomeLost, jobs.OutcomeUnknown,
		}
		effects := []jobs.EffectStatus{jobs.EffectNone, jobs.EffectCompleted, jobs.EffectPartial, jobs.EffectUnknown}
		generation := int64(0)
		for _, outcome := range outcomes {
			for _, effect := range effects {
				generation++
				if _, err := pool.PGX().Exec(ctx, `
					INSERT INTO postgres_job_attempts (
						logical_job_id, attempt_generation, recovery_generation, attempt_number, worker_id,
						lease_expires_at, finalized_at, final_state, outcome, effect_status,
						attempts_used, elapsed_used_milliseconds
					) VALUES ('attempt-job', $1::bigint, 0, ($1::bigint)::integer, 'worker', clock_timestamp() + interval '1 minute',
						clock_timestamp(), 'succeeded', $2, $3, ($1::bigint)::integer, 0)`, generation, outcome, effect); err != nil {
					t.Fatalf("insert outcome/effect %q/%q: %v", outcome, effect, err)
				}
			}
		}
		if _, err := pool.PGX().Exec(ctx, `
			INSERT INTO postgres_job_attempts (
				logical_job_id, attempt_generation, recovery_generation, attempt_number, worker_id,
				lease_expires_at, finalized_at, final_state, outcome, effect_status,
				attempts_used, elapsed_used_milliseconds
			) VALUES ('attempt-job', 100, 0, 100, 'worker', clock_timestamp() + interval '1 minute',
				clock_timestamp(), 'succeeded', 'future', 'none', 100, 0)`); err == nil {
			t.Fatal("database accepted unknown attempt outcome")
		}
		if _, err := pool.PGX().Exec(ctx, `
			INSERT INTO postgres_job_attempts (
				logical_job_id, attempt_generation, recovery_generation, attempt_number, worker_id,
				lease_expires_at, finalized_at, final_state, outcome, effect_status,
				attempts_used, elapsed_used_milliseconds
			) VALUES ('attempt-job', 101, 0, 101, 'worker', clock_timestamp() + interval '1 minute',
				clock_timestamp(), 'succeeded', 'success', 'future', 101, 0)`); err == nil {
			t.Fatal("database accepted unknown attempt effect status")
		}
	})
}

func postgresJobsSchemaMutations(t *testing.T) []postgresJobsSchemaMutation {
	t.Helper()
	ctx, pool, _ := newPostgresJobsFixture(t)
	mutations := make([]postgresJobsSchemaMutation, 0)
	for _, table := range postgresJobsSchemaTables {
		mutations = append(mutations, postgresJobsSchemaMutation{
			name:      "relation " + table,
			authority: table,
			sql:       "DROP TABLE " + pgx.Identifier{table}.Sanitize() + " CASCADE",
		})
	}

	for _, catalog := range []struct {
		name  string
		query string
		drop  func(string, string) string
	}{
		{
			name: "column",
			query: `SELECT table_name, column_name
				FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = ANY($1::text[])
				ORDER BY table_name, ordinal_position`,
			drop: func(table, name string) string {
				return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s CASCADE", pgx.Identifier{table}.Sanitize(), pgx.Identifier{name}.Sanitize())
			},
		},
		{
			name: "constraint",
			query: `SELECT relation.relname, constraint_row.conname
				FROM pg_constraint AS constraint_row
				JOIN pg_class AS relation ON relation.oid = constraint_row.conrelid
				JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
				WHERE namespace.nspname = 'public' AND relation.relname = ANY($1::text[])
				ORDER BY relation.relname, constraint_row.conname`,
			drop: func(table, name string) string {
				return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s CASCADE", pgx.Identifier{table}.Sanitize(), pgx.Identifier{name}.Sanitize())
			},
		},
		{
			name: "index",
			query: `SELECT relation.relname, index_relation.relname
				FROM pg_index AS index_row
				JOIN pg_class AS relation ON relation.oid = index_row.indrelid
				JOIN pg_class AS index_relation ON index_relation.oid = index_row.indexrelid
				JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
				LEFT JOIN pg_constraint AS constraint_row ON constraint_row.conindid = index_row.indexrelid
				WHERE namespace.nspname = 'public'
					AND relation.relname = ANY($1::text[])
					AND constraint_row.oid IS NULL
				ORDER BY relation.relname, index_relation.relname`,
			drop: func(_, name string) string {
				return "DROP INDEX " + pgx.Identifier{name}.Sanitize()
			},
		},
	} {
		rows, err := pool.PGX().Query(ctx, catalog.query, postgresJobsSchemaTables)
		if err != nil {
			t.Fatalf("list %s authority: %v", catalog.name, err)
		}
		values, err := pgx.CollectRows(rows, pgx.RowToStructByPos[struct {
			Table string
			Name  string
		}])
		if err != nil {
			t.Fatalf("collect %s authority: %v", catalog.name, err)
		}
		for _, value := range values {
			authority := value.Table + "." + value.Name
			mutations = append(mutations, postgresJobsSchemaMutation{
				name:      catalog.name + " " + authority,
				authority: authority,
				sql:       catalog.drop(value.Table, value.Name),
			})
		}
	}

	return append(mutations,
		postgresJobsSchemaMutation{
			name:      "column default postgres_job_actions.completed_at",
			authority: "postgres_job_actions.completed_at",
			sql:       "ALTER TABLE postgres_job_actions ALTER COLUMN completed_at SET DEFAULT statement_timestamp()",
		},
		postgresJobsSchemaMutation{
			name:      "column collation postgres_job_actions.reason",
			authority: "postgres_job_actions.reason",
			sql:       `ALTER TABLE postgres_job_actions ALTER COLUMN reason TYPE text COLLATE "POSIX"`,
		},
		postgresJobsSchemaMutation{
			name:      "constraint definition postgres_job_actions_result_check",
			authority: "postgres_job_actions.postgres_job_actions_result_check",
			sql:       "ALTER TABLE postgres_job_actions DROP CONSTRAINT postgres_job_actions_result_check, ADD CONSTRAINT postgres_job_actions_result_check CHECK (result IS NOT NULL)",
		},
		postgresJobsSchemaMutation{
			name:      "index definition postgres_job_actions_job_idx",
			authority: "postgres_job_actions.postgres_job_actions_job_idx",
			sql:       "DROP INDEX postgres_job_actions_job_idx; CREATE INDEX postgres_job_actions_job_idx ON postgres_job_actions (logical_job_id) WHERE logical_job_id IS NOT NULL",
		},
		postgresJobsSchemaMutation{
			name:      "neutral scope",
			authority: "neutral claim scope",
			sql:       "DELETE FROM postgres_job_claim_scopes WHERE work_class = 'neutral'",
		},
	)
}
