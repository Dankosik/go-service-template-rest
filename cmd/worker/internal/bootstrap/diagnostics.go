package bootstrap

import (
	"github.com/example/go-service-template-rest/internal/health"
)

// workerReady is this binary's readiness verdict, which runtimeopts.DiagnosticsServer
// serves as one answer.
//
// The two conditions are separate facts and both have to hold. The broker state
// is read live rather than through a probe, because a consumer that lost its
// connection is not ready now and waiting for the next refresh would keep it in
// rotation for up to one interval. The cached verdict covers everything else the
// process probes.
func workerReady(messagingReady func() bool, healthSvc *health.Service) func() bool {
	return func() bool {
		return messagingReady() && healthSvc.Cached() == nil
	}
}
