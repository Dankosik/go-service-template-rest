//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go/jetstream"
)

func TestPostgresInboxNATSLogicalIdentityAndAcknowledgement(t *testing.T) {
	t.Parallel()
	ctx, pool := newInboxFixture(t)
	createInboxEffects(t, ctx, pool)
	fixture := newNATSFixture(t)
	const consumer = "postgres-inbox-worker"
	consumerIdentity := sourceStream + "/" + consumer
	if consumerIdentity != "EVENTS/postgres-inbox-worker" {
		t.Fatalf("consumer identity = %q", consumerIdentity)
	}

	type delivery struct {
		messageID     string
		publicationID string
		delivered     uint64
		applied       bool
	}
	redeliveries := make(chan delivery, 2)
	redrives := make(chan delivery, 1)
	var redeliveryCalls atomic.Int32
	const (
		redeliveryMessage     = "logical-redelivery"
		redeliveryPublication = "publication-redelivery"
		redriveMessage        = "logical-redrive"
		originalPublication   = "publication-before-dlq"
		redrivePublication    = "publication-after-dlq"
	)
	client, _, _ := fixture.worker(t, func(handlerCtx context.Context, message natsjs.Message) error {
		switch message.MessageID() {
		case redeliveryMessage:
			applied, err := applyInbox(handlerCtx, pool, consumerIdentity, message.MessageID(),
				recordInboxEffect(handlerCtx, consumerIdentity, message.MessageID()))
			if err != nil {
				return err
			}
			redeliveries <- delivery{
				messageID: message.MessageID(), publicationID: message.PublicationID(),
				delivered: message.Metadata().NumDelivered, applied: applied,
			}
			if redeliveryCalls.Add(1) == 1 {
				return errors.New("simulate lost handler completion")
			}
			return nil
		case redriveMessage:
			if message.PublicationID() == originalPublication {
				return natsjs.Permanent(errors.New("send original delivery to DLQ"))
			}
			applied, err := applyInbox(handlerCtx, pool, consumerIdentity, message.MessageID(),
				recordInboxEffect(handlerCtx, consumerIdentity, message.MessageID()))
			if err != nil {
				return err
			}
			redrives <- delivery{
				messageID: message.MessageID(), publicationID: message.PublicationID(),
				delivered: message.Metadata().NumDelivered, applied: applied,
			}
			return nil
		default:
			return fmt.Errorf("unexpected logical message %q", message.MessageID())
		}
	}, func(cfg *natsjs.WorkerConfig) {
		cfg.Consumer = consumer
		cfg.RetryDelays = []time.Duration{50 * time.Millisecond}
	})

	redeliveryEvent := testEvent("redelivery")
	redeliveryEvent.MessageID = redeliveryMessage
	redeliveryEvent.PublicationID = redeliveryPublication
	if _, err := client.Producer().Publish(ctx, redeliveryEvent); err != nil {
		t.Fatalf("publish redelivery event: %v", err)
	}
	first := waittest.Receive(t, redeliveries, 5*time.Second, "first inbox delivery")
	second := waittest.Receive(t, redeliveries, 10*time.Second, "inbox redelivery")
	if !first.applied || second.applied || first.delivered != 1 || second.delivered != 2 ||
		first.messageID != redeliveryMessage || second.messageID != redeliveryMessage ||
		first.publicationID != redeliveryPublication || second.publicationID != redeliveryPublication {
		t.Fatalf("ordinary deliveries = first %+v second %+v", first, second)
	}
	waitConsumerSettled(t, fixture, consumer)
	assertInboxIdentityCount(t, ctx, pool, consumerIdentity, redeliveryMessage, 1)

	renamedConsumer := sourceStream + "/postgres-inbox-worker-renamed"
	applied, err := applyInbox(ctx, pool, renamedConsumer, redeliveryMessage,
		recordInboxEffect(ctx, renamedConsumer, redeliveryMessage))
	if err != nil || !applied {
		t.Fatalf("renamed consumer apply = %t, %v", applied, err)
	}
	assertInboxIdentityCount(t, ctx, pool, renamedConsumer, redeliveryMessage, 1)

	original := testEvent("dead letter then redrive")
	original.MessageID = redriveMessage
	original.PublicationID = originalPublication
	if _, err := client.Producer().Publish(ctx, original); err != nil {
		t.Fatalf("publish original DLQ event: %v", err)
	}
	var deadLetter *jetstream.RawStreamMsg
	waittest.Until(t, 5*time.Second, func() bool {
		stream, streamErr := fixture.js.Stream(ctx, deadLetterStream)
		if streamErr != nil {
			return false
		}
		deadLetter, streamErr = stream.GetLastMsgForSubject(ctx, deadLetterSubject)
		return streamErr == nil
	}, "inbox source dead-letter transfer")
	if deadLetter.Header.Get("Message-Id") != redriveMessage ||
		deadLetter.Header.Get("Original-Publication-Id") != originalPublication {
		t.Fatalf("dead-letter identity = message %q publication %q",
			deadLetter.Header.Get("Message-Id"), deadLetter.Header.Get("Original-Publication-Id"))
	}
	seeded, err := applyInbox(ctx, pool, consumerIdentity, redriveMessage,
		recordInboxEffect(ctx, consumerIdentity, redriveMessage))
	if err != nil || !seeded {
		t.Fatalf("seed committed redrive claim = %t, %v", seeded, err)
	}

	redrive := original
	redrive.PublicationID = redrivePublication
	redrive.Subject = deadLetter.Header.Get("Original-Subject")
	if _, err := client.Producer().Publish(ctx, redrive); err != nil {
		t.Fatalf("publish redrive: %v", err)
	}
	redriven := waittest.Receive(t, redrives, 5*time.Second, "inbox redrive")
	if redriven.messageID != redriveMessage || redriven.publicationID != redrivePublication || redriven.applied {
		t.Fatalf("redriven delivery = %+v", redriven)
	}
	waitConsumerSettled(t, fixture, consumer)
	assertInboxIdentityCount(t, ctx, pool, consumerIdentity, redriveMessage, 1)
	assertStreamMessages(t, fixture, deadLetterStream, 1)

	var effects int
	if err := pool.PGX().QueryRow(ctx, `
		SELECT count(*) FROM inbox_test_effects
		WHERE consumer_identity = $1 AND message_id IN ($2, $3)
	`, consumerIdentity, redeliveryMessage, redriveMessage).Scan(&effects); err != nil {
		t.Fatalf("count inbox NATS effects: %v", err)
	}
	if effects != 2 {
		t.Fatalf("inbox NATS effects = %d, want one per logical message", effects)
	}

	t.Run("forced shutdown rolls back before redelivery", func(t *testing.T) {
		t.Parallel()
		const (
			shutdownConsumer    = "postgres-inbox-shutdown-worker"
			shutdownMessage     = "logical-shutdown"
			shutdownPublication = "publication-shutdown"
			shutdownSubject     = "events.inbox-shutdown"
		)
		shutdownIdentity := sourceStream + "/" + shutdownConsumer
		entered := make(chan struct{}, 1)
		shutdownClient, shutdownWorker, runErr := fixture.worker(t, func(handlerCtx context.Context, message natsjs.Message) error {
			_, err := applyInbox(handlerCtx, pool, shutdownIdentity, message.MessageID(), func(pgx.Tx) error {
				entered <- struct{}{}
				<-handlerCtx.Done()
				return handlerCtx.Err()
			})
			return err
		}, func(cfg *natsjs.WorkerConfig) {
			cfg.Consumer = shutdownConsumer
			cfg.FilterSubject = shutdownSubject
			cfg.HandlerTimeout = 100 * time.Millisecond
			cfg.RetryDelays = []time.Duration{50 * time.Millisecond}
		})

		shutdownEvent := testEvent("forced shutdown")
		shutdownEvent.MessageID = shutdownMessage
		shutdownEvent.PublicationID = shutdownPublication
		shutdownEvent.Subject = shutdownSubject
		if _, err := shutdownClient.Producer().Publish(ctx, shutdownEvent); err != nil {
			t.Fatalf("publish shutdown event: %v", err)
		}
		waittest.ReceiveSignal(t, entered, 5*time.Second, "claimed inbox handler")
		shutdownCtx, cancelShutdown := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancelShutdown()
		if err := shutdownWorker.Shutdown(shutdownCtx); err == nil {
			t.Fatal("Shutdown() error = nil, want forced shutdown")
		}
		_ = waittest.Receive(t, runErr, 5*time.Second, "forced worker join")
		assertInboxIdentityCount(t, ctx, pool, shutdownIdentity, shutdownMessage, 0)
		waittest.Until(t, 5*time.Second, func() bool {
			consumer, err := fixture.js.Consumer(ctx, sourceStream, shutdownConsumer)
			if err != nil {
				return false
			}
			info, err := consumer.Info(ctx)
			return err == nil && info.NumAckPending == 1
		}, "unacknowledged shutdown delivery")

		redelivered := make(chan delivery, 1)
		_, _, _ = fixture.worker(t, func(handlerCtx context.Context, message natsjs.Message) error {
			applied, err := applyInbox(handlerCtx, pool, shutdownIdentity, message.MessageID(),
				recordInboxEffect(handlerCtx, shutdownIdentity, message.MessageID()))
			if err != nil {
				return err
			}
			redelivered <- delivery{
				messageID: message.MessageID(), publicationID: message.PublicationID(),
				delivered: message.Metadata().NumDelivered, applied: applied,
			}
			return nil
		}, func(cfg *natsjs.WorkerConfig) {
			cfg.Consumer = shutdownConsumer
			cfg.FilterSubject = shutdownSubject
			cfg.HandlerTimeout = 100 * time.Millisecond
			cfg.RetryDelays = []time.Duration{50 * time.Millisecond}
		})
		got := waittest.Receive(t, redelivered, 15*time.Second, "redelivery after forced inbox shutdown")
		if !got.applied || got.messageID != shutdownMessage || got.publicationID != shutdownPublication || got.delivered < 2 {
			t.Fatalf("shutdown redelivery = %+v", got)
		}
		waitConsumerSettled(t, fixture, shutdownConsumer)
		assertInboxIdentityCount(t, ctx, pool, shutdownIdentity, shutdownMessage, 1)
	})
}
