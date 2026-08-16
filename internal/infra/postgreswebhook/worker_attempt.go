package postgreswebhook

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math/bits"
	"time"
)

func (worker *Worker) runAttempt(ctx context.Context, attempt ClaimedAttempt) error {
	prepared, err := PrepareSend(ctx, worker.resolver, attempt, worker.manifest)
	if err != nil {
		evidence := TransportEvidence{DefinitelyNotSent: true, LocalDenial: errors.Is(err, ErrDestinationDenied)}
		return worker.finalizeWithoutCancel(ctx, attempt, Finalization{Evidence: evidence, LocalRetryDelay: retryDelay(attempt.Policy, attempt.AttemptNumber)})
	}
	authorization := AuthorizationEvidence{KeyReference: prepared.KeyReference, SignatureHeaderDigest: prepared.SignatureDigest, DNSSetDigest: prepared.DNSSetDigest, SelectedAddress: prepared.SelectedAddress}
	if err := worker.store.AuthorizeAttempt(ctx, attempt, worker.manifest, authorization); err != nil {
		return worker.finalizeWithoutCancel(ctx, attempt, Finalization{Evidence: TransportEvidence{DefinitelyNotSent: true}, LocalRetryDelay: retryDelay(attempt.Policy, attempt.AttemptNumber)})
	}
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), attempt.Policy.AttemptTimeout)
	result, _ := Send(sendCtx, prepared)
	cancel()
	return worker.finalizeWithoutCancel(ctx, attempt, Finalization{Evidence: result.Evidence, ResponseHeaderBytes: result.ResponseHeaderBytes, ResponseBodyBytes: result.ResponseBodyBytes, RetryAfter: result.RetryAfter, ResponseDate: result.ResponseDate, LocalRetryDelay: retryDelay(attempt.Policy, attempt.AttemptNumber)})
}

func (worker *Worker) finalizeWithoutCancel(ctx context.Context, attempt ClaimedAttempt, final Finalization) error {
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), worker.config.StoreOperationTimeout)
	defer cancel()
	outcome, err := worker.store.FinalizeAttempt(finalizeCtx, attempt, final)
	if err == nil {
		worker.telemetry.Record(finalizeCtx, "attempt", outcome)
	}
	return err
}

func retryDelay(policy DeliveryPolicy, attempt int) time.Duration {
	if policy.BackoffBase <= 0 || policy.BackoffCap < policy.BackoffBase {
		return 0
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return policy.BackoffBase
	}
	return retryDelayWithRandom(policy, attempt, binary.BigEndian.Uint64(random[:]))
}

func retryDelayWithRandom(policy DeliveryPolicy, attempt int, random uint64) time.Duration {
	limit := policy.BackoffBase
	for range max(attempt-1, 0) {
		if limit > policy.BackoffCap/2 {
			limit = policy.BackoffCap
			break
		}
		limit *= 2
	}
	limit = min(limit, policy.BackoffCap)
	if limit <= policy.BackoffBase {
		return policy.BackoffBase
	}
	span, err := uint64Value(int64(limit - policy.BackoffBase))
	if err != nil {
		return policy.BackoffBase
	}
	offset, _ := bits.Mul64(random, span+1)
	jitter, err := durationValue(offset)
	if err != nil {
		return policy.BackoffBase
	}
	return policy.BackoffBase + jitter
}
