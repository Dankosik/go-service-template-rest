package postgreswebhook

const (
	boundedOther                  = "other"
	failureNone                   = "none"
	failureSSRFDenied             = "ssrf_denied"
	failureSecretRotation         = "secret_rotation_failure"
	failureResponseBound          = "response_bound"
	failureReconciliationConflict = "reconciliation_conflict"
	failureTLSDenied              = "tls_denied"
	failureTimeout                = "timeout"
	failureCanceled               = "canceled"
)

var (
	boundedEvents = map[string]struct{}{
		"attempt": {}, "claim": {}, "claim_progress": {}, "maintenance": {}, "observation": {},
		"reconciliation": {}, "privacy_retirement": {}, "cleanup": {}, boundedOther: {},
	}
	boundedOutcomes = map[string]struct{}{
		"http_accepted": {}, "definitely_not_sent_retryable": {}, "retryable_http_ambiguous": {},
		"transport_ambiguous": {}, "http_rejected": {}, "locally_denied": {}, "attempts_exhausted": {},
		"outcome_unknown": {}, "closed_unknown": {}, "retained": {}, boundedOther: {},
	}
	boundedFailures = map[string]struct{}{
		failureNone: {}, failureSSRFDenied: {}, failureSecretRotation: {}, failureResponseBound: {},
		failureReconciliationConflict: {}, failureTLSDenied: {}, failureTimeout: {}, failureCanceled: {}, boundedOther: {},
	}
)

func boundedValue(vocabulary map[string]struct{}, value string) string {
	if _, ok := vocabulary[value]; ok {
		return value
	}
	return boundedOther
}
