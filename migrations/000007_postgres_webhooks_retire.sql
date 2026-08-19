-- +goose Up

-- Preserve the published schema for rollback while removing every live
-- webhook relation name from the runtime contract. A later migration may drop
-- these deprecated relations after the previous webhook worker leaves the
-- rollback window.
ALTER SEQUENCE webhook_fairness_sequence RENAME TO deprecated_webhook_fairness_sequence;
ALTER FUNCTION webhook_delivery_policy_valid(jsonb) RENAME TO deprecated_webhook_delivery_policy_valid;

ALTER TABLE webhook_clock RENAME TO deprecated_webhook_clock;
ALTER TABLE webhook_destinations RENAME TO deprecated_webhook_destinations;
ALTER TABLE webhook_events RENAME TO deprecated_webhook_events;
ALTER TABLE webhook_fanouts RENAME TO deprecated_webhook_fanouts;
ALTER TABLE webhook_deliveries RENAME TO deprecated_webhook_deliveries;
ALTER TABLE webhook_cycles RENAME TO deprecated_webhook_cycles;
ALTER TABLE webhook_attempts RENAME TO deprecated_webhook_attempts;
ALTER TABLE webhook_capacity_slots RENAME TO deprecated_webhook_capacity_slots;
ALTER TABLE webhook_operator_actions RENAME TO deprecated_webhook_operator_actions;
ALTER TABLE webhook_tombstones RENAME TO deprecated_webhook_tombstones;
ALTER TABLE webhook_destination_tombstones RENAME TO deprecated_webhook_destination_tombstones;

-- +goose Down

ALTER TABLE deprecated_webhook_destination_tombstones RENAME TO webhook_destination_tombstones;
ALTER TABLE deprecated_webhook_tombstones RENAME TO webhook_tombstones;
ALTER TABLE deprecated_webhook_operator_actions RENAME TO webhook_operator_actions;
ALTER TABLE deprecated_webhook_capacity_slots RENAME TO webhook_capacity_slots;
ALTER TABLE deprecated_webhook_attempts RENAME TO webhook_attempts;
ALTER TABLE deprecated_webhook_cycles RENAME TO webhook_cycles;
ALTER TABLE deprecated_webhook_deliveries RENAME TO webhook_deliveries;
ALTER TABLE deprecated_webhook_fanouts RENAME TO webhook_fanouts;
ALTER TABLE deprecated_webhook_events RENAME TO webhook_events;
ALTER TABLE deprecated_webhook_destinations RENAME TO webhook_destinations;
ALTER TABLE deprecated_webhook_clock RENAME TO webhook_clock;

ALTER FUNCTION deprecated_webhook_delivery_policy_valid(jsonb) RENAME TO webhook_delivery_policy_valid;
ALTER SEQUENCE deprecated_webhook_fairness_sequence RENAME TO webhook_fairness_sequence;
