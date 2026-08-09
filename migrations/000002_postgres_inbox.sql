-- +goose Up

CREATE TABLE postgres_inbox_claims (
    consumer_identity text COLLATE "C" NOT NULL,
    message_id text COLLATE "C" NOT NULL,
    PRIMARY KEY (consumer_identity, message_id),
    CONSTRAINT postgres_inbox_claims_consumer_check CHECK (
        octet_length(consumer_identity) BETWEEN 1 AND 1024
        AND consumer_identity !~ '[[:cntrl:]]'
    ),
    CONSTRAINT postgres_inbox_claims_message_check CHECK (
        octet_length(message_id) BETWEEN 1 AND 256
        AND message_id !~ '[[:cntrl:]]'
    )
);

-- +goose Down

DROP TABLE postgres_inbox_claims;
