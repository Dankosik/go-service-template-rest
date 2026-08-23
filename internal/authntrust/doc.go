// Package authntrust owns the deployment values shared by configuration and the
// concrete authentication engines: trusted provider URLs, introspection target
// classes, and the selected JWT token profile.
//
// It is a leaf on purpose, and it exists because two owners need one answer at
// two different times. internal/config must refuse a bad value at configuration
// load, before any verifier exists, so a mistyped key fails the process instead
// of the authentication boundary. Each concrete engine must enforce the same
// rules when it builds its policy. The depguard rule config_no_runtime_owners
// stops internal/config from importing runtime adapters, so neither owner may
// import the other and the rules would otherwise live in two copies held in
// step by a parity test.
//
// What stays out: this package holds no configured value and builds no policy
// object. It answers about strings, so a caller may ask before it has anywhere
// to put the answer.
package authntrust
