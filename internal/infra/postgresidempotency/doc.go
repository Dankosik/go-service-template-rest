// Package postgresidempotency owns PostgreSQL-backed HTTP idempotency.
//
// Reserve publishes a short-lived writer reservation before an endpoint starts
// its caller-owned transaction. Acquire, feature work, outbox append, and
// Complete share that transaction; Release is only for a definitely rolled-back
// transaction. Reconcile and MaterializeEpoch use the writer after an ambiguous
// commit or a successful completion.
//
// The store prefix is a lint boundary: only store*.go may import pgx or sqlc.
package postgresidempotency
