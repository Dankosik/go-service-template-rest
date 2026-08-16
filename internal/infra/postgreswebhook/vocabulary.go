package postgreswebhook

var boundedVocabulary = map[string]struct{}{
	"ready": {}, "scheduled": {}, "in_flight": {}, "suspended": {}, "terminal": {}, "quarantined": {},
	"http_accepted": {}, "definitely_not_sent_retryable": {}, "retryable_http_ambiguous": {},
	"transport_ambiguous": {}, "http_rejected": {}, "locally_denied": {}, "attempts_exhausted": {},
	"outcome_unknown": {}, "closed_unknown": {}, "claim": {}, "attempt": {}, "maintenance": {}, "observation": {}, "other": {},
	"reconcile": {}, "deadline": {}, "cleanup": {},
}

func boundedValue(value string) string {
	if _, ok := boundedVocabulary[value]; ok {
		return value
	}
	return "other"
}
