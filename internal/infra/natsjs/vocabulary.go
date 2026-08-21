package natsjs

// This file is the label vocabulary: every literal this package puts on a
// metric attribute or an operator log field is named here, so a maintainer
// adding one has a list to join rather than a string to invent — and so the
// values that repeat across positions with different meanings are told apart by
// name.
//
// outcome and event reach metric attributes and are bounded below, because an
// unrecognized value there mints a time series. reason is a log field only, so
// it is named for the reader and not collapsed.
//
// The dead-letter reasons are deliberately not here. They travel on the wire to
// whatever consumes the dead-letter stream, so they are a published contract
// rather than an internal label, and they live with the transfer that writes
// them in message_deadletter.go.

// Publication, handler, and drain outcomes.
const (
	// Publication outcomes.
	outcomeAccepted  = "accepted"
	outcomeRejected  = "rejected"
	outcomeAmbiguous = "ambiguous"
	// Handler outcomes.
	outcomeSuccess   = "success"
	outcomeTerminal  = "terminal"
	outcomePermanent = "permanent"
	outcomeCanceled  = "canceled"
	outcomeTimeout   = "timeout"
	outcomeRetryable = "retryable"
	// boundedOther is what an unrecognized outcome or connection event
	// collapses to, so a value this list forgot cannot mint a time series.
	boundedOther = "other"
)

const connectionAsyncError = "async_error"

// The reason field on an operator log record. It never reaches a metric.
const (
	reasonNone                = "none"
	reasonInvalidMessage      = "invalid_message"
	reasonContextDone         = "context_done_before_dispatch"
	reasonDraining            = "draining"
	reasonBrokerRejected      = "broker_rejected"
	reasonAckUnavailable      = "ack_unavailable"
	reasonAckAmbiguous        = "ack_ambiguous"
	reasonShutdown            = "shutdown"
	reasonHandlerPanic        = "handler_panic"
	reasonHandlerPermanent    = "handler_permanent"
	reasonHandlerExhausted    = "handler_exhausted"
	reasonHandlerRetry        = "handler_retry"
	reasonMetadataUnavailable = "metadata_unavailable"
	reasonDeliveryBound       = "delivery_bound"
	reasonDeadLetterEnvelope  = "dlq_envelope"
	reasonDeadLetterRejected  = "dlq_rejected"
	// The asynchronous connection faults asyncErrorReason classes.
	reasonSlowConsumer   = "slow_consumer"
	reasonAuthentication = "authentication"
	reasonPermission     = "permission"
	reasonMessageBound   = "message_bound"
	reasonConnection     = "connection"
)

// A redelivery request that the broker itself refused is logged as this
// reason plus "_rejected", so the log names which settlement path failed.
const (
	redeliveryHandler    = "handler_redelivery"
	redeliveryDeadLetter = "dlq_redelivery"
	redeliverySourceAck  = "source_ack_redelivery"
)

// boundedOutcome is the closed vocabulary for the outcome attribute, shared by
// messaging.publish.operations, messaging.handler.operations,
// messaging.dlq.transfers. Each metric uses a
// subset; the union is bounded once because one unrecognized value is the same
// failure however it arrives.
func boundedOutcome(value string) string {
	switch value {
	case outcomeAccepted, outcomeRejected, outcomeAmbiguous,
		outcomeSuccess, outcomeTerminal, outcomePermanent, outcomeCanceled,
		outcomeTimeout, outcomeRetryable:
		return value
	default:
		return boundedOther
	}
}
