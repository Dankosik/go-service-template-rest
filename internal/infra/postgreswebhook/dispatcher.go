package postgreswebhook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
)

const (
	MaxEventDataBytes = 128 << 10
	MaxReceivers      = 1000
	ownerScopeField   = "owner_scope"
	receiverIDField   = "receiver_id"
)

var deliveryRevision = jobs.Revision{
	Kind:          "outbound_webhook",
	ArgsVersion:   "v1",
	PolicyVersion: "standard_webhooks_v1",
}

type Event struct {
	OwnerScope string
	ID         string
	Type       string
	OccurredAt time.Time
	Data       json.RawMessage
}

type ReceiverID string

type Dispatcher struct {
	endpoints  *EndpointManifest
	definition jobs.Definition[deliveryArgs]
}

type preparedDelivery struct {
	id  string
	job jobs.Prepared
}

type Prepared struct {
	deliveries []preparedDelivery
}

type AcceptanceStatus string

const (
	AcceptanceNew         AcceptanceStatus = "new"
	AcceptanceExisting    AcceptanceStatus = "existing"
	AcceptanceAccepted    AcceptanceStatus = "accepted"
	AcceptanceNotAccepted AcceptanceStatus = "not_accepted"
	AcceptanceConflict    AcceptanceStatus = "conflict"
	AcceptanceUnknown     AcceptanceStatus = "unknown"
)

type deliveryArgs struct {
	OwnerScope              string          `json:"owner_scope"`
	ReceiverID              string          `json:"receiver_id"`
	ReceiverGeneration      int64           `json:"receiver_generation"`
	URL                     string          `json:"url"`
	ActiveKeyReference      string          `json:"active_key_reference"`
	PredecessorKeyReference string          `json:"predecessor_key_reference,omitempty"`
	FanoutFingerprint       string          `json:"fanout_fingerprint"`
	Body                    json.RawMessage `json:"body"`
}

func NewDispatcher(endpoints *EndpointManifest) (*Dispatcher, error) {
	if endpoints == nil || len(endpoints.entries) == 0 {
		return nil, fmt.Errorf("%w: endpoint manifest is required", ErrConfig)
	}
	definition, err := deliveryDefinition()
	if err != nil {
		return nil, err
	}
	return &Dispatcher{endpoints: endpoints, definition: definition}, nil
}

func (d *Dispatcher) Prepare(event Event, receivers []ReceiverID) (Prepared, error) {
	if d == nil || d.endpoints == nil {
		return Prepared{}, fmt.Errorf("%w: dispatcher is required", ErrConfig)
	}
	if err := validateEvent(event); err != nil {
		return Prepared{}, err
	}
	if len(receivers) == 0 || len(receivers) > MaxReceivers {
		return Prepared{}, fmt.Errorf("%w: receivers must contain 1..%d entries", ErrConfig, MaxReceivers)
	}

	ordered := slices.Clone(receivers)
	slices.Sort(ordered)
	for i, receiver := range ordered {
		if err := validateToken("receiver_id", string(receiver)); err != nil {
			return Prepared{}, err
		}
		if i > 0 && receiver == ordered[i-1] {
			return Prepared{}, fmt.Errorf("%w: duplicate receiver", ErrConflict)
		}
	}

	body, err := json.Marshal(struct {
		Type      string          `json:"type"`
		Timestamp string          `json:"timestamp"`
		Data      json.RawMessage `json:"data"`
	}{Type: event.Type, Timestamp: event.OccurredAt.UTC().Format(time.RFC3339Nano), Data: event.Data})
	if err != nil {
		return Prepared{}, fmt.Errorf("%w: encode event: %w", ErrConfig, err)
	}

	resolved := make([]Endpoint, len(ordered))
	for i, receiver := range ordered {
		endpoint, err := d.endpoints.Resolve(event.OwnerScope, string(receiver))
		if err != nil {
			return Prepared{}, err
		}
		resolved[i] = endpoint
	}
	fanoutFingerprint := fingerprintFanout(event.OwnerScope, event.ID, resolved)
	eventAnchor := deriveStableID("whe_", event.OwnerScope, event.ID)
	prepared := Prepared{deliveries: make([]preparedDelivery, 0, len(ordered))}
	availableAt := event.OccurredAt.UTC()
	for i, receiver := range ordered {
		endpoint := resolved[i]
		deliveryID := deriveJobID(event.OwnerScope, event.ID, string(receiver), endpoint.Generation)
		occurrenceID := deliveryID
		if i == 0 {
			occurrenceID = eventAnchor
		}
		producerKey := deriveStableID("whr_", event.OwnerScope, event.ID, string(receiver))
		args := deliveryArgs{
			OwnerScope: event.OwnerScope, ReceiverID: string(receiver), ReceiverGeneration: endpoint.Generation,
			URL: endpoint.URL, ActiveKeyReference: endpoint.ActiveKeyReference,
			PredecessorKeyReference: endpoint.PredecessorKeyReference,
			FanoutFingerprint:       fanoutFingerprint, Body: body,
		}
		job, err := d.definition.Prepare(args, deliveryIdentity(deliveryID, producerKey, occurrenceID), availableAt)
		if err != nil {
			return Prepared{}, fmt.Errorf("prepare webhook delivery %s: %w", deliveryID, err)
		}
		prepared.deliveries = append(prepared.deliveries, preparedDelivery{id: deliveryID, job: job})
	}
	return prepared, nil
}

func (p Prepared) DeliveryIDs() []string {
	ids := make([]string, len(p.deliveries))
	for i := range p.deliveries {
		ids[i] = p.deliveries[i].id
	}
	return ids
}

// Stage must be the final operation in the caller-owned transaction. The
// caller rolls the transaction back when Stage returns an error.
func (p Prepared) Stage(ctx context.Context, store *postgresjobs.Store, tx pgx.Tx) (AcceptanceStatus, error) {
	if store == nil || tx == nil || len(p.deliveries) == 0 {
		return AcceptanceUnknown, fmt.Errorf("%w: prepared deliveries, store, and transaction are required", ErrConfig)
	}
	newCount := 0
	existingCount := 0
	for _, delivery := range p.deliveries {
		result, err := store.Stage(ctx, tx, delivery.job)
		if err != nil {
			return AcceptanceUnknown, fmt.Errorf("stage webhook delivery: %w", err)
		}
		switch result.Outcome {
		case jobs.StageNew:
			newCount++
		case jobs.StageExisting:
			existingCount++
		case jobs.StageConflict:
			return AcceptanceConflict, ErrConflict
		case jobs.StageRejected:
			return AcceptanceUnknown, fmt.Errorf("%w: jobs stage returned %q", ErrConfig, result.Outcome)
		}
	}
	if newCount != 0 && existingCount != 0 {
		return AcceptanceConflict, fmt.Errorf("%w: partial fanout replay", ErrConflict)
	}
	if existingCount != 0 {
		return AcceptanceExisting, nil
	}
	return AcceptanceNew, nil
}

func (p Prepared) Resolve(ctx context.Context, store *postgresjobs.Store) (AcceptanceStatus, error) {
	if store == nil || len(p.deliveries) == 0 {
		return AcceptanceUnknown, fmt.Errorf("%w: prepared deliveries and store are required", ErrConfig)
	}
	accepted := 0
	notAccepted := 0
	for _, delivery := range p.deliveries {
		result, err := store.ResolveAcceptance(ctx, delivery.job.ReadbackExpectation())
		if err != nil {
			return AcceptanceUnknown, fmt.Errorf("resolve webhook acceptance: %w", err)
		}
		switch result.Outcome {
		case jobs.ReadbackAccepted:
			accepted++
		case jobs.ReadbackNotAccepted:
			notAccepted++
		case jobs.ReadbackConflict:
			return AcceptanceConflict, ErrConflict
		case jobs.ReadbackUnknown:
			return AcceptanceUnknown, errors.New("webhook acceptance remains unknown")
		}
	}
	if accepted == len(p.deliveries) {
		return AcceptanceAccepted, nil
	}
	if notAccepted == len(p.deliveries) {
		return AcceptanceNotAccepted, nil
	}
	return AcceptanceConflict, fmt.Errorf("%w: fanout readback is not atomic", ErrConflict)
}

func deliveryDefinition() (jobs.Definition[deliveryArgs], error) {
	definition, err := jobs.NewDefinition(jobs.DefinitionInput[deliveryArgs]{
		Revision:        deliveryRevision,
		MaxPayloadBytes: jobs.MaxPayloadBytes,
		Validate:        deliveryArgs.validate,
		Policy: jobs.Policy{
			Effect: jobs.EffectPolicy{AmbiguousAction: jobs.AmbiguousEffectRetry},
			Retry: jobs.RetryPolicy{
				MaxAttempts: 20, MaxElapsed: 4 * 24 * time.Hour,
				InitialBackoff: 5 * time.Second, MaxBackoff: 24 * time.Hour,
				HintPolicy: jobs.RetryHintBackoffFloor, Jitter: jobs.JitterSHA256,
				JitterPermille: 100, MaxRecoveryWave: 8,
			},
			Recovery: jobs.RecoveryPolicy{
				Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved,
			},
			MaxAttemptDuration:  30 * time.Second,
			TerminationEnvelope: 30 * time.Second,
		},
	})
	if err != nil {
		return jobs.Definition[deliveryArgs]{}, fmt.Errorf("define webhook job: %w", err)
	}
	return definition, nil
}

func (a deliveryArgs) validate() error {
	for name, value := range map[string]string{
		ownerScopeField: a.OwnerScope, receiverIDField: a.ReceiverID, "active_key_reference": a.ActiveKeyReference,
		"fanout_fingerprint": a.FanoutFingerprint,
	} {
		if err := validateToken(name, value); err != nil {
			return err
		}
	}
	if a.PredecessorKeyReference != "" {
		if err := validateToken("predecessor_key_reference", a.PredecessorKeyReference); err != nil {
			return err
		}
	}
	if a.ReceiverGeneration <= 0 || len(a.Body) == 0 || !json.Valid(a.Body) {
		return fmt.Errorf("%w: receiver generation and JSON body are required", ErrConfig)
	}
	_, err := parseWebhookURL(a.URL)
	return err
}

func validateEvent(event Event) error {
	for name, value := range map[string]string{ownerScopeField: event.OwnerScope, "event_id": event.ID, "event_type": event.Type} {
		if err := validateToken(name, value); err != nil {
			return err
		}
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at is required", ErrConfig)
	}
	if len(event.Data) == 0 || len(event.Data) > MaxEventDataBytes || !json.Valid(event.Data) {
		return fmt.Errorf("%w: data must be valid JSON no larger than %d bytes", ErrConfig, MaxEventDataBytes)
	}
	return nil
}

func deliveryIdentity(deliveryID, producerKey, occurrenceID string) jobs.AcceptanceIdentity {
	return jobs.AcceptanceIdentity{
		LogicalJobID:  jobs.LogicalJobID(deliveryID),
		ProducerScope: "webhook_receiver", ProducerKey: jobs.ProducerKey(producerKey),
		OccurrenceScope: "webhook_event", OccurrenceID: jobs.OccurrenceID(occurrenceID),
		EffectScope: "webhook", EffectKey: jobs.EffectKey(deliveryID),
	}
}

func deriveJobID(owner, eventID, receiver string, generation int64) string {
	return deriveStableID("whd_", owner, eventID, receiver, strconv.FormatInt(generation, 10))
}

func deriveStableID(prefix string, values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return prefix + hex.EncodeToString(hash.Sum(nil))
}

func fingerprintFanout(owner, eventID string, endpoints []Endpoint) string {
	hash := sha256.New()
	for _, value := range []string{owner, eventID} {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	for _, endpoint := range endpoints {
		for _, value := range []string{
			endpoint.ReceiverID, strconv.FormatInt(endpoint.Generation, 10), endpoint.URL,
			endpoint.ActiveKeyReference, endpoint.PredecessorKeyReference,
		} {
			_, _ = hash.Write([]byte(value))
			_, _ = hash.Write([]byte{0})
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func validateToken(name, value string) error {
	if value == "" || len(value) > jobs.MaxIdentityBytes || !utf8.ValidString(value) ||
		strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return fmt.Errorf("%w: %s is invalid", ErrConfig, name)
	}
	return nil
}
