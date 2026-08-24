// Package postgreswebhook stages one immutable job per receiver and delivers it
// with the Standard Webhooks protocol. PostgreSQL job infrastructure owns job
// durability, retries, recovery, concurrency, telemetry, and worker lifecycle;
// the caller's business transaction owns replay after job retention.
package postgreswebhook
