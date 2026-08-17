package postgresjobs

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
)

var postgresJobsTables = []string{
	"postgres_job_attempts",
	"postgres_job_claim_scopes",
	"postgres_jobs",
}

var expectedPostgresJobsColumns = []string{
	"postgres_job_attempts.logical_job_id|text|text|NO|",
	"postgres_job_attempts.attempt_generation|bigint|int8|NO|",
	"postgres_job_attempts.recovery_generation|bigint|int8|NO|",
	"postgres_job_attempts.attempt_number|integer|int4|NO|",
	"postgres_job_attempts.worker_id|text|text|NO|",
	"postgres_job_attempts.started_at|timestamp with time zone|timestamptz|NO|clock_timestamp()",
	"postgres_job_attempts.lease_expires_at|timestamp with time zone|timestamptz|NO|",
	"postgres_job_attempts.finalized_at|timestamp with time zone|timestamptz|YES|",
	"postgres_job_attempts.final_state|text|text|YES|",
	"postgres_job_attempts.outcome|text|text|YES|",
	"postgres_job_attempts.effect_status|text|text|YES|",
	"postgres_job_attempts.failure_code|text|text|YES|",
	"postgres_job_attempts.retry_at|timestamp with time zone|timestamptz|YES|",
	"postgres_job_attempts.attempts_used|integer|int4|YES|",
	"postgres_job_attempts.elapsed_used_milliseconds|bigint|int8|YES|",
	"postgres_job_claim_scopes.work_class|text|text|NO|",
	"postgres_job_claim_scopes.paused|boolean|bool|NO|false",
	"postgres_job_claim_scopes.scope_generation|bigint|int8|NO|0",
	"postgres_job_claim_scopes.updated_at|timestamp with time zone|timestamptz|NO|clock_timestamp()",
	"postgres_jobs.logical_job_id|text|text|NO|",
	"postgres_jobs.producer_scope|text|text|NO|",
	"postgres_jobs.producer_key|text|text|NO|",
	"postgres_jobs.occurrence_scope|text|text|NO|",
	"postgres_jobs.occurrence_id|text|text|NO|",
	"postgres_jobs.effect_scope|text|text|NO|",
	"postgres_jobs.effect_key|text|text|NO|",
	"postgres_jobs.intent_fingerprint|bytea|bytea|NO|",
	"postgres_jobs.kind|text|text|NO|",
	"postgres_jobs.args_version|text|text|NO|",
	"postgres_jobs.policy_version|text|text|NO|",
	"postgres_jobs.payload|bytea|bytea|NO|",
	"postgres_jobs.work_class|text|text|NO|",
	"postgres_jobs.state|text|text|NO|",
	"postgres_jobs.available_at|timestamp with time zone|timestamptz|NO|",
	"postgres_jobs.recovery_generation|bigint|int8|NO|0",
	"postgres_jobs.attempt_generation|bigint|int8|NO|0",
	"postgres_jobs.attempts_used|integer|int4|NO|0",
	"postgres_jobs.budget_started_at|timestamp with time zone|timestamptz|NO|clock_timestamp()",
	"postgres_jobs.current_worker_id|text|text|YES|",
	"postgres_jobs.lease_expires_at|timestamp with time zone|timestamptz|YES|",
	"postgres_jobs.created_at|timestamp with time zone|timestamptz|NO|clock_timestamp()",
	"postgres_jobs.updated_at|timestamp with time zone|timestamptz|NO|clock_timestamp()",
	"postgres_jobs.terminal_at|timestamp with time zone|timestamptz|YES|",
}

var expectedPostgresJobsCollations = []string{
	"postgres_job_attempts.logical_job_id|C",
	"postgres_job_attempts.worker_id|C",
	"postgres_job_attempts.final_state|C",
	"postgres_job_attempts.outcome|C",
	"postgres_job_attempts.effect_status|C",
	"postgres_job_attempts.failure_code|C",
	"postgres_job_claim_scopes.work_class|C",
	"postgres_jobs.logical_job_id|C",
	"postgres_jobs.producer_scope|C",
	"postgres_jobs.producer_key|C",
	"postgres_jobs.occurrence_scope|C",
	"postgres_jobs.occurrence_id|C",
	"postgres_jobs.effect_scope|C",
	"postgres_jobs.effect_key|C",
	"postgres_jobs.kind|C",
	"postgres_jobs.args_version|C",
	"postgres_jobs.policy_version|C",
	"postgres_jobs.work_class|C",
	"postgres_jobs.state|C",
	"postgres_jobs.current_worker_id|C",
}

var expectedPostgresJobsConstraints = []string{
	"postgres_job_attempts.postgres_job_attempts_effect_check|c|dc80aa27de228d01c2b772155c04ca71",
	"postgres_job_attempts.postgres_job_attempts_failure_check|c|e87ac3a7edbefa6552cb329abfb4518a",
	"postgres_job_attempts.postgres_job_attempts_final_check|c|171905e856df100583f25a71ca2c5f5d",
	"postgres_job_attempts.postgres_job_attempts_generation_check|c|5b5c6c9576b5b6c4b343ef92adac1688",
	"postgres_job_attempts.postgres_job_attempts_logical_job_id_fkey|f|1bfc49c3607374056a4960c58c8672e3",
	"postgres_job_attempts.postgres_job_attempts_outcome_check|c|ff2e21d254365a87cd3cc7b6426da811",
	"postgres_job_attempts.postgres_job_attempts_pkey|p|f49fd02ceec8e859230193a80614b194",
	"postgres_job_attempts.postgres_job_attempts_state_check|c|689268940c4a476047b8cf06f107c790",
	"postgres_job_attempts.postgres_job_attempts_timestamp_check|c|27743b4ada7d1b5d4730fc97050eabef",
	"postgres_job_attempts.postgres_job_attempts_worker_check|c|3ac857b0c45d7097fc97460f00dd298c",
	"postgres_job_claim_scopes.postgres_job_claim_scopes_class_check|c|cc837cbe0916fd18107fbd922fb6be7f",
	"postgres_job_claim_scopes.postgres_job_claim_scopes_generation_check|c|7d8e60da6e704eda93e580f45d02352b",
	"postgres_job_claim_scopes.postgres_job_claim_scopes_pkey|p|439ac44e4639f0385124509bc531ecd1",
	"postgres_job_claim_scopes.postgres_job_claim_scopes_timestamp_check|c|dcb719d90262517412d1357688ab5b5b",
	"postgres_jobs.postgres_jobs_effect_key|u|1e55024b41513279f4d5d2d2712adf30",
	"postgres_jobs.postgres_jobs_generation_check|c|a6297f8fae07b54bab8c6933cf3866b0",
	"postgres_jobs.postgres_jobs_identity_check|c|25fb554bd05b7f666c83e4d9ccbb7d02",
	"postgres_jobs.postgres_jobs_intent_check|c|92e9eca19c375bf52309bd4d940bb472",
	"postgres_jobs.postgres_jobs_occurrence_key|u|35728ae23ea7ff82666e71776b34b2c8",
	"postgres_jobs.postgres_jobs_owner_check|c|c0023c799c670c538b6c7dc6c26919f3",
	"postgres_jobs.postgres_jobs_pkey|p|8038af6420f53700ed1a4addac65d74d",
	"postgres_jobs.postgres_jobs_producer_key|u|d70d74b22b42fa62ff7d607021bf2cc1",
	"postgres_jobs.postgres_jobs_revision_check|c|566c86bd00bdecd8b6e86d1105cc5194",
	"postgres_jobs.postgres_jobs_state_check|c|bf828e2a197242d3f437ce3be00ce71a",
	"postgres_jobs.postgres_jobs_terminal_check|c|bba3a44605202315afc114bbb58f5dd7",
	"postgres_jobs.postgres_jobs_timestamp_check|c|b26fe4d69166d972dc8494fb21e48b8f",
	"postgres_jobs.postgres_jobs_work_class_check|c|cc837cbe0916fd18107fbd922fb6be7f",
}

var expectedPostgresJobsIndexes = []string{
	"postgres_job_attempts.postgres_job_attempts_lease_idx|a99885f8ebeb14eabcc8fe487744945e",
	"postgres_job_attempts.postgres_job_attempts_pkey|26fe4859f7a8b8d9b21ce3cb6b6e7ce9",
	"postgres_job_claim_scopes.postgres_job_claim_scopes_pkey|c16a31c40f3c07303794585ab0efa6a0",
	"postgres_jobs.postgres_jobs_claim_idx|c85a8aaab60d3485ccca52b59ae78aad",
	"postgres_jobs.postgres_jobs_effect_key|42a4a1f0015429c965ebbf0c4250316b",
	"postgres_jobs.postgres_jobs_lease_idx|443e73c501c920a471cc91ea68298b51",
	"postgres_jobs.postgres_jobs_observation_idx|e53bfcb7bd2d08e24c9c2f2b067839d6",
	"postgres_jobs.postgres_jobs_occurrence_key|945b03bdf2830dd6186f00e01033a143",
	"postgres_jobs.postgres_jobs_pkey|db544e4f7ec2d121d6f99c50ff63cfae",
	"postgres_jobs.postgres_jobs_producer_key|9b3b4be27f80133635ba4fe214ffae6c",
	"postgres_jobs.postgres_jobs_revision_idx|98e9822811d7383a20d239a694aad563",
}

func (s *Session) CheckSchema(ctx context.Context) error {
	return s.withOperation(ctx, "check_schema", pgx.ReadOnly, func(ctx context.Context, queries *sqlcgen.Queries) error {
		columns, err := queries.ListPostgresJobsSchemaColumns(ctx, postgresJobsTables)
		if err != nil {
			return fmt.Errorf("list postgres jobs schema columns: %w", err)
		}
		gotColumns := lo.Map(columns, func(column sqlcgen.ListPostgresJobsSchemaColumnsRow, _ int) string {
			name := column.TableName + "." + column.ColumnName
			return strings.Join([]string{
				name,
				column.DataType,
				column.UdtName,
				column.IsNullable,
				column.ColumnDefault,
			}, "|")
		})
		gotCollations := lo.FilterMap(columns, func(column sqlcgen.ListPostgresJobsSchemaColumnsRow, _ int) (string, bool) {
			name := column.TableName + "." + column.ColumnName
			return name + "|" + column.CollationName, column.CollationName != ""
		})
		if !schemaContains(gotColumns, expectedPostgresJobsColumns) {
			return schemaMismatch("columns", gotColumns, expectedPostgresJobsColumns)
		}
		if !schemaContains(gotCollations, expectedPostgresJobsCollations) {
			return schemaMismatch("collations", gotCollations, expectedPostgresJobsCollations)
		}

		constraints, err := queries.ListPostgresJobsSchemaConstraints(ctx, postgresJobsTables)
		if err != nil {
			return fmt.Errorf("list postgres jobs schema constraints: %w", err)
		}
		gotConstraints := lo.Map(constraints, func(constraint sqlcgen.ListPostgresJobsSchemaConstraintsRow, _ int) string {
			return constraint.TableName + "." + constraint.ConstraintName + "|" + constraint.ConstraintType + "|" + constraint.DefinitionHash
		})
		if !schemaContains(gotConstraints, expectedPostgresJobsConstraints) {
			return schemaMismatch("constraints", gotConstraints, expectedPostgresJobsConstraints)
		}

		indexes, err := queries.ListPostgresJobsSchemaIndexes(ctx, postgresJobsTables)
		if err != nil {
			return fmt.Errorf("list postgres jobs schema indexes: %w", err)
		}
		gotIndexes := lo.Map(indexes, func(index sqlcgen.ListPostgresJobsSchemaIndexesRow, _ int) string {
			return index.TableName + "." + index.IndexName + "|" + index.DefinitionHash
		})
		if !schemaContains(gotIndexes, expectedPostgresJobsIndexes) {
			return schemaMismatch("indexes", gotIndexes, expectedPostgresJobsIndexes)
		}

		scope, err := queries.GetPostgresJobsNeutralScope(ctx)
		if err != nil {
			return fmt.Errorf("%w: neutral claim scope: %w", ErrSchemaIncompatible, err)
		}
		if scope != "neutral" {
			return fmt.Errorf("%w: neutral claim scope has unexpected identity", ErrSchemaIncompatible)
		}
		return nil
	})
}

func schemaContains(got, required []string) bool {
	return !slices.ContainsFunc(required, func(authority string) bool {
		return !slices.Contains(got, authority)
	})
}

func schemaMismatch(authority string, got, want []string) error {
	return fmt.Errorf("%w: %s = %v, want %v", ErrSchemaIncompatible, authority, got, want)
}
