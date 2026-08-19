// Package postgreswebhook stages one immutable job per receiver and delivers it
// with the Standard Webhooks protocol. PostgreSQL job infrastructure owns
// durability, retries, recovery, concurrency, telemetry, and worker lifecycle.
package postgreswebhook
