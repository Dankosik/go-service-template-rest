package postgreswebhook

import (
	"bytes"
	"context"
	"fmt"
	"net/netip"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type AuthorizationEvidence struct {
	KeyReference          string
	SignatureHeaderDigest [32]byte
	DNSSetDigest          [32]byte
	SelectedAddress       netip.Addr
}

func (s *Store) AuthorizeAttempt(ctx context.Context, attempt ClaimedAttempt, manifest *SecretManifest, evidence AuthorizationEvidence) error {
	if !s.valid() || manifest == nil || manifest.Revision() != s.options.ManifestRevision ||
		evidence.KeyReference == "" || !evidence.SelectedAddress.IsValid() || len(evidence.SelectedAddress.AsSlice()) != 4 && len(evidence.SelectedAddress.AsSlice()) != 16 {
		return fmt.Errorf("%w: send authorization inputs are invalid", ErrConfig)
	}
	return s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		if _, err := advanceClock(ctx, queries); err != nil {
			return err
		}
		identity := attempt.Identity
		barrier, err := queries.LockWebhookSendBarrier(ctx, sqlcgen.LockWebhookSendBarrierParams{OwnerScope: identity.OwnerScope, DeliveryID: identity.DeliveryID, CycleNumber: identity.Cycle, AttemptID: identity.AttemptID, Fence: identity.Fence})
		if err != nil {
			return ErrStaleAttempt
		}
		if barrier.ControlRevision != attempt.ControlRevision || barrier.KeyStateRevision != attempt.KeyStateRevision ||
			barrier.RequiredSecretRevision > manifest.Revision() || barrier.ActiveKeyReference == nil || *barrier.ActiveKeyReference != evidence.KeyReference ||
			barrier.DestinationID != attempt.DestinationID || barrier.DestinationGeneration != attempt.DestinationGeneration {
			return ErrStaleAttempt
		}
		if _, err := manifest.Resolve(identity.OwnerScope, attempt.DestinationID, evidence.KeyReference); err != nil {
			return fmt.Errorf("%w: authorized secret binding is missing", ErrConfig)
		}
		rows, err := queries.AuthorizeWebhookAttempt(ctx, sqlcgen.AuthorizeWebhookAttemptParams{
			KeyReference: &evidence.KeyReference, SignatureHeaderDigest: evidence.SignatureHeaderDigest[:],
			DnsSetDigest: evidence.DNSSetDigest[:], SelectedAddress: bytes.Clone(evidence.SelectedAddress.AsSlice()),
			OwnerScope: identity.OwnerScope, DeliveryID: identity.DeliveryID, CycleNumber: identity.Cycle,
			AttemptID: identity.AttemptID, Fence: identity.Fence, ControlRevision: attempt.ControlRevision,
			KeyStateRevision: attempt.KeyStateRevision, ManifestRevision: manifest.Revision(),
		})
		if err != nil {
			return fmt.Errorf("authorize webhook attempt: %w", err)
		}
		if rows != 1 {
			return ErrStaleAttempt
		}
		return nil
	})
}
