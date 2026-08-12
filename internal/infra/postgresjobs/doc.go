// Package postgresjobs owns the concrete PostgreSQL job Store, reserved control
// Session, and engine persistence vocabulary. It never migrates schema, owns a
// public operator transport, or replaces a broken Session in process.
package postgresjobs
