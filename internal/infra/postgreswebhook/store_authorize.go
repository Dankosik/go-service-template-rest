package postgreswebhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type AuthorizationEvidence struct {
	KeyReference          string
	KeyReferences         []string
	SignatureHeaderDigest [32]byte
	DNSSetDigest          [32]byte
	SelectedAddress       netip.Addr
}

//nolint:cyclop // Authorization keeps every fail-closed identity and fence check together.
func (s *Store) AuthorizeAttempt(ctx context.Context, attempt ClaimedAttempt, manifest *SecretManifest, evidence AuthorizationEvidence) error {
	if !s.valid() || manifest == nil || manifest.Revision() != s.options.ManifestRevision ||
		attempt.CapacityRevision != s.options.CapacityRevision ||
		evidence.KeyReference == "" || len(evidence.KeyReferences) < 1 || len(evidence.KeyReferences) > 2 || evidence.KeyReferences[0] != evidence.KeyReference ||
		!evidence.SelectedAddress.IsValid() || len(evidence.SelectedAddress.AsSlice()) != 4 && len(evidence.SelectedAddress.AsSlice()) != 16 {
		return fmt.Errorf("%w: send authorization inputs are invalid", ErrConfig)
	}
	return s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		sampledAt, err := advanceClock(ctx, queries)
		if err != nil {
			return err
		}
		identity := attempt.Identity
		barrier, err := queries.LockWebhookSendBarrier(ctx, sqlcgen.LockWebhookSendBarrierParams{
			CapacityRevision: attempt.CapacityRevision, SampledAt: pgtime(sampledAt), OwnerScope: identity.OwnerScope,
			DeliveryID: identity.DeliveryID, CycleNumber: identity.Cycle, AttemptID: identity.AttemptID, Fence: identity.Fence,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStaleAttempt
		}
		if err != nil {
			return fmt.Errorf("lock webhook send barrier: %w", err)
		}
		if barrier.ControlRevision != attempt.ControlRevision || barrier.KeyStateRevision != attempt.KeyStateRevision ||
			barrier.RequiredSecretRevision > manifest.Revision() || barrier.ActiveKeyReference != evidence.KeyReference ||
			barrier.DestinationID != attempt.DestinationID || barrier.DestinationGeneration != attempt.DestinationGeneration {
			return ErrStaleAttempt
		}
		expected := []string{attempt.KeyReference}
		if attempt.PredecessorReference != "" {
			expected = append(expected, attempt.PredecessorReference)
		}
		if !slices.Equal(evidence.KeyReferences, expected) {
			return ErrStaleAttempt
		}
		for _, reference := range evidence.KeyReferences {
			if _, err := manifest.Resolve(identity.OwnerScope, attempt.DestinationID, reference); err != nil {
				return fmt.Errorf("%w: authorized secret binding is missing", ErrConfig)
			}
		}
		rows, err := queries.AuthorizeWebhookAttempt(ctx, sqlcgen.AuthorizeWebhookAttemptParams{
			KeyReference: &evidence.KeyReference, KeyReferences: evidence.KeyReferences, SignatureHeaderDigest: evidence.SignatureHeaderDigest[:],
			DnsSetDigest: evidence.DNSSetDigest[:], SelectedAddress: bytes.Clone(evidence.SelectedAddress.AsSlice()),
			OwnerScope: identity.OwnerScope, DeliveryID: identity.DeliveryID, CycleNumber: identity.Cycle,
			AttemptID: identity.AttemptID, Fence: identity.Fence, ControlRevision: attempt.ControlRevision,
			KeyStateRevision: attempt.KeyStateRevision, ManifestRevision: manifest.Revision(),
			CapacityRevision: attempt.CapacityRevision, SampledAt: pgtime(sampledAt),
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
