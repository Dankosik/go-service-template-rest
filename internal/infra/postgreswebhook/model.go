package postgreswebhook

import "time"

type DeliveryState string

const activeDisposition = "active"

const (
	DeliveryReady       DeliveryState = "ready"
	DeliveryScheduled   DeliveryState = "scheduled"
	DeliveryInFlight    DeliveryState = "in_flight"
	DeliverySuspended   DeliveryState = "suspended"
	DeliveryTerminal    DeliveryState = "terminal"
	DeliveryQuarantined DeliveryState = "quarantined"
)

type OutcomeClass string

const (
	OutcomeHTTPAccepted           OutcomeClass = "http_accepted"
	OutcomeDefinitelyNotSentRetry OutcomeClass = "definitely_not_sent_retryable"
	OutcomeRetryableHTTPAmbiguous OutcomeClass = "retryable_http_ambiguous"
	OutcomeTransportAmbiguous     OutcomeClass = "transport_ambiguous"
	OutcomeHTTPRejected           OutcomeClass = "http_rejected"
	OutcomeLocallyDenied          OutcomeClass = "locally_denied"
	OutcomeAttemptsExhausted      OutcomeClass = "attempts_exhausted"
	OutcomeUnknown                OutcomeClass = "outcome_unknown"
	OutcomeClosedUnknown          OutcomeClass = "closed_unknown"
	OutcomeRetained               OutcomeClass = "retained"
)

type AttemptIdentity struct {
	OwnerScope string
	DeliveryID string
	Cycle      int64
	AttemptID  string
	Fence      int64
}

type ClaimedAttempt struct {
	Identity              AttemptIdentity
	CapacityRevision      int64
	DestinationID         string
	DestinationGeneration int64
	URL                   string
	Body                  []byte
	ContentType           string
	AttemptedAt           time.Time
	Deadline              time.Time
	PreviousRetryDelay    time.Duration
	KeyReference          string
	PredecessorReference  string
	ManifestRevision      int64
	SignatureProfile      string
	ControlRevision       int64
	KeyStateRevision      int64
	Policy                DeliveryPolicy
}
