-- name: ClaimInboxMessage :execrows
INSERT INTO postgres_inbox_claims (consumer_identity, message_id)
VALUES (sqlc.arg(consumer_identity), sqlc.arg(message_id))
ON CONFLICT (consumer_identity, message_id) DO NOTHING;
