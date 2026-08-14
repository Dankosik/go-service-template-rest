package postgreswebhook

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"
)

func (worker *Worker) runAttempt(ctx context.Context, attempt ClaimedAttempt) error {
	prepared, err := PrepareSend(ctx, worker.resolver, attempt, worker.manifest)
	if err != nil {
		evidence := TransportEvidence{DefinitelyNotSent: true, LocalDenial: errors.Is(err, ErrDestinationDenied)}
		return worker.finalizeWithoutCancel(ctx, attempt, Finalization{Evidence: evidence, LocalRetryDelay: retryDelay(attempt.Policy)})
	}
	authorization := AuthorizationEvidence{KeyReference: prepared.KeyReference, SignatureHeaderDigest: prepared.SignatureDigest, DNSSetDigest: prepared.DNSSetDigest, SelectedAddress: prepared.SelectedAddress}
	if err := worker.store.AuthorizeAttempt(ctx, attempt, worker.manifest, authorization); err != nil {
		return worker.finalizeWithoutCancel(ctx, attempt, Finalization{Evidence: TransportEvidence{DefinitelyNotSent: true}, LocalRetryDelay: retryDelay(attempt.Policy)})
	}
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), attempt.Policy.AttemptTimeout)
	result, _ := Send(sendCtx, prepared)
	cancel()
	return worker.finalizeWithoutCancel(ctx, attempt, Finalization{Evidence: result.Evidence, ResponseHeaderBytes: result.ResponseHeaderBytes, ResponseBodyBytes: result.ResponseBodyBytes, RetryAfter: result.RetryAfter, ResponseDate: result.ResponseDate, LocalRetryDelay: retryDelay(attempt.Policy)})
}

func (worker *Worker) finalizeWithoutCancel(ctx context.Context, attempt ClaimedAttempt, final Finalization) error {
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), worker.config.StoreOperationTimeout)
	defer cancel()
	_, err := worker.store.FinalizeAttempt(finalizeCtx, attempt, final)
	return err
}

func retryDelay(policy DeliveryPolicy) time.Duration {
	if policy.BackoffBase <= 0 || policy.BackoffCap < policy.BackoffBase {
		return 0
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return policy.BackoffBase
	}
	return DecorrelatedJitter(policy.BackoffBase, policy.BackoffBase, policy.BackoffCap, binary.BigEndian.Uint64(random[:]))
}
