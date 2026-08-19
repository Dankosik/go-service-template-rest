package postgreswebhook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
)

const (
	MaxEventDataBytes = 128 << 10
	MaxReceivers      = 1000
	maxIdentityBytes  = 256
	deliveryKind      = "outbound_webhook"
	maxAttempts       = 20
	ownerScopeField   = "owner_scope"
	receiverIDField   = "receiver_id"
)

type Event struct {
	OwnerScope string
	ID         string
	Type       string
	OccurredAt time.Time
	Data       json.RawMessage
}

type ReceiverID string

type Dispatcher struct {
	client    *river.Client[pgx.Tx]
	endpoints *EndpointManifest
}

type preparedDelivery struct {
	args        deliveryArgs
	scheduledAt time.Time
}

type Prepared struct {
	client     *river.Client[pgx.Tx]
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
	AcceptanceID            string          `json:"acceptance_id" river:"unique"`
	DeliveryID              string          `json:"delivery_id"`
	OwnerScope              string          `json:"owner_scope"`
	ReceiverID              string          `json:"receiver_id"`
	ReceiverGeneration      int64           `json:"receiver_generation"`
	URL                     string          `json:"url"`
	ActiveKeyReference      string          `json:"active_key_reference"`
	PredecessorKeyReference string          `json:"predecessor_key_reference,omitempty"`
	FanoutFingerprint       string          `json:"fanout_fingerprint"`
	Body                    json.RawMessage `json:"body"`
}

func (deliveryArgs) Kind() string { return deliveryKind }

func NewDispatcher(endpoints *EndpointManifest) (*Dispatcher, error) {
	if endpoints == nil || len(endpoints.entries) == 0 {
		return nil, fmt.Errorf("%w: endpoint manifest is required", ErrConfig)
	}
	telemetry := otelriver.NewMiddleware(&otelriver.MiddlewareConfig{EnableTracePropagation: true})
	client, err := river.NewClient(riverpgxv5.New(nil), &river.Config{Plugins: []rivertype.Plugin{telemetry}})
	if err != nil {
		return nil, fmt.Errorf("initialize River webhook dispatcher: %w", err)
	}
	return &Dispatcher{client: client, endpoints: endpoints}, nil
}

func (d *Dispatcher) Prepare(event Event, receivers []ReceiverID) (Prepared, error) {
	if d == nil || d.client == nil || d.endpoints == nil {
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
		if err := validateToken(receiverIDField, string(receiver)); err != nil {
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
	fanoutFingerprint := fingerprintFanout(event.OwnerScope, event.ID, body, resolved)
	eventAnchor := deriveStableID("whe_", event.OwnerScope, event.ID)
	prepared := Prepared{client: d.client, deliveries: make([]preparedDelivery, 0, len(ordered))}
	for i, receiver := range ordered {
		endpoint := resolved[i]
		deliveryID := deriveJobID(event.OwnerScope, event.ID, string(receiver), endpoint.Generation)
		acceptanceID := deliveryID
		if i == 0 {
			acceptanceID = eventAnchor
		}
		args := deliveryArgs{
			AcceptanceID: acceptanceID, DeliveryID: deliveryID,
			OwnerScope: event.OwnerScope, ReceiverID: string(receiver), ReceiverGeneration: endpoint.Generation,
			URL: endpoint.URL, ActiveKeyReference: endpoint.ActiveKeyReference,
			PredecessorKeyReference: endpoint.PredecessorKeyReference,
			FanoutFingerprint:       fanoutFingerprint, Body: body,
		}
		if err := args.validate(); err != nil {
			return Prepared{}, fmt.Errorf("prepare webhook delivery %s: %w", args.DeliveryID, err)
		}
		prepared.deliveries = append(prepared.deliveries, preparedDelivery{args: args, scheduledAt: event.OccurredAt.UTC()})
	}
	return prepared, nil
}

func (p Prepared) DeliveryIDs() []string {
	ids := make([]string, len(p.deliveries))
	for i := range p.deliveries {
		ids[i] = p.deliveries[i].args.DeliveryID
	}
	return ids
}

// Stage must be the final operation in the caller-owned transaction. The
// caller rolls the transaction back when Stage returns an error.
func (p Prepared) Stage(ctx context.Context, tx pgx.Tx) (AcceptanceStatus, error) {
	if p.client == nil || tx == nil || len(p.deliveries) == 0 {
		return AcceptanceUnknown, fmt.Errorf("%w: prepared deliveries and transaction are required", ErrConfig)
	}
	newCount := 0
	existingCount := 0
	for _, delivery := range p.deliveries {
		result, err := p.client.InsertTx(ctx, tx, delivery.args, deliveryInsertOpts(delivery.scheduledAt))
		if err != nil {
			return AcceptanceUnknown, fmt.Errorf("stage webhook delivery: %w", err)
		}
		if !result.UniqueSkippedAsDuplicate {
			newCount++
			continue
		}
		if result.Job == nil || !sameDelivery(result.Job.EncodedArgs, delivery.args) {
			return AcceptanceConflict, ErrConflict
		}
		existingCount++
	}
	if newCount != 0 && existingCount != 0 {
		return AcceptanceConflict, fmt.Errorf("%w: partial fanout replay", ErrConflict)
	}
	if existingCount != 0 {
		return AcceptanceExisting, nil
	}
	return AcceptanceNew, nil
}

func (p Prepared) Resolve(ctx context.Context, pool *pgxpool.Pool) (AcceptanceStatus, error) {
	if pool == nil || len(p.deliveries) == 0 {
		return AcceptanceUnknown, fmt.Errorf("%w: prepared deliveries and pool are required", ErrConfig)
	}
	found := make(map[string]deliveryArgs, len(p.deliveries))
	rows, err := pool.Query(ctx, `SELECT args FROM river_job WHERE kind = $1 AND args->>'delivery_id' = ANY($2)`, deliveryKind, p.DeliveryIDs())
	if err != nil {
		return AcceptanceUnknown, fmt.Errorf("resolve webhook acceptance: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var encoded []byte
		if err := rows.Scan(&encoded); err != nil {
			return AcceptanceUnknown, fmt.Errorf("resolve webhook acceptance: %w", err)
		}
		var args deliveryArgs
		if err := json.Unmarshal(encoded, &args); err != nil {
			return AcceptanceConflict, fmt.Errorf("%w: stored webhook arguments are invalid", ErrConflict)
		}
		found[args.DeliveryID] = args
	}
	if err := rows.Err(); err != nil {
		return AcceptanceUnknown, fmt.Errorf("resolve webhook acceptance: %w", err)
	}
	if len(found) == 0 {
		return AcceptanceNotAccepted, nil
	}
	if len(found) != len(p.deliveries) {
		return AcceptanceConflict, fmt.Errorf("%w: fanout readback is not atomic", ErrConflict)
	}
	for _, delivery := range p.deliveries {
		stored, ok := found[delivery.args.DeliveryID]
		if !ok || !equalDelivery(stored, delivery.args) {
			return AcceptanceConflict, ErrConflict
		}
	}
	return AcceptanceAccepted, nil
}

func deliveryInsertOpts(scheduledAt time.Time) *river.InsertOpts {
	states := append(rivertype.UniqueOptsByStateDefault(), rivertype.JobStateCancelled, rivertype.JobStateDiscarded)
	return &river.InsertOpts{
		MaxAttempts: maxAttempts,
		ScheduledAt: scheduledAt,
		UniqueOpts:  river.UniqueOpts{ByArgs: true, ByState: states},
	}
}

func sameDelivery(encoded []byte, want deliveryArgs) bool {
	var got deliveryArgs
	return json.Unmarshal(encoded, &got) == nil && equalDelivery(got, want)
}

func equalDelivery(got, want deliveryArgs) bool {
	return got.DeliveryID == want.DeliveryID &&
		got.FanoutFingerprint == want.FanoutFingerprint
}

func (a deliveryArgs) validate() error {
	for name, value := range map[string]string{
		"acceptance_id": a.AcceptanceID, "delivery_id": a.DeliveryID,
		ownerScopeField: a.OwnerScope, receiverIDField: a.ReceiverID,
		"active_key_reference": a.ActiveKeyReference, "fanout_fingerprint": a.FanoutFingerprint,
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

func fingerprintFanout(owner, eventID string, body []byte, endpoints []Endpoint) string {
	hash := sha256.New()
	for _, value := range []string{owner, eventID} {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	_, _ = hash.Write(body)
	_, _ = hash.Write([]byte{0})
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
	if value == "" || len(value) > maxIdentityBytes || !utf8.ValidString(value) ||
		strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return fmt.Errorf("%w: %s is invalid", ErrConfig, name)
	}
	return nil
}
