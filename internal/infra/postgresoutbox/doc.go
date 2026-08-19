// Package postgresoutbox appends River publication jobs inside caller-owned
// PostgreSQL transactions. River owns delivery state, retry, rescue, and
// maintenance; this package owns only the atomic append boundary.
package postgresoutbox
