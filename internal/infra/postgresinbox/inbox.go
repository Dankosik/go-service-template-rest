// Package postgresinbox atomically claims one logical message for one consumer
// inside the caller's PostgreSQL transaction.
//
// A true result means the caller may apply its database effect in that same
// transaction. False means an earlier delivery already committed the claim, so
// the caller skips the effect and acknowledges the delivery. Effects outside
// this transaction are not made idempotent; route those through a downstream
// idempotency key or the transactional outbox.
package postgresinbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

const (
	maxConsumerIdentityBytes = 1024
	maxMessageIDBytes        = 256
)

var errInvalidInput = errors.New("invalid inbox claim")

// Claim records consumerIdentity and messageID in tx. It never starts or
// commits a transaction: the caller owns the transaction containing both this
// claim and the guarded effect.
func Claim(ctx context.Context, tx pgx.Tx, consumerIdentity, messageID string) (bool, error) {
	if tx == nil {
		return false, fmt.Errorf("%w: transaction is required", errInvalidInput)
	}
	if err := validateIdentity("consumer identity", consumerIdentity, maxConsumerIdentityBytes); err != nil {
		return false, err
	}
	if err := validateIdentity("message id", messageID, maxMessageIDBytes); err != nil {
		return false, err
	}

	rows, err := sqlcgen.New(tx).ClaimInboxMessage(ctx, sqlcgen.ClaimInboxMessageParams{
		ConsumerIdentity: consumerIdentity,
		MessageID:        messageID,
	})
	if err != nil {
		return false, fmt.Errorf("claim inbox message: %w", err)
	}
	if rows > 1 {
		return false, fmt.Errorf("claim inbox message: affected %d rows", rows)
	}
	return rows == 1, nil
}

func validateIdentity(name, value string, limit int) error {
	if value == "" || len(value) > limit || !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s must be valid UTF-8 between 1 and %d bytes", errInvalidInput, name, limit)
	}
	if strings.ContainsFunc(value, unicode.IsControl) {
		return fmt.Errorf("%w: %s contains a control character", errInvalidInput, name)
	}
	return nil
}
