// Package jobs defines immutable, standard-library-only contracts for durable
// one-off background work. It owns job definitions, acceptance values, pure
// transition policy, and exact-revision handler registration; persistence,
// scheduling loops, transports, and operator controls belong elsewhere.
package jobs
