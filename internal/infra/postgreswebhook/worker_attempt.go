package postgreswebhook

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net/netip"
	"time"
)

func (worker *Worker) runAttempt(ctx context.Context, attempt ClaimedAttempt) (OutcomeClass, string, error) {
	preSendCtx, cancelPreSend := context.WithDeadline(ctx, attempt.Deadline)
	defer cancelPreSend()
	prepared, err := PrepareSend(preSendCtx, worker.resolver, attempt, worker.manifest)
	if err != nil {
		evidence := TransportEvidence{DefinitelyNotSent: true, LocalDenial: errors.Is(err, ErrDestinationDenied)}
		outcome, finalizeErr := worker.finalizeWithoutCancel(ctx, attempt, Finalization{Evidence: evidence, LocalRetryDelay: retryDelay(attempt.Policy, attempt.PreviousRetryDelay)})
		return outcome, failureClass(err), finalizeErr
	}
	sendCtx, cancelSend := context.WithDeadline(context.WithoutCancel(ctx), attempt.Deadline)
	defer cancelSend()
	result, sendErr := tryPreparedAddresses(sendCtx, prepared, func(candidate PreparedSend) error {
		authorization := AuthorizationEvidence{KeyReference: candidate.KeyReference, KeyReferences: candidate.KeyReferences, SignatureHeaderDigest: candidate.SignatureDigest, DNSSetDigest: candidate.DNSSetDigest, SelectedAddress: candidate.SelectedAddress}
		return worker.store.AuthorizeAttempt(preSendCtx, attempt, worker.manifest, authorization)
	}, Send)
	outcome, finalizeErr := worker.finalizeWithoutCancel(ctx, attempt, Finalization{Evidence: result.Evidence, ResponseHeaderBytes: result.ResponseHeaderBytes, ResponseBodyBytes: result.ResponseBodyBytes, RetryAfter: result.RetryAfter, ResponseDate: result.ResponseDate, LocalRetryDelay: retryDelay(attempt.Policy, attempt.PreviousRetryDelay)})
	return outcome, failureClass(sendErr), finalizeErr
}

func tryPreparedAddresses(ctx context.Context, prepared PreparedSend, authorize func(PreparedSend) error, send func(context.Context, PreparedSend) (SendResult, error)) (SendResult, error) {
	addresses := prepared.Addresses
	if len(addresses) == 0 && prepared.SelectedAddress.IsValid() {
		addresses = []netip.Addr{prepared.SelectedAddress}
	}
	if len(addresses) == 0 || authorize == nil || send == nil {
		return SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, ErrConfig
	}
	var last SendResult
	var lastErr error
	for _, address := range addresses {
		prepared.SelectedAddress = address
		if err := authorize(prepared); err != nil {
			return SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true}}, err
		}
		last, lastErr = send(ctx, prepared)
		if lastErr == nil || !last.Evidence.DefinitelyNotSent {
			return last, lastErr
		}
	}
	return last, lastErr
}

func (worker *Worker) finalizeWithoutCancel(ctx context.Context, attempt ClaimedAttempt, final Finalization) (OutcomeClass, error) {
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), worker.config.StoreOperationTimeout)
	defer cancel()
	return worker.store.FinalizeAttempt(finalizeCtx, attempt, final)
}

func retryDelay(policy DeliveryPolicy, previous time.Duration) time.Duration {
	if policy.BackoffBase <= 0 || policy.BackoffCap < policy.BackoffBase {
		return 0
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return min(max(previous, policy.BackoffBase), policy.BackoffCap)
	}
	return DecorrelatedJitter(previous, policy.BackoffBase, policy.BackoffCap, binary.BigEndian.Uint64(random[:]))
}

func failureClass(err error) string {
	switch {
	case err == nil:
		return failureNone
	case errors.Is(err, ErrDestinationDenied):
		return failureSSRFDenied
	case errors.Is(err, ErrSecretUnavailable):
		return failureSecretRotation
	case errors.Is(err, ErrResponseLimit):
		return failureResponseBound
	case errors.Is(err, ErrConflict), errors.Is(err, ErrClockRegression), errors.Is(err, ErrStaleAttempt):
		return failureReconciliationConflict
	case errors.Is(err, context.DeadlineExceeded):
		return failureTimeout
	case errors.Is(err, context.Canceled):
		return failureCanceled
	case permanentTLSValidationError(err):
		return failureTLSDenied
	default:
		return boundedOther
	}
}
