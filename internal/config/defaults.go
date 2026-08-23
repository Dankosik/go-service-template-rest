package config

import "maps"

// buildVersion is set by the production Docker build and remains "dev" for local builds.
var buildVersion = "dev"

// buildCommit is the source revision the binary was built from, stamped by the
// production Docker build.
//
// It is a linker variable rather than something read back from
// runtime/debug.ReadBuildInfo: the image build runs with -buildvcs=false, because
// .dockerignore keeps .git out of the build context, so the binary carries no
// vcs.revision of its own. Without this the commit existed only as an OCI image
// label — unreachable from inside the container, and unjoinable to a span, a
// metric, or a log line.
var buildCommit = "unknown"

// defaultValues merges each section's own defaults over the ones every profile
// keeps. A section that a build profile removes contributes through a single
// call that leaves with its file, rather than through a marked block cut out of
// one shared map.
func defaultValues() map[string]any {
	values := map[string]any{
		"app.env":         "local",
		"app.version":     buildVersion,
		"app.commit":      buildCommit,
		"app.instance_id": "",

		"health.refresh_interval":  "2s",
		"health.failure_threshold": 3,

		"log.level": "info",

		"runtime.memory_limit_ratio": 0.9,
	}

	maps.Copy(values, httpDefaults())
	maps.Copy(values, observabilityDefaults())
	// profile:authn-oidc-jwt:start
	maps.Copy(values, authnDefaults())
	// profile:authn-oidc-jwt:end
	// profile:outbound-auth-oauth2-client-credentials:start
	maps.Copy(values, outboundAuthDefaults())
	// profile:outbound-auth-oauth2-client-credentials:end
	// profile:grpc:start
	maps.Copy(values, grpcDefaults())
	// profile:grpc:end
	// profile:messaging-nats-jetstream:start
	maps.Copy(values, messagingDefaults())
	// profile:messaging-nats-jetstream:end
	// profile:database-postgres:start
	maps.Copy(values, postgresDefaults())
	// profile:database-postgres:end
	// profile:jobs-postgres:start
	maps.Copy(values, jobsDefaults())
	// profile:jobs-postgres:end
	// profile:webhooks-durable:start
	maps.Copy(values, webhooksDefaults())
	// profile:webhooks-durable:end
	// profile:inbound-webhooks-standard:start
	maps.Copy(values, inboundWebhooksDefaults())
	// profile:inbound-webhooks-standard:end
	// profile:object-storage:start
	maps.Copy(values, objectStorageDefaults())
	// profile:object-storage:end

	return values
}
